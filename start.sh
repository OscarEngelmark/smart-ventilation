#!/bin/sh
# Starts a session: ./start.sh <speed>, e.g. ./start.sh 10.
# The session continues room time where the stored history ends, or at ORIGIN
# when nothing later is stored. Reasoning: project_notes.md §6.
set -eu

ORIGIN=2026-09-25T09:00:00+02:00 # room time the first session starts at
STORAGE_URL=http://localhost:8081

if [ $# -ne 1 ]; then
  echo "usage: $0 <speed>, e.g. $0 10" >&2
  exit 1
fi
speed=$1

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
session_start=$(date -u +%Y-%m-%dT%H:%M:%SZ)

cat > run.env <<EOF
SIM_SESSION_START=$session_start
SIM_ROOM_START=$room_start
SIM_SPEED=$speed
EOF

# No --build: rebuilding would also restart BuildSim and lose the room's state.
docker compose up -d
echo "session started at room time $room_start, speed $speed"
