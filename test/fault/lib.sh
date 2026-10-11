# Shared steps of the day-long fault tests (project_notes.md §10), sourced by
# frozen.sh and broker.sh, and by test/stress/stress.sh for history_end and
# restore. Each fault test runs one weekday at speed 250 on a snapshot of the
# History store, restored at the end, and judges NFR-3 by the room's real CO2,
# read from the physical model's log, since the fault keeps the stored readings
# from showing it.
set -euo pipefail
export LC_ALL=C # numbers with a decimal point, whatever the machine's locale

STORAGE_URL=http://localhost:8081
ROOM=level0/A125
SPEED=250          # the evaluation speed, at which every reading kept its 10 s spacing
THRESHOLD_PPM=1000 # NFR-3
TIMEOUT_S=1200     # real seconds to wait for the day to end; it takes about 6
MARGIN_S=600       # room seconds to run past midnight, so the day's last readings are stored

cd "$(dirname "$0")/../.."
tz=$(sed -n 's/^TZ=//p' sim.env)

# setting prints the value run.env gives for the name $1.
setting() {
  sed -n "s/^$1=//p" run.env
}

# epoch prints a moment in seconds since 1970; one without an offset is read in the room's zone.
epoch() {
  TZ=$tz date -d "$1" +%s
}

# clock prints a moment, given as an ISO time or seconds since 1970, as the room's hour and minute.
clock() {
  TZ=$tz date -d "$1" +%H:%M
}

# room_now prints the current room time in seconds since 1970, from the session's clock.
room_now() {
  local real0 room0
  real0=$(date -d "$(setting SIM_SESSION_START)" +%s)
  room0=$(date -d "$(setting SIM_ROOM_START)" +%s)
  jq -n "$room0 + $SPEED * ($(date +%s.%N) - $real0) | floor" # real time to the nanosecond, since each real second is SPEED room seconds
}

# history_end prints the newest stored room time, once the Storage service answers.
history_end() {
  local tries=0
  until curl -sf "$STORAGE_URL/latest"; do
    tries=$((tries + 1))
    if [ "$tries" -ge 30 ]; then
      echo "storage-service did not answer at $STORAGE_URL" >&2
      exit 1
    fi
    sleep 1
  done
}

# session_start prints the room time a session starts at to cover the day $1: 23:00 the evening before.
session_start() {
  echo "$(date -d "$1 - 1 day" +%F)T23:00"
}

# test_day prints the first weekday whose evening before, at 23:00, lies after the room time $1.
test_day() {
  local end day
  end=$(epoch "$1")
  day=$(TZ=$tz date -d "$1" +%F)
  while :; do
    day=$(date -d "$day + 1 day" +%F)
    if [ "$(date -d "$day" +%u)" -le 5 ] && [ "$(epoch "$(session_start "$day")")" -gt "$end" ]; then
      echo "$day"
      return
    fi
  done
}

# restore puts the History store back as it was before the test, and stops every container.
restore() {
  ./eval/snapshot.sh restore "$snapshot"
}

# begin_test saves the History store, picks the test day and its results folder
# results/<$1>_<day>, and from then on also saves everything printed in test.txt there.
begin_test() {
  snapshot=$1_before
  ./eval/snapshot.sh save "$snapshot" # also stops every container
  trap restore EXIT # runs when the script ends, passed or failed
  docker compose build --quiet # the stack is stopped, so no state is lost
  docker compose up -d storage-service
  day=$(test_day "$(history_end)")
  day_start=$(epoch "$day 00:00")
  day_end=$(epoch "$day 00:00 + 1 day")
  out=test/results/$1_$day
  mkdir -p "$out"
  exec > >(tee "$out/test.txt") 2>&1 # everything printed from here on is also saved
}

# wait_for_day_end waits until room time lies MARGIN_S past the end of the test day.
wait_for_day_end() {
  local waited=0
  until [ "$(room_now)" -ge $((day_end + MARGIN_S)) ]; do
    if [ "$waited" -ge "$TIMEOUT_S" ]; then
      echo "FAIL: the day did not end within $TIMEOUT_S real seconds"
      exit 1
    fi
    sleep 5
    waited=$((waited + 5))
  done
}

# real_co2 prints the room's real CO2 over the test day as a JSON array, from the
# physical model's log: one entry per step, its room time computed from the line's real time.
real_co2() {
  docker compose logs --no-color --no-log-prefix --timestamps --since "$(setting SIM_SESSION_START)" physical-model |
    jq -R -n \
      --argjson real0 "$(date -d "$(setting SIM_SESSION_START)" +%s)" \
      --argjson room0 "$(date -d "$(setting SIM_ROOM_START)" +%s)" \
      --argjson speed "$SPEED" --argjson from "$day_start" --argjson to "$day_end" '
      [inputs
       | capture("^(?<s>[^.Z]+)(?<frac>[.][0-9]+)?Z .* (?<N>[0-9]+) people, damper (?<d>[0-9.]+), [0-9]+ -> (?<ppm>[0-9]+) ppm$")
       | ((.s + "Z" | fromdateiso8601) + ("0" + (.frac // ".0") | tonumber)) as $real
       | ($room0 + $speed * ($real - $real0)) as $room
       | select($room >= $from and $room < $to)
       | {ts: ($room | floor | todate), people: (.N | tonumber), damper: (.d | tonumber), ppm: (.ppm | tonumber)}]'
}

# judge saves the room's real CO2 over the test day in real_co2.json, prints when
# it first reached THRESHOLD_PPM after the first person arrived, its peak and the
# time at or above, and exits with a failure if it reached THRESHOLD_PPM.
judge() {
  local series=$out/real_co2.json steps first_in first_over peak peak_ppm over_min
  real_co2 > "$series"
  steps=$(jq length "$series")
  first_in=$(jq -r '[.[] | select(.people > 0)][0].ts' "$series")
  peak=$(jq 'max_by(.ppm)' "$series")
  peak_ppm=$(jq .ppm <<< "$peak")
  over_min=$(jq --argjson t "$THRESHOLD_PPM" '[.[] | select(.ppm >= $t)] | length / 6' "$series")
  echo "real CO2 over the day: $steps steps, peak $peak_ppm ppm at $(clock "$(jq -r .ts <<< "$peak")")," \
    "$over_min room minutes at or above $THRESHOLD_PPM ppm"
  echo "first person in at $(clock "$first_in")"
  if [ "$peak_ppm" -lt "$THRESHOLD_PPM" ]; then
    echo "PASS: the room's real CO2 stayed under $THRESHOLD_PPM ppm, peaking at $peak_ppm ppm"
    return
  fi
  first_over=$(jq -r --argjson t "$THRESHOLD_PPM" '[.[] | select(.ppm >= $t)][0].ts' "$series")
  echo "real CO2 first at or above $THRESHOLD_PPM ppm at $(clock "$first_over")," \
    "$((($(date -d "$first_over" +%s) - $(date -d "$first_in" +%s)) / 60)) room minutes after the first person came in"
  echo "FAIL: the room's real CO2 reached $peak_ppm ppm"
  exit 1
}
