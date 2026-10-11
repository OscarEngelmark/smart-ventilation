#!/bin/bash
# Recovery check after a Message broker outage (project_notes.md §10): on a
# session at speed 1, stops the broker for 7 minutes, past the Decision
# service's 5 minutes without a reading, then starts it again:
# ./test/fault/reconnect.sh
# It judges nothing itself: the Decision service's log, saved with the data
# stored in the session in test/results/reconnect_<date>/, shows the recovery.
source "$(dirname "$0")/lib.sh"

OUTAGE_S=420 # real seconds the broker is stopped, at speed 1 also room seconds

snapshot=reconnect_before
./eval/snapshot.sh save "$snapshot" # also stops every container
trap restore EXIT # runs when the script ends, passed or failed
docker compose build --quiet # the stack is stopped, so no state is lost
out=test/results/reconnect_$(date +%F)
mkdir -p "$out"
exec > >(tee "$out/test.txt") 2>&1 # everything printed from here on is also saved

./start.sh 1
echo "waiting for the first decision"
until docker compose logs decision | grep -q ': damper '; do sleep 2; done
sleep 30
docker compose stop mosquitto
echo "broker stopped at $(date +%T)"
sleep "$OUTAGE_S"
docker compose start mosquitto
echo "broker started at $(date +%T)"
sleep 60

save_stored
echo "--- the Decision service's log over the session"
docker compose logs --no-color --no-log-prefix --since "$(setting SIM_SESSION_START)" decision
