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

Risks are grouped under an `operational-risk` `#RiskCategory`. The model has no risk-appetite
field, so the schema-required `appetite` is set to a default (`Low`) and the category
description says "Risk appetite not set in the model; default applied". The optional
`max-severity` is left unset.

### Policy (L3)

The policy states only what the model records:

| Policy field | Source | When the model does not state it |
|---|---|---|
| `contacts.responsible` / `contacts.accountable` | distinct `owner` actor ids of `risks` and `controlVerifications`, resolved to actor names (architecture `actors`, then catalog `actors`) | a single contact named `not set`, plus the export diagnostic `gemara.policy_contacts_not_set` |
| `adherence.evaluation-methods` | `EM-INTENT` (automated) when any control verification has method `test`/`automated`; `EM-MANUAL` (manual) for analysis, inspection or review | omitted |
| `adherence.assessment-plans[*]` | one per control verification, with that verification's method | `frequency: not set in the model` |
| `adherence.enforcement-methods` | `ENM-REMEDIATION` (manual) when the model has `poamItems` | omitted; no automated change gate is claimed |
| `implementation-plan.*.notes` | — | `not set in the model` |

No accepted method is marked `required`: the model does not say which methods the policy owner
mandates. `metadata.author` stays the generator identity (`--author`, default
`Engineering Model`, typed `Software`; `GemaraExportOptions.AuthorType` overrides the type); it
is never used as a policy contact.

`risks` follows the model risk `status`: `mitigated` lists only risks whose status is `closed`;
`accepted` lists risks whose status is `accepted` or whose response is `accept`. Open and
mitigating risks are still being treated. Gemara `Risks` has only these two lists and no notes
field, so they are left out of both and the export diagnostic `gemara.policy_risks_untreated`
names them.

### POA&M item → Enforcement Log action (L6)

Each action's `disposition` follows the POA&M item's `status`: `completed` is `Enforced`;
`accepted` (or `risk-accepted`, `accept`, `waived`, `tolerated`) is `Tolerated`; `planned`,
`in-progress`, `deferred` or no status is `Undetermined` (a value the go-gemara SDK
serializes and accepts), because no enforcement outcome exists yet. The log's aggregate
`disposition` is `Undetermined` while any action is, otherwise `Enforced` if any action is,
otherwise `Tolerated`.

Each action cites the policy's `ENM-REMEDIATION` method. Its `justification.assessments` cite
the Evaluation Log entries (control verifications) whose `riskRefs` include the POA&M item's
`riskRef`, with the verification's result. A POA&M item without such a verification cites no
assessment (empty list) and raises the export diagnostic `gemara.enforcement_unjustified`; no
entry id is invented. Missing milestone, due date, status or responsible role are rendered as
`not set`.

Mapping-reference ids for the Policy and the Evaluation Log are those artifacts' own
`metadata.id` (`<model-id>-POLICY`, `<model-id>-EVALUATION-LOG`), so references resolve within
the rendered set.

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

- **Cross-references (in-repo):** `CheckGemaraCrossReferences` resolves, across the rendered set
  (including the Evaluation Log), every `reference-id` against the document's declared
  `mapping-references` and the rendered artifacts' `metadata.id`, and every `entry-id`,
  `entries[].reference-id` and mapping-document `source`/`targets[].entry-id` against the ids in
  the target artifact. The MCP tool `gemara.validate` returns `crossReferenceErrors` and sets
  `valid: false` when any reference does not resolve or any artifact is not SDK-recognized
  (`allValid` keeps the type-discriminator result alone).
- **Model consistency (in-repo):** `CheckGemaraModelConsistency` compares the rendered claims
  with the model: each enforcement action's disposition and the aggregate must follow the POA&M
  statuses as above, and a risk listed as `mitigated` must be `closed` and one listed as
  `accepted` must be accepted. `gemara.validate` returns these as `modelConsistencyErrors` and
  sets `valid: false` when there are any.
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
