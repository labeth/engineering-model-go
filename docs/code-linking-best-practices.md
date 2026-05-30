# Engmod Code Linking Best Practices

This document defines how source code, tests, runtime artifacts, and model entities should be linked in engmod projects. It is intended to be the policy that strict linting enforces.

## Goals

Code links should make the model useful for implementation work, impact analysis, verification coverage, and generated documentation.

The linking model should answer four questions:

1. Which functional unit owns this code?
2. Which requirements does this code implement or verify?
3. Which architecture entities does this API, interface, schema, runtime resource, or adapter represent?
4. Which files and declarations need attention when a requirement or model entity changes?

## Link Types

### File Ownership

Use file ownership for the functional unit that owns a source file.

```go
// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap
```

Rules:

- Every first-party source file in strict mode should have one owning functional unit.
- Ownership should point to a functional unit, not a functional group.
- If a file genuinely spans multiple units, prefer splitting it. If that is not practical, use the dominant owner and add declaration-level links for the mixed parts.
- Generated files should either be excluded from scanning or marked as generated and not require ownership links.

### Requirement Links

Use `TRLC-LINKS` for requirements implemented or verified by a declaration.

```go
// TRLC-LINKS: REQ-EMG-010
func inferCodeItems(...) (...) {
}
```

Rules:

- Requirement links must use requirement IDs, for example `REQ-EMG-010`.
- Requirement links belong on behavior-bearing declarations, not random line comments.
- A declaration may link to multiple requirements when it directly implements or verifies all of them.
- Do not add broad requirement links just because a file is in the same functional unit.
- Prefer linking the smallest stable behavior boundary that is useful for review and impact analysis.

### Model Entity Links

Use `ENGMODEL-LINKS` for declarations that represent architecture entities or engineering-model concepts.

Marker shape:

```go
// ENGMODEL-LINKS: IF-GITHUB-WEBHOOK, FLOW-PR-OPENED-REVIEW, DATA-PULL-REQUEST-EVENT
```

Rules:

- Model links should point to authored model entity IDs.
- Use model links for interfaces, APIs, schemas, DTOs, events, adapters, runtime entrypoints, controls, trust-boundary code, and model contract types.
- Use requirement links for why behavior exists; use model links for where the code sits in the architecture.

## Strict Mode Policy

Strict mode should mean that missing links are fatal for in-scope first-party code.

Expected strict behavior:

- Scan configured code roots from `architecture.inferenceHints.codeSources`.
- If no code roots are configured, scan the model directory or repository root by default.
- Emit a fatal diagnostic when strict mode cannot find any supported first-party source files.
- Emit fatal diagnostics for required declarations without `TRLC-LINKS`.
- Emit fatal diagnostics for first-party source files without `ENGMODEL-OWNER-UNIT`.
- Emit fatal diagnostics for public APIs and schemas without model links once strict model-link enforcement is enabled.
- Ignore generated, vendor, third-party, and dependency cache directories.

Current implementation note:

- `lintRun.mode: strict` currently controls requirements/EARS linting.
- Code scanning captures `TRLC-LINKS` and `ENGMODEL-LINKS`, and emits missing `TRLC-LINKS` diagnostics for selected declaration kinds, but strict code-linking is not yet a first-class mode.
- The scanner and command behavior should be tightened to match this document.

## What Must Be Linked

### Go

Required requirement links in strict mode:

- Package-level functions.
- Receiver methods.
- Test functions that verify modeled behavior.
- HTTP, gRPC, CLI, queue, or event handler functions.
- Middleware functions that implement controls or boundary behavior.

Required model links once model-entity linking is available:

- Public API handlers to `interface`, `flow`, and owning `functionalUnit`.
- Request and response structs to `interface` or `dataObject`.
- Event structs to `event`, `flow`, or `dataObject`.
- Service interfaces to `functionalUnit` or `interface`.
- External client adapters to `referencedElement` or outbound `interface`.
- Security middleware to `control` or `trustBoundary`.

Optional or normally ignored:

- Imports.
- Local variables.
- Simple constants.
- Private helper functions that do not express independent modeled behavior.
- Struct fields, unless field-level linking is explicitly required for a schema use case.

Example:

```go
// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

// ENGMODEL-LINKS: IF-MCP-TOOLS, FLOW-MCP-REQUEST
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func handleToolCall(...) (...) {
}

// ENGMODEL-LINKS: IF-MCP-TOOLS, DATA-TOOL-CALL-REQUEST
type ToolCallRequest struct {
}
```

### TypeScript and TSX

Required requirement links in strict mode:

- Function declarations.
- Class methods.
- Object methods used as handlers.
- Exported arrow/function variables.
- React components that implement product behavior.
- Hooks that implement domain or integration behavior.
- Route, action, loader, resolver, and API handler functions.

Required model links once model-entity linking is available:

- Route handlers to `interface` and `flow`.
- Exported request/response types to `interface` or `dataObject`.
- Event types to `event` or `dataObject`.
- Client adapters to outbound `interface` or `referencedElement`.
- Components that represent modeled UI boundaries to `actor`, `interface`, or `functionalUnit` when applicable.

Optional or normally ignored:

- Local callbacks used only inside a linked component or handler.
- Test setup helpers.
- Type aliases that only simplify local implementation details.

Example:

