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
#   5. Artifact freshness: every maintained example artifact must regenerate
#      bit-for-bit.
#   6. Authoritative format validation for AsciiDoc, Mermaid, Structurizr,
#      SysML, TRLC, LOBSTER, OSCAL, Gemara, Threat Dragon, Open OTM, JSON, CSV.
#
# External validation may only be skipped locally by explicitly setting the
# relevant ENGMOD_*_SKIP_EXTERNAL flag. CI installs and requires pinned tools.
set -uo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

fail=0
section() { printf '\n========== %s ==========\n' "$*"; }
log_dir="$repo_root/.engmod/validation/logs"
mkdir -p "$log_dir"
generated_before="$log_dir/generated.before.sha256"
generated_after="$log_dir/generated.after.sha256"
find generated examples -type f \( -path 'generated/*' -o -path '*/generated/*' \) -print0 |
  sort -z | xargs -0 sha256sum >"$generated_before"

# name|model-dir — each dir has engmod.yml plus schema-v2 domain documents and
# inferenceHints.codeSources, so engdoc/engtrace need no --code-root.
MODELS=(
  "self|."
  "dal-c-flight-control|examples/dal-c-flight-control-sample"
)

section "Build"
go build ./... && echo "  ok" || fail=1

section "Enterprise OCI workspace"
if go test . -run '^TestAtlasIndustriesCompanyExample$' -count=1; then
  echo "  ok   Atlas Industries multi-repository composition"
else
  echo "  FAIL Atlas Industries multi-repository composition"
  fail=1
fi
if scripts/generate-examples.sh >"$log_dir/example-generation.log" 2>&1; then
  echo "  ok   all maintained example projections"
else
  echo "  FAIL maintained example projections"
  tail -12 "$log_dir/example-generation.log" | sed 's/^/         /'
  fail=1
fi

section "Generation gates (engdoc 0 errors, engtrace 0 dangling) + regeneration"
for entry in "${MODELS[@]}"; do
  name="${entry%%|*}"; dir="${entry#*|}"; g="$dir/generated"
  mkdir -p "$g"
  if go run ./cmd/engdoc --model "$dir/engmod.yml" --requirements "$dir/model/requirements.yml" \
       --design "$dir/model/views.yml" --out "$g/ARCHITECTURE.adoc" --decisions-out "$g/DECISIONS.adoc" 2>"$log_dir/$name.engdoc.err"; then
    echo "  ok   engdoc   $name"
  else
    echo "  FAIL engdoc   $name"; grep -E '\[error\]' "$log_dir/$name.engdoc.err" | sed 's/^/         /' | head; fail=1
  fi
  if go run ./cmd/engtrace --model "$dir/engmod.yml" --requirements "$dir/model/requirements.yml" \
       --out "$g/TRACE-MATRIX.json" 2>"$log_dir/$name.engtrace.err"; then
    echo "  ok   engtrace $name"
  else
    echo "  FAIL engtrace $name"; grep -E 'dangling' "$log_dir/$name.engtrace.err" | sed 's/^/         /' | head; fail=1
  fi
  if grep -q '^naf:' "$dir/model/views.yml"; then
    if go run ./cmd/engnaf --model "$dir/engmod.yml" --out "$g/ARCHITECTURE.naf.adoc" 2>"$log_dir/$name.engnaf.err"; then
      echo "  ok   engnaf   $name"
    else
      echo "  FAIL engnaf   $name"; head -10 "$log_dir/$name.engnaf.err" | sed 's/^/         /'; fail=1
    fi
  fi

done

go run ./cmd/engoscal --model engmod.yml --requirements model/requirements.yml --code-root . \
  --profile-out generated/ARCHITECTURE.profile.json \
  --ssp-out generated/ARCHITECTURE.ssp.json \
  --ap-out generated/ARCHITECTURE.ap.json \
  --ar-out generated/ARCHITECTURE.ar.json \
  --poam-out generated/ARCHITECTURE.poam.json \
  --import-profile-href ./ARCHITECTURE.profile.json \
  --ap-href ./ARCHITECTURE.ap.json \
  --ssp-href ./ARCHITECTURE.ssp.json \
  --last-modified 2026-09-19T00:00:00Z >/dev/null
go run ./cmd/engoscal --model engmod.yml --requirements model/requirements.yml \
  --ar-out generated/compliance/OSCAL-ASSESSMENT-RESULTS.json \
  --ap-href ./ASSESSMENT-PLAN.json \
  --last-modified 2026-09-19T00:00:00Z >/dev/null

section "SysML v2 official parser and KPAR round trip"
if bash scripts/validate-sysml.sh; then
  echo "  ok   SysML v2 conformance gates"
else
  echo "  FAIL SysML v2 conformance gates"
  fail=1
fi

section "Artifact freshness (regeneration is a no-op)"
find generated examples -type f \( -path 'generated/*' -o -path '*/generated/*' \) -print0 |
  sort -z | xargs -0 sha256sum >"$generated_after"
if cmp -s "$generated_before" "$generated_after"; then
  echo "  ok   no drift"
else
  echo "  FAIL generated artifacts changed during regeneration:"
  diff -u "$generated_before" "$generated_after" | sed 's/^/         /' || true
  fail=1
fi

section "Generated format conformance"
if [ "${ENGMOD_FORMAT_SKIP_EXTERNAL:-0}" = "1" ]; then
  echo "  skip external generated-format validation explicitly disabled"
elif scripts/validate-generated-formats.sh >"$log_dir/generated-formats.log" 2>&1; then
  echo "  ok   all generated formats"
else
  echo "  FAIL generated format conformance"
  tail -20 "$log_dir/generated-formats.log" | sed 's/^/         /'
  fail=1
fi

section "Result"
[ "$fail" -eq 0 ] && echo "  PASS" || echo "  FAIL"
exit "$fail"
