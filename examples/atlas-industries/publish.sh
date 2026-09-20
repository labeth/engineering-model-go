#!/usr/bin/env bash
set -euo pipefail

if ! command -v cue >/dev/null 2>&1; then
  echo "cue is required: go install cuelang.org/go/cmd/cue@v0.17.1" >&2
  exit 1
fi
if [[ -z "${CUE_REGISTRY:-}" ]]; then
  echo "CUE_REGISTRY must select the target OCI registry" >&2
  exit 1
fi

example_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repositories=(
  shared-edge-platform
  shared-cloud-platform
  shared-security-services
  shared-compliance-baseline
  aegis-sentinel
  orion-relay
  company-portfolio
)

for name in "${repositories[@]}"; do
  echo "publishing $name v0.1.0"
  (cd "$example_root/repos/$name" && cue mod publish v0.1.0)
done
