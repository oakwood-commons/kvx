#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="$ROOT/tests/snapshots"
TEST_FILE="$ROOT/tests/sample.yaml"
GOCACHE_DIR="${GOCACHE:-$ROOT/.cache/go-build}"

mkdir -p "$OUT_DIR"

run_snapshot() {
  local out="$1"
  shift
  GOCACHE="$GOCACHE_DIR" go run . "$TEST_FILE" \
    --snapshot \
    --no-color \
    --width 80 \
    --height 30 \
    "$@" > "$OUT_DIR/$out"
}

run_snapshot default-80x30.txt
run_snapshot f1-80x30.txt --press "<F1>"
