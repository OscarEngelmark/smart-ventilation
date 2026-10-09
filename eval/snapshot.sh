#!/bin/bash
# Saves or restores the History store, so that one stretch of room days can be
# run twice from the same history: ./eval/snapshot.sh save|restore <name>
# Both stop every container first, since the Storage service must not write
# while its files are copied. A snapshot is a tar of the storage-data volume in
# eval/snapshots/<name>.tar. Reasoning: project_notes.md §11.
set -euo pipefail

VOLUME=smart-ventilation_storage-data # the storage-data volume, as Compose names it
SNAPSHOT_DIR=eval/snapshots

# save writes every file in the volume into the tar $1.
save() {
  docker run --rm -v "$VOLUME:/data:ro" alpine tar -C /data -cf - . > "$1"
}

# restore empties the volume and fills it from the tar $1.
restore() {
  docker run --rm -i -v "$VOLUME:/data" alpine \
    sh -c 'find /data -mindepth 1 -delete && tar -C /data -xf -' < "$1"
}

if [ $# -ne 2 ] || { [ "$1" != save ] && [ "$1" != restore ]; }; then
  echo "usage: $0 save|restore <name>" >&2
  exit 1
fi
cd "$(dirname "$0")/.."
mkdir -p "$SNAPSHOT_DIR"
tar_path="$SNAPSHOT_DIR/$2.tar"

docker compose stop
if [ "$1" = save ]; then
  save "$tar_path"
  echo "saved the History store to $tar_path"
else
  if [ ! -f "$tar_path" ]; then
    echo "no snapshot at $tar_path" >&2
    exit 1
  fi
  restore "$tar_path"
  echo "restored the History store from $tar_path"
fi
