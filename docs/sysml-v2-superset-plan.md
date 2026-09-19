# SysML v2 Superset Migration

## Outcome

Engineering Model will remain YAML-authored and have one canonical semantic
model aligned with SysML v2 and KerML. YAML instances are the canonical authored
and persisted data, while modular CUE files are the authoritative schema and
cross-field constraint layer. Go types are runtime API representations checked
against that contract, not a separately authored semantic model. The YAML schema
may be redesigned where that produces clearer or more complete semantics. All
documentation, analysis, assurance, traceability, and interchange artifacts are
projections from that model.

The current YAML schema is not a compatibility constraint. During migration it
is accepted through a compatibility adapter and may be replaced after equivalent
output and validation behavior is proven. YAML remains the canonical authoring
and persistence format after that migration; SysML textual syntax is an output,
not the source of truth.

## Non-duplication rules

1. A standard SysML or KerML concept has exactly one canonical representation.
2. Format-specific exporters cannot introduce authored domain entities.
3. The canonical YAML schema maps directly to the canonical semantic model.
   Compatibility input adapters cannot retain a second independently mutable
   semantic graph.
4. Engineering-specific concepts extend standard elements through typed
   metadata or explicit external relationships.
5. A concept that cannot be mapped without loss produces a diagnostic; it is
   never silently flattened into prose or a generic edge.
6. Legacy syntax is removed only after all compatibility fixtures pass against
   the canonical model.

## Canonical ownership

| Existing concept | Canonical SysML/KerML direction | Rule |
|---|---|---|
| Functional group/unit, subsystem, hardware item, deployment target | Part definitions and usages | Do not add parallel SysML part fields |
| Data object | Item or attribute definition/usage | Preserve schema, classification, and lifecycle as typed metadata |
| Interface and hardware interface | Ports, interfaces, connections, and flows | Do not collapse port, interface, and connection semantics |
| Flow and flow step | Actions, parameters, flows, and successions | Preserve transfer separately from temporal ordering |
| State and event | State and event occurrences | Transitions own triggers, guards, effects, source, and target |
| Requirement | Requirement definition/usage | Preserve subject, assumptions, constraints, satisfaction, and verification |
| Allocation and satisfaction | SysML allocation and satisfaction relationships | Preserve relationship identity and endpoint semantics |
| View | View and viewpoint definitions/usages | Rendering remains separate presentation data |
| Mapping | Specific typed relationship | Generic mappings remain migration input only |
| Control, threat, risk, POA&M, compliance | Typed engineering metadata/domain library | Extend SysML rather than redefining requirements or cases |
| ADR, source ownership, code/test evidence | Typed engineering metadata and external relationships | Keep repository evidence addressable by stable IDs |

## Compatibility contract

Migration must preserve the observable behavior of:

- AsciiDoc and PDF architecture publication
- Mermaid views
- Structurizr DSL
- Threat Dragon and Open Threat Model
- TRLC and LOBSTER
- OSCAL SSP, assessment results, and POA&M
- Gemara catalogs and logs
- trace-matrix JSON and CSV
- system-of-systems composition and requirement delegation
- MCP model, ownership, impact, coverage, and verification tools
- strict YAML, EARS, reference, path-boundary, composition, trace-link, and
  deterministic-generation validation

Parity is measured with normalized golden outputs and diagnostic fixtures.
Intentional output changes require a model decision and updated fixture.

## Delivery sequence

1. **Baseline:** capture representative outputs and diagnostics for the root
   model and all example systems.
2. **Semantic kernel:** introduce identity, namespace, ownership,
   definition/usage, typed feature, relationship, expression, multiplicity,
   metadata, and extension primitives.
3. **YAML schema and legacy adapter:** define the canonical YAML serialization
   of the semantic kernel in CUE, validate YAML before strict Go decoding,
   continuously check the Go runtime representation for schema drift, translate
   the current YAML schema into the semantic model, and reject duplicate or
   lossy representations.
4. **Structural and behavioral coverage:** implement parts, items, attributes,
   ports, interfaces, connections, allocations, actions, flows, successions,
   states, and transitions.
5. **Requirements and analysis coverage:** implement constraints,
   calculations, requirements, concerns, stakeholders, cases, verification,
   views, and viewpoints.
6. **Remaining KerML/SysML semantics:** expressions, quantities, units,
   occurrences, time portions, variability, metadata, imports, and libraries.
7. **Exporter migration:** move every existing generator and MCP projection to
   the canonical model, one output at a time, under parity tests.
8. **Interchange and conformance:** add textual and project interchange,
   official-tool validation, round-trip reconstruction, and an executable
   normative coverage manifest.
9. **Cutover:** remove the legacy YAML schema only after all outputs,
   validations, and conformance evidence pass from the canonical YAML model.

## Claim policy

Before complete conformance evidence exists, generated output is described as a
documented SysML v2 subset. A semantic-superset claim requires:

