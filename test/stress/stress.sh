#!/bin/bash
# Stress test (project_notes.md §10): publishes CO2 readings for 1000 made-up
# rooms at rising rates, with only the broker and the storage-service running,
# and counts how many were stored at each rate: ./test/stress/stress.sh
# Its output, each container's CPU and memory every 3 s and the broker's
# dropped-message lines are saved in test/results/stress_<date>/.
source "$(dirname "$0")/../fault/lib.sh"

# sample_stats prints each container's CPU and memory every 3 s until stopped.
sample_stats() {
  while :; do
    docker stats --no-stream --format '{{.Name}} {{.CPUPerc}} {{.MemUsage}}' | sed "s/^/$(date +%T) /"
    sleep 3
  done
}

snapshot=stress_before
./eval/snapshot.sh save "$snapshot" # also stops every container
trap 'kill "${sampler:-}" 2>/dev/null; restore' EXIT # runs when the script ends, passed or failed
docker compose build --quiet storage-service
docker compose up -d storage-service # also starts the broker it depends on
start=$(date -u -d "$(history_end) + 1 hour" +%FT%TZ)
out=test/results/stress_$(date +%F)
mkdir -p "$out"
exec > >(tee "$out/test.txt") 2>&1 # everything printed from here on is also saved

sample_stats > "$out/stats.txt" &
sampler=$! # the sampler's process ID, to stop it

echo "first reading at $start"
began=$(date -u +%FT%TZ)
go run ./test/stress -start "$start"
docker compose logs --no-color --since "$began" mosquitto | grep 'dropped' > "$out/broker_dropped.txt" || true
echo "broker log lines on dropped messages: $(wc -l < "$out/broker_dropped.txt")"

echo "one save to the History store, file on disk, then in memory:"
go test -run '^$' -bench SaveCO2Reading -benchtime 2000x ./internal/store | grep Benchmark
TMPDIR=/dev/shm go test -run '^$' -bench SaveCO2Reading -benchtime 2000x ./internal/store | grep Benchmark
