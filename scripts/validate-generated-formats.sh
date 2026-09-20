#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ASCIIDOCTOR_IMAGE="docker.io/asciidoctor/docker-asciidoctor@sha256:438f00d931a2b0775114bb294712db9273971e7f9a3dfda21fc786f1ba4e2dea"
MERMAID_IMAGE="docker.io/minlag/mermaid-cli@sha256:e8edd23a12b7939cabdb7fd9d2c6cbe7d7b8c37f68d62c782fd8e405b61529d6"
OSCAL_IMAGE="docker.io/matthewruge/oscal-cli@sha256:4c9ff87a5ac79d08236910d4251fa87219adef4d6112662a1a5c0df2873db798"
work="$ROOT_DIR/.engmod/validation/formats"

cd "$ROOT_DIR"
rm -rf "$work"
mkdir -p "$work"
if [[ -d "$ROOT_DIR/.engmod/tooling/python/bin" ]]; then
  export PATH="$ROOT_DIR/.engmod/tooling/python/bin:$PATH"
fi

container() {
  if command -v podman >/dev/null 2>&1; then
    podman run --rm -v "$ROOT_DIR:/workspace:Z" -w /workspace "$@"
  elif command -v docker >/dev/null 2>&1; then
    docker run --rm -v "$ROOT_DIR:/workspace" -w /workspace "$@"
  else
    echo "docker or podman is required for generated-format validation" >&2
    return 1
  fi
}

generated_files() {
  find generated examples -type f "$@" | sort
}

echo "> AsciiDoc"
while IFS= read -r file; do
  container "$ASCIIDOCTOR_IMAGE" asciidoctor --failure-level WARN -o /dev/null "$file" >/dev/null
done < <(generated_files -name '*.adoc')

echo "> Structurizr"
while IFS= read -r file; do
  scripts/validate-structurizr.sh "$file" >/dev/null
done < <(generated_files -name '*.dsl')

echo "> Mermaid"
while IFS= read -r file; do
  # The image's unprivileged default user writes only inside its own home.
  # This avoids host bind-mount UID mismatches while still parsing every file.
  container "$MERMAID_IMAGE" \
    -i "/workspace/$file" -o "/home/mermaidcli/validated.svg" >/dev/null
done < <(generated_files -name '*.mmd')

echo "> Threat Dragon and Open OTM"
while IFS= read -r file; do
  scripts/validate-threat-dragon.sh td-v2 "$file" >/dev/null
done < <(generated_files \( -name '*threat-dragon*.json' -o -name '*THREAT-MODEL.threat-dragon.json' \))
while IFS= read -r file; do
  scripts/validate-threat-dragon.sh open-otm "$file" >/dev/null
done < <(generated_files \( -name '*open-threat-model*.json' -o -name '*THREAT-MODEL.open-otm.json' \))

echo "> OSCAL"
while IFS= read -r file; do
  base="${file##*/}"
  case "$base" in
    *catalog*.json) kind=catalog ;;
    *profile*.json|*PROFILE*.json) kind=profile ;;
    *ssp*.json|*SSP*.json) kind=ssp ;;
    *ASSESSMENT-PLAN*.json|*.ap.json) kind=ap ;;
    *ASSESSMENT-RESULTS*.json|*.ar.json|*oscal-ar.json) kind=ar ;;
    *poam*.json|*POAM*.json) kind=poam ;;
    *) continue ;;
  esac
  container "$OSCAL_IMAGE" "$kind" validate "$file" >/dev/null
done < <(generated_files -name '*.json')

python3 - <<'PY'
import csv
import json
import xml.etree.ElementTree as ET
from pathlib import Path
from urllib.parse import urlparse

roots = [Path("generated"), Path("examples")]
generated = []
for root in roots:
    if not root.exists():
        continue
    generated.extend(path for path in root.rglob("*") if path.is_file() and (root.name == "generated" or "generated" in path.parts))