- complete normative KerML and SysML feature-family coverage;
- preservation of ownership, typing, relationship, expression, multiplicity,
  library, and derived/implied semantics;
- project interchange with stable cross-project identity;
- round-trip equivalence tests;
- no unsupported or lossy mapping diagnostics; and
- successful validation with the configured SysML v2 toolchain.

## Current implemented slice

<!-- TRLC-LINKS: REQ-EMG-038, REQ-EMG-039 -->

The formal 2026-04 release has no normative metamodel XMI/Ecore. Tasks 1.1-1.3
therefore use the aligned official Pilot Implementation Ecore as the closest
machine-readable implementation artifact and label it non-normative. The pinned
sources and hashes are in `tools/sysml/metamodel-sources.json`; the normalized
inventory and strict per-metaclass/per-property binding are checked offline by
`scripts/check-sysml-metamodel.sh`. The inventory generates both the Go runtime
binding and CUE property validation. All 175 metaclasses and 415 owned
properties are now represented; derived properties remain explicitly marked
read-only rather than being counted as unimplemented.

Tasks 2.1–2.3 add an official-qualified metaclass and typed property container
to the single canonical graph. Existing YAML requirements, actors, control
verifications, views, design narratives, threats, controls, compliance, risks,
POA&M, ADRs, composition, ownership policy, and evidence normalize into those
official instances or stable `Engineering::` extension metaclasses. Legacy
fields remain input/export compatibility fields, not a second canonical graph.

The bounded prerequisite before exporter migration is complete: the modular
schemas under `model/schema/` cover all five authored YAML document types,
including the canonical `semantics` section. Loading validates those instances
against CUE before strict Go decoding, rejects unknown fields with source paths,
checks multiplicity ordering and typed semantic value/extension invariants, and
uses a schema-shape test to detect CUE/Go field drift. CUE does not contain a
second set of authored domain instances.

Project interchange is implemented as a Sysand-managed model interchange
project and KPAR, not as a custom ZIP. Generated source embeds the canonical
semantic model in typed `EngineeringProject` metadata so reopening the KPAR can
reconstruct and compare identity, ownership, relationships, expressions,
library/project references, and typed extensions. Deterministic unit tests cover
the payload import and comparison; `scripts/validate-sysml.sh` installs no tools
implicitly, but uses the pinned external tools to build/reopen the KPAR and run
the official parser.

Textual generation now uses a table-driven native SysML production for every
canonical standard kind. Definitions and usages retain their distinction;
packages/imports, typing, specialization, subsetting, redefinition,
conjugation, multiplicity, ordering, uniqueness, features, values, and
relationship endpoints are emitted as SysML syntax. There is no generic
`item def` recovery path: a canonical official metaclass without a rule is a
generation error. The coverage manifest lists all 175 official metaclasses as
either native syntax or non-instantiable semantic metamodel machinery. The
latter are created implicitly by parsing native ownership, membership, typing,
expression, and relationship productions and cannot be authored as standalone
textual elements.

The typed `EngineeringProject` payload is the normative lossless interchange
sidecar, not a substitute for valid SysML. It preserves official property values
and Engineering extensions across KPAR packaging. Derived and implied
properties may be absent from the text because the official importer derives
them from the native syntax; round-trip comparison normalizes only those
read-only values. Both the generated source and the reopened KPAR source must
pass the pinned official parser before semantic equivalence is accepted.

The conformance baseline is SysML 2.0 / KerML 1.0 using the official Pilot
Implementation release `2026-04`, artifact `0.59.0`, commit
`20897e3122f2c2f8b29389745f0caaaeb7c6e21a`. The CI-friendly validator wrapper is
DeciSym/sysmlv2-validator commit
`63abbd9fbc7851dc437d01b2dc07836b919770b8`, configured with
`-Dsysml.release.tag=2026-04 -Dsysml.artifact.version=0.59.0`. KPAR packaging and
reopening use Sysand `0.2.1`, commit
`8430ab04d4d825fb1c196670692d2dafa344e713`, under MIT OR Apache-2.0. Release
asset checksums are pinned in `tools/sysml-toolchain.env`; source repositories
without separately published archive checksums are pinned and verified by Git
commit. Release `2026-08` is intentionally excluded because it targets SysML 2.1
Beta 2 rather than this formal baseline.

The canonical YAML/CUE model is a semantic superset of the pinned formal SysML
2.0/KerML 1.0 abstract syntax, with typed `Engineering::` extensions kept
separate from normative entries. `generated/SYSML-COVERAGE.json` provides
executable evidence for every metaclass, owned property, and relationship:
54 metaclasses are native-rendered, 121 are abstract/implicit, all 415 owned
properties are mapped or derived/implied, and all 66 relationships are
classified. The claim is bounded to abstract-syntax representation, textual
rendering, and semantic interchange round trip; it does not include
execution/simulation semantics or graphical concrete syntax.
