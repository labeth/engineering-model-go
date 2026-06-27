#!/usr/bin/env bash
# Validate generated Gemara documents against the official OpenSSF Gemara CUE
# schemas using `cue vet`. Complements the in-repo SDK round-trip test
# (gemara_export_test.go) with full structural validation.
#
# Requirements:
#   - cue CLI (https://cuelang.org); install: go install cuelang.org/go/cmd/cue@v0.15.4
#   - the Gemara CUE schemas. Set GEMARA_SCHEMA_DIR to a checkout of
#     github.com/gemaraproj/gemara (the directory containing *.cue). If unset and
#     ~/ws/gemara exists it is used; otherwise the repo is cloned to a temp dir.
#
# Usage: scripts/validate-gemara.sh [example-dir ...]
#   Defaults to all examples/*-sample directories.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

if ! command -v cue >/dev/null 2>&1; then
  echo "ERROR: cue not found on PATH. Install: go install cuelang.org/go/cmd/cue@v0.15.4 (then add \$(go env GOPATH)/bin to PATH)" >&2
  exit 1
fi

schema_dir="${GEMARA_SCHEMA_DIR:-$HOME/ws/gemara}"
if [ ! -f "$schema_dir/controlcatalog.cue" ]; then
  schema_dir="$(mktemp -d)/gemara"
  echo ">  cloning gemara schemas to $schema_dir"
  git clone --depth 1 https://github.com/gemaraproj/gemara.git "$schema_dir" >/dev/null 2>&1
fi
echo ">  using schemas: $schema_dir"

# Map artifact file -> CUE definition.
declare -A DEFS=(
  [vector-catalog.yaml]='#VectorCatalog'
  [capability-catalog.yaml]='#CapabilityCatalog'
  [control-catalog.yaml]='#ControlCatalog'
  [threat-catalog.yaml]='#ThreatCatalog'
  [risk-catalog.yaml]='#RiskCatalog'
  [evaluation-log.yaml]='#EvaluationLog'
  [principle-catalog.yaml]='#PrincipleCatalog'
  [guidance-catalog.yaml]='#GuidanceCatalog'
  [policy.yaml]='#Policy'
  [lexicon.yaml]='#Lexicon'
  [control-threat-mapping.yaml]='#MappingDocument'
  [audit-log.yaml]='#AuditLog'
  [enforcement-log.yaml]='#EnforcementLog'
)

examples=("$@")
if [ ${#examples[@]} -eq 0 ]; then
  examples=(examples/*-sample)
fi

fail=0
for ex in "${examples[@]}"; do
  model="$ex/architecture.yml"
  [ -f "$model" ] || continue
  reqs="$ex/requirements.yml"
  out="$(mktemp -d)"
  echo ">  generating Gemara for $ex"
  args=(--model "$model" --out-dir "$out" --version 1.0.0 --date 2026-06-26T00:00:00Z)
  [ -f "$reqs" ] && args+=(--requirements "$reqs")
  go run ./cmd/enggemara "${args[@]}" >/dev/null

  for file in "${!DEFS[@]}"; do
    [ -f "$out/$file" ] || continue
    # cue requires the schema referenced as a package from its own dir (relative '.').
    if ( cd "$schema_dir" && cue vet -d "${DEFS[$file]}" . "$out/$file" ) 2>/tmp/cue-err; then
      echo "   PASS  ${DEFS[$file]}  $file"
    else
      echo "   FAIL  ${DEFS[$file]}  $file"
      sed 's/^/         /' /tmp/cue-err | head -8
      fail=1
    fi
  done
done

if [ $fail -eq 0 ]; then
  echo ">  ALL GEMARA ARTIFACTS VALID"
else
  echo ">  GEMARA VALIDATION FAILED" >&2
fi
exit $fail
