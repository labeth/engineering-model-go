#!/usr/bin/env bash
# Full validation gauntlet — single source of truth for local dev and CI.
#
# Strict (always enforced):
#   1. go build.
#   2. Zero-gap SysML/KerML inventory, renderer, property, relationship, and
#      generated coverage-manifest validation.
#   3. Generation gates: engdoc exits non-zero on any error, which transitively
#      enforces trace integrity (dangling links), EARS lint, and composition checks.
#   4. engtrace: 0 dangling code trace links per model.
#   5. Artifact freshness: regenerated ARCHITECTURE.adoc / DECISIONS.adoc /
#      TRACE-MATRIX.json / ARCHITECTURE.naf.adoc must match what is committed.
#
# External validation may only be skipped locally by explicitly setting the
# relevant ENGMOD_*_SKIP_EXTERNAL flag. CI installs and requires pinned tools.
# Best-effort:
#   6. Gemara CUE schema validation (cue).
#   7. Structurizr DSL validation (docker/podman).
#   8. TRLC validation (trlc).
set -uo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

fail=0
section() { printf '\n========== %s ==========\n' "$*"; }
log_dir="$repo_root/.engmod/validation/logs"
mkdir -p "$log_dir"
generated_before="$log_dir/generated.before.sha256"
generated_after="$log_dir/generated.after.sha256"
find . -path ./.git -prune -o -type f \( -name 'ARCHITECTURE.adoc' -o -name 'DECISIONS.adoc' -o -name 'TRACE-MATRIX.json' -o -name 'ARCHITECTURE.naf.adoc' \) -print0 |
  sort -z | xargs -0 sha256sum >"$generated_before"

# name|model-dir — each dir has architecture.yml, requirements.yml, design.yml and
# inferenceHints.codeSources, so engdoc/engtrace need no --code-root.
MODELS=(
  "self|."
  "payments|examples/payments-engineering-sample"
  "bedrock|examples/bedrock-pr-review-github-app-sample"
  "coffee-fleet|examples/coffee-fleet-ota-cloud-sample"
  "telemetry|examples/coffee-fleet-ota-cloud-sample/subsystems/telemetry"
  "ota-agent|examples/coffee-fleet-ota-cloud-sample/subsystems/ota-agent"
  "cloud-api|examples/coffee-fleet-ota-cloud-sample/subsystems/cloud-api"
)

section "Build"
go build ./... && echo "  ok" || fail=1

section "Generation gates (engdoc 0 errors, engtrace 0 dangling) + regeneration"
for entry in "${MODELS[@]}"; do
  name="${entry%%|*}"; dir="${entry#*|}"; g="$dir/generated"
  mkdir -p "$g"
  if go run ./cmd/engdoc --model "$dir/architecture.yml" --requirements "$dir/requirements.yml" \
       --design "$dir/design.yml" --out "$g/ARCHITECTURE.adoc" --decisions-out "$g/DECISIONS.adoc" 2>"$log_dir/$name.engdoc.err"; then
    echo "  ok   engdoc   $name"
  else
    echo "  FAIL engdoc   $name"; grep -E '\[error\]' "$log_dir/$name.engdoc.err" | sed 's/^/         /' | head; fail=1
  fi
  if go run ./cmd/engtrace --model "$dir/architecture.yml" --requirements "$dir/requirements.yml" \
       --out "$g/TRACE-MATRIX.json" 2>"$log_dir/$name.engtrace.err"; then
    echo "  ok   engtrace $name"
  else
    echo "  FAIL engtrace $name"; grep -E 'dangling' "$log_dir/$name.engtrace.err" | sed 's/^/         /' | head; fail=1
  fi
  if grep -q '^naf:' "$dir/architecture.yml"; then
    if go run ./cmd/engnaf --model "$dir/architecture.yml" --out "$g/ARCHITECTURE.naf.adoc" 2>"$log_dir/$name.engnaf.err"; then
      echo "  ok   engnaf   $name"
    else
      echo "  FAIL engnaf   $name"; head -10 "$log_dir/$name.engnaf.err" | sed 's/^/         /'; fail=1
    fi
  fi

done

section "SysML v2 official parser and KPAR round trip"
if bash scripts/validate-sysml.sh; then
  echo "  ok   SysML v2 conformance gates"
else
  echo "  FAIL SysML v2 conformance gates"
  fail=1
fi

section "Artifact freshness (regeneration is a no-op)"
find . -path ./.git -prune -o -type f \( -name 'ARCHITECTURE.adoc' -o -name 'DECISIONS.adoc' -o -name 'TRACE-MATRIX.json' -o -name 'ARCHITECTURE.naf.adoc' \) -print0 |
  sort -z | xargs -0 sha256sum >"$generated_after"
if cmp -s "$generated_before" "$generated_after"; then
  echo "  ok   no drift"
else
  echo "  FAIL generated artifacts changed during regeneration:"
  diff -u "$generated_before" "$generated_after" | sed 's/^/         /' || true
  fail=1
fi

section "Gemara schema validation (cue)"
if command -v cue >/dev/null 2>&1; then
  if bash scripts/validate-gemara.sh >"$log_dir/gemara.log" 2>&1; then echo "  ok   all Gemara artifacts valid"; else echo "  FAIL"; tail -8 "$log_dir/gemara.log" | sed 's/^/         /'; fail=1; fi
else
  echo "  skip cue not on PATH (go install cuelang.org/go/cmd/cue@v0.15.4)"
fi

section "Structurizr DSL validation (docker/podman)"
if [ "${ENGMOD_VALIDATE_STRUCTURIZR:-0}" != "1" ]; then
  echo "  skip set ENGMOD_VALIDATE_STRUCTURIZR=1 to validate (pulls the structurizr docker image)"
elif command -v docker >/dev/null 2>&1 || command -v podman >/dev/null 2>&1; then
  while IFS= read -r dsl; do
    if bash scripts/validate-structurizr.sh "$dsl" >"$log_dir/struct.log" 2>&1; then echo "  ok   $dsl"; else echo "  FAIL $dsl"; tail -4 "$log_dir/struct.log" | sed 's/^/         /'; fail=1; fi
  done < <(find . -path ./.git -prune -o -name STRUCTURIZR.dsl -print)
else
  echo "  skip no docker/podman on PATH"
fi

section "TRLC validation (trlc)"
if command -v trlc >/dev/null 2>&1; then
  while IFS= read -r tdir; do
    if bash scripts/validate-trlc.sh "$tdir" >"$log_dir/trlc.log" 2>&1; then echo "  ok   $tdir"; else echo "  FAIL $tdir"; tail -4 "$log_dir/trlc.log" | sed 's/^/         /'; fail=1; fi
  done < <(find . -path ./.git -prune -o -type d -name trlc -print)
else
  echo "  skip trlc not on PATH (python3 -m pip install --user trlc)"
fi

section "Result"
[ "$fail" -eq 0 ] && echo "  PASS" || echo "  FAIL"
exit "$fail"
