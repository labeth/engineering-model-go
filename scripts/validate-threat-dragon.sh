#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCHEMA_DIR="${ROOT_DIR}/tools/threat-dragon-schemas"

usage() {
  cat <<'EOF'
Usage:
  scripts/validate-threat-dragon.sh <format> <json-file>

Formats:
  td-v2      Validate against threat-dragon-v2.schema.json
  open-otm   Validate against open-threat-model.schema.json

Examples:
  scripts/validate-threat-dragon.sh td-v2 out/threat-dragon.json
  scripts/validate-threat-dragon.sh open-otm out/open-threat-model.json
EOF
}

if [[ $# -ne 2 ]]; then
  usage
  exit 2
fi

FORMAT="$1"
MODEL_FILE="$2"

case "${FORMAT}" in
  td-v2)
    SCHEMA_FILE="${SCHEMA_DIR}/threat-dragon-v2.schema.json"
    ;;
  open-otm)
    SCHEMA_FILE="${SCHEMA_DIR}/open-threat-model.schema.json"
    ;;
  *)
    printf "Unknown format: %s\n\n" "${FORMAT}" >&2
    usage
    exit 2
    ;;
esac

if [[ ! -f "${SCHEMA_FILE}" ]]; then
  printf "Schema not found: %s\n" "${SCHEMA_FILE}" >&2
  printf "Run scripts/fetch-threat-dragon-schemas.sh first.\n" >&2
  exit 1
fi

if [[ ! -f "${MODEL_FILE}" ]]; then
  printf "JSON model not found: %s\n" "${MODEL_FILE}" >&2
  exit 1
fi

npx --yes ajv-cli validate --spec=draft7 --allow-union-types -s "${SCHEMA_FILE}" -d "${MODEL_FILE}"
