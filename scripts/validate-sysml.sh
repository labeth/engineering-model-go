#!/usr/bin/env bash
# Blocking SysML syntax, interchange freshness, KPAR reopen, and round-trip gate.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

tool_bin="$repo_root/.engmod/tooling/bin"
validator="${ENGMOD_SYSML_VALIDATOR:-$tool_bin/validate-sysml}"
sysand="${ENGMOD_SYSAND:-$tool_bin/sysand}"
work="$repo_root/.engmod/validation/sysml"
trap 'rm -rf "$work"' EXIT

bash scripts/check-sysml-metamodel.sh

run_validator() {
  local source="$1" log="$2"
  if "$validator" "$source" >"$log" 2>&1; then
    return
  fi
  cat "$log" >&2
  return 1
}

if [ ! -x "$validator" ] || [ ! -x "$sysand" ]; then
  if [ "${ENGMOD_SYSML_SKIP_EXTERNAL:-0}" = "1" ]; then
    echo "SKIP SysML external validation explicitly disabled with ENGMOD_SYSML_SKIP_EXTERNAL=1"
    exit 0
  fi
  echo "pinned SysML tools are required; run scripts/install-sysml-toolchain.sh" >&2
  exit 1
fi

rm -rf "$work"
mkdir -p "$work"

go run ./cmd/engsysml --model architecture.yml --out "$work/ARCHITECTURE.sysml" 2>"$work/diagnostics.log"
if grep -E '(lossy|unsupported|unknown)' "$work/diagnostics.log"; then
  echo "blocking SysML projection diagnostic detected" >&2
  exit 1
fi
cmp "$work/ARCHITECTURE.sysml" generated/ARCHITECTURE.sysml

go run ./cmd/engsysml --coverage --out "$work/SYSML-COVERAGE.json"
cmp "$work/SYSML-COVERAGE.json" generated/SYSML-COVERAGE.json

run_validator "$work/ARCHITECTURE.sysml" "$work/generated-validator.log"

go run ./cmd/engsysml --model architecture.yml \
  --project-out "$work/project" \
  --kpar-out "$work/engineering-model.kpar" \
  --sysand "$sysand"
run_validator "$work/project/model.sysml" "$work/project-validator.log"

go run ./cmd/engsysml --model architecture.yml \
  --verify-kpar "$work/engineering-model.kpar" \
  --reopen-out "$work/reopened" \
  --sysand "$sysand"
run_validator "$work/reopened/model.sysml" "$work/reopened-validator.log"

cmp "$work/project/model.sysml" "$work/reopened/model.sysml"
echo "PASS SysML official-parser, freshness, normative KPAR, reopen, and semantic round-trip validation"
