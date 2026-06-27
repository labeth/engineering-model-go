# Gemara GRC Rendering

`enggemara` renders the engineering model into [OpenSSF Gemara](https://gemara.openssf.org)
documents. Gemara is a first-class rendering: the exporter builds the official
`github.com/gemaraproj/go-gemara` SDK structs, serializes them with the same YAML codec the
SDK uses, and every artifact validates against the published Gemara CUE schemas.

## Artifacts produced (all 13 Gemara types)

| Gemara layer | Artifact | Source model data |
|---|---|---|
| L1 | Vector Catalog | `attackVectors` (+ environment `applicability` from `deploymentTargets`) |
| L1 | Principle Catalog | standard governance principles (least-privilege, defense-in-depth, …) |
| L1 | Guidance Catalog | one guideline per distinct control `category`; controls link back via `guidelines` |
| L2 | Capability Catalog | `functionalUnits` (+ a synthetic `CAP-SYSTEM`) grouped by `functionalGroups` |
| L2 | Threat Catalog | `threatScenarios` (group from STRIDE/category; capabilities from `appliesTo`; vectors from `attackVectorRef`; derived `actors`) |
| L2 | Control Catalog | `controls` + `controlVerifications` (+ `guidelines`/`threats` links) |
| L3 | Risk Catalog | `risks` (derived severity + unique `rank`) |
| L3 | Policy | scope from data/actors/interfaces; imports control+guidance catalogs; mitigated/accepted risks; adherence from verifications |
| L5 | Evaluation Log | `controlVerifications` (pass and fail; with `evidence`, `recommendation`, `confidence-level`) |
| L6 | Enforcement Log | `poamItems` (remediation `actions`) |
| L7 | Audit Log | `controlVerifications` + residual risks (`results` with evidence/recommendations) |
| — | Mapping Document | control → threat relationships |
| — | Lexicon | catalog terms (controlled vocabulary) |

Artifacts that need supporting data (Guidance, Policy, Mapping, Audit, Enforcement) are emitted
only when the model provides it; the Vector/Capability/Control/Threat/Risk/Principle catalogs and
the Lexicon are always produced.

## Field mapping

### Control → `#Control` (controlcatalog)

| Model `Control` | Gemara `#Control` | Notes |
|---|---|---|
| `id` | `id` | |
| `name` | `title` | |
| `description` | `objective` | required; falls back to name |
| `category` | `group` | lifted into `groups[]`; default `general` |
| `controlVerifications[*]` (by `controlRef`) | `assessment-requirements[]` | one requirement per verification; a control with none gets a default requirement |
| `controlVerification.findings` | `assessment-requirement.recommendation` | |
| `threatMitigations`/`threatScenario.relatedControls` | `threats[]` (MultiEntryMapping → threat catalog) | |
| — | `state` | defaults to `Active` |

Assessment-requirement `applicability` references applicability-groups authored from the
functional units (plus an `all-systems` catch-all), resolved from `compliance.mappings.appliesTo`.

### Risk → `#Risk` (riskcatalog)

| Model `Risk` | Gemara `#Risk` | Notes |
|---|---|---|
| `id` / `title` | `id` / `title` | |
| `statement` | `description` | |
| `likelihood` × `impact` | `severity` | derived: `Low`/`Medium`/`High`/`Critical` |
| `owner` | `owner` (RACI) | mapped to responsible + accountable |
| `threatScenarios` | `threats[]` | |
| `rationale` | `impact` (prose) | |

Risks are grouped under an authored `operational-risk` `#RiskCategory` carrying the
organization's appetite/`max-severity` (the appetite is an authored input, not inferable).

### ThreatScenario → `#Threat` (threatcatalog)

| Model `ThreatScenario` | Gemara `#Threat` | Notes |
|---|---|---|
| `id` / `title` | `id` / `title` | |
| `summary` | `description` | |
| `stride`/`category` | `group` | |
| `appliesTo` | `capabilities[]` (required) | falls back to `CAP-SYSTEM` |
| `attackVectorRef` | `vectors[]` | when it resolves to a vector |

Assessment-only fields (`likelihood`/`impact`/`severity`/`status`/`exploitPath`/`evidence`)
are deliberately **not** placed on the L2 threat — Gemara keeps those in L3 Risk and the
L5 Evaluation Log.

## Validation

- **SDK (in-repo):** `go test -run TestGemara ./...` builds every artifact, loads it back
  through `gemara.Load[T]`, confirms `gemara.DetectType`, and checks structural invariants.
- **CUE schemas:** `scripts/validate-gemara.sh` runs `cue vet` for each artifact against the
  official schema definitions. Requires the `cue` CLI and the Gemara schema module
  (`GEMARA_SCHEMA_DIR=<checkout of github.com/gemaraproj/gemara>`).

The Evaluation Log is validated via the schema and the SDK type discriminator but is not
round-tripped through `gemara.Load`, because `AssessmentStep` is a func type that serializes
to a name string yet cannot be unmarshaled back; the SDK consumes evaluation logs in memory
(e.g. for OSCAL conversion).

## OSCAL: decision

The Gemara SDK converts a Gemara Control Catalog to an OSCAL **Catalog** and a Gemara
Evaluation Log to OSCAL **Assessment Results**. `enggemara --oscal-catalog-out` /
`--oscal-ar-out` expose this bridge.

engmod's hand-written OSCAL **SSP**, **AR**, and **POA&M** (`engoscal`) are **retained**.
They encode system characteristics, POA&M items, and compliance-profile–resolved reviewed
controls with deterministic UUIDs — none of which the Gemara schema represents — so moving
them onto Gemara would lose information rather than simplify. The Gemara→OSCAL path is
therefore additive: it adds a Gemara-sourced OSCAL control catalog (which engmod did not
previously emit) and demonstrates the assessment-results bridge.
