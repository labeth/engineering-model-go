#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${ROOT_DIR}/tools/threat-dragon-schemas"

mkdir -p "${OUT_DIR}"

curl -fsSL "https://raw.githubusercontent.com/OWASP/threat-dragon/main/td.vue/src/assets/schema/threat-dragon-v2.schema.json" \
  -o "${OUT_DIR}/threat-dragon-v2.schema.json"

curl -fsSL "https://raw.githubusercontent.com/OWASP/threat-dragon/main/td.vue/src/assets/schema/open-threat-model.schema.json" \
  -o "${OUT_DIR}/open-threat-model.schema.json"

printf "Downloaded schemas to %s\n" "${OUT_DIR}"
