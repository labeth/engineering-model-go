#!/usr/bin/env bash
# Render the maintained architecture publications after generate-examples.sh.
# Keep this separate from deterministic export validation: PDF metadata and the
# local proven-docs rendering stack are not part of the model export contract.
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
command -v proven-docs >/dev/null || { echo "proven-docs is required" >&2; exit 1; }
log_dir="$repo_root/.engmod/validation/pdf"
mkdir -p "$log_dir"
index=0
while IFS= read -r input; do
  output="${input%.adoc}.proven.pdf"
  printf 'Rendering %s\n' "$input"
  if ! proven-docs render "$input" --output "$output" >"$log_dir/$index.log" 2>&1; then
    cat "$log_dir/$index.log" >&2
    exit 1
  fi
  if grep -Fq '[mermaid] Diagram parse error:' "$log_dir/$index.log"; then
    cat "$log_dir/$index.log" >&2
    echo "Diagram rendering failed: $input" >&2
    exit 1
  fi
  test -s "$output"
  index=$((index + 1))
done < <(find generated examples -type f -name ARCHITECTURE.adoc \( -path 'generated/*' -o -path '*/generated/*' \) | sort)
if [[ "$index" -eq 0 ]]; then
  echo "No architecture publications found; run scripts/generate-examples.sh first" >&2
  exit 1
fi
printf 'Rendered %s architecture PDFs; inspect pages before release. Logs: %s\n' "$index" "$log_dir"
