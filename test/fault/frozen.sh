#!/bin/bash
# Fault test of NFR-3 with a frozen CO2 sensor (project_notes.md §10): runs one
# weekday with the CO2 sensor stuck at its first reading, and checks that the
# room's real CO2 stays under 1000 ppm: ./test/fault/frozen.sh
# Its output, the day's stored data and the room's real CO2 are saved in
# test/results/frozen_<room date>/.
source "$(dirname "$0")/lib.sh"

begin_test frozen
echo "test day $day, frozen CO2 sensor, speed $SPEED"
FREEZE_CO2=true ./start.sh "$SPEED" "$(session_start "$day")"
wait_for_day_end
docker compose logs --no-color --no-log-prefix --since "$(setting SIM_SESSION_START)" co2-sensor | grep 'frozen at'

./eval/save_day.sh "$day" "frozen_$day/day"
echo "stored CO2 readings over the day: $(jq -c '[.[].ppm] | unique' "$out/day_co2.json") ppm"
judge
