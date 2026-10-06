#!/bin/bash
# Crash-and-restart test of NFR-1 (project_notes.md §3): kills the CO2 sensor and
# the decision service, writes a CO2 level above the target into BuildSim while
# both are down, and measures the real time until the damper is open in BuildSim.
# Run it with the stack started at speed 1 (./start.sh 1), where room time is real
# time: ./test/crash/crash.sh
# It needs sudo, since Docker restarts a container only when its process dies on
# its own, not when Docker is told to stop it. Its output, the stored data of the
# run and the two services' logs are saved in test/results/crash_<date>*.
set -euo pipefail
export LC_ALL=C # numbers with a decimal point, whatever the machine's locale

BUILDSIM_URL=http://localhost:9090
STORAGE_URL=http://localhost:8081
ROOM=level0/A125
CO2_SENSOR=A125-co2
DAMPER=A125-damper
INJECTED_PPM=980 # above the plan's target, below the 1000 ppm threshold
LIMIT_S=120      # NFR-1: the loop resumes within 2 minutes
TIMEOUT_S=300    # real seconds to wait, past the limit so a slow recovery is still measured
SERVICES="co2-sensor decision"

cd "$(dirname "$0")/../.."
out=test/results/crash_$(date +%F)
exec > >(tee "$out.txt") 2>&1 # everything printed from here on is also saved

# history prints the room's stored history of one kind (co2, occupancy, commands) from a room time on.
history() {
  curl -sfG "$STORAGE_URL/$1" --data-urlencode "room=$ROOM" --data-urlencode "since=$2"
}

# container prints the ID of a service's container.
container() {
  docker compose ps -q "$1"
}

# restarts prints how many times Docker has restarted a service's container.
restarts() {
  docker inspect -f '{{.RestartCount}}' "$(container "$1")"
}

# pid prints the host's process ID of a service's main process.
pid() {
  docker inspect -f '{{.State.Pid}}' "$(container "$1")"
}

# damper_state prints the damper's level as BuildSim holds it, 0 to 1.
damper_state() {
  curl -sf "$BUILDSIM_URL/api/actuators/$DAMPER" | jq -r .state
}

# damper_open succeeds when BuildSim holds the damper fully open.
damper_open() {
  [ "$(jq -n "$(damper_state) >= 1")" = true ]
}

# open_command_stored succeeds when a command to open the damper fully, newer than $since, is stored.
open_command_stored() {
  history commands "$since" |
    jq -e --arg s "$since" 'any(.[]; .ts > $s and .level == 1)' > /dev/null
}

# seconds_since_kill prints the real seconds since the kill.
seconds_since_kill() {
  jq -n "$(date +%s.%N) - $kill_real"
}

# wait_for runs a check every half second until it succeeds, and stops the test
# if that hasn't happened within TIMEOUT_S of the kill. $1 names the awaited event.
wait_for() {
  until "$2"; do
    if [ "$(jq -n "$(seconds_since_kill) > $TIMEOUT_S")" = true ]; then
      echo "FAIL: no $1 within $TIMEOUT_S real seconds of the kill"
      exit 1
    fi
    sleep 0.5
  done
}

speed=$(sed -n 's/^SIM_SPEED=//p' run.env)
if [ "$speed" != 1 ]; then
  echo "the session runs at speed $speed; start it at speed 1 (./start.sh 1) so room time is real time"
  exit 1
fi
if damper_open; then
  echo "the damper is already fully open, so no change can be seen; run again later"
  exit 0
fi

sudo -v # asks for the password now, so the typing isn't timed

since=$(curl -sf "$STORAGE_URL/latest") # newest stored room time before the kill
echo "speed $speed; damper at $(damper_state); newest stored room time $since"
declare -A restarts_before
for service in $SERVICES; do
  restarts_before[$service]=$(restarts "$service")
done

sensor_pid=$(pid co2-sensor)
decision_pid=$(pid decision)
kill_real=$(date +%s.%N)
sudo kill -9 "$sensor_pid" "$decision_pid"
curl -sf -X PUT "$BUILDSIM_URL/api/sensors/$CO2_SENSOR/value" \
  -H 'Content-Type: application/json' \
  -d "{\"data_type\":\"text\",\"value\":\"$INJECTED_PPM.0\"}" > /dev/null
echo "killed co2-sensor and decision; wrote $INJECTED_PPM ppm to BuildSim"

wait_for "fully open damper in BuildSim" damper_open
recovery_s=$(seconds_since_kill)
printf 'damper fully open in BuildSim %.1f real s after the kill\n' "$recovery_s"
wait_for "stored command to open the damper fully" open_command_stored

first_reading=$(history co2 "$since" |
  jq -r --arg s "$since" '[.[] | select(.ts > $s)][0] | "\(.ppm) ppm at \(.ts)"')
echo "first stored reading after the kill: $first_reading"
restarted=true
for service in $SERVICES; do
  after=$(restarts "$service")
  echo "$service restarted by Docker: ${restarts_before[$service]} -> $after"
  if [ "$after" -ne $((${restarts_before[$service]} + 1)) ]; then
    restarted=false
  fi
done

for kind in co2 commands occupancy; do
  history "$kind" "$since" | jq . > "${out}_$kind.json"
done
logs_from=$((${kill_real%.*} - 5)) # a few seconds before the kill, to show the old process's last lines
docker compose logs --no-color --timestamps --since "$logs_from" $SERVICES > "${out}_logs.txt"

if [ "$restarted" != true ]; then
  echo "FAIL: Docker did not restart each service exactly once"
  exit 1
fi
if [ "$(jq -n "$recovery_s > $LIMIT_S")" = true ]; then
  printf 'FAIL: the loop took %.1f real s, over %s\n' "$recovery_s" "$LIMIT_S"
  exit 1
fi
printf 'PASS: the loop resumed in %.1f real s, within %s\n' "$recovery_s" "$LIMIT_S"