```ts
// ENGMODEL-LINKS: IF-CHECKOUT-API, FLOW-CHECKOUT-SUBMIT
// TRLC-LINKS: REQ-PAY-001
export const submitCheckout = async (request: CheckoutRequest) => {
};

// ENGMODEL-LINKS: IF-CHECKOUT-API, DATA-CHECKOUT-REQUEST
export interface CheckoutRequest {
}
```

### Rust

Required requirement links in strict mode:

- Free functions.
- Impl methods.
- Trait methods that define behavior.
- Test functions that verify modeled behavior.
- HTTP, CLI, queue, or event handler functions.

Required model links once model-entity linking is available:

- Public structs and enums used as API contracts to `interface` or `dataObject`.
- Traits that define ports/adapters to `interface` or `functionalUnit`.
- External adapters to `referencedElement` or outbound `interface`.
- Security or policy modules to `control` or `trustBoundary`.

Optional or normally ignored:

- Local helper functions inside a linked module when they do not express independent modeled behavior.
- Private data structures that are pure implementation detail.

Example:

```rust
// ENGMODEL-LINKS: IF-RISK-SCORE, FLOW-RISK-SCORING
// TRLC-LINKS: REQ-PAY-004
pub fn calculate_risk_score(input: RiskInput) -> RiskScore {
}

// ENGMODEL-LINKS: IF-RISK-SCORE, DATA-RISK-INPUT
pub struct RiskInput {
}
```

## APIs, Interfaces, and Schemas

APIs and interfaces are architecture boundaries. They should be linked even when the implementation body is small.

Use these mapping rules:

| Code element | Link to |
| --- | --- |
| HTTP route handler | Interface, flow, functional unit, requirements |
| gRPC method | Interface, flow, data objects, requirements |
| GraphQL resolver | Interface, data objects, requirements |
| CLI command handler | Interface or functional unit, requirements |
| Queue/topic publisher | Interface, event, flow, requirements |
| Queue/topic consumer | Interface, event, flow, requirements |
| Request DTO | Interface or data object |
| Response DTO | Interface or data object |
| Domain event | Event and data object |
| External service client | Referenced element and outbound interface |
| Auth middleware | Control, trust boundary, requirements |
| Validation middleware | Control, interface, requirements |
| Repository/persistence adapter | Data object, deployment target or referenced element |

## Runtime Links

Runtime artifacts should link deployed things back to the model.

Supported ownership markers include comments and annotations such as:

```yaml
engmodel.dev/owner-unit: FU-MCP-SERVER
engmodel.dev/runtime-description: Serves model-aware MCP tool calls.
```

Rules:

- Workloads, services, functions, jobs, topics, buckets, and databases should link to an owning functional unit.
- Runtime descriptions should explain the deployed responsibility, not restate the file path.
- Runtime links should support deployment and threat-model diagrams.
- External managed resources should be represented as referenced elements, deployment targets, interfaces, data objects, or controls as appropriate.

## Verification Links

Tests and verification artifacts should link to requirements they verify.

```go
// TRLC-LINKS: REQ-EMG-010
func TestScanRequiresTRLCLinks(t *testing.T) {
}
```

Rules:

- Link test functions, contract tests, e2e tests, and verification scripts to the requirements they prove.
- Prefer test-level links over file-level links when a test file covers multiple requirements.
- Test helpers should not require requirement links unless they independently assert modeled behavior.
- If a requirement has implementation links but no verification links, strict verification coverage should report a gap.

## Choosing the Right Level

Prefer this order:

1. File owner link for functional unit ownership.
2. Declaration requirement link for implemented or verified behavior.
3. Declaration model link for APIs, contracts, events, controls, adapters, and public schema.
4. Runtime owner link for deployment resources.

Avoid these patterns:

- Linking every helper to every nearby requirement.
- Linking a whole file to a requirement when only one function implements it.
- Linking to a functional group when a functional unit is available.
- Using non-requirement text in `TRLC-LINKS`.
- Adding links to silence lint without checking the requirement meaning.

## Scanner and Linter Expectations

Strict linting should enforce the documented policy in two layers.

Layer 1: existing markers

- `ENGMODEL-OWNER-UNIT` on in-scope first-party files.
- `TRLC-LINKS` on required behavior declarations.
- Valid requirement ID syntax.
- Attached markers must be adjacent to the declaration they describe.

Layer 2: model-entity links

- `ENGMODEL-LINKS` or equivalent on public API and schema declarations.
- Linked model IDs must exist in the loaded architecture model.
- Model link target kinds should be compatible with the declaration kind.
- Diagnostics should group repeated findings by file and list line numbers as comma-separated values.

## Supported Language Coverage

The scanner should define required and optional declaration kinds per supported language.

| Language | Required in strict mode | Optional but linkable |
| --- | --- | --- |
| Go | Functions, methods, behavior tests, API handlers | Type declarations, API/schema structs, interfaces |
| TypeScript | Function declarations, methods, exported function variables, handlers | Classes, interfaces, type aliases, components |
| TSX | Function declarations, methods, exported function variables, components, hooks | Props types, interfaces, type aliases |
| Rust | Free functions, impl methods, trait methods, behavior tests | Structs, enums, traits, type aliases |

The scanner should document unsupported syntax explicitly so gaps are visible.

## Review Checklist

Before merging model-affecting code:

1. Every changed first-party source file has an owner functional unit.
2. Every changed behavior declaration has accurate requirement links.
3. Public APIs, schemas, events, controls, and adapters are linked to model entities when supported.
4. Tests that verify changed behavior have requirement links.
5. Generated docs or AI views show the expected implementation and verification links.
6. Strict linting fails when links are missing or malformed.
