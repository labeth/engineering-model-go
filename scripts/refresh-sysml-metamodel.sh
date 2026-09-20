#!/usr/bin/env bash
# Download the pinned official implementation Ecore and refresh tracked compact artifacts.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

go run ./cmd/sysmlmetamodel --mode fetch
go run ./cmd/sysmlmetamodel --mode refresh
go run ./cmd/sysmlmetamodel --mode check
