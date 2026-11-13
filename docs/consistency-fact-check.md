# Consistency checks: what exists today (fact check of ADR-EMG-056, 2026-11-13)

Read-only audit of `internal/validate`, `internal/trace`, `internal/codemap`, `internal/codescan` and the tier table.
Tiers read poc, prototype, mvp, product, operational; D deferred, W warning, E error; `E*` an untiered error; `w*` an
untiered warning (stays a warning at operational because no tier row keys it).

| Pairing | Diagnostic | Severity |
|---|---|---|
| requirement to code or test | `requirement.orphan` (one marker anywhere satisfies it; retired and rejected exempt) | DDWEE |
| requirement to code, composing systems only | `requirement.unsatisfied` | DDEEE |
| requirement to test | no check, only the authored status (`lifecycle.requirement_unverified`) | DDDWE |
| requirement to element (`appliesTo`) | `requirement.invalid_applies_to` (the field is optional) | E* |
| behavior requirement to state, transition, invariant or flow | no check | none |
| invariant to requirement | `behavior.invariant_unlinked` | DDWWE |
| machine, state, transition, event, flow, unit, interface, data object to requirement | no check | none |
| state reachable, state has an exit, final state exits, event declared, event used, determinism | `behavior.state_unreachable`, `machine_deadlock`, `final_state_has_transitions`, `transition_unknown_event`, `event_unused`, `transition_nondeterministic` | E* or DDWWE |
| machine has an invariant, invariant has a formal result | no check (`verification.formal_stale` is untiered and only when evidence exists) | none |
| any model element to code | no check (only MCP lookups that never diagnose) | none |
| state or transition to a code site | no check | none |
| marker names an ID that does not exist | `code.dangling_model_link`, `code.dangling_requirement_link` | E all tiers |
| marker names an ID of the wrong kind | no check (the ID set is flat and includes requirements and catalog terms) | none |
| function lacks `TRLC-LINKS` | `code.missing_trlc_link` (functions and methods only) | WWWEE |
| `ENGMODEL-OWNER-UNIT` missing or naming no unit | no check (defaults to "unresolved") | none |
| unlinked state-like declaration, enum members against machine states | no check | none |
| threat to control verification | `threat.verification_missing` | DDWEE |
| control to verification | `gemara.control_needs_review` | w* |
| control verification to tests | `verification.unlinked`, `model.invalid_control_verification_ref`, `verification.status_stale` | DDWEE, E*, DDWWE |
| threat to mitigation | no check (MCP `threats.unmitigated` only) | none |
| flow references, steps, boundaries, interface owner | `model.invalid_flow_*`, `model.missing_flow_boundary_ref`, `model.invalid_interface_owner` | E* |
| trust boundary, data object or interface used by no flow | no check | none |

## Claims of ADR-EMG-056 that were too strong

- "Every function carries a marker": only `TRLC-LINKS` is required, only on functions and methods, and only a warning
  until product. `ENGMODEL-LINKS` and `ENGMODEL-OWNER-UNIT` are never required.
- "An `ENGMODEL-LINKS` target must be a model element": only existence in a flat ID set is checked.
- `requirement.unsatisfied` runs only for models with composition subsystems; `requirement.orphan` is satisfied by one
  marker and is deferred before mvp.
- Stacked `ENGMODEL-LINKS` lines overwrite each other in the scanner while `TRLC-LINKS` lines accumulate; element to code
  coverage must fix that first.

## Never expected to have code

Actors, attack vectors, trust boundaries, threat scenarios, assumptions, out-of-scope entries, risks, POA&M items,
deployment targets, decisions, catalog terms, views, and imported `alias::ID` entities. Elements owned by something
external need an explicit exemption: no generic element waiver exists today.
