#!/usr/bin/env bash
# Full validation gauntlet — single source of truth for local dev and CI.
#
# Strict (always enforced):
#   1. go build.
#   2. Generation gates: engdoc exits non-zero on any error, which transitively
#      enforces trace integrity (dangling links), EARS lint, and composition checks.
#   3. engtrace: 0 dangling code trace links per model.
#   4. Artifact freshness: regenerated ARCHITECTURE.adoc / DECISIONS.adoc /
#      TRACE-MATRIX.json must match what is committed (no stale generated docs).
#
# Best-effort (run only when the external tool is on PATH; never fails on a
# missing tool, but a real validation failure does fail):
#   5. Gemara CUE schema validation (cue).
#   6. Structurizr DSL validation (docker/podman).
#   7. TRLC validation (trlc).
set -uo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

fail=0
section() { printf '\n========== %s ==========\n' "$*"; }

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
       --design "$dir/design.yml" --out "$g/ARCHITECTURE.adoc" --decisions-out "$g/DECISIONS.adoc" 2>"/tmp/$name.engdoc.err"; then
    echo "  ok   engdoc   $name"
  else
    echo "  FAIL engdoc   $name"; grep -E '\[error\]' "/tmp/$name.engdoc.err" | sed 's/^/         /' | head; fail=1
  fi
  if go run ./cmd/engtrace --model "$dir/architecture.yml" --requirements "$dir/requirements.yml" \
       --out "$g/TRACE-MATRIX.json" 2>"/tmp/$name.engtrace.err"; then
    echo "  ok   engtrace $name"
  else
    echo "  FAIL engtrace $name"; grep -E 'dangling' "/tmp/$name.engtrace.err" | sed 's/^/         /' | head; fail=1
  fi
done

section "Artifact freshness (regenerated must match committed)"
if git diff --quiet -- '*ARCHITECTURE.adoc' '*DECISIONS.adoc' '*TRACE-MATRIX.json'; then
  echo "  ok   no drift"
else
  echo "  FAIL committed generated artifacts are stale — regenerate and commit:"
  git diff --name-only -- '*ARCHITECTURE.adoc' '*DECISIONS.adoc' '*TRACE-MATRIX.json' | sed 's/^/         /'
  fail=1
fi

section "Gemara schema validation (cue)"
if command -v cue >/dev/null 2>&1; then
  if bash scripts/validate-gemara.sh >/tmp/gemara.log 2>&1; then echo "  ok   all Gemara artifacts valid"; else echo "  FAIL"; tail -8 /tmp/gemara.log | sed 's/^/         /'; fail=1; fi
else
  echo "  skip cue not on PATH (go install cuelang.org/go/cmd/cue@v0.15.4)"
fi

section "Structurizr DSL validation (docker/podman)"
if [ "${ENGMOD_VALIDATE_STRUCTURIZR:-0}" != "1" ]; then
  echo "  skip set ENGMOD_VALIDATE_STRUCTURIZR=1 to validate (pulls the structurizr docker image)"
elif command -v docker >/dev/null 2>&1 || command -v podman >/dev/null 2>&1; then
  while IFS= read -r dsl; do
    if bash scripts/validate-structurizr.sh "$dsl" >/tmp/struct.log 2>&1; then echo "  ok   $dsl"; else echo "  FAIL $dsl"; tail -4 /tmp/struct.log | sed 's/^/         /'; fail=1; fi
  done < <(find . -path ./.git -prune -o -name STRUCTURIZR.dsl -print)
else
  echo "  skip no docker/podman on PATH"
fi

section "TRLC validation (trlc)"
if command -v trlc >/dev/null 2>&1; then
  while IFS= read -r tdir; do
    if bash scripts/validate-trlc.sh "$tdir" >/tmp/trlc.log 2>&1; then echo "  ok   $tdir"; else echo "  FAIL $tdir"; tail -4 /tmp/trlc.log | sed 's/^/         /'; fail=1; fi
  done < <(find . -path ./.git -prune -o -type d -name trlc -print)
else
  echo "  skip trlc not on PATH (python3 -m pip install --user trlc)"
fi

section "Result"
[ "$fail" -eq 0 ] && echo "  PASS" || echo "  FAIL"
exit "$fail"
