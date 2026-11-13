# engineering-model-go

`engineering-model-go` is a Go library and CLI for generating architecture views and AsciiDoc documentation from a typed engineering model.

It combines:
- architecture model loading and validation
- viewpoint projection (architecture-intent/communication/deployment/security/traceability)
- Mermaid rendering
- Structurizr DSL export
- Threat Dragon/Open Threat Model JSON export
- TRLC requirements export
- LOBSTER activity trace export
- design + requirement narrative generation to AsciiDoc
- EARS requirement preflight linting
- code trace mapping (Go, Python, JavaScript, TypeScript, Rust, Verilog modules)

YAML remains the canonical authored and persistence format, with CUE as the
authoritative schema and cross-field constraint layer. The canonical YAML/CUE
model is a semantic superset of the pinned formal SysML 2.0/KerML 1.0 abstract
syntax, with typed `Engineering::` extensions. This is evidenced by exhaustive
inventory coverage, official-parser validation, and semantic project-interchange
round trip. It does not claim execution/simulation semantics or graphical
concrete syntax. SysML textual DSL is generated output, not another source
model. See
[`docs/sysml-v2-superset-plan.md`](docs/sysml-v2-superset-plan.md).

Each model directory has one schema-v2 manifest entry point and eight required
domain documents. An optional ninth `documents.aviation` document adds an
airborne software evidence-readiness profile without changing models that omit it:

```yaml
schemaVersion: 2
module:
  path: models.example/example-system@v0
  version: v0.1.0
  modelId: example-system
  title: Example System
  introduction: Example schema-v2 model.
  kind: system
documents:
  catalog: model/catalog.yml
  requirements: model/requirements.yml
  architecture: model/architecture.yml
  behavior: model/behavior.yml
  assurance: model/assurance.yml
  compliance: model/compliance.yml
  views: model/views.yml
  decisions: model/decisions.yml
  # aviation: model/aviation.yml
dependencies: []
publications:
  - id: public
    architecture: [FU-EXAMPLE]
    requirements: [REQ-EXAMPLE-001]
inferenceHints: {}
```

`engmod.yml` is the only model entry point. Every manifest and domain document
uses `schemaVersion: 2`; paths are explicit and no companion filename is
inferred. AI agents should call the no-argument MCP tool
`model.authoringContract` first; it returns the resolved
document set, schema files, top-level YAML fields, stable-ID conventions,
dependency/source rules, and recommended editing order without requiring a broad
repository scan, plus `rules`: enum values, relationship pairs, field rules, the
fields each entity kind accepts (`fieldSets`) and their YAML types (`fieldTypes`,
for example `threatScenario.stride: string`, `preconditions: list<string>`,
`flow.steps: list<object> of flowStep`). Validate each edited document with
`model.diagnose`, or edit one entity with `model.patch`.

## Scope

This project models development-state architecture inputs from YAML and produces deterministic documentation/view artifacts.

It is not a runtime observability or incident/compliance runtime system.

## Features

- CUE-backed, closed-schema YAML loading for architecture, catalog,
  requirements, decisions, and design documents, followed by strict Go decoding
- format-neutral document definitions and document-control metadata for downstream
  formal-document generators
- model validation for IDs, references, relations, and viewpoint configuration
- expanded mapping relation taxonomy for communication/deployment/security/traceability/lifecycle semantics
- catalog-linked architecture relation labeling
- deployment view extraction from:
  - Terraform (`.tf`) using HashiCorp HCL parsing
  - Flux resources (GitRepository/Kustomization/HelmRelease)
  - Helm chart metadata via Helm SDK chart loader
- deterministic Mermaid view rendering
- canonical semantic projection with stable IDs, typed relationships, ownership,
  source metadata, and explicit unsupported/lossy diagnostics
- deterministic SysML v2 textual projection and executable coverage manifest
- optional DO-178C/ED-12C evidence-readiness modeling and deterministic draft
  package generation; this is not a compliance or certification determination
- AsciiDoc architecture generation with chapter scope diagrams
- view-level filtering controls (`includeKinds`, `excludeKinds`, `includeMappings`, `excludeMappings`, `maxDepth`)
- EARS preflight linting via `github.com/labeth/ears-lint-go`
  - `lintRun.mode` is required (`lint.mode_missing`): `strict` is the project mode from tier mvp
  - `guided` is allowed only below mvp, for drafting (`lint.mode_guided_at_tier` from mvp)
- Tree-sitter based code symbol extraction and trace mapping for:
  - Go
  - TypeScript/TSX
  - JavaScript (`.js`, `.mjs`, `.cjs`): named functions/generators, methods, classes and directly bound function values. Anonymous callbacks and top-level statements are not independent declarations; this does not infer classic-script load order or runtime behavior. Syntax errors and multiple declarations on one line produce errors rather than ambiguous trace credit.
  - Python (`.py`): functions (including async and nested definitions), methods and classes. Decorated declarations retain their definition-line coordinates; multiline decorators do not detach preceding trace comments. Strings and docstrings cannot create trace or ownership markers. Syntax errors fail declaration coverage. Lambdas and top-level script effects are outside this declaration scope. `test_*.py` and `*_test.py` are verification sources, with `not-run` status until result evidence is supplied.
  - Rust
- Lexical Verilog (`.v`) module trace extraction, with one requirement-link block per module. This does not validate HDL syntax, elaborate designs, or expand macros. Unresolved macro invocations that may affect module declarations and malformed module boundaries produce errors. Macros recognized in expression positions inside a module produce warnings requiring independent build evidence; conditional branches are all inspected. The whole module root is scanned; `inferenceHints.exclude` takes paths and globs relative to the module root to leave out: `dir`, `dir/`, `dir/*` and `dir/**` each leave the whole directory out, `*.gen.go` matches within one path segment; any other form (absolute, `..`, `**` not at the end) is `model.invalid_exclude_pattern`. Git-ignored files are left out too; the MCP code tools use the same scope. Linked `tb_*.v` and `*_tb.v` testbenches produce verification links with `not-run` status until result evidence is supplied. Test-source links do not count as production implementation in the trace matrix.

## Installation

```bash
go get github.com/labeth/engineering-model-go
```

The code scanner parses source with tree-sitter, so building the library or its
commands needs cgo and a C compiler: `CGO_ENABLED=0` does not build. For a static
binary (for example a `distroless/static` image), link externally:

```bash
CGO_ENABLED=1 go build -tags netgo,osusergo \
  -ldflags '-linkmode external -extldflags -static' ./cmd/engdoc
```

## Library API

The root package `engmodel` is the only public Go API; every other package is
under `internal/`. It re-exports the model and validation types, so callers do
not import the internal packages:

```go
bundle, err := engmodel.LoadBundle("engmod.yml") // engmodel.Bundle, engmodel.Requirement, ...
if err != nil {
    panic(err)
}
diags := engmodel.SortDiagnostics(engmodel.ValidateBundle(bundle))
if engmodel.HasErrors(diags) {
    // report []engmodel.Diagnostic
}
```

Model: `LoadBundle`, `LoadCanonicalBundle`, `NewCanonicalBundle`,
`ProjectSemanticModel`, `BuildAuthoringContract`, `LoadWorkspace`,
`FindWorkspace`, `FindWorkspaceWith`. Validation: `ValidateBundle`,
`HasErrors`, `SortDiagnostics`, `DedupeDiagnostics`, `AuthoringRules`
(returns `ValidationRules`).

Primary entry points:
- `GenerateFromFile(architecturePath, viewID)`
- `Generate(bundle, viewID)`
- `GenerateAsciiDocFromFiles(architecturePath, requirementsPath, designPath, options)`
- `GenerateStructurizrDSLFromFile(architecturePath)`
- `ProjectSemanticModel(bundle)`
- `GenerateSysMLV2FromFile(architecturePath)`
- `GenerateSysMLV2(bundle)`
- `GenerateAirborneAssuranceFromFile(manifestPath, options)`

For a model that declares `documents.aviation`, generate the readiness package:

```bash
go run ./cmd/engair --model engmod.yml --out-dir generated/do178c \
  --date 2026-09-19 --version v0.1.0
```

The package is visibly marked `DRAFT / NOT A COMPLIANCE OR CERTIFICATION
DETERMINATION`, identifies exact authored baselines and open gaps, and uses only
customer-supplied opaque objective references. See
`examples/dal-c-flight-control-sample/`.
- `GenerateSysMLV2Projection(semanticModel)`
- `SysMLV2Coverage()`
- `GenerateThreatModelExportFromFile(architecturePath, options)`
- `GenerateTRLCRequirementsFromFile(requirementsPath, options)`
- `GenerateLobsterActivityTraceFromDir(testsDir, options)`
- `BuildTraceMatrixFromFiles(architecturePath, requirementsPath, options)`

Example:

```go
res, err := engmodel.GenerateFromFile("examples/payments-engineering-sample/engmod.yml", "VIEW-ARCHITECTURE-INTENT")
if err != nil {
    panic(err)
}

fmt.Println(res.Mermaid)
for _, d := range res.Diagnostics {
    fmt.Printf("%s [%s] %s\n", d.Code, d.Severity, d.Message)
}
```

View IDs are free-form, but view `kind` must be one of:
- `architecture-intent`
- `communication`
- `deployment`
- `security`
- `traceability`
- `state-lifecycle` (optional)
- `interaction-flow` (optional)

Optional view projection controls in `model/views.yml`:

- `includeKinds` / `excludeKinds`: node kind filtering
- `includeMappings` / `excludeMappings`: relation type filtering
- `maxDepth`: bounded traversal from roots (`0` means unlimited/default behavior)
- `audience`, `abstraction`: optional publication metadata

Formal-document consumers can select canonical model content without adding
format-specific payloads:

```yaml
documents:
  - id: DOC-SYSTEM-SPECIFICATION
    title: System Specification
    kind: system-requirements-specification
    purpose: Define the system requirement baseline.
    audience: [systems-engineering, verification]
    stakeholderRefs: [ACT-SYSTEMS-ENGINEER]
    referenceRefs: [REF-SOURCE-SPECIFICATION]
    contentRefs: [REQ-SYSTEM-001, FU-CONTROL, VIEW-LOGICAL]
    sections:
      - id: scope
        title: Scope
        narrative: Defines the system boundary and intended use.
        includeRefs: [FU-CONTROL]
    control:
      identifier: SYS-SPEC-001
      revision: "1.0"
      status: draft
      issuedBy: Example Organization
      issueDate: 2026-09-20
      language: en
      documentType: specification
      confidentiality: internal
      securityClassification: unclassified
      exportControlled: false
      countryOfOrigin: Sweden
      confidentialityStamp: false
```

Requirements may additionally declare `title`, `category`, `rationale`,
`sourceRefs`, `verificationMethods`, `verificationCriteria`, `priority`,
`criticality`, `status`, `derived`, `derivedRationale`, and `tags`. These fields
remain generic model input; exporters use only fields that have a native
representation in their target format.

Authored architecture optionally supports additional first-class entities:

- `interfaces`
- `dataObjects`
- `deploymentTargets`
- `controls`
- `trustBoundaries`
- `states`
- `events`
- `flows` (with ordered `steps`)
- `hardwareItems` (physical items with DO-254 DAL safety levels, part numbers, suppliers)
- `hardwareInterfaces` (HW/SW interface contracts: ICD/IRS with buses such as `arinc429`, `can`, `spi`, `i2c`, `ethernet`, `cellular`)

Structural composition entities in `model/architecture.yml`:

