# Requirement links sample

Three independently publishable repositories with `requirementLinks: strict`:

- `repos/child-a` (position encoder) provides `CAP-A-1` (publish position sentences) and
  `CAP-A-2` (checksum). `REQ-A-001` and `REQ-A-002` realize `CAP-A-1`, `REQ-A-003` realizes
  `CAP-A-2`, and `REQ-A-004` is derived.
- `repos/child-b` (lab bus bridge) provides `CAP-B-1` and the assumption of use `AOU-B-1`
  (the lab bus is isolated). `REQ-B-001` realizes both.
- `repos/system` (position simulator) composes both. `model/req-links.yml` links each
  system requirement to the contract items that satisfy it, and evidences `bridge::AOU-B-1`
  with the derived requirement `REQ-SYS-003`. Each link is pinned to `v0.1.0` and the
  fingerprint of the item at that version.

A child never names its users. The links live only in the system. `engmod.work.yml`
replaces both children with their directories under `repos/`, so the system resolves
them offline.

```sh
go run ./cmd/engdoc --model examples/requirement-links/repos/system/engmod.yml \
  --requirements examples/requirement-links/repos/system/model/requirements.yml \
  --design examples/requirement-links/repos/system/model/views.yml --out /tmp/system.adoc
go test ./internal/integration -run RequirementLinks
```

When a requirement that realizes `CAP-A-1` changes, the system reports `link.suspect` until
someone reviews the link and re-stamps its fingerprint. When `CAP-A-2` is removed, it reports
`link.broken`. When child-b publishes a new assumption of use, it reports
`link.aou_unverified` until a link with `evidence` names it.
