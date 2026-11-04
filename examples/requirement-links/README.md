# Requirement links sample

Three independently publishable repositories with `requirementLinks: strict`:

- `repos/child-a` (position encoder) provides `CAP-A-1` (publish position sentences) and
  `CAP-A-2` (checksum). `REQ-A-001` and `REQ-A-002` realize `CAP-A-1`, `REQ-A-003` realizes
  `CAP-A-2`, and `REQ-A-004` is derived.
- `repos/child-b` (lab bus bridge) provides `CAP-B-1` and the assumption of use `AOU-B-1`
  (the lab bus is isolated). `REQ-B-001` realizes both.
- `repos/system` (position simulator) composes both. `model/req-links.yml` links each
  system requirement to the contract items that satisfy it, and evidences `bridge::AOU-B-1`
  with the derived requirement `REQ-SYS-003`. Each link is pinned to `v0.1.0`, the
  published version it was reviewed at.

A child never names its users. The links live only in the system. Rendering child-a with
`engdoc --used-by` in this workspace adds a Used By section with the system's links; that
render is for local reading only and is never committed into the child. `engmod.work.yml`
replaces both children with their directories under `repos/`, so the system resolves
them offline.

```sh
go run ./cmd/engdoc --model examples/requirement-links/repos/system/engmod.yml \
  --requirements examples/requirement-links/repos/system/model/requirements.yml \
  --design examples/requirement-links/repos/system/model/views.yml --out /tmp/system.adoc
go test ./internal/integration -run RequirementLinks
```

Editing a child checkout in this workspace never changes a link's state. When child-a
publishes a release that changes the text of a requirement realizing `CAP-A-1`,
`engws bump system encoder <version>` leaves that link at `v0.1.0`, and the system reports
`link.version_mismatch` until someone reviews it (`engws review-links --show`, then
`--accept`). A release without `CAP-A-2` reports `link.broken`. When child-b publishes a new
assumption of use, the system reports `link.aou_unverified` until a link with `evidence`
names it (`engws link add --evidence REQ-… bridge::AOU-…`).
