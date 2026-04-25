#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

usage() {
  cat <<'EOF'
Usage:
  scripts/generate-lobster-report.sh <example-dir> <trlc-package>

Example:
  scripts/generate-lobster-report.sh examples/payments-engineering-sample PaymentsRequirements
EOF
}

if [[ $# -ne 2 ]]; then
  usage
  exit 2
fi

EXAMPLE_DIR="$1"
TRLC_PACKAGE="$2"

if [[ ! -d "${ROOT_DIR}/${EXAMPLE_DIR}" ]]; then
  printf "Example directory not found: %s\n" "${EXAMPLE_DIR}" >&2
  exit 1
fi

for bin in trlc lobster-trlc lobster-report lobster-html-report; do
  if ! command -v "$bin" >/dev/null 2>&1; then
    printf "Missing required tool: %s\n" "$bin" >&2
    printf "Install with: python3 -m pip install --user trlc bmw-lobster-core bmw-lobster-tool-trlc\n" >&2
    exit 1
  fi
done

REQ_FILE="${EXAMPLE_DIR}/requirements.yml"
TESTS_DIR="${EXAMPLE_DIR}/tests"
OUT_DIR="${ROOT_DIR}/${EXAMPLE_DIR}/generated/lobster"
TRLC_OUT_DIR="${ROOT_DIR}/${EXAMPLE_DIR}/generated/trlc"

mkdir -p "$OUT_DIR"

go run ./cmd/engtrlc --requirements "$REQ_FILE" --out-dir "${EXAMPLE_DIR}/generated/trlc" --package "$TRLC_PACKAGE"

TRLC_CONFIG="${OUT_DIR}/lobster-trlc-config.yml"
cat > "$TRLC_CONFIG" <<EOF
inputs:
  - ${TRLC_OUT_DIR}/model.rsl
  - ${TRLC_OUT_DIR}/requirements.trlc
conversion-rules:
  - package: ${TRLC_PACKAGE}
    record-type: Requirement
    namespace: req
    description-fields:
      - text
EOF

REQ_LOBSTER="${OUT_DIR}/requirements.lobster"
ACT_LOBSTER="${OUT_DIR}/activities.lobster"
CONF_FILE="${OUT_DIR}/lobster.conf"
REPORT_FILE="${OUT_DIR}/report.lobster"
HTML_FILE="${OUT_DIR}/report.html"

lobster-trlc --config "$TRLC_CONFIG" --out "$REQ_LOBSTER"

go run ./cmd/englobster --tests-dir "$TESTS_DIR" --requirements-package "$TRLC_PACKAGE" --activity-namespace tests --out "$ACT_LOBSTER"

cat > "$CONF_FILE" <<EOF
requirements "Requirements" {
  source: "${REQ_LOBSTER}";
}

activity "Tests" {
  source: "${ACT_LOBSTER}";
  trace to: "Requirements";
}
EOF

lobster-report --lobster-config "$CONF_FILE" --out "$REPORT_FILE"
lobster-html-report --out "$HTML_FILE" "$REPORT_FILE"

printf "Generated LOBSTER outputs:\n"
printf -- "- %s\n" "$REQ_LOBSTER"
printf -- "- %s\n" "$ACT_LOBSTER"
printf -- "- %s\n" "$CONF_FILE"
printf -- "- %s\n" "$REPORT_FILE"
printf -- "- %s\n" "$HTML_FILE"
