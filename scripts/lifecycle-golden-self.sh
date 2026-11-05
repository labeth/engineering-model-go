#!/usr/bin/env bash
# No-lifecycle golden for this repository's own model (REQ-EMG-124): builds
# engdoc and engtrace at a reference tag (default v0.9.0, the last engine
# without the life-cycle tier) and at the working tree, runs both on the
# same tree, and fails unless the documents, decisions documents, trace
# matrices and diagnostics are byte-identical. The example models are
# covered by TestNoLifecycleGoldenOverExamples; this model's trace changes
# with every code edit, so it is compared live instead of against a record.
#
#   scripts/lifecycle-golden-self.sh [reference-tag]
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
ref="${1:-v0.9.0}"
work="$(mktemp -d "${TMPDIR:-/tmp}/engmod-golden-self.XXXXXX")"
trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf "$work"' EXIT

mkdir -p "$work/src" "$work/bin-ref" "$work/bin-cur" "$work/out-ref" "$work/out-cur"
git archive "$ref" | tar -x -C "$work/src"
(cd "$work/src" && go build -o "$work/bin-ref/" ./cmd/engdoc ./cmd/engtrace)
go build -o "$work/bin-cur/" ./cmd/engdoc ./cmd/engtrace

if grep -q '^lifecycle:' engmod.yml; then
  echo "engmod.yml declares lifecycle; this golden covers a model without it" >&2
  exit 2
fi

run() {
  local bin="$1" out="$2" status=0
  "$bin/engdoc" --model engmod.yml --requirements model/requirements.yml --design model/views.yml \
    --out "$out/ARCHITECTURE.adoc" --decisions-out "$out/DECISIONS.adoc" 2>"$out/engdoc.err" || status=$?
  echo "$status" >"$out/engdoc.status"
  status=0
  "$bin/engtrace" --model engmod.yml --requirements model/requirements.yml \
    --out "$out/TRACE-MATRIX.json" >/dev/null 2>"$out/engtrace.err" || status=$?
  echo "$status" >"$out/engtrace.status"
}
run "$work/bin-ref" "$work/out-ref"
run "$work/bin-cur" "$work/out-cur"

fail=0
for f in ARCHITECTURE.adoc DECISIONS.adoc TRACE-MATRIX.json engdoc.err engtrace.err engdoc.status engtrace.status; do
  if cmp -s "$work/out-ref/$f" "$work/out-cur/$f"; then
    echo "  ok   $f identical to $ref"
  else
    echo "  FAIL $f differs from $ref"
    diff "$work/out-ref/$f" "$work/out-cur/$f" | head -20 | sed 's/^/         /'
    fail=1
  fi
done
exit "$fail"