- `contract` (the system's `provides`/`requires` public interface contract)
- `composition.subsystems` (downward references to child system models)
- `composition.allocations` (parent-requirement allocation onto a subsystem or hardware item)
- `composition.satisfactions` (records of how a subsystem's required interface is satisfied by a provider)

`engdoc` renders HW/SW interface allocation views from `hardwareItems`, `hardwareInterfaces`, and
`composition.allocations`. See System-of-Systems Composition for the composition gates.

Flow step fields (`model/behavior.yml` `behavior.flows[].steps[]`) support:

- `id`, `ref`, `kind`, `action`
- `dataIn`, `dataOut`
- `next`, `onError`
- `async`, `optional`

Assurance authoring fields in `model/assurance.yml`:

- `controls`
- `risks`
- `poamItems`

OSCAL catalog/profile integration is authored in `model/compliance.yml`:

- `compliance.profiles[]`: OSCAL profile/catalog sources for the selected baseline
- `compliance.mappings[]`: maps local `CTRL-*` implementation controls to selected OSCAL `controlIds` and architecture `appliesTo` entities; unmapped controls remain local model controls and are not exported as OSCAL implementations

Expanded mapping relation vocabulary includes:

- Base: `contains`, `depends_on`, `interacts_with`, `targets`
- Communication/data: `calls`, `publishes`, `subscribes`, `reads`, `writes`, `streams`
- Traceability/deployment: `implements`, `satisfies`, `allocated_to`, `verified_by`, `deployed_to`
- Security/guards: `mitigated_by`, `bounded_by`, `guarded_by` (a machine or machine state is protected by a control, trust boundary or referenced element)
- Flow projection (view-only): `flow_next`, `flow_error`, `flow_async`, `flow_ref`
- State lifecycle projection (view-only): `transition` (state to state, label `TR-ID on EVT [guard] / effect`), `on_event` (state to the event a transition from it waits for)

`bounded_by` links a functional unit, deployment target, interface, actor, hardware item or data object to a trust boundary. Default security views retain these member kinds. Membership describes the modeled boundary; it does not establish authentication or enforcement. Author an explicit relationship when it should appear in the projected view.

Optional per-view publication metadata (in `model/views.yml`):
- `authoredStatus` (for example `draft`, `in-review`, `stable`)
- `authoredStatusExplanation` (short rationale shown in Document Health Snapshot)

Verification metadata is inferred from test artifacts:
- test sources under `tests/` (for inferred verification checks and test code element links)
- result artifacts under `test-results/` (for inferred requirement-level outcomes)
- requirement IDs in test sources are inferred from `TRLC-LINKS: REQ-*` markers
- requirement IDs in result artifacts are inferred by matching `REQ-*` tokens in result content
- published verification chain is `Verification Check -> Test Code Element -> Requirement`
- functional ownership in verification tables is shown as derived context from requirement ownership
- strict EARS lint also warns when catalog terms (systems, actors, events, states, features, modes, conditions, data terms) are referenced nowhere in the model (`catalog.term_unreferenced`): a term counts as referenced when its id is used by a model document or its name or an alias appears in requirement text, notes, verification criteria or rationale, another catalog entry's name or definition, or any architecture, behavior, assurance, compliance, views or decisions text
- `lint.multiple_responses` is not reported for one response verb whose conjunctions all sit in a trailing `with`, `including` or `of` field list (`shall publish one world state message with UTC time, latitude, longitude and altitude`); a conjunction before the list, a second `shall`, a `then` in the list, or a list item that reads as a verb with its object (`… with the timestamp and notify the operator`) is still reported
- a risk with `status: accepted` or `response: accept`, and a threat scenario with `status: accepted`, carries the owner's acceptance record `acceptedBy: {actor: ACT-…, date: YYYY-MM-DD, statement: "<the owner's words>"}` (or `decisionRef: DEC-…` naming an accepted decision whose context quotes the owner; without `statement` that is the warning `risk.acceptance_by_decision_only`); without it `risk.acceptance_unrecorded` is an error at `acceptedBy`, and an actor, date or decisionRef that does not resolve is `model.invalid_acceptance`. The Gemara policy lists a risk under `risks.accepted` only with a complete, resolving record and status `accepted`, or response `accept` with a status other than `open`. The record is a claim in the model: the review of the change that adds it is what shows the owner gave it. Authored risk and threat scenario IDs must not contain `::`, which is reserved for imported `alias::ID` entities
- a control verification names its tests in `verificationRefs`: inferred `CODE-<PATH>--<NAME>` IDs of test declarations or `REQ-*` IDs whose linked tests verify it. `verification.status_stale` warns at the `status` of a control verification that is unset or `not-run`, and of a threat mitigation that is unset or `planned`, while those tests exist in the code; the status stays authored

### Model link and acceptance diagnostics

Beyond the schema and reference errors, validation reports model content that is declared
but not wired. Warnings do not fail `engdoc`; the rows marked (error) do:

| Code | Reported at | Rule |
| --- | --- | --- |
| `behavior.event_unused` | `authoredArchitecture.events[i]` | a behavior event no machine transition names in `on`, no semantic transition lists as a trigger and no semantic relationship, flow or flow step references |
| `behavior.state_unreachable` | `authoredArchitecture.machines[i].states[j]` | a machine state that no path of transitions leads to from the machine's `initial` state |
| `behavior.machine_deadlock` | `authoredArchitecture.machines[i].states[j]` | a state that is not `final: true` and has no outgoing transition |
| `behavior.transition_nondeterministic` | `authoredArchitecture.machines[i].transitions[j]` | the transition leaves a state on the same `on` event (or both have none) as an earlier transition and neither has a guard or both have the same guard |
| `behavior.transition_no_trigger` | `authoredArchitecture.machines[i].transitions[j]` | a transition with neither `on` nor `guard` |
| `behavior.invariant_unlinked` | `authoredArchitecture.machines[i].invariants[j].requirements` | an invariant that lists no existing `REQ-*` |
| `behavior.transition_unknown_event` (error) | `…machines[i].transitions[j].on` | `on` names no `behavior.events` entry |
| `behavior.expression_unknown_variable` (error) | `…transitions[j].guard`, `.effect`, `…invariants[k].expr` | the expression reads or assigns a variable the machine does not declare |
| `behavior.expression_type` (error) | `…transitions[j].guard`, `.effect`, `…invariants[k].expr` | an operand or result type the operator table rejects: a guard or invariant that is not Boolean, `<`, `<=`, `>`, `>=`, `+`, `-`, `*` on a non-Integer, `==` or `!=` on different types, `and`, `or`, `not`, `implies` on a non-Boolean, or `:=` of a value whose type is not the variable's |
| `behavior.in_state_foreign` (error) | `…transitions[j].guard`, `…invariants[k].expr` | `in_state(STATE-…)` names a state of another machine or no state |
| `behavior.final_state_has_transitions` (error) | `…machines[i].transitions[j].from` | a transition leaves a `final: true` state |
| `behavior.flow_step_no_onerror` | `authoredArchitecture.flows[i].steps[j].onError` | a step with `boundaryCrossing: true` and no `onError` |
| `catalog.name_has_operator` | `catalog.<group>[i].name` or `.aliases[j]` | an actor, event, state, feature, mode, condition or data term name or alias with a standalone `and`, `or`, `not`, a comma or a parenthesis, which the EARS clause parser reads as operators |
| `verification.status_stale` | `…controlVerifications[i].status`, `…threatMitigations[i].status` | the referenced tests exist but the status still says they have not run (see above); reported by `engdoc` and `engtrace` |
| `verification.benchmark_only` | `requirements[i]` | the requirement's `verificationCriteria` contain a number and every test linking it is a benchmark (Go `Benchmark*`, Rust `#[bench]`): a benchmark asserts nothing, add a `Test*` that checks the value; the trace row has `benchmarkOnly` |
| `threat.verification_missing` | `…threatScenarios[i].verificationRefs` | the threat does not list a control verification that one of its mitigations references or that names the threat in `threatScenarioRefs` |
| `mitigation.verification_control_mismatch` (error) | `…threatMitigations[i].verificationRefs[j]` | the referenced control verification's `controlRef` is another control than the mitigation's |
| `verification.unlinked` | `…controlVerifications[i].verificationRefs` | a `method: test` verification with status `pass` or `partial` and no `verificationRefs`, so no test or requirement evidences it |
| `risk.acceptance_unrecorded` (error) | `…risks[i].acceptedBy`, `…threatScenarios[i].acceptedBy` | an accepted risk or threat scenario without a complete `acceptedBy` record (the message lists the missing parts) |
| `model.invalid_acceptance` (error) | `…acceptedBy.actor`, `.date`, `.decisionRef` | the actor is no actor, the date is not a YYYY-MM-DD calendar date, or the decisionRef is no decision or a decision whose status is not `accepted` |
| `risk.acceptance_not_owner` | `…risks[i].acceptedBy.actor` | the accepting actor is not the risk's `owner` (only when an owner is set) |
| `risk.acceptance_by_decision_only` | `…risks[i].acceptedBy.statement`, `…threatScenarios[i].acceptedBy.statement` | the record names a `decisionRef` but has no `statement`: add the owner's own words accepting this risk (a decision can bundle the acceptance with other choices) |
| `risk.accept_while_open` | `…risks[i].status` | `response: accept` with `status: open`: the risk still waits for a decision, so the Gemara policy does not list it as accepted |
| `risk.severity_inconsistent` | `…threatScenarios[i].severity` | the scenario's (likelihood, impact) pair is rated with another severity by another threat scenario of the model; apply one scoring method |
| `controlVerification.risk_control_unlinked` | `…controlVerifications[i].riskRefs[j]` | the verification names a risk whose `relatedControls` omit the verification's `controlRef` (trace both ways) |
| `threat.attack_vector_missing` | `…threatScenarios[i].attackVectorRef` | the scenario names no attack vector while other threat scenarios of the model name theirs |
| `gemara.control_needs_review` | `…controls[i]` | the Gemara evaluation log marks the control Needs Review: no control verification names it (the message names another control's verification when that is its only evidence), or its verifications need review (for example `partial`) and none failed |

State behavior is modeled with `behavior.machines` only (v0.11.0, ADR-EMG-037, ADR-EMG-038).
Every state belongs to one machine and keeps its `STATE-*` ID. The four advisory machine rows
above (`machine_deadlock`, `transition_nondeterministic`, `transition_no_trigger`,
`invariant_unlinked`) and `state_unreachable` follow the hygiene row of the tier table; the
structural errors marked (error) are errors at every tier. A missing or unresolved
initial state (initial is a single field) and expression syntax errors are reported by projection as
`semantic.invalid_machine` and `semantic.invalid_expression`. A `behavior.states` list and the
relationship types `transitions_to` and `triggered_by` are rejected when the model loads (see the
changelog for the rewrite):

```yaml
behavior:
  events:
    - {id: EVT-PAY-REQUESTED, name: Payment requested}
  machines:
    - id: SM-PAYMENT
      owner: FU-PAYMENTS
      initial: STATE-PAY-IDLE
      variables:
        - {name: attempts, type: Integer, initial: 0}
      states:
        - {id: STATE-PAY-IDLE, name: Idle}
        - {id: STATE-PAY-AUTH, name: Authorizing}
        - {id: STATE-PAY-DONE, name: Done, final: true}
      transitions:
        - {id: TR-PAY-START, from: STATE-PAY-IDLE, to: STATE-PAY-AUTH, on: EVT-PAY-REQUESTED, guard: "attempts < 3", effect: "attempts := attempts + 1"}
        - {id: TR-PAY-FINISH, from: STATE-PAY-AUTH, to: STATE-PAY-DONE, on: EVT-PAY-REQUESTED, guard: "attempts >= 3"}
      invariants:
        - {id: INV-PAY-BOUND, requirements: [REQ-PAY-001], expr: "attempts <= 3"}
  relationships:
    - {type: guarded_by, from: STATE-PAY-AUTH, to: CTRL-PAY-GATE}
```

A `state-lifecycle` view is driven by machines: a machine ID as root selects all of its states,
its transitions are `transition` edges and the events they wait for `on_event` edges.

### Code trace markers

Source files link back to the model with two comment markers, scanned under `--code-root`:

- `TRLC-LINKS:` lists requirement IDs (`REQ-*`) implemented or exercised by the code
- `ENGMODEL-LINKS:` lists model element IDs (functional units, interfaces, controls, flows,
  data objects, etc.) realized by the code

Both markers are validated against the loaded model. Unresolved IDs are hard errors
(`code.dangling_requirement_link` for `TRLC-LINKS`, `code.dangling_model_link` for
`ENGMODEL-LINKS`), enforced by the `engdoc` zero-error gate and by `engtrace` (non-zero exit).
Container build files (`Dockerfile`, `Containerfile`, `*.Dockerfile`, `*.Containerfile`) may
carry `# TRLC-LINKS:`, `# ENGMODEL-LINKS:` and `# ENGMODEL-OWNER-UNIT:` comment lines, and
Markdown files the same markers in HTML comments (`<!-- TRLC-LINKS: REQ-… -->`) outside code
fences (a fence closes only on one at least as long, CommonMark) and outside lines indented four
or more columns. The markers apply to the whole file, which implements (never verifies) the
linked requirements under its `CODE-<PATH>` identifier; they are optional, so an unmarked
Dockerfile or README raises nothing. Only regular files up to 512 KiB are read; larger files and
symlinks are skipped.

A Go test is *environment gated* when it skips depending on an environment variable or
`testing.Short()`: one of its first three statements is an `if` whose condition or initializer
calls `os.Getenv`, `os.LookupEnv` or `testing.Short()` and whose body calls `t.Skip`, `t.Skipf`
or `t.SkipNow`. A gate in a later statement or in a helper the test calls is not detected, so
such a test counts as ungated.
Code is scoped to its nearest enclosing model root, so links resolve against the closest model
rather than the whole tree. Requirements with no implementing or delegating code surface a
`requirement.orphan` warning, and a requirement linked from code that does not exist raises
`requirement.internal_link`.

## CLI Usage

Plan, inspect, and atomically apply a requirements delta:

```yaml
# requirements-delta.yml
version: 1
requirements:
  add:
    - id: REQ-PAY-011
      text: "When payment authorized event is received, the payments system shall persist authorization record."
      notes: "New audit behavior."
      appliesTo: [FU-PAYMENT-AUTHORIZATION]
  update: []
  remove: []
```

```bash
go run ./cmd/engchange validate --root examples/payments-engineering-sample --delta requirements-delta.yml
go run ./cmd/engchange diff --root examples/payments-engineering-sample --delta requirements-delta.yml
go run ./cmd/engchange apply --root examples/payments-engineering-sample --delta requirements-delta.yml
```

Delta version 1 supports requirement `add`, whole-entity `update`, and `remove` operations.
Planning and diffing never write canonical files. Apply validates the complete candidate model,
preserves unchanged YAML nodes and comments, and atomically replaces `model/requirements.yml` only
when no blocking diagnostics are present.

Render OpenSpec spec delta text from model requirements (also
`engmodel.RenderSpecDelta`); each `REQ-…:` line starts with the model text, so a
pasted spec stays consistent with the model:

```bash
go run ./cmd/engspec --model examples/dal-c-flight-control-sample/engmod.yml \
  --prefix REQ-YD- --capability yaw-damping
```

Generate a single Mermaid view:

```bash
go run ./cmd/engview \
  --model examples/payments-engineering-sample/engmod.yml \
  --view VIEW-DEPLOYMENT \
  --out out.mmd
```

Generate every authored view as deterministic Mermaid and native SVG:

```bash
go run ./cmd/engview \
  --model examples/coffee-appliance-six-view/engmod.yml \
  --out-dir examples/coffee-appliance-six-view/generated
```

The coffee appliance fixture demonstrates use-case, logical, process, physical,
implementation, and deployment concerns from canonical schema-v2 semantics.
Its process output is a Mermaid sequence diagram, and every SVG includes an
accessible title plus source and view provenance.

Generate architecture AsciiDoc:

```bash
go run ./cmd/engdoc \
  --model examples/payments-engineering-sample/engmod.yml \
  --requirements examples/payments-engineering-sample/model/requirements.yml \
  --design examples/payments-engineering-sample/model/views.yml \
  --code-root ./src \
  --view VIEW-ARCHITECTURE-INTENT \
  --view VIEW-TRACEABILITY \
  --decisions-out examples/payments-engineering-sample/generated/DECISIONS.adoc \
  --out examples/payments-engineering-sample/generated/ARCHITECTURE.adoc
```

`--view` is repeatable to scope the document to specific viewpoint IDs (omit to include all
configured views). When `--decisions-out` is set, `engdoc` emits the architecture decision
records (from `model/decisions.yml`) as a separate `DECISIONS.adoc` document.

Each authored view and each requirement coverage graph is published as one complete
diagram. Large diagrams scale to fit the page instead of being split into repeated
panels. Standalone Mermaid and SVG exports preserve the same complete graph and
can be viewed at a larger scale.
Deployment publications shorten relationship labels longer than 32 characters to
numbered arrows and retain their full text in a table directly below the diagram.
Extra routing space separates keyed edges; standalone graph exports retain the
original labels. Browser and requirement flowcharts use wider spacing to reduce
label collisions without removing nodes or relationships.

Generate the machine-readable traceability matrix:

```bash
go run ./cmd/engtrace \
  --model examples/payments-engineering-sample/engmod.yml \
  --requirements examples/payments-engineering-sample/model/requirements.yml \
  --code-root examples/payments-engineering-sample/src \
  --format json \
  --out examples/payments-engineering-sample/generated/TRACE-MATRIX.json
```

Use `--format csv` for a spreadsheet-friendly matrix instead. The emitted matrix rolls up
per-requirement status (`implemented`/`verified`/`delegated`/`orphan`) together with code
references and delegations. `engtrace` exits non-zero when it finds dangling code trace links
(`code.dangling_requirement_link` / `code.dangling_model_link`), so it doubles as a CI gate.

Generate Threat Dragon/Open Threat Model exports:

```bash
go run ./cmd/engdragon --model examples/payments-engineering-sample/engmod.yml --format threat-dragon-v2 --out examples/payments-engineering-sample/generated/threat-dragon-v2.json
go run ./cmd/engdragon --model examples/payments-engineering-sample/engmod.yml --format open-otm --out examples/payments-engineering-sample/generated/open-threat-model.json
```

Generate Structurizr DSL:

```bash
go run ./cmd/engstruct --model examples/payments-engineering-sample/engmod.yml --out examples/payments-engineering-sample/generated/STRUCTURIZR.dsl
```

Generate the deterministic SysML v2 textual projection:

```bash
go run ./cmd/engsysml \
  --model examples/payments-engineering-sample/engmod.yml \
  --out examples/payments-engineering-sample/generated/architecture.sysml
```

Author NATO Architecture Framework 4.1 metadata in `model/views.yml`
without duplicating the canonical architecture graph:

```yaml
naf:
  framework: NAF
  version: "4.1"
  architectureDescription: Example NAF Architecture Description
  stakeholders:
    - actorRef: ACT-ARCHITECT
      role: Owns architecture decisions.
  concerns:
    - id: NAF-CONCERN-STRUCTURE
      name: System structure
      stakeholderRefs: [ACT-ARCHITECT]
  products:
    - id: NAF-P2-RESOURCE-STRUCTURE
      viewpoint: P2
      title: Physical resource structure
      viewRef: VIEW-DEPLOYMENT
      concernRefs: [NAF-CONCERN-STRUCTURE]
```

Each NAF product references an existing Engmod view. Capability, service,
logical, resource, interface, behavior, and requirement concepts therefore
remain represented once in the canonical SysML/KerML-aligned model. Generate
the deterministic NAF architecture document with:

```bash
go run ./cmd/engnaf \
  --model engmod.yml \
  --out generated/ARCHITECTURE.naf.adoc
```

The profile accepts the official NAF 4.1 viewpoint codes (`C*`, `S*`, `L*`,
`P*`, and `A*`, including roadmap and cross-domain viewpoints). CUE rejects
unsupported framework versions and viewpoint codes; model validation also
checks actor, stakeholder, concern, and Engmod view references. This output is
a framework-aligned architecture description and is not a claim of NATO
certification.

Emit the executable metaclass/property-level coverage manifest:

```bash
go run ./cmd/engsysml --coverage
```

The formal metamodel inventory is separate from that projection report. The
formal `Systems-Modeling/SysML-v2-Release` tag `2026-04` is pinned at commit
`9baca5908ca28b53da085de69336fde48420ea8f`. That release does **not** contain a
normative metamodel XMI/Ecore: its XMI directories are standard-library model
instances, and the release notes explicitly state that their Eclipse XMI is not
fully normative OMG XMI.

The closest official machine-readable metamodel artifact is therefore the Ecore
published by the aligned official Pilot Implementation tag `2026-04`, commit
`20897e3122f2c2f8b29389745f0caaaeb7c6e21a`:

| Origin | Path | SHA-256 |
| --- | --- | --- |
| KerML 1.0 | `org.omg.sysml/model/kerml.ecore` | `62db52cbccf41266ccfa9b3b9cbb4a40384f16ab513f1d8522fd29d9d0fda8e7` |
| SysML 2.0 | `org.omg.sysml.model/src/main/resources/model/SysML.ecore` | `be71998ca0d9bb7e6081a25e8ead93ce49d623a47ca414b30ee8c572d6d0d753` |

Exact immutable URLs and authority notes are tracked in
`internal/export/sysml/metamodel/metamodel-sources.json`. Refresh downloads into the ignored
`.engmod/cache` tree, verifies both hashes, and regenerates the compact inventory
and semantic coverage binding:

```bash
scripts/refresh-sysml-metamodel.sh
scripts/check-sysml-metamodel.sh
```

The offline check uses only tracked artifacts. It inventories exactly 175
metaclasses, 415 owned properties, 13,318 effective properties, 209 inheritance
edges, 328 derived properties, and 66 relationships. The strict binding records
262 mapped entries, 328 derived/implied entries, and zero unimplemented entries.
Textual rendering classifies 54 metaclasses as native-rendered and 121 as
abstract/implicit, with zero missing rules; all 66 relationships are classified.
CI rejects stale inventory hashes, omitted properties or renderer
classifications, partial/unsupported normative statuses, stale coverage JSON,
official-parser failures, and semantic round-trip differences.

Install the pinned official parser wrapper and Sysand KPAR packager into the
ignored `.engmod` tool/cache tree:

```bash
scripts/install-sysml-toolchain.sh
```

The installer verifies the official `2026-04` Pilot Implementation
`jupyter-sysml-kernel-0.59.0.zip` checksum and the platform-specific Sysand
`0.2.1` release checksum. The validator wrapper is checked out at commit
`63abbd9fbc7851dc437d01b2dc07836b919770b8`; GitHub does not publish a separate
source-archive checksum for that commit, so the immutable Git object ID is the
documented verification boundary.

Build and reopen a normative KPAR through Sysand:

```bash
go run ./cmd/engsysml \
  --model engmod.yml \
  --project-out .engmod/validation/sysml/project \
  --kpar-out .engmod/validation/sysml/engineering-model.kpar \
  --sysand .engmod/tooling/bin/sysand

go run ./cmd/engsysml \
  --model engmod.yml \
  --verify-kpar .engmod/validation/sysml/engineering-model.kpar \
  --reopen-out .engmod/validation/sysml/reopened \
  --sysand .engmod/tooling/bin/sysand
```

`engsysml` delegates KPAR creation and reopening to Sysand; it does not create
or label a custom ZIP as KPAR. The generated project contains only native SysML
source; engineering-model data without a native SysML representation is omitted
rather than embedded in descriptions, metadata payloads, or extension bags.
Verification reopens the KPAR and compares its SysML source byte-for-byte with a
fresh deterministic projection.
Run the complete blocking gate with `scripts/validate-sysml.sh`. Local users may
explicitly skip unavailable external tools with
`ENGMOD_SYSML_SKIP_EXTERNAL=1 scripts/validate-all.sh`; CI installs the pinned
tools and never sets that skip.

The exporter preserves engineering-specific concepts and source properties in a
typed metadata payload and reports unsupported, lossy, external-endpoint, and
toolchain-conformance limitations as diagnostics. The generated text and KPAR
are validated by the official parser baseline, but the coverage manifest still
describes a documented subset/projection rather than claiming complete SysML v2
semantic conformance.

Every authored YAML document is validated against the embedded modular schemas
in `internal/model/schema/` before strict Go decoding. CUE is authoritative for document
shape and cross-field constraints, produces path-aware diagnostics, and rejects
unknown fields recursively. A reflection-based contract test detects drift
between CUE fields and the Go runtime API representations. The loader aggregates
the domain documents into one canonical semantic graph for validation and
projection.

Generate TRLC requirements package:

```bash
go run ./cmd/engtrlc --model examples/payments-engineering-sample/engmod.yml --requirements examples/payments-engineering-sample/model/requirements.yml --out-dir examples/payments-engineering-sample/generated/trlc --package PaymentsRequirements
```

Generate LOBSTER activity trace from tests:

```bash
go run ./cmd/englobster --model examples/payments-engineering-sample/engmod.yml --tests-dir examples/payments-engineering-sample/tests --requirements-package PaymentsRequirements --activity-namespace tests --out examples/payments-engineering-sample/generated/lobster/activities.lobster
```

Regenerate all maintained examples with `scripts/generate-examples.sh`.
After generation, `scripts/render-publication-pdfs.sh` renders the self-model
and every example architecture publication using the locally installed
`proven-docs`. Review the rendered pages before release; a successful render
does not establish visual quality. PDF rendering is separate from deterministic
model-export validation.

Render an individual PDF with proven-docs:

```bash
proven-docs render \
  examples/payments-engineering-sample/generated/ARCHITECTURE.adoc \
  --output examples/payments-engineering-sample/generated/ARCHITECTURE.proven.pdf
```

The generated document starts with:
- `Introduction` (from `model.introduction`)
- `Scope and Assumptions`
- `How To Read This Document`
- `Document Health Snapshot` (per-view coverage and confidence)
- `Terms and Definitions` as a compact `Term | Definition` table
  - terms are sorted A-Z by term text
  - term IDs are shown as `Term (ID)` in the term column

Each generated view includes:
- `What This View Answers`
- `Coverage Gaps` and `Recommended Next Evidence Additions`
- view-scoped narratives from `model/views.yml`
  - for `security`, content is organized as attack-vector chapters with per-attack diagrams and related FU sections

The Traceability View includes:
- `Requirement-to-Unit Mapping (Compact)`
- `Verification Result Mapping` (inferred per-requirement outcomes)

The traceability appendix includes:
- `Requirement Details`
- `Verification Inventory` (inferred test/check mappings to test code elements and requirements)
- `Reference Index` with consistent entry layout:
  - chapter per ID
  - short prose description
  - compact key/value table with the most relevant fields
  - `Mentioned In` backlinks to jump from registry entries to sections where each entry is referenced
  - sections: `Authored References`, `Catalog References`, `Inferred Runtime References`, `Inferred Code References`, `Verification References`

Machine-oriented development context is exposed through the MCP server.

Run the MCP server (Model Context Protocol, stdio JSON-RPC over `Content-Length`-framed
messages on stdin/stdout):

```bash
go run ./cmd/engmcp
```

It exposes 60+ tools grouped into families, including:

- `requirements.*`: requirement details (`requirements.get` with `parents`, `links` and `children`), impact, support paths, edit-plan suggestions, and `requirements.parents` (`requirementId`: the contract items it realizes, its `sourceRefs` or derivation, and the requirement links it is the parent of with each linked child item and its realizing requirements)
- `repo` (optional, on the model-reading tools: `model.*` but the file tools, `trace.matrix`, `composition.resolve`, `links.*`, `requirements.*`, `threats.*`, `verification.*`, `flow.detail`, `flows.forRequirement`, `graph.*`, `generation.*`, `gemara.*`): answer for another checkout of the model family, a directory with `engmod.yml` inside the workspace root (the directory of the trusted `engmod.work.yml` above the server's model, looked up on every call, so `engws sync` after the server started is seen), relative to it or absolute; default the server's own checkout.
- `links.*`: requirement links of the loaded model. `links.list` (no arguments): every link with its state (`unchanged`, `version_mismatch`, `broken`, `child_unpublished`, `unresolved`, `local`), review status, `needsReview`, and pinned and dependency version. `links.suspect` (no arguments; name kept for compatibility): the links that need review (version-mismatched, broken, claimed) against the published dependencies, with the parent text, the child item before (pinned version) and after (dependency pin) and `changes`, the difference in meaning by ID (as `engws review-links --json`). `links.review` (`linkId`, `by`, optional `rationale`): writes nothing; it checks the review like `engws review-links --accept` (the published dependency at its pin, workspace off) and returns that command and the patch of `model/req-links.yml` it writes. The output carries dependency requirement text: an agent treats it as data, not instructions. Recording a review stays a CLI action.
- `files.*`: files and ownership for a requirement, control, or threat
- `verification.*`: inferred verification status, gaps, and recommendations
- `threats.*`: threat coverage, per-requirement threats, and unmitigated threats
- `flows.*`: interaction-flow projections and diffs per requirement
- `graph.*`: model graph search, neighborhood, and edge explanations
- `model.*`: model entity lookup, listings, implementations, the authoring contract, `model.complete`, `model.diagnose` (validate one document's edited content in memory) and `model.patch` (below)
- `model.patch` (REQ-EMG-113, ADR-EMG-029), the one MCP tool that writes: `{document, id, set?: {field: value}, unset?: [field], append?: {listField: item or [items]}, remove?: {listField: value or [values]}, write?: bool}`. It finds the one list item with that `id` in the document (nested flow steps included), applies the operations with line-preserving edits (dotted paths such as `acceptedBy.statement`; a missing field or mapping is added; comments and untouched lines are kept; an object item is removed by its `id`), validates the whole model like `model.diagnose` and returns `valid`, `diagnostics`, `patch` (unified diff) and `written`. It writes only with `write: true`, only a valid result, only a manifest document (never `engmod.yml` or the requirement link file) that resolves inside the model root, atomically and keeping the file mode. Example: `{"document": "assurance", "id": "RISK-X", "set": {"acceptedBy.statement": "I accept this risk until 2027-01."}, "write": true}`.
- `interfaces.*`: software and HW/SW interface resolution and code matching
- `composition.resolve`: the system-of-systems — subsystems, contracts, requirement allocations with the delegated subsystem requirement, and composition diagnostics
- `trace.matrix`: summary (`requirements`, `implemented`, `tested`/`testLinked`, `passed`, `delegated`, `orphan`, `dangling`, `gatedOnly` and `gatedOnlyRequirements`: test-linked requirements whose every linked Go test skips depending on an environment variable or `testing.Short` (see Code trace markers), with rows `gated` and `gates`; `verified` is a deprecated alias of `testLinked`; `linksSuspect`, `linksBroken`, `linksClaimed` and `unparented` for requirement links), per-requirement status, `parents` (the contract items a requirement realizes, or its stakeholder `sourceRefs`), `children` (for a system requirement: each linked `alias::CAP-…` and its realizing `alias::REQ-…`), `links` with their state, `satisfiedBy: links` (a system requirement implemented or test linked through its linked child items), code references, and dangling code trace links; `summaryOnly: true` returns only the summary and diagnostics
- `gemara.*`: one tool per OpenSSF Gemara artifact type plus `gemara.validate`

Additional export/validation docs:

- MCP-first agent workflow and tagging contract: `docs/skills/architecture-mcp-workflow.md`
- OSCAL profile/catalog and compliance mapping workflow: `docs/oscal-compliance-mapping.md`
- Threat Dragon export and schema checks: `docs/threat-dragon-export-testing.md`
- Structurizr export and validation: `docs/structurizr-export-testing.md`
- TRLC requirements export and validation: `docs/trlc-export-testing.md`
- TRLC + LOBSTER traceability report pipeline: `docs/lobster-traceability.md`

## Formal behavior verification (engquint)

`engquint` projects each behavior machine (`behavior.machines`) to a deterministic Quint
module and checks its invariants (ADR-EMG-040, ADR-EMG-041, ADR-EMG-042). A model without
machines produces nothing and `engquint check` prints "no machines" and exits 0.

```bash
go run ./cmd/engquint generate --model engmod.yml          # writes generated/quint/<machine>.qnt
go run ./cmd/engquint check    --model engmod.yml          # quint run, once per invariant
go run ./cmd/engquint check    --model engmod.yml --mode both   # also quint verify (Apalache)
```

`check` flags: `--mode run|verify|both` (default `run`), `--quint path`, `--quint-home dir`,
`--max-steps N` (20, at most 1000), `--max-samples M` (10000, at most 1000000), `--seed S` (`1`),
`--verify-steps N` (10, at most 100), `--timeout D` (30m overall; each quint call is limited to 10m and
is killed with its whole process group), `--pass-env a,b` (extra environment names for the quint
processes; they get an allow-listed environment only),
`--out dir` (committed modules, default `generated/quint`), `--results path`,
`--evidence path` and `--no-evidence`. Exit codes: 0 everything passed (warnings allowed);
1 a violated invariant, a stale (drifted) module, a vacuous or erroring run or a type error;
2 a setup error (missing quint, missing Apalache or Java 21 in verify mode, model that does
not load). The report names the violated invariant with its requirements and the
counterexample as the ordered transitions, for example
`INV-PAYMENTS-001 violated: TR-PAYMENTS-001 -> TR-PAYMENTS-003 -> TR-PAYMENTS-001`.

- Tools: `ENGMOD_QUINT` names the quint executable (else `quint` on `PATH`); `QUINT_HOME`
  names the directory with the pinned Apalache for verify mode. `engquint` never downloads a
  tool. `scripts/install-quint-toolchain.sh` installs the pinned Quint (npm 0.33.1) under
  `.engmod/tooling/quint` and Apalache 0.62.4 (sha256-checked) under
  `.engmod/tooling/quint-home/apalache-dist-0.62.4`; verify mode also needs Java 21 or later
  on `PATH`. Run mode needs neither.
- `quint run` always uses `--backend=typescript`: the default Rust backend downloads its
  binary at run time. Verify mode switches Apalache's deadlock check off (a machine with a
  final state has no enabled transition there); a machine that deadlocks in its initial state
  is a `deadlock` warning in run mode, and a trace of length zero is an error because it
  checks nothing. A state no trace reaches is an `unreachable state` warning.
- Drift is stale: `check` compares a fresh in-memory generation with the committed modules;
  any changed, missing or extra generated module fails the check, nothing runs for it and no
  evidence is written for it. Regenerate with `engquint generate` and commit.
- `generated/quint/results.json` records, per module, the sha256 of the `.qnt`, the Quint
  version, backend, mode, bounds and seed, and per invariant the run and verify verdicts,
  trace statistics and the counterexample. It holds no timings or timestamps, so identical
  inputs give identical bytes.
- `test-results/quint-formal.json` is the evidence the trace reads: one entry per
  requirement of an invariant (`pass`, or `fail` for a violated invariant) with
  `method: analysis`, `kind: formal` and the `qntFile` and `qntSha256` it was produced from.
  Modules with drift or an error contribute no entry. When the code scan reads it, a
  result whose module no longer hashes to `qntSha256` is stale: it counts as no evidence and
  yields the warning `verification.formal_stale`.
- `scripts/generate-examples.sh` runs `engquint generate` and `scripts/validate-all.sh`
  runs `generate` and `check --mode both` for every model that has machines (set
  `ENGMOD_QUINT_SKIP_EXTERNAL=1` to skip the check locally; CI installs the pinned tools).

## Required model switches

Everything the system needs in order to function is an error when it is absent, and the error says how to
fix it (ADR-EMG-061: there is no backwards compatibility and there are no silent defaults). Each
diagnostic below carries a `remedy` (JSON field; the CLIs print it on a `fix:` line, MCP rows have a
`remedy` key). The codes are rank-0 core codes: no tier, `lifecycle.raise` or POA&M item changes or
defers them. `LoadBundle` stays lenient so that tools which embed the engine can load an incomplete
model and report it; `ValidateBundle` and every CLI error. `rules.required` of the authoring contract
(v7) lists the same table, and `TestRequiredSwitchesHaveRemedies` keeps the table and its inventory equal.

| Switch | Error code | Remedy |
|---|---|---|
| engmod.yml lifecycle | `lifecycle.missing` | run the ef-tier-up skill (engineering-flow) to choose and record the tier, then write engmod.yml lifecycle {tier, bundles, tierDecision} |
| lifecycle.tier | `lifecycle.invalid_tier` | run the ef-tier-up skill (engineering-flow) to choose and record the tier, then write engmod.yml lifecycle {tier, bundles, tierDecision} |
| lifecycle.bundles | `lifecycle.bundles_missing` | run the ef-tier-up skill (engineering-flow) to choose and record the tier, then write engmod.yml lifecycle {tier, bundles, tierDecision} |
| lifecycle.bundles (value) | `lifecycle.invalid_bundle` | run the ef-tier-up skill (engineering-flow) to choose and record the tier, then write engmod.yml lifecycle {tier, bundles, tierDecision} |
| lifecycle.tierDecision | `lifecycle.tier_decision_missing` | run the ef-tier-up skill (engineering-flow) to choose and record the tier, then write engmod.yml lifecycle {tier, bundles, tierDecision} |
| lifecycle.tierDecision (accepted decision with Tier: and Bundles: lines) | `lifecycle.tier_decision_mismatch` | run the ef-tier-up skill (engineering-flow) to choose and record the tier, then write engmod.yml lifecycle {tier, bundles, tierDecision} |
| engmod.yml requirementLinks | `model.requirement_links_missing` | run the ef-tier-up skill (engineering-flow) to choose the tier, then set requirementLinks: strict in engmod.yml (guided is allowed only below mvp) |
| engmod.yml requirementLinks (value) | `model.invalid_requirement_links` | run the ef-tier-up skill (engineering-flow) to choose the tier, then set requirementLinks: strict in engmod.yml (guided is allowed only below mvp) |
| requirementLinks guided at mvp or above | `lifecycle.requirement_links_conflict` | run the ef-tier-up skill (engineering-flow) to choose the tier, then set requirementLinks: strict in engmod.yml (guided is allowed only below mvp) |
| requirements.yml lintRun.mode | `lint.mode_missing` | run the ef-tier-up skill (engineering-flow) to choose the tier, then set lintRun.mode: strict in requirements.yml (guided is allowed only below mvp) |
| lintRun.mode (value) | `lint.mode_invalid` | run the ef-tier-up skill (engineering-flow) to choose the tier, then set lintRun.mode: strict in requirements.yml (guided is allowed only below mvp) |
| lintRun.mode guided at mvp or above | `lint.mode_guided_at_tier` | run the ef-tier-up skill (engineering-flow) to choose the tier, then set lintRun.mode: strict in requirements.yml (guided is allowed only below mvp) |
| engmod.yml health block (tier product and operational) | `health.block_missing` | run the ef-tier-up skill (engineering-flow) to write the health block with its debt list, then record the baseline with enghealth baseline write |
| schemaVersion 2 in every document | `model.unsupported_schema_version` | set schemaVersion: 2 at the top of the file (edit the file; an upgrade command is planned in engineering-flow and does not exist yet) |
| module.modelId | `model.id_missing` | set module.modelId in engmod.yml to the stable identifier of the model (for example MY-SYSTEM) |

The tier decision named by `lifecycle.tierDecision` must be accepted and carry a `Tier:` line and a
`Bundles:` line; a decision without a `Bundles:` line no longer means Foundation.

## Life-cycle tier

A module declares its life-cycle tier in `engmod.yml` (required since v0.11.0, ADR-EMG-031, ADR-EMG-061). The tier is the
bar for the default branch and every release; work below it happens only in experiments, which the
engine does not model.

```yaml
lifecycle:
  tier: product                      # poc < prototype < mvp < product < operational
  bundles: [foundation, security]    # required, no default; safety adds the DO-178C rules
  tierDecision: DEC-TIER-PRODUCT     # accepted decision with "Tier: Product" and "Bundles: ..." lines
  raise: [behavior.*]                # codes raised to error regardless of tier; never lowers
```

A model without `lifecycle` is the error `lifecycle.missing` (see Required model switches below). The
library still loads such a model and applies no tier rule to it, so that tools can report; the validator
and every CLI treat it as an error. Every public
entry (`ValidateBundle`, `GenerateAsciiDoc`, the trace builders, `engws check`, MCP) applies
`ApplyTierPolicy` once to its merged diagnostics:

- a tier-keyed code gets the severity of its row at the project tier. **D** (deferred) is a
  warning with `deferred: true`: evidence debt that is reported and counted but never fails a check;
  every tier-keyed diagnostic carries `requiredFrom`, the tier from which it is an error;
- an open POA&M item with `milestone: "tier:<project tier>"` and `defers: [{code, path?}]` defers the
  diagnostics it names (warning, `deferred`, `poamRef`); naming a rank-0 code is
  `lifecycle.poam_defers_core` (error);
- `lifecycle.raise` makes the named codes (or `family.*`) errors;
- rank-0 core codes (`CoreCodes()`: `model.duplicate_id`, `model.broken_reference`, `model.invalid_*`,
  `model.missing_*`, `code.dangling_*`, `composition.cycle`, `composition.lock_*`, the link integrity
  codes, `risk.acceptance_unrecorded`, the structural `lifecycle.*` codes, the required-switch codes
  and the AI-credit and authority-evidence aviation codes) are errors at every tier and never deferred, and every code the
  table does not key keeps its severity;
- requirement links are strict from mvp whatever `requirementLinks` says; `requirementLinks: guided`
  at mvp or above is `lifecycle.requirement_links_conflict` (error);
- only rank-0 and other untiered errors stop the AsciiDoc render; an error the tier policy sets fails
  the check, but the document is still produced for reviewers (D15 extended, ADR-EMG-032).

New codes with `lifecycle`: `lifecycle.tier_decision_mismatch`, `lifecycle.requirement_links_conflict`,
`lifecycle.poam_defers_core`, `lifecycle.invalid_tier`, `lifecycle.invalid_bundle` (errors),
`lifecycle.requirement_unapproved` (status draft or proposed), `lifecycle.requirement_unverified`
(status not verified), `verification.failed` (a linked test result failed), `verification.gated_only`
(every linked test is environment gated), the safety bundle's `lifecycle.aviation_profile_missing`,
`lifecycle.software_level_missing`, `aviation.state_below_tier`, `aviation.authority_state_unevidenced`
and `aviation.ai_actor_as_credit` (ADR-EMG-035), and the composition rules
`composition.dependency_below_tier` and `composition.dependency_tier_unknown` (ADR-EMG-034).
To accept a dependency below the system's tier, record a risk with `status: accepted` (or
`response: accept`) and a complete `acceptedBy`, and name the dependency alias from `engmod.yml`
in its `dependencies` field (preferred); naming the composing subsystem in `appliesTo` still works.
A `dependencies` entry that is no declared alias is `model.invalid_risk_dependency` (error):

```yaml
risks:
- id: RISK-ENCODER-TIER
  title: Encoder below tier
  statement: The encoder is still a prototype.
  status: accepted
  owner: ACT-SYSTEM-OWNER
  dependencies: [encoder]
  acceptedBy: {actor: ACT-SYSTEM-OWNER, date: "2026-11-05", statement: I accept the prototype encoder until v1.}
```
`engtrace --tier-gap <tier>` and the MCP tool `tier.gap` list what a target tier would require.

Severity by tier (`TierPolicy()`, `TierPolicyMarkdown()`; D deferred, W warning, E error;
`requirement.no_parent`, `requirement.unsatisfied` and `link.aou_unverified` are errors at every tier
with `requirementLinks: strict`):

| Code | poc | prototype | mvp | product | operational |
|---|---|---|---|---|---|
| `aviation.derived.incomplete_safety_feedback` | D | D | W | E | E |
| `aviation.lifecycle.missing_category` | D | D | E | E | E |
| `aviation.state_below_tier` | D | D | E | E | E |
| `aviation.tool.incomplete_credit_assessment` | D | D | W | E | E |
| `behavior.event_unused` | D | D | W | W | E |
| `behavior.flow_step_no_onerror` | D | D | W | W | E |
| `behavior.invariant_unlinked` | D | D | W | W | E |
| `behavior.machine_deadlock` | D | D | W | W | E |
| `behavior.state_unreachable` | D | D | W | W | E |
| `behavior.transition_no_trigger` | D | D | W | W | E |
| `behavior.transition_nondeterministic` | D | D | W | W | E |
| `code.invalid_trlc_link` | W | W | W | E | E |
| `code.missing_trlc_link` | W | W | W | E | E |
| `code.scan_failed` | W | W | W | E | E |
| `composition.dependency_below_tier` | D | D | W | E | E |
| `composition.dependency_tier_unknown` | W | W | W | W | W |
| `composition.dependency_tier_unknown` with safety | W | W | W | E | E |
| `contract.unrealized` | D | D | W | W | E |
| `health.baseline_stale` | W | W | W | W | E |
| `health.brain_method` | W | W | W | E | E |
| `health.brain_method` with safety | W | W | E | E | E |
| `health.bumpy_road` | W | W | W | E | E |
| `health.bumpy_road` with safety | W | W | E | E | E |
| `health.complex_conditional` | W | W | W | E | E |
| `health.complex_conditional` with safety | W | W | E | E | E |
| `health.complex_method` | W | W | W | E | E |
| `health.complex_method` with safety | W | W | E | E | E |
| `health.deep_nesting` | W | W | W | E | E |
| `health.deep_nesting` with safety | W | W | E | E | E |
| `health.duplicated_assertion_block` | W | W | W | E | E |
| `health.duplicated_assertion_block` with safety | W | W | E | E | E |
| `health.excess_arguments` | W | W | W | E | E |
| `health.excess_arguments` with safety | W | W | E | E | E |
| `health.large_assertion_block` | W | W | W | E | E |
| `health.large_assertion_block` with safety | W | W | E | E | E |
| `health.large_method` | W | W | W | E | E |
| `health.large_method` with safety | W | W | E | E | E |
| `health.policy_weakened` | W | W | W | E | E |
| `health.policy_weakened` with safety | W | W | E | E | E |
| `health.primitive_obsession` | W | W | W | E | E |
| `health.primitive_obsession` with safety | W | W | E | E | E |
| `health.regression` | W | W | W | E | E |
| `health.regression` with safety | W | W | E | E | E |
| `health.waiver_unused` | W | W | W | E | E |
| `health.waiver_unused` with safety | W | W | E | E | E |
| `lifecycle.aviation_profile_missing` (safety bundle) | D | D | E | E | E |
| `lifecycle.requirement_unapproved` | D | D | W | E | E |
| `lifecycle.requirement_unverified` | D | D | D | W | E |
| `lifecycle.software_level_missing` (safety bundle) | D | D | E | E | E |
| `link.aou_unverified` | D | D | E | E | E |
| `link.claimed_only` | D | D | W | E | E |
| `requirement.no_parent` | D | D | E | E | E |
| `requirement.orphan` | D | D | W | E | E |
| `requirement.unsatisfied` | D | D | E | E | E |
| `threat.verification_missing` | D | D | W | E | E |
| `verification.failed` | D | D | D | W | E |
| `verification.gated_only` | D | D | D | W | E |
| `verification.status_stale` | D | D | W | W | E |
| `verification.unlinked` | D | D | W | E | E |

### Code health configuration

The `health:` block (ADR-EMG-051, ADR-EMG-052) is required from tier product (`health.block_missing`,
see Required model switches; an empty block `health: {}` is complete when the project has no debt).
Below product a manifest without one reports nothing about health. The block configures the report and
is checked on load and by `ValidateBundle`:

```yaml
health:
  exclude: [vendor/**, "*.gen.go"]            # the inferenceHints.exclude pattern dialect
  thresholds:                                  # per smell code with numeric limits
    health.complex_method: {warn: 10, high: 15}
  overrides:                                   # different limits for some paths
  - path: internal/legacy/**
    thresholds:
      health.large_method: {warn: 120, high: 240}
  waivers:                                     # accept one finding on an accepted decision
  - {code: health.complex_method, path: internal/legacy/parse.go, function: parseAll, decision: ADR-LEGACY-PARSER}
  baseline:                                    # accept all existing debt on an accepted decision
    {file: model/health-baseline.json, decision: ADR-DEBT-BASELINE}
  policyDecision: ADR-RAISE-LIMITS             # records a weakening of this very block (see below)
```

Every key is optional. `thresholds` and `overrides[].thresholds` accept only the smell codes with
numeric limits (`health.complex_method`, `health.large_method`, `health.deep_nesting`,
`health.bumpy_road`, `health.complex_conditional`, `health.excess_arguments` and
`health.large_assertion_block`); `warn` is at least 1 and `high` at least `warn`. The structural smells
`health.primitive_obsession`, `health.brain_method` and `health.duplicated_assertion_block` take no limits.
`exclude` and `overrides[].path` use the same dialect as `inferenceHints.exclude`: relative to the module
root, `*` `?` `[...]` within one path segment, `**` only as a trailing `/**`, no negation.

A configuration error is `model.invalid_health_exclude`, `model.invalid_health_threshold`,
`model.invalid_health_override` or `model.invalid_health_waiver` (errors at every tier, never deferred).
A waiver names a smell code, a file, a function and a decision. If the decision does not exist or is not
accepted, the waiver is `health.waiver_unjustified` (an untiered error: no tier or POA&M item changes or
defers it) and it waives nothing. The path of a waiver is a file path or a pattern of the exclude dialect.
`health.waiver_unused` (a waiver that matches no finding) is reported by the health report.

Severity follows the table above. The bar for code quality is highest and the same at product and
operational, the two highest tiers, because AI-assisted work needs cleaner code than human-only work to
stay free of bugs (ADR-EMG-057): every smell code and `health.waiver_unused` is a warning up to mvp and
an error at product and operational (from mvp with the safety bundle), and the scan applies the stricter
threshold set at both. `health.regression`,
a new or worsened high finding against the base revision, is a warning at poc, prototype and mvp and an
error from product (from mvp with the safety bundle). `lifecycle.raise: [health.*]` is the escape hatch
that makes the debt codes errors at a lower tier.

**Baseline.** A project that already has debt cannot raise its tier to product, where every finding is an
error, unless it can accept that debt as a whole (ADR-EMG-059). `health.baseline` names a JSON file,
relative to the manifest directory (no absolute path, no `..`), and an accepted decision that justifies
it. The file, `{"schema": "engmod-health-baseline/1", "entries": [{"path", "function", "code", "value"}]}`,
lists the accepted findings by path, qualified function name and smell code with the metric value each
had when the file was written (at least 1), sorted, without timestamps;
`enghealth baseline write --model engmod.yml [--tier t] [--out file] [paths...]` creates it from a scan
that ignores any existing baseline (every finding, or with `--tier` only those that are errors at that
tier; it refuses to write outside the manifest directory and is idempotent), and `--check` exits 0 when the
file is exactly what `write` would produce, 1 when it differs (it prints the stale and the missing
entries, each with its value, so a function whose metric changed is both stale and missing) and 2 on a
setup error. A finding in the baseline is *baselined* while its current value is at most the recorded
one: the report lists it under `baselined`, it still counts toward the score (the score stays honest about
the code), and it is neither an error nor a warning of the summary, so `enghealth scan --fail-on-error`
ignores it. A finding whose value grew past the recorded one is not baselined: it counts at the level of
its code (an error from product) and its message says what the baseline accepted, so small increases
cannot ratchet in silently; a finding that shrank stays baselined and `--check` asks for a rewrite. A
baseline whose decision is missing or not accepted suppresses nothing: the scan does not read the file
and reports `health.waiver_unjustified` (an error at every tier, naming the baseline), which counts for
`--fail-on-error`. An entry that
matches no finding is `health.baseline_stale`, a warning at poc to product and an error at operational
(row `WWWWE`), so the baseline can only shrink; a scan restricted to some paths does not report entries of
files outside them. A regression is never baselined: the delta check compares the working tree with a base
revision and does not read the baseline. The validation errors are `health.waiver_unjustified` (the
decision of a baseline is missing or not accepted; untiered, as for a waiver), `health.baseline_unreadable`
(the file is missing, not valid JSON, not `engmod-health-baseline/1` or holds an invalid entry; untiered,
read by validation through the format of `internal/model`) and `model.invalid_health_baseline` (an empty or
unclean file path, or no decision). A missing or malformed file is a setup error (exit 2) for `enghealth`.

**The policy is defined by the pull request.** A change can raise a threshold, widen `health.exclude` or an
override, or lower the tier, and the weakened configuration then applies to both sides of a delta. So
`enghealth check` also reads the manifest of the *base* revision (`git cat-file`; one that is not there has
the default limits of the tier) and reports `health.policy_weakened` when the head has a limit (warn or
high) above the effective limit of the base for any code (the default of the tier for a code the manifest
does not set), a new exclude pattern, an override that raises a limit, or a lower tier. The code has the row
`WWWEE` (`WWEEE` with the safety bundle) and is evaluated at the stricter of the base and the head tier, so
lowering the tier does not lower the level of its own weakening: it makes `check` exit 1 from product. The
weakening is recorded, and then does not fail the check, when the head's `health.policyDecision` names an
accepted decision that the base manifest did not already name; `decisions.yml` is code-owned, so the
weakening is reviewable. A `policyDecision` that does not exist or is not accepted is
`health.waiver_unjustified` and records nothing. `--tier` applies to both sides.

The scores come from this project's own heuristics over the scanned source (cyclomatic complexity, lines,
nesting depth and similar); they are not CodeScene's Code Health and make no claim to match it. The
authoring contract (`engineering-model.authoring.v6`) documents the block in its `health` section and the
field sets `health` (with `policyDecision`), `healthThreshold`, `healthOverride`, `healthWaiver` and
`healthBaseline`.


## Code health (enghealth)

`enghealth scan` measures the maintainability of the source of a model and writes a report. It
reads Go, Rust, TypeScript and TSX, JavaScript and Python with the tree-sitter grammars of the code
scan. It needs no `health:` block (the block only configures it, see above) and it is separate from
`engdoc`: nothing in validation or generation changes.

**What it measures.** For each function, method and named closure (anonymous closures count toward
the function around them; a named closure or nested function is a unit of its own and its rows
are not counted again in the function around it): cyclomatic complexity, logical lines, nesting depth, bumps (top-level control
structures that contain another control structure, "bumpy road"), boolean operators in one condition, parameters and, in tests, the
longest run of consecutive assertions. Ten smells follow, `health.complex_method`, `health.large_method`,
`health.deep_nesting`, `health.bumpy_road`, `health.complex_conditional`, `health.excess_arguments`,
`health.primitive_obsession`, `health.brain_method`, `health.large_assertion_block` and
`health.duplicated_assertion_block`, each with a `warn` and a `high` limit (ADR-EMG-047). A smell that
cannot be computed for a language (primitive obsession needs known types, so never for JavaScript) is
listed as unsupported in the report and is never a silent zero.

**Whose scores these are.** The definitions, limits, weights and bands are this project's own, built
from public ideas (McCabe complexity, the brain method of Lanza and Marinescu, the bumpy road of
Tornhill's public writing). They are not CodeScene's Code Health and make no claim to match it; no
CodeScene code or weights are used.

**Score.** A file starts at 10.0. Each finding deducts the weight of its code times a severity factor
(warn 1, high 2). The deductions of one function add up; across the functions of the file the worst
function counts fully, the next times 0.5 and every other function times 0.25, but what the functions
beyond the second count adds up to at most 2.0 points in total (scoring version 2, ADR-EMG-059; if it
would be more, those deductions are scaled down together, so the number of mediocre functions in a file
cannot decide its band alone). The result is clamped to 1.0..10.0 and rounded to one decimal. Bands:
green from 8.0, yellow from 4.0, red below. Weights (`scoringVersion` 2; version 1 had the same weights
and no cap): complex_method 1.0, large_method 0.8, deep_nesting 1.0, bumpy_road 1.0,
complex_conditional 0.6, excess_arguments 0.5, primitive_obsession 0.3, brain_method 1.5,
large_assertion_block 0.3, duplicated_assertion_block 0.3. Every score carries the deductions that
produced it. Example: function A with a high complex_method (2.0) and a warn deep_nesting (1.0), function
B with a warn large_method (0.8) and function C with a warn excess_arguments (0.5) count
2.0 + 1.0 + 0.8 x 0.5 + 0.5 x 0.25 = 3.525, so the file scores 6.475, rounded to 6.5, yellow. With the
cap: a file whose worst function deducts 3.0, whose second deducts 2.0 and whose other eight functions
deduct 2.0 each counts 3.0 + 2.0 x 0.5 + min(8 x 2.0 x 0.25 = 4.0, 2.0) = 6.0 and scores 4.0, yellow
(version 1 counted 8.0 and scored 2.0, red); each of the eight is shown with `counted` 0.25 x 2.0 x 0.5
= 0.25 in the deductions, so the counted amounts still add up to what was deducted. The report carries
`scoringVersion`; scores of different versions are not comparable. Test files are scored apart and are
left out of the production roll-ups.

**Roll-ups.** File, functional unit (the LOC-weighted mean of its production files and the worst file,
attributed by the file-level `ENGMODEL-OWNER-UNIT` marker), requirement (the functions that
marker-linked symbols with `TRLC-LINKS` lie in, joined by path and line range; the findings and the
worst file score), and project (LOC-weighted mean, worst file, files per band). A `tests` block reports
the test files.

```bash
go run ./cmd/enghealth scan --model engmod.yml                         # text summary
go run ./cmd/enghealth scan --model engmod.yml --format json --out health.json
go run ./cmd/enghealth scan --model engmod.yml --format sarif --out health.sarif
go run ./cmd/enghealth scan --model engmod.yml --tier operational --fail-on-error internal/health
go run ./cmd/enghealth scan --import other-engine.json --format sarif  # re-render a conforming report
```

Flags: `--model` (engmod.yml), `--format text|json|sarif` (default text), `--out file`, `--tier` (scan at
this life-cycle tier instead of the declared one), `--fail-on-error`, `--import report.json`, `--cache`,
`--no-cache` and paths
(files or directories inside the project root; they must come after the flags). The scan covers the code
scope the other tools use: the module root minus git-ignored files, nested modules, dot directories,
`node_modules`, `vendor`, `inferenceHints.exclude`, `health.exclude` and the default excludes
`generated/**`, `vendor/**`, `node_modules/**`, `.engmod/**` and `.git/**`. A source file above 4 MiB, one
that is not a regular file or cannot be read, one whose path holds a control character, and one the engine
cannot analyse (including syntax nested more than 10,000 levels deep or an analysis that exceeds its 15
second budget) is listed as a `health.file_skipped` or `health.file_failed` diagnostic; so is a directory
the walk cannot read (`health.file_skipped`, with the directory as its path). A file the parser could only
partly read (ERROR or MISSING syntax nodes, counted in `syntaxErrors` of the file, at most 1000) is
analysed as far as the parser recovered it and gets a `health.syntax_error` diagnostic, since the functions
it lost are not measured. When the code scan of the model fails, the unit and requirement roll-ups are
empty and a `health.rollup_unavailable` diagnostic says so (for example one line longer than the scanner
accepts). A file that was not analysed is not a clean file: these diagnostics have the level of
`health.regression` at the tier (computed by the same tier policy: a warning up to mvp, an error from
product and from mvp with the safety bundle), so from product they count in the errors of the summary and
fail `--fail-on-error`. The parse has a 10 second timeout.

**Result cache.** `scan` and `check` cache the result of each file under `.engmod/cache/health` (keyed by
the content, the limits, the scoring version and the engine version), which is what makes a repeated run
fast. The cache is not trusted by default, because its key can be computed from public inputs and a
committed entry could turn a finding into a pass. It is therefore off where the environment variable `CI`
is non-empty unless `--cache` is given, and it is refused for a run (a note on stderr; the run goes on
without it) while `git ls-files .engmod` lists any tracked file. Each entry also carries a structural
digest (sha256 over the key and the canonical JSON of the result), so an entry that was edited or
truncated is a miss. The digest is not a signature and does not stop a forged entry made with the public
key rule, which is why the two rules above exist. `--no-cache` turns the cache off.

**Formats.** `json` is the report `engmod-health/1` (`docs/health-report.schema.json`, JSON Schema draft
2020-12): engine and scoring version, the tier, the level (error or warning) of each code at that tier,
files with scores, deductions and functions, waived findings, roll-ups and a summary. It has no timestamp
and no absolute path, every list is sorted and the encoding is canonical, so two runs on one tree are
byte-identical. `sarif` is SARIF 2.1.0 (one run, driver `enghealth`, one rule per code, one result per
finding, stable `partialFingerprints`; a waived finding is a suppressed result). `--import` loads a report
that another engine wrote, validates it against the schema (another schema string, a missing required
field, an unknown field, a `scoringVersion` other than 1 or 2, a score outside 1.0..10.0, a band that does
not match its score or a path outside the repository is a setup error) and renders it in any format;
with `--tier` or `--model` the levels are recomputed for that tier. The scores of an imported report are
taken as they are and not recomputed; the scoring version that was read is printed on stderr. The levels
of an imported report are not trusted either: `--fail-on-error` with `--import` needs `--model` or
`--tier` (otherwise exit 2). Text output escapes every control character of a path or message
(`\x0a`, `\x1b`), so a file name cannot start a workflow command in a CI log or move a terminal cursor;
`--out` writes through a temporary file and a rename and refuses an existing target that is not a regular
file (a symbolic link, for instance).

**Exit codes.** 0 ok; 1 findings or diagnostics (a skipped or failed file included) that are errors at the
tier, only with `--fail-on-error`; 2 a setup or
usage error (the model does not load, a bad flag, tier or path, an unreadable output file, a report that
does not conform), which is never a pass. `enghealth check` has its own rules (below).

**Tiers.** Severity follows the tier policy of `engdoc` (`ValidateBundle` uses the same table): every
smell code is a warning up to mvp and an error at product and operational (from mvp with the safety
bundle), `lifecycle.raise: [health.*]` makes them errors at any tier, and a project without a tier gets
warnings. At product and operational the scan also applies stricter default limits (complex_method 8 and
15, large_method 50 and 100, deep_nesting 3 and 4, bumpy_road 2 and 3, complex_conditional 3 and 5,
excess_arguments 4 and 6, large_assertion_block 5 and 8; ADR-EMG-059); below product the defaults apply
(10 and 20, 70 and 120, 4 and 5, 2 and 3, 3 and 5, 5 and 7, 6 and 10). A `health.thresholds` entry or an
override still replaces the limits of its code. The strict limits are calibrated by the dogfood report
(`docs/code-health-dogfood.md`); the project owner confirms or moves them.

**Waiving.** To keep a finding at product or operational, add a decision with status `accepted` that
explains why, and a waiver for the code, file and qualified function name (`Type.Method`) in the
`health` block. The finding then appears under `waived` with its decision instead of in the function and
no longer lowers the score. A waiver that matches no finding is reported as `health.waiver_unused`
(an error where the debt is an error), so a stale waiver does not hide debt. A decision that is missing or
not accepted is `health.waiver_unjustified`.

### Gate on regressions: enghealth check

`enghealth check` compares the source files that changed against a base revision and fails only on a
regression, so a project with existing debt can hold new code to the bar at once (ADR-EMG-053,
ADR-EMG-059). A finding is keyed by file, qualified function name and code. It is **new** when it is
absent at base, **worsened** when it is present at base and at head either of a higher severity or with a
metric that grew by at least 20 percent, and **existing** otherwise. A **regression** (`health.regression`)
is a new or worsened finding of severity high. A function that was renamed or moved with the same body is
not new, a deleted file or function produces nothing, and a finding that a waiver covers is left out of
both sides, so it is existing and never a regression (it is counted as waived). The base file is read with
`git show` as raw bytes; nothing is checked out.

```bash
go run ./cmd/enghealth check --model engmod.yml                        # uncommitted work against HEAD
go run ./cmd/enghealth check --model engmod.yml --base origin/main      # the work tree against a branch
go run ./cmd/enghealth check --model engmod.yml --base origin/main...HEAD --format sarif --out regressions.sarif
go run ./cmd/enghealth check --model engmod.yml --base v1.2.0 --head HEAD~1 --tier product internal/health
```

Flags: `--model` (required), `--base REV` (default `HEAD`; the comparison is always with the merge base of
`REV` and the head, as `git diff base...head` does, so a branch that is behind `REV` does not see the later
work of `REV` as its own regressions; `Base` of the report is that merge base and a note says when it differs
from the tip of `REV`; without a common ancestor the tip of `REV` is used, with a note; `base...head` names
both revisions), `--head REV` (default: the working tree, untracked files included),
`--tier`, `--format text|json|sarif`, `--out file`, `--cache`, `--no-cache` and paths (files or
directories inside the project root, after the flags). The limits, excludes, thresholds, overrides and
waivers are those of `scan`, and so is the result cache (see above: `.engmod/cache/health`, off in CI
unless `--cache`, refused while `.engmod` is tracked, off with `--no-cache`). The check
reads only the model, git and the changed files; it does not build the roll-ups, so it takes about 0.3 s
on a one-file change with a warm cache.

**Level and exit codes.** Whether a regression fails the gate follows the tier policy like any other code:
`health.regression` is a warning below product and an error from product (from mvp with the safety bundle;
`lifecycle.raise` is honoured). Exit 0: no regression, or only regressions that are warnings at the
tier. Exit 1: at least one regression that is an error at the tier; each is printed as
`file:line function: code (new|worsened) metric before -> metric after` with its message. Exit 1 also when
a *changed* source file was skipped (above 4 MiB, the same cap for the work tree and for `--head REV`, or
not a regular file) or has syntax errors the change brought in (the parser lost functions) at a tier where a regression is an error, since its findings are unknown, and when the
policy is weakened against the base revision without a recorded `policyDecision` where
`health.policy_weakened` is an error (see above). Exit 2: a setup
error, never a pass: the model does not load, an unknown tier, a path outside the project or a bad
flag or format, the project is not in a git work tree, an invalid revision, or a base revision that is not
available (an unknown name, or a shallow clone that lacks it; the message says to fetch more history, for
example `fetch-depth: 0` in CI) or a manifest at the base that is not YAML. In a repository without a
commit there is no base: every finding is new, and a note says so. The git child gets an allow-listed
environment with the user and system configuration switched off (`GIT_CONFIG_GLOBAL=/dev/null`,
`GIT_CONFIG_NOSYSTEM=1`, so a `safe.directory` setting does not apply), the hooks, the filesystem monitor,
external diff and textconv drivers disabled, and at most 64 KiB of its stderr kept; a file name with a
control character is refused (exit 2).

**Formats.** `text` is a short summary (the changed files with score and band before and after, the
regressions, the counts of new, worsened, existing and waived findings, the verdict). `json` is the delta
report `engmod-health-delta/1` (base and head commits, notes, files, all entries, the blocking entries and
the skipped files) plus `levels`, `regressionLevel`, `counts`, `waived`, `policyWeakened`, `policyLevel`,
`policyDecision` and `failed`; it is sorted, canonical and without
timestamps or absolute paths, so two runs are byte-identical. `sarif` is SARIF 2.1.0 with the regressions
as its results (rule `health.regression`, the level it has at the tier, the smell code in the
`properties` of the result; the artifact `uri` is the repository-relative path with each segment
percent-encoded, for example `app/%5Bid%5D.tsx`; plus `health.file_skipped` for each skipped changed file and
`health.policy_weakened` for each unrecorded weakening, their rules present only when there are results). With `--out` the failure is still printed on stderr.

The same engine is available in Go as `ScanHealth`, `CheckHealth`, `ImportHealthReport`,
`EncodeHealthReport`, `EncodeHealthCheck`, `HealthSARIF` and `HealthCheckSARIF`.


## System-of-Systems Composition

A model declares exact dependencies in `engmod.yml` and composes their selected
publications from `model/architecture.yml`.

```yaml
# engmod.yml
dependencies:
  - alias: telemetry
    path: models.example/coffee-fleet/telemetry@v0
    version: v0.1.0
    publications: [public]

# model/architecture.yml
architecture:
  composition:
    subsystems:
      - id: SUB-TELEMETRY
        name: Machine Telemetry Subsystem
        dependency: telemetry
        publication: public
    satisfactions:
      - need: ota-agent::NEED-TELEMETRY-FEED
        by: telemetry::CAP-TELEM-REPORT

# model/req-links.yml (documents.reqLinks)
reqLinks:
  - id: LNK-COF-001-TELEMETRY
    parent: REQ-COF-001
    child: telemetry::CAP-TELEM-REPORT
    pinned: {version: v0.1.0}
    status: claimed
```

Dependencies are standard CUE modules fetched from the OCI registry selected by
`CUE_REGISTRY` and materialized under `.engmod/modules`. A module contains
`cue.mod/module.cue`, `engmod.yml`, and the eight domain documents:

```cue
module: "models.example.com/company/security@v0"
language: version: "v0.17.0"
```

Publish with the standard CUE workflow:

```sh
export CUE_REGISTRY=registry.example.com/company-models
cue mod tidy
cue mod publish v0.3.1
```

For multi-repository local development, place `engmod.work.yml` above the
repositories. Published coordinates stay unchanged while dependencies resolve
from explicit sibling paths:

```yaml
schemaVersion: 2
replacements:
  - module: models.example.com/company/security@v0
    path: ./repos/security
```

Replacement paths must remain inside the workspace and their
`cue.mod/module.cue` identity must match. Composition rejects floating or
invalid versions, path escapes, cycles, missing manifests, unpublished targets,
and unsatisfied requirements.

To prove a family's release order and run the pre-push gate before anything is
pushed, stage every checkout under the workspace root (`go run ./cmd/engws help`
has the details):

```sh
engws stage --serve          # foreground: loopback OCI registry for the staged CUE modules
# in another shell, export the printed lines, then in each checkout, dependencies first:
export CUE_REGISTRY=<printed> GOPROXY=file://<root>/.engmod/staging/goproxy,off
export GONOPROXY=none GONOSUMDB=<printed> GOWORK=off ENGMOD_WORK=off
ENGMOD_LOCK=check <pre-push gate>
```

`engws stage` writes only `<root>/.engmod/staging`: `registry/<module path>/`
(an OCI image layout per CUE module, the content `cue mod publish` would push:
committed files for source kind `git`) and `goproxy/` (a GOPROXY file tree for
each checkout with `go.mod`). CUE has no file-based registry, so `--serve` serves
the layouts from memory on 127.0.0.1.

### Requirement link commands

`engws` maintains the requirement links of `model/req-links.yml` (design and decisions:
`docs/requirement-links.md`; fixture: `examples/requirement-links`).

Every requirement traces upward through a capability, because no model can know it is top
level: any system may later be used by one above. A requirement satisfies the upward trace
(`requirement.no_parent` otherwise; an error with `requirementLinks: strict`, a warning in
`guided` mode where the model publishes contract items or has links) iff (a) it `realizes` a
local capability or assumption of use, (b) it is the `parent` of a link to a dependency's
capability, or (c) it is `derived: true` with a `derivedRationale`. Stakeholder `sourceRefs`
do not count. A system publishes its own capabilities so a system above can link to them.

A link records the published dependency version it was reviewed at (`pinned.version`);
published versions are immutable, so that version identifies the reviewed content. The
model check compares no content: a pin other than the dependency pin in `engmod.yml` is
`link.version_mismatch`, an item missing at the published dependency pin is `link.broken`,
and a dependency resolved from a workspace replacement (a checkout, not a version) raises no
link finding at all, so editing a child in a workspace never churns its users. Review
happens when a dependency is bumped.

**`engws link add [--repo DIR] (--parent REQ | --evidence REQ[,REQ]) <alias::CAP-…|alias::AOU-…> [--rationale TEXT]`**
writes a claimed link pinned to the dependency version, after checking that the parent
exists and the child is a published capability or assumption of use at that version
(workspace off); a duplicate is refused. It creates `model/req-links.yml` and
`documents.reqLinks` when missing:

```text
engws: added LNK-SYS-002-ENCODER-2 REQ-SYS-002 -> encoder::CAP-A-1 (v0.1.0, claimed); review it with engws review-links --accept LNK-SYS-002-ENCODER-2 --by ACT-... --rationale TEXT
```

**`engws bump <repo> <alias> <version> [--allow-review] [--show]`** moves a dependency pin
(see `engws help`) and then compares every link of `<repo>` on `<alias>` between its pinned
version and the new one, both published (workspace off), by meaning: the item's kind, text
and verification criteria and each realizing requirement's text and verification criteria.
Status, notes, criticality, verification methods, title and tags are not compared.

```text
  review LNK-SYS-001-ENCODER (encoder::CAP-A-1: REQ-A-001 text)
  moved LNK-SYS-002-ENCODER v0.2.0 -> v0.3.0
```

The same meaning moves `pinned.version` (`moved`). A changed meaning, a pinned version that
cannot be resolved (`review`) or an item gone at the new version (`broken`) keeps the old
pin, so the model check reports `link.version_mismatch` (or `link.broken`) until the link is
reviewed. The bump is written either way; it exits 1 when a link needs review, 0 with
`--allow-review`. `--show` adds the item text before and after.

**`engws review-links [--repo DIR] [--json] [--show]`** lists the links of the model in
`DIR` (default `.`) that need review (pinned to another version than the dependency, broken
or claimed), one line each, resolved from the published versions (module cache, refetched
when the registry's digest differs, for example after `engws stage` re-staged a version):

```text
LNK-SYS-001-ENCODER encoder::CAP-A-1: version_mismatch (pinned v0.2.0, dependency v0.3.0): changed REQ-A-001 text
```

`--show` adds the parent requirement and the item before and after:

```text
  parent REQ-SYS-001: When simulation epoch elapses, the position simulator shall publish ownship position on the lab bus.
  before (v0.2.0)
  after (v0.3.0)
    capability CAP-A-1: Publish ownship position sentences at the configured rate.
  - REQ-A-001: When epoch period elapses, the position encoder shall publish one position sentence. (verification: ...)
  + REQ-A-001: When epoch period elapses, the position encoder shall publish two position sentences. (verification: ...)
```

After the review, record it (the actor must be in `architecture.actors`; a broken link
cannot be accepted):

```sh
engws review-links --accept LNK-SYS-001-ENCODER --by ACT-SYSTEM-OWNER \
  --rationale "Two sentences per epoch still give one position per epoch."
```

This sets `pinned.version` to the dependency pin (the published dependency, composed with
the workspace off), `reviewedBy`, `reviewedOn`, the rationale and `status: accepted` in
`model/req-links.yml`, changing only those lines, then evaluates the link again and exits 1
when it still does not pass. An accepted link without a model actor in `reviewedBy` or a
calendar date in `reviewedOn` is `link.review_unrecorded`; only accepted links at the
dependency pin count toward the trace roll-up of the system requirement.

`pinned.fingerprint` (v0.8.0) is deprecated: still accepted for one release, ignored, and
reported as `link.fingerprint_deprecated` (warning). `engws migrate-links` removes it.

**`engws status-suggest [--repo DIR] [--tree HASH] [--json]`** prints the status each
requirement and control verification earns from this tree's evidence, and writes nothing:
`implemented` when code links the requirement; `verified` when also a test links it that is
neither environment gated nor only a benchmark, and `.engflow/red-checks.jsonl` (written by
engineering-flow's `red-check.mjs`) has a RED line for this tree naming it in `reqs`; a control
verification `pass` when every requirement in its `verificationRefs` is verified. The tree is
the working-tree hash (`git write-tree` on a temporary index, as `tree-hash.mjs`), or `--tree`.
The owner approves the list; set the statuses in the model by hand or with `model.patch`.

**`engws contract-diff <from-ref> <to-ref> [--repo DIR] [--json]`** compares the
published contract items of the model as committed at two git refs:

```text
contract v1.0.0..v1.1.0
  changed   capability CAP-A-1 (text): text changed: treated as major [major]
  removed   capability CAP-A-2: Protect each position sentence with a checksum. [major]
impact: major
```

| Change | Impact |
|---|---|
| capability removed, capability text changed, assumption of use added, assumption text or `verificationCriteria` changed, item kind changed | major |
| assumption removed, capability added, only realizing requirements changed | minor |
| anything else | patch |

**`engws release-check [--repo DIR] [--tag vX.Y.Z]`** is the release gate. This repository's
CI runs it on every `v*` tag (`release-check` job of `.github/workflows/ci.yml`); in a
project, run it in the pre-push hook for a tag push as well. It fails (exit 1) with:

- `dependency.unreleased_pin` when a dependency version in `engmod.yml` or
  `engmod.lock.yml` is a pre-release or a pseudo-version: a release pins release tags only;
- `contract.semver_too_small` when the bump of the tag (default: the release tag at HEAD)
  over the previous release (the nearest release tag in the commit's history) is smaller
  than `contract-diff` between them requires. In v0.x a breaking change needs a minor bump (`v0.1.0` to `v0.2.0`).

Exit 2 is a usage, git or model error.

**`engws migrate-links [--repo DIR] [--dry-run]`** converts a model from the deprecated
`composition.allocations` and `contract.provides[].ref` (run it in the dependencies
first, then in the system):

```text
  ref CAP-TELEM-REPORT -> REQ-TELEM-001 realizes [CAP-TELEM-REPORT]
  link LNK-COF-001-TELEMETRY REQ-COF-001 -> telemetry::CAP-TELEM-REPORT (v0.1.0, claimed)
  added documents.reqLinks: model/req-links.yml
engws: wrote engmod.yml, model/architecture.yml, model/req-links.yml
engws: 9 requirement(s) without an upward trace (requirement.no_parent):
  requirement.no_parent: REQ-COF-004 [warning]
  ...
```

Each allocation becomes a `status: claimed` link pinned to the dependency version (a
`parent` link to a capability, an `evidence` link to an assumption of use; a note when the
dependency cannot be resolved); each capability `ref` becomes `realizes` on the requirement
it names; every deprecated `pinned.fingerprint` is removed and counted. An allocation to an
interface entry is not a link target and stays in place (`kept allocation REQ-… →
alias::IF-…: interface entries are not link targets; publish a capability that realizes it
and link that`), still reported as `composition.allocation_deprecated`. Only the touched lines of
each file change, and a second run reports `nothing to migrate`. Then review the claimed
links and give each listed requirement a `realizes` on a capability the model publishes, a
link to a dependency capability, or mark it `derived` (the link parents written by the migration
are already traced).

The architecture document shows the links in both directions. A system with `reqLinks`
gets a **Requirement Links** chapter: each requirement, its links with their state
(version-mismatched, broken and claimed links are marked), the linked child items and the
child requirements that realize them with their implementation and test status. A child
that publishes contract items gets a **Contract Items** chapter (each item and its realizing
requirements) and no list of users, also when it is rendered in a workspace. Only with
`engdoc --used-by` (`AsciiDocOptions.UsedBy`) and an active workspace whose other checkouts
link its items does it get a **Used By** section, computed at render time; never commit a
document rendered that way into the child. The trace matrix and the MCP `links.*` and
`requirements.parents` tools carry the same information.

Compatibility: upgrade the engine of every using system to v0.8.0 before a child publishes a
release that uses `realizes` or `contract.assumptions`; older engines reject them.

See `examples/coffee-fleet-ota-cloud-sample` for local composition and
`examples/atlas-industries` for a seven-repository company portfolio with two
products reusing shared hardware/software, security, compliance, and cloud
modules through OCI coordinates.

### Requirement delegation (no tiers, deprecated)

`composition.allocations` (a parent requirement bound `to` a subsystem with a `target` contract
id) is deprecated in favour of requirement links (`composition.allocation_deprecated`); it is
still read for one release. A delegation without a resolvable target raises
`composition.untraceable_delegation`, and a delegated requirement is not an orphan. Convert
allocations with `engws migrate-links`; both maintained composition examples are migrated.

## OSCAL chain generation

Use `engoscal` to generate SSP, Assessment Results, and POA&M artifacts from architecture inputs.

SSP only:

```bash
go run ./cmd/engoscal --model examples/payments-engineering-sample/engmod.yml --ssp-out examples/payments-engineering-sample/generated/ARCHITECTURE.ssp.json
```

Full chain:

```bash
go run ./cmd/engoscal \
  --model examples/payments-engineering-sample/engmod.yml \
  --requirements examples/payments-engineering-sample/model/requirements.yml \
  --code-root examples/payments-engineering-sample/src \
  --profile examples/payments-engineering-sample/oscal/profile-nist-800-53-low.json \
  --catalog examples/payments-engineering-sample/oscal/catalog-nist-800-53-rev5-subset.json \
  --ssp-out examples/payments-engineering-sample/generated/ARCHITECTURE.ssp.json \
  --ar-out examples/payments-engineering-sample/generated/ARCHITECTURE.ar.json \
  --poam-out examples/payments-engineering-sample/generated/ARCHITECTURE.poam.json \
  --ap-href ./ASSESSMENT-PLAN.json \
  --ssp-href ./ARCHITECTURE.ssp.json
```

Validation with OSCAL CLI (containerized):

```bash
podman run --rm -v "$PWD":/work:Z -w /work docker.io/matthewruge/oscal-cli:0.2.0 ssp validate --as json examples/payments-engineering-sample/generated/ARCHITECTURE.ssp.json
podman run --rm -v "$PWD":/work:Z -w /work docker.io/matthewruge/oscal-cli:0.2.0 ar validate --as json examples/payments-engineering-sample/generated/ARCHITECTURE.ar.json
podman run --rm -v "$PWD":/work:Z -w /work docker.io/matthewruge/oscal-cli:0.2.0 poam validate --as json examples/payments-engineering-sample/generated/ARCHITECTURE.poam.json
```

## Gemara GRC rendering

Gemara is a first-class rendering of the model: `enggemara` projects the engineering
model into [OpenSSF Gemara](https://gemara.openssf.org) documents using the official
`go-gemara` SDK types, and every artifact validates against the published Gemara CUE schemas.

All 13 Gemara artifact types are produced:
- L1 Vector Catalog ← `attackVectors` (with environment applicability)
- L1 Principle Catalog ← standard governance principles
- L1 Guidance Catalog ← control families (control `guidelines` link back to it)
- L2 Capability Catalog ← `functionalUnits` (+ a synthetic system capability)
- L2 Threat Catalog ← `threatScenarios` (capabilities, vectors, and threat `actors`)
- L2 Control Catalog ← `controls` + `controlVerifications` (synthesized assessment-requirements)
- L3 Risk Catalog ← `risks` (severity derived from likelihood × impact; unique rank)
- L3 Policy ← compliance/risks/controls (scope, imports, risk treatment, adherence)
- L5 Evaluation Log ← `controlVerifications` (pass **and** fail, with evidence)
- L6 Enforcement Log ← `poamItems` (remediation actions)
- L7 Audit Log ← `controlVerifications` + residual risks
- Mapping Document ← control → threat relationships
- Lexicon ← catalog terms (controlled vocabulary)

Generate the catalogs and logs:

```bash
go run ./cmd/enggemara \
  --model examples/payments-engineering-sample/engmod.yml \
  --requirements examples/payments-engineering-sample/model/requirements.yml \
  --out-dir examples/payments-engineering-sample/generated/gemara \
  --version 1.0.0 --date 2026-06-26T00:00:00Z
```

Optionally emit OSCAL via the Gemara SDK bridge (additive; the hand-written OSCAL
SSP/AR/POA&M under `engoscal` are retained):

```bash
go run ./cmd/enggemara --model <engmod.yml> --requirements <model/requirements.yml> \
  --out-dir <dir> \
  --oscal-catalog-out <dir>/oscal-catalog.json \
  --oscal-ar-out <dir>/oscal-ar.json
```

Validation:
- in-repo, via the go-gemara SDK loader + type discriminator: `go test -run TestGemara ./...`
- against the official CUE schemas: `scripts/validate-gemara.sh` (needs the `cue` CLI and the
  Gemara schema module; set `GEMARA_SCHEMA_DIR` to a checkout of `github.com/gemaraproj/gemara`)

The generated AsciiDoc document includes a `Gemara GRC Model` chapter, and the MCP server
exposes a `gemara.*` tool per artifact type (`gemara.controlCatalog`, `gemara.threatCatalog`,
`gemara.riskCatalog`, `gemara.vectorCatalog`, `gemara.capabilityCatalog`, `gemara.principleCatalog`,
`gemara.guidanceCatalog`, `gemara.policy`, `gemara.lexicon`, `gemara.mappingDocument`,
`gemara.auditLog`, `gemara.enforcementLog`, `gemara.evaluationLog`) plus `gemara.validate`.

See `docs/gemara-rendering.md` for the field-by-field mapping and the OSCAL decision.

## Agent Skills

Use these docs to drive AI-agent development workflows and tagging conventions:

- OpenCode-ready skill (auto-discoverable by `opencode debug skill`): `.opencode/skills/engineering-model-architecture-ai/SKILL.md`
- framework-neutral workflow and tagging contract: `docs/skills/architecture-mcp-workflow.md`
- OpenCode adapter prompt: `docs/skills/adapters/opencode-skill-prompt.md`
- generic adapter prompt for other frameworks: `docs/skills/adapters/generic-agent-prompt.md`

## Example Project

End-to-end sample inputs and generated outputs are under:
- `examples/payments-engineering-sample`
- `examples/bedrock-pr-review-github-app-sample`
- `examples/coffee-fleet-ota-cloud-sample`

Core files:
- `engmod.yml`
- `model/catalog.yml`
- `model/requirements.yml`
- `model/architecture.yml`
- `model/behavior.yml`
- `model/assurance.yml`
- `model/compliance.yml`
- `model/views.yml`
- `model/decisions.yml`
- `infra/terraform`
- `src` (Go/Rust/TypeScript traced code)

## Development

### Repository layout

```text
.
├── doc.go, model_api.go,       public engmodel facade: type aliases, constants and thin
│   validate_api.go, api_*.go   wrappers, one api_<area>.go per area (old public names kept)
├── facade_test.go              every exported implementation name is re-exported at the root
├── layering_test.go            the package import direction below holds
├── cmd/                        command-line tools (engdoc, engview, engtrace, engchange, engspec,
│                               enggemara, engoscal, engsysml, engstruct, engdragon, engnaf,
│                               engtrlc, englobster, engair, engquint, enghealth, engmcp, engws, sysmlmetamodel)
├── internal/
│   ├── model/                  model types, loader, canonical bundle, embedded CUE schema (schema/)
│   ├── validate/               diagnostics and bundle validation rules
│   ├── codemap/                tree-sitter and lexical source symbol scanner
│   ├── view/                   view projection of the architecture
│   ├── render/                 diagramstyle, mermaid and svg renderers of a projected view
│   ├── composition/            system-of-systems composition (CUE modules, workspace, lock)
│   ├── codescan/               code, runtime and verification evidence inference
│   ├── health/                 code health: report, scoring, roll-ups, delta check, baseline, cache, git
│   │   ├── engine/treesitter/  the tree-sitter engine (functions, metrics, smells)
│   │   └── scan/               scan and check pipelines (the one package that builds the engine)
│   ├── trace/                  traceability matrix
│   ├── reqlint/                EARS preflight, catalog description and requirement link lint
│   ├── reqdelta/               requirements delta planning and atomic application, spec delta rendering
│   ├── generate/               single-view Mermaid/SVG generation (Generate, GenerateFromFile)
│   ├── asciidoc/               AsciiDoc publication (templates/asciidoc.tmpl)
│   ├── export/                 one package per exporter: gemara, oscal, sysml (metamodel/),
│   │                           structurizr, threatmodel, naf, trlc, lobster, airborne, exportutil
│   ├── integration/            test-only: end-to-end tests through the facade and across packages
│   ├── mcp/                    MCP server
│   ├── engws/                  engmod.work.yml workspace tool
│   ├── cliflags/               shared command-line flags
│   ├── sysmlmetamodel/         SysML v2 metamodel inventory tooling
│   ├── textutil/               shared string helpers
│   └── testutil/               shared test fixtures and helpers
├── engmod.yml, model/          this repository's own engineering model (YAML only)
├── generated/                  this repository's generated artifacts
├── examples/                   sample models with their generated outputs
├── docs/                       design notes and format guides
└── scripts/                    validation and tooling scripts (sysml-toolchain.env pins the SysML tools)
```

Package layering: imports point down only. `textutil` and `model` import nothing in the
module; `validate` builds on `model`; `codemap` and `view` on those; `render/*` on `view`;
`composition` on `model`, `validate` and `textutil`; `codescan` on `composition` and
`codemap`; `health` on `validate`, `codescan` and `codemap` (it never imports an engine);
`health/engine/treesitter` on `health` and `codemap`; `health/scan` on `health`, the engine and
`codescan`; `trace` on `codescan`; `reqlint` on `model` and `validate`; `reqdelta` on
`reqlint` and `trace`; `generate` on `view`, `render/*` and `codescan`; `export/*` on the
layers below `trace`; `asciidoc` on all of these, never on `mcp` or `engws`; `cmd/enghealth` uses the
root facade (and `internal/health` for the baseline helpers). Only `cmd/*`,
`internal/mcp` and `internal/engws` import the root facade (among tests, only
`internal/integration`), and `mcp` and `engws` do not import each other. `layering_test.go` enforces the rules with `go list`.

The tests of the `internal/...` implementation packages run from the repository root (their `TestMain` changes
to the directory holding `go.mod`), so they name repository paths such as
`examples/...` and `engmod.yml` directly:
`go test ./internal/integration -run TestAtlasIndustriesCompanyExample`.

Unit checks:

```bash
go test ./...
go vet ./...
```

The canonical validation gauntlet is `scripts/validate-all.sh` — the single source of truth for
both local development and CI:

```bash
bash scripts/validate-all.sh
```

It runs strict gates that always fail the build:

- `go build ./...`
- generation gates: `engdoc` exits non-zero on any error (transitively enforcing trace
  integrity / dangling links, EARS lint, and composition checks)
- `engtrace`: zero dangling code trace links per model
- artifact freshness: regenerated `ARCHITECTURE.adoc` / `DECISIONS.adoc` / `TRACE-MATRIX.json`
  must match what is committed (detected via `git diff`)

plus best-effort gates that run only when their tool is on `PATH` (a missing tool is skipped, a
real failure still fails the build):

- Gemara CUE schema validation (`cue`)
- Structurizr DSL validation (behind `ENGMOD_VALIDATE_STRUCTURIZR=1`, via docker/podman)
- TRLC validation (`trlc`)

`.github/workflows/ci.yml` runs `scripts/validate-all.sh`, so the same gates apply in CI.

Generated `ARCHITECTURE.adoc`, `DECISIONS.adoc`, and `TRACE-MATRIX.json` are deterministic and
reproducible; CI fails on any drift, so when you change a model you must regenerate these
artifacts and commit them alongside the model change.

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md).

## License

This project is licensed under the MIT License.
See [LICENSE](./LICENSE).
