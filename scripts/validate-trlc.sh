#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/validate-trlc.sh <trlc-dir>

Example:
  scripts/validate-trlc.sh examples/payments-engineering-sample/generated/trlc
EOF
}

if [[ $# -ne 1 ]]; then
  usage
  exit 2
fi

DIR="$1"
if [[ ! -d "$DIR" ]]; then
  printf "TRLC directory not found: %s\n" "$DIR" >&2
  exit 1
fi

if ! command -v trlc >/dev/null 2>&1; then
  printf "trlc binary not found on PATH. Install with: python3 -m pip install --user trlc\n" >&2
  exit 1
fi

trlc "$DIR"
printf "TRLC files are valid: %s\n" "$DIR"
