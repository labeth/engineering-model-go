#!/usr/bin/env bash
set -euo pipefail

example_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$example_root/../.." && pwd)"
output_root="${1:-$example_root/generated}"
oscal_last_modified="${ENGMOD_OSCAL_LAST_MODIFIED:-2026-09-19T00:00:00Z}"
repositories=(
  shared-edge-platform
  shared-cloud-platform
  shared-security-services
  shared-compliance-baseline
  aegis-sentinel
  orion-relay
  company-portfolio
)

cd "$repo_root"
for name in "${repositories[@]}"; do
  model_dir="$example_root/repos/$name"
  out="$output_root/$name"
  mkdir -p "$out"/{governance,requirements,security}

  go run ./cmd/engdoc --model "$model_dir/engmod.yml" --requirements "$model_dir/model/requirements.yml" \
    --design "$model_dir/model/views.yml" --out "$out/ARCHITECTURE.adoc" --decisions-out "$out/DECISIONS.adoc"
  go run ./cmd/engview --model "$model_dir/engmod.yml" --out-dir "$out"
  go run ./cmd/engtrace --model "$model_dir/engmod.yml" --requirements "$model_dir/model/requirements.yml" \
    --out "$out/TRACE-MATRIX.json"
  go run ./cmd/engtrace --model "$model_dir/engmod.yml" --requirements "$model_dir/model/requirements.yml" \
    --format csv --out "$out/TRACE-MATRIX.csv"
  go run ./cmd/engstruct --model "$model_dir/engmod.yml" --out "$out/STRUCTURIZR.dsl"
  go run ./cmd/engsysml --model "$model_dir/engmod.yml" --out "$out/ARCHITECTURE.sysml"
  go run ./cmd/engtrlc --requirements "$model_dir/model/requirements.yml" --out-dir "$out/requirements"
  go run ./cmd/englobster --tests-dir "$model_dir" --out "$out/requirements/tests.lobster.json"
  go run ./cmd/engdragon --model "$model_dir/engmod.yml" --format threat-dragon-v2 --out "$out/security/THREAT-MODEL.threat-dragon.json"
  go run ./cmd/engdragon --model "$model_dir/engmod.yml" --format open-otm --out "$out/security/THREAT-MODEL.open-otm.json"
  go run ./cmd/engoscal --model "$model_dir/engmod.yml" --requirements "$model_dir/model/requirements.yml" \
    --profile-out "$out/OSCAL-PROFILE.json" \
    --ssp-out "$out/OSCAL-SSP.json" --ap-out "$out/OSCAL-ASSESSMENT-PLAN.json" \
    --ar-out "$out/OSCAL-ASSESSMENT-RESULTS.json" --poam-out "$out/OSCAL-POAM.json" \
    --import-profile-href ./OSCAL-PROFILE.json \
    --ap-href ./OSCAL-ASSESSMENT-PLAN.json --ssp-href ./OSCAL-SSP.json --last-modified "$oscal_last_modified"
  rm -rf "$out/profiles"
  if [[ -f "$out/OSCAL-PROFILE.json" ]]; then
    mkdir -p "$out/profiles"
    cp -a "$example_root/repos/shared-compliance-baseline/profiles/." "$out/profiles/"
  fi
  rm -rf "$out/governance"
  mkdir -p "$out/governance"
  go run ./cmd/enggemara --model "$model_dir/engmod.yml" --requirements "$model_dir/model/requirements.yml" \
    --out-dir "$out/governance" --author "Atlas Industries" --author-id atlas-industries \
    --version v0.1.0 --date "${oscal_last_modified%%T*}"
done
