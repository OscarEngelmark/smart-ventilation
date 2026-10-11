#!/bin/bash
# Fault test of NFR-3 with the Message broker stopped (project_notes.md §10): runs
# one weekday with the broker stopped from before anyone arrives until the day
# ends, and checks that the room's real CO2 stays under 1000 ppm:
# ./test/fault/broker.sh
# Its output, the data stored since the session started and the room's real CO2
# are saved in test/results/broker_<room date>/.
source "$(dirname "$0")/lib.sh"

DAMPER=A125-damper
BUILDSIM_URL=http://localhost:9090

# decisions prints how many decisions the Decision service has logged since the real time $1.
decisions() {
  docker compose logs --no-color --no-log-prefix --since "$1" decision | grep -c ': damper ' || true
}

# wait_for_first_decision waits until the Decision service has decided on a reading.
wait_for_first_decision() {
  local waited=0
  until [ "$(decisions "$(setting SIM_SESSION_START)")" -gt 0 ]; do
    if [ "$waited" -ge 120 ]; then
      echo "FAIL: no decision within 120 real seconds of the start"
      exit 1
    fi
    sleep 1
    waited=$((waited + 1))
  done
}

begin_test broker
echo "test day $day, Message broker stopped, speed $SPEED"
./start.sh "$SPEED" "$(session_start "$day")"
wait_for_first_decision
docker compose stop mosquitto
stop_real=$(date -u +%Y-%m-%dT%H:%M:%S.%NZ)
stop_room=$(room_now)
damper=$(curl -sf "$BUILDSIM_URL/api/actuators/$DAMPER" | jq -r .state)
echo "stopped the broker at room time $(clock "@$stop_room"), with the damper at $damper"
wait_for_day_end

save_stored
echo "decisions logged after the stop: $(decisions "$stop_real")"
echo "CO2 readings stored after the stop: $(jq --argjson t "$stop_room" '[.[] | select((.ts | fromdateiso8601) > $t)] | length' "$out/stored_co2.json")"
judge
