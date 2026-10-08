#!/bin/sh
# Starts a session: ./start.sh <speed> [room time], e.g. ./start.sh 10.
# The session continues room time where the stored history ends, or at ORIGIN
# when nothing later is stored. A room time given as the second argument, such
# as 2026-11-23T12:35, starts the session later instead, at most FIT_DAYS after
# the history ends. Reasoning: project_notes.md §6.
set -eu

ORIGIN=2026-09-25T09:00:00+02:00 # room time the first session starts at
STORAGE_URL=http://localhost:8081

# setting prints the value sim.env gives for the name $1.
setting() {
  sed -n "s/^$1=//p" sim.env
}

# check_start fails unless room time $2 lies after the history's end $1 and
# at most fit_days after it, so the room model still has history to fit.
check_start() {
  end=$(date -d "$1" +%s)
  start=$(date -d "$2" +%s)
  if [ "$start" -le "$end" ]; then
    echo "$2 is not after the stored history, which ends at $1" >&2
    exit 1
  fi
  if [ "$start" -gt $((end + fit_days * 86400)) ]; then
    echo "$2 is more than $fit_days days after the stored history, which ends at $1" >&2
    exit 1
  fi
}

if [ $# -lt 1 ] || [ $# -gt 2 ]; then
  echo "usage: $0 <speed> [room time], e.g. $0 10 or $0 10 2026-11-23T12:35" >&2
  exit 1
fi
speed=$1
fit_days=$(setting FIT_DAYS)
if [ $# -eq 2 ]; then
  # A room time given without an offset is read in the room's zone.
  requested=$(TZ=$(setting TZ) date -d "$2" --iso-8601=seconds)
fi

# Started on its own first, since it holds the history the session continues from.
docker compose up -d storage-service

tries=0
until latest=$(curl -sf "$STORAGE_URL/latest"); do
  tries=$((tries + 1))
  if [ "$tries" -ge 30 ]; then
    echo "storage-service did not answer at $STORAGE_URL" >&2
    exit 1
  fi
  sleep 1
done

room_start=$ORIGIN
if [ -n "$latest" ] && [ "$(date -d "$latest" +%s)" -gt "$(date -d "$ORIGIN" +%s)" ]; then
  room_start=$latest
fi
if [ $# -eq 2 ]; then
  check_start "$room_start" "$requested"
  room_start=$requested
fi
session_start=$(date -u +%Y-%m-%dT%H:%M:%SZ)

cat > run.env <<EOF
SIM_SESSION_START=$session_start
SIM_ROOM_START=$room_start
SIM_SPEED=$speed
EOF

# No --build: rebuilding would also restart BuildSim and lose the room's state.
docker compose up -d
echo "session started at room time $room_start, speed $speed"
