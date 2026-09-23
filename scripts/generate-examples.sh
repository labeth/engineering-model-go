#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GENERATED_DATE="${ENGMOD_GENERATED_DATE:-2026-09-19}"
OSCAL_LAST_MODIFIED="${ENGMOD_OSCAL_LAST_MODIFIED:-${GENERATED_DATE}T00:00:00Z}"

examples=(
  examples/payments-engineering-sample
  examples/bedrock-pr-review-github-app-sample
  examples/coffee-fleet-ota-cloud-sample
  examples/coffee-fleet-ota-cloud-sample/subsystems/cloud-api
  examples/coffee-fleet-ota-cloud-sample/subsystems/ota-agent
  examples/coffee-fleet-ota-cloud-sample/subsystems/telemetry
  examples/dal-c-flight-control-sample
)

cd "$ROOT_DIR"
if [[ -d "$ROOT_DIR/.engmod/tooling/python/bin" ]]; then
  export PATH="$ROOT_DIR/.engmod/tooling/python/bin:$PATH"
fi

mkdir -p generated
go run ./cmd/engsysml --model engmod.yml --out generated/ARCHITECTURE.sysml

for model_dir in "${examples[@]}"; do
  out="$model_dir/generated"
  model="$model_dir/engmod.yml"
  requirements="$model_dir/model/requirements.yml"
  design="$model_dir/model/views.yml"
  mkdir -p "$out" "$out/trlc"

  go run ./cmd/engdoc --model "$model" --requirements "$requirements" --design "$design" \
    --code-root "$model_dir" --out "$out/ARCHITECTURE.adoc" --decisions-out "$out/DECISIONS.adoc"
  go run ./cmd/engview --model "$model" --out-dir "$out"
  go run ./cmd/engtrace --model "$model" --requirements "$requirements" --code-root "$model_dir" \
    --out "$out/TRACE-MATRIX.json"
  go run ./cmd/engtrace --model "$model" --requirements "$requirements" --code-root "$model_dir" \
    --format csv --out "$out/TRACE-MATRIX.csv"
  go run ./cmd/engstruct --model "$model" --out "$out/STRUCTURIZR.dsl"
  go run ./cmd/engsysml --model "$model" --out "$out/ARCHITECTURE.sysml"
  go run ./cmd/engtrlc --requirements "$requirements" --out-dir "$out/trlc"
  go run ./cmd/engdragon --model "$model" --format threat-dragon-v2 --out "$out/threat-dragon-v2.json"
  go run ./cmd/engdragon --model "$model" --format open-otm --out "$out/open-threat-model.json"
  rm -rf "$out/do178c"
  go run ./cmd/engair --model "$model" --out-dir "$out/do178c" --date "$GENERATED_DATE" --version v0.1.0

  if [[ -d "$model_dir/oscal" ]]; then
    rm -rf "$out/oscal"
    mkdir -p "$out/oscal"
    cp -a "$model_dir/oscal/." "$out/oscal/"
  fi
  go run ./cmd/engoscal --model "$model" --requirements "$requirements" --code-root "$model_dir" \
    --profile-out "$out/ARCHITECTURE.profile.json" \
    --ssp-out "$out/ARCHITECTURE.ssp.json" \
    --ap-out "$out/ARCHITECTURE.ap.json" \
    --ar-out "$out/ARCHITECTURE.ar.json" \
    --poam-out "$out/ARCHITECTURE.poam.json" \
    --import-profile-href ./ARCHITECTURE.profile.json \
    --ap-href ./ARCHITECTURE.ap.json \
    --ssp-href ./ARCHITECTURE.ssp.json \
    --last-modified "$OSCAL_LAST_MODIFIED"

  rm -rf "$out/gemara"
  mkdir -p "$out/gemara"
  gemara_args=(
    --model "$model"
    --requirements "$requirements"
    --code-root "$model_dir"
    --out-dir "$out/gemara"
    --author "Engineering Model Go Examples"
    --author-id engineering-model-go
    --version v0.1.0
    --date "$GENERATED_DATE"
  )
  if grep -q '^  controls:' "$model_dir/model/assurance.yml"; then
    gemara_args+=(--oscal-catalog-out "$out/gemara/oscal-catalog.json")
  fi
  if [[ -f "$out/ARCHITECTURE.ap.json" ]]; then
    gemara_args+=(--oscal-ar-out "$out/gemara/oscal-ar.json" --oscal-ap-href ../ARCHITECTURE.ap.json)
  fi
  go run ./cmd/enggemara "${gemara_args[@]}"

  if [[ -d "$model_dir/tests" ]]; then
    trlc_package="$(awk '/^package / { print $2; exit }' "$out/trlc/model.rsl")"
    scripts/generate-lobster-report.sh "$model_dir" "$trlc_package"
  else
    rm -rf "$out/lobster"
  fi
done

coffee_six_view="examples/coffee-appliance-six-view"
mkdir -p "$coffee_six_view/generated"
go run ./cmd/engdoc --model "$coffee_six_view/engmod.yml" \
  --requirements "$coffee_six_view/model/requirements.yml" --design "$coffee_six_view/model/views.yml" \
  --out "$coffee_six_view/generated/ARCHITECTURE.adoc" --decisions-out "$coffee_six_view/generated/DECISIONS.adoc"
go run ./cmd/engtrace --model "$coffee_six_view/engmod.yml" \
  --requirements "$coffee_six_view/model/requirements.yml" --out "$coffee_six_view/generated/TRACE-MATRIX.json"
go run ./cmd/engview --model "$coffee_six_view/engmod.yml" --out-dir "$coffee_six_view/generated"
go run ./cmd/engsysml --model "$coffee_six_view/engmod.yml" --out "$coffee_six_view/generated/ARCHITECTURE.sysml"

examples/atlas-industries/generate.sh
