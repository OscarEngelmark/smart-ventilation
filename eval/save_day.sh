#!/bin/bash
# Saves one room day as evaluation evidence: ./eval/save_day.sh <date> <name>,
# e.g. ./eval/save_day.sh 2026-11-23 comparison_week_2026-11-23/day_model_2026-11-23,
# where the folder in <name> groups the days of one run. Writes the day's
# summary (eval/day_summary.py) to test/results/<name>.txt, and its stored CO2
# readings, head counts and commands, midnight to midnight in the room's time
# zone, to test/results/<name>_{co2,occupancy,commands}.json. The commands are
# led by the one in force at midnight. Then checks that the day is complete:
# 8640 CO2 readings, each 9-11 s after the previous one, and 1440 head counts.
# Needs the Storage service running.
set -euo pipefail

STORAGE_URL=http://localhost:8081
ROOM=level0/A125

if [ $# -ne 2 ]; then
  echo "usage: $0 <date> <name>, e.g. $0 2026-11-23 comparison_week_2026-11-23/day_model_2026-11-23" >&2
  exit 1
fi
day=$1
cd "$(dirname "$0")/.."
out=test/results/$2
mkdir -p "$(dirname "$out")"
tz=$(sed -n 's/^TZ=//p' sim.env)

# utc prints a room-time moment, given in the room's time zone, in UTC as stored.
utc() {
  date -u -d "$(TZ=$tz date -d "$1" --iso-8601=seconds)" +%Y-%m-%dT%H:%M:%SZ
}

# check_day prints whether the saved day holds every reading and head count.
check_day() {
  local readings counts gaps shortest longest result
  readings=$(jq length "${out}_co2.json")
  counts=$(jq length "${out}_occupancy.json")
  # seconds between each stored reading and the one before it
  gaps=$(jq '[.[].ts | fromdateiso8601] | [.[1:], .[:-1]] | transpose | map(.[0] - .[1])' "${out}_co2.json")
  shortest=$(jq min <<< "$gaps")
  longest=$(jq max <<< "$gaps")
  result=INCOMPLETE
  if [ "$readings" = 8640 ] && [ "$counts" = 1440 ] && [ "$shortest" -ge 9 ] && [ "$longest" -le 11 ]; then
    result=complete
  fi
  echo "check: $result, $readings readings $shortest-$longest s apart, $counts head counts"
}

start=$(utc "$day 00:00")
end=$(utc "$day 00:00 + 1 day")

python3 eval/day_summary.py "$day" > "$out.txt"
cat "$out.txt"
for kind in co2 occupancy commands; do
  curl -sfG "$STORAGE_URL/$kind" --data-urlencode "room=$ROOM" --data-urlencode "since=$start" |
    jq --arg end "$end" '[.[] | select(.ts < $end)]' > "${out}_$kind.json"
  echo "$kind: $(jq length "${out}_$kind.json") rows in ${out}_$kind.json"
done
check_day
