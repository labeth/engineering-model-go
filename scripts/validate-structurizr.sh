#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STRUCTURIZR_IMAGE="docker.io/structurizr/structurizr@sha256:dcacb88f392fb51cb75fdc5beb7584e67e8dd58fb4b52d3a96d05bcb51c6be48"

usage() {
  cat <<'EOF'
Usage:
  scripts/validate-structurizr.sh <dsl-file>

Example:
  scripts/validate-structurizr.sh examples/payments-engineering-sample/generated/STRUCTURIZR.dsl
EOF
}

if [[ $# -ne 1 ]]; then
  usage
  exit 2
fi

DSL_FILE="$1"
if [[ ! -f "${DSL_FILE}" ]]; then
  printf "DSL file not found: %s\n" "${DSL_FILE}" >&2
  exit 1
fi

if command -v podman >/dev/null 2>&1; then
  podman run --rm -v "${ROOT_DIR}:/workspace:Z" "${STRUCTURIZR_IMAGE}" validate -w "/workspace/${DSL_FILE}"
elif command -v docker >/dev/null 2>&1; then
  docker run --rm -v "${ROOT_DIR}:/workspace" "${STRUCTURIZR_IMAGE}" validate -w "/workspace/${DSL_FILE}"
else
  printf "Neither podman nor docker is installed.\n" >&2
  exit 1
fi

printf "Structurizr DSL is valid: %s\n" "${DSL_FILE}"
