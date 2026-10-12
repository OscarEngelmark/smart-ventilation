#!/bin/bash
# Runs the comparison week end to end, unattended: ./eval/run_comparison.sh
# Restores the History store to the snapshot taken before the week, runs the
# week with the CO2-only switch, restores it again and runs the same week with
# the room model, saving every day of both. Then writes the comparison table,
# the delay from reading to command, and the report's meeting plot data. The
# room model runs last, so the stored history continues with it. Everything
# printed is also saved in run.txt in the run folder. Reasoning:
# project_notes.md §11.
source "$(dirname "$0")/../test/fault/lib.sh"

SNAPSHOT=before   # eval/snapshots/before.tar, the History store before the week
FIRST_DAY=2026-11-23
DAYS=5
PLOT_DAY=2026-11-27 # the day the report's meeting plot shows
WEEK_TIMEOUT_S=3600 # real seconds to wait for the week to end; it takes about 30
RUN=comparison_week_$FIRST_DAY
out=test/results/$RUN

# days prints the run's dates, one per line.
days() {
  local i
  for i in $(seq 0 $((DAYS - 1))); do
    date -d "$FIRST_DAY + $i day" +%F
  done
}

# run_week restores the snapshot, runs the week under the method $1 (model or
# switch) and saves each of its days in the run folder.
run_week() {
  local method=$1 ignore_model=false week_end day
  if [ "$method" = switch ]; then
    ignore_model=true
  fi
  ./eval/snapshot.sh restore "$SNAPSHOT" # also stops every container
  echo "week with the $method, speed $SPEED"
  IGNORE_ROOM_MODEL=$ignore_model ./start.sh "$SPEED" "$(session_start "$FIRST_DAY")"
  week_end=$(epoch "$(days | tail -1) 00:00 + 1 day")
  wait_until $((week_end + MARGIN_S)) "$WEEK_TIMEOUT_S"
  for day in $(days); do
    ./eval/save_day.sh "$day" "$RUN/day_${method}_$day"
  done
}

mkdir -p "$out"
exec > >(tee "$out/run.txt") 2>&1 # everything printed from here on is also saved
docker compose stop
docker compose build --quiet # the stack is stopped, so no state is lost

run_week switch
run_week model
docker compose stop

python3 eval/compare_week.py "$out" "$FIRST_DAY" --days "$DAYS" > "$out/$RUN.txt"
python3 eval/delay.py "$out" "$FIRST_DAY" --days "$DAYS" > "$out/delay.txt"
python3 eval/meeting_series.py "$out" "$PLOT_DAY" "latex/report/data/meeting_$PLOT_DAY.dat"
cat "$out/$RUN.txt" "$out/delay.txt"
