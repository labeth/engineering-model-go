#!/usr/bin/env bash
# Offline freshness and strict inventory-coverage binding check.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

go run ./cmd/sysmlmetamodel --mode check

work="$repo_root/.engmod/validation/sysml-coverage"
mkdir -p "$work"
go run ./cmd/engsysml --coverage --out "$work/SYSML-COVERAGE.json"
cmp "$work/SYSML-COVERAGE.json" generated/SYSML-COVERAGE.json
go run ./cmd/engsysml --validate-coverage generated/SYSML-COVERAGE.json