for path in sorted(generated):
    if path.suffix == ".svg":
        root = ET.parse(path).getroot()
        if root.tag != "{http://www.w3.org/2000/svg}svg":
            raise SystemExit(f"{path}: root element is not SVG")
        if root.get("data-view"):
            if not root.get("viewBox") or root.get("viewBox") == "0 0 0 0":
                raise SystemExit(f"{path}: missing nonzero viewBox")
            if root.get("role") != "img" or not root.get("aria-labelledby"):
                raise SystemExit(f"{path}: missing accessible image metadata")
    elif path.suffix == ".json":
        with path.open(encoding="utf-8") as handle:
            document = json.load(handle)
        refs = []
        if isinstance(document, dict):
            for root_name, import_name in (
                ("system-security-plan", "import-profile"),
                ("assessment-plan", "import-ssp"),
                ("assessment-results", "import-ap"),
                ("plan-of-action-and-milestones", "import-ssp"),
            ):
                root_object = document.get(root_name)
                if isinstance(root_object, dict):
                    imported = root_object.get(import_name)
                    if isinstance(imported, dict):
                        refs.append(imported.get("href", ""))
            profile = document.get("profile")
            if isinstance(profile, dict):
                refs.extend(item.get("href", "") for item in profile.get("imports", []) if isinstance(item, dict))
        for href in refs:
            parsed = urlparse(href)
            if not href or href.startswith("#") or parsed.scheme:
                continue
            target = (path.parent / href).resolve()
            if not target.is_file():
                raise SystemExit(f"{path}: unresolved OSCAL import {href}")
    elif path.suffix == ".csv":
        with path.open(newline="", encoding="utf-8") as handle:
            rows = list(csv.reader(handle))
        if rows and any(len(row) != len(rows[0]) for row in rows[1:]):
            raise SystemExit(f"{path}: inconsistent CSV field count")

airborne = Path("examples/dal-c-flight-control-sample/generated/do178c")
required = {
    "READINESS-SUMMARY.md", "PSAC-OUTLINE-DRAFT.md", "LIFECYCLE-DATA-INDEX.json",
    "OBJECTIVE-EVIDENCE-MATRIX.csv", "REQUIREMENTS-TRACE-MATRIX.csv",
    "VERIFICATION-SUMMARY.json", "INDEPENDENCE-REPORT.json",
    "STRUCTURAL-COVERAGE-SUMMARY.json", "TOOL-ASSESSMENT-REGISTER.json",
    "SOFTWARE-CONFIGURATION-INDEX.json", "PROBLEM-REPORTS.json",
    "SQA-RECORDS.json", "CERTIFICATION-LIAISON-LOG.json", "OPEN-GAPS.json",
    "SOFTWARE-ACCOMPLISHMENT-SUMMARY-DRAFT.md",
}
if airborne.is_dir():
    actual = {path.name for path in airborne.iterdir() if path.is_file()}
    if actual != required:
        raise SystemExit(f"{airborne}: output inventory mismatch: {sorted(actual ^ required)}")
    for path in airborne.iterdir():
        text = path.read_text(encoding="utf-8")
        if "DRAFT / NOT A COMPLIANCE OR CERTIFICATION DETERMINATION" not in text:
            raise SystemExit(f"{path}: missing readiness disclaimer")
        lowered = text.lower()
        if "objectiveText" in text or "normativeText" in text or "satisfies do-178c" in lowered:
            raise SystemExit(f"{path}: contains forbidden normative/compliance claim")
    lifecycle = json.loads((airborne / "LIFECYCLE-DATA-INDEX.json").read_text())
    if len(lifecycle["data"]) != 22:
        raise SystemExit(f"{airborne}: lifecycle-data index must contain 22 categories")
    coverage = json.loads((airborne / "STRUCTURAL-COVERAGE-SUMMARY.json").read_text())
    for item in coverage["data"]:
        if item["total"] != item["covered"] + item["uncovered"]:
            raise SystemExit(f"{airborne}: inconsistent coverage arithmetic")
        if sum(entry["count"] for entry in item["dispositions"]) != item["uncovered"]:
            raise SystemExit(f"{airborne}: uncovered dispositions do not account for uncovered count")
PY

echo "> TRLC"
while IFS= read -r dir; do
  trlc --brief "$dir" >/dev/null
done < <(find generated examples -type d -name trlc | sort)

echo "> LOBSTER"
while IFS= read -r config; do
  lobster-report --lobster-config "$config" --out "$work/$(printf '%s' "$config" | sha256sum | cut -c1-16).lobster" >/dev/null
done < <(generated_files -name lobster.conf)

echo "> Gemara"
scripts/validate-gemara.sh

echo "PASS generated format validation"
