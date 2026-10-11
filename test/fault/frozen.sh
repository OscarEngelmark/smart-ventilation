#!/bin/bash
# Fault test of NFR-3 with a frozen CO2 sensor (project_notes.md §10): runs one
# weekday at speed 250 with the CO2 sensor stuck at its first reading, and checks
# that the room's real CO2, read from the physical model's log, stays under
# 1000 ppm. The History store is saved first and restored at the end, so the
# frozen readings never reach later room-model fits: ./test/fault/frozen.sh
# Its output, the day's stored data and the room's real CO2 are saved in
# test/results/frozen_<room date>/.
set -euo pipefail
export LC_ALL=C # numbers with a decimal point, whatever the machine's locale

STORAGE_URL=http://localhost:8081
SPEED=250           # the evaluation speed, at which every reading kept its 10 s spacing
THRESHOLD_PPM=1000  # NFR-3
SNAPSHOT=frozen_before
TIMEOUT_S=1200      # real seconds to wait for the day to end; it takes about 6

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

# session_start prints the room time a session starts at to cover the day $1: 23:00 the evening before.
session_start() {
  echo "$(date -d "$1 - 1 day" +%F)T23:00"
}

# restore puts the History store back as it was before the test, and stops every container.
restore() {
  ./eval/snapshot.sh restore "$SNAPSHOT"
}

# wait_for_day_end waits until a stored room time lies past the end of the test day.
wait_for_day_end() {
  local waited=0
  until [ "$(epoch "$(curl -sf "$STORAGE_URL/latest")")" -ge "$day_end" ]; do
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

./eval/snapshot.sh save "$SNAPSHOT" # also stops every container
trap restore EXIT # runs when the script ends, passed or failed
docker compose build --quiet # the stack is stopped, so no state is lost

docker compose up -d storage-service
day=$(test_day "$(history_end)")
day_start=$(epoch "$day 00:00")
day_end=$(epoch "$day 00:00 + 1 day")
out=test/results/frozen_$day
mkdir -p "$out"
exec > >(tee "$out/test.txt") 2>&1 # everything printed from here on is also saved

echo "test day $day, frozen CO2 sensor, speed $SPEED"
FREEZE_CO2=true ./start.sh "$SPEED" "$(session_start "$day")"
wait_for_day_end
docker compose logs --no-color --no-log-prefix --since "$(setting SIM_SESSION_START)" co2-sensor | grep 'frozen at'

./eval/save_day.sh "$day" "frozen_$day/day"
echo "stored CO2 readings over the day: $(jq -c '[.[].ppm] | unique' "$out/day_co2.json") ppm"
real_co2 > "$out/real_co2.json"
steps=$(jq length "$out/real_co2.json")
peak=$(jq 'max_by(.ppm)' "$out/real_co2.json")
peak_ppm=$(jq .ppm <<< "$peak")
peak_time=$(TZ=$tz date -d "$(jq -r .ts <<< "$peak")" +%H:%M)
over_min=$(jq --argjson t "$THRESHOLD_PPM" '[.[] | select(.ppm >= $t)] | length / 6' "$out/real_co2.json")
echo "real CO2 over the day: $steps steps, peak $peak_ppm ppm at $peak_time, $over_min room minutes at or above $THRESHOLD_PPM ppm"

if [ "$peak_ppm" -ge "$THRESHOLD_PPM" ]; then
  echo "FAIL: the room's real CO2 reached $peak_ppm ppm"
  exit 1
fi
echo "PASS: the room's real CO2 stayed under $THRESHOLD_PPM ppm, peaking at $peak_ppm ppm"
