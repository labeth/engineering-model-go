# Release validation: source tracing and hardware publications

Validation date: 2026-09-23. Base: v0.2.0 (5833375).

The release adds JavaScript and lexical Verilog source tracing, individual-file
inference roots, distinct source identities, hardware view/MCP support, and
publication fixes. Test-only links remain verification evidence rather than
production implementation.

Publications retain one complete diagram per authored view and requirement.
Large graphs scale to fit rather than being split across repeated panels.
Standalone SVG and Mermaid exports retain complete graphs.

## Checks

- Full Go tests and go vet: passed.
- Formatting and whitespace: passed.
- All 15 example models and the repository model regenerated.
- All 16 architecture PDFs rendered without Mermaid parse-error warnings.
- All 1,050 PDF pages checked for text outside page bounds and renderer error text:
  none found. Representative diagram, table, opening and closing pages were
  visually reviewed. Dense graphs intentionally use smaller labels.
- All 28 standalone SVGs rendered and visually reviewed; text stays within their
  view boxes.
- SysML metamodel, official parser, all-example syntax, KPAR reopen and native
  source round trip: passed.
- Artifact freshness: passed; regeneration produced no changes.
- Full scripts/validate-all.sh gate, including external generated-format
  conformance checks: passed.

The example models retain their authored gaps and not-run evidence states.
Successful export validation does not change those states into passed tests.

## Architecture PDFs

| Publication | Pages |
| --- | ---: |
| examples/atlas-industries/generated/aegis-sentinel/ARCHITECTURE.proven.pdf | 41 |
| examples/atlas-industries/generated/company-portfolio/ARCHITECTURE.proven.pdf | 36 |
| examples/atlas-industries/generated/orion-relay/ARCHITECTURE.proven.pdf | 40 |
| examples/atlas-industries/generated/shared-cloud-platform/ARCHITECTURE.proven.pdf | 29 |
| examples/atlas-industries/generated/shared-compliance-baseline/ARCHITECTURE.proven.pdf | 38 |
| examples/atlas-industries/generated/shared-edge-platform/ARCHITECTURE.proven.pdf | 29 |
| examples/atlas-industries/generated/shared-security-services/ARCHITECTURE.proven.pdf | 30 |
| examples/bedrock-pr-review-github-app-sample/generated/ARCHITECTURE.proven.pdf | 111 |
| examples/coffee-appliance-six-view/generated/ARCHITECTURE.proven.pdf | 36 |
| examples/coffee-fleet-ota-cloud-sample/generated/ARCHITECTURE.proven.pdf | 129 |
| examples/coffee-fleet-ota-cloud-sample/subsystems/cloud-api/generated/ARCHITECTURE.proven.pdf | 29 |
| examples/coffee-fleet-ota-cloud-sample/subsystems/ota-agent/generated/ARCHITECTURE.proven.pdf | 31 |
| examples/coffee-fleet-ota-cloud-sample/subsystems/telemetry/generated/ARCHITECTURE.proven.pdf | 31 |
| examples/dal-c-flight-control-sample/generated/ARCHITECTURE.proven.pdf | 32 |
| examples/payments-engineering-sample/generated/ARCHITECTURE.proven.pdf | 122 |
| generated/ARCHITECTURE.proven.pdf | 286 |

## Subsequent Python tracing work (unreleased)

The PDF review above applies to the release-preparation commit `d0da9a3`.
The subsequent Python declaration-tracing changes pass the complete Go test
suite, go vet, focused parser/inference regressions and native MCP lookup of
REQ-EMG-010. All example and self-model exports regenerate successfully.
An independent comparison against the SDS pinned Python AST inventory matches
51 declarations in 25 files, using synthetic comments in temporary copies only.
Those comments do not count as authored SDS requirement links.

Consecutive-generation comparison passed for 499 maintained artifacts. PDF rendering and visual
review of the updated architecture publication remain deferred to the next
larger publication batch; the existing PDFs have not been revalidated for
these subsequent changes.

## Subsequent complete-diagram layout work (unreleased)

Authored and requirement flowcharts use wider spacing. Long deployment edge labels
use collision-safe numbered references with full descriptions below the single
diagram; keyed edges request an extra routing rank. Focused tests cover description
preservation, repeated edges, preexisting reference-like labels, unrelated view
isolation and actual AsciiDoc table emission. The SDS candidate preserves all nodes
and relationships across 13 complete views after expanding the keys.

Standalone browser previews improved browser/requirement spacing and resolved the
hardware-label collision with keyed edge routing. This is not final PDF validation.
The prior SDS PDF audit covered 949 pages with no text outside page bounds and 39
small-text flags; five pages were visually inspected. All 15 examples and the self-model architecture documents regenerated successfully.
The complete Go test suite and go vet pass for the latest layout batch. Updated
PDFs and final visual review remain required before declaring this batch release-ready.

## Current PDF batch audit (2026-09-23)

All 16 refreshed architecture PDFs rendered successfully (1,077 pages).
The complete text-boundary scan found no words outside page bounds and no
renderer-error text. Contact-sheet review so far covers Aegis Sentinel, the
coffee appliance, Bedrock review app, coffee fleet, and the repository model.
Dense complete graphs remain small at page scale and require zoomed review.
The final SDS hardware page was visually checked: numbered edges avoid the
previous long-label collision, and full descriptions follow in a table.
Remaining visual checks and final test completion are still pending; this
batch is not yet declared release-ready.

The final complete Go test suite and `go vet ./...` passed. Contact sheets
from all 16 publications have now been reviewed for sampled page structure
and clipping. This is a representative review, not a visual inspection of
every page or every dense diagram label. Per-publication PDF hashes and
reviewed page numbers are recorded in the local layout audit.

## Complete overview changes (pending refreshed publications)

Manhattan and requirement-alignment matrices now render as one inline SVG each,
without functional-group or unit column bands. System boundary and decomposition
figures have keep-together wrappers and a bounded SVG height. Boundary groups
use a wider layout. Requirement implementation arrows use an extra routing rank.
The full Go suite passed for the matrix changes; focused functional-diagram tests
also pass after the boundary orientation change. A 19-group/153-unit Manhattan
preview rendered on one PDF page. The largest boundary and decomposition graphs
selected from the instrument publication each rendered on one page; this checks
pagination, not detailed label readability. The wider boundary candidate was
subsequently inspected in a browser preview.

The preceding all-format validation passed external format checks and SysML
checks, but failed artifact freshness because the self architecture document
changed during regeneration. That run is not a passing release gate. All example,
self-model and instrument publications are being refreshed together before final
freshness and PDF review.
