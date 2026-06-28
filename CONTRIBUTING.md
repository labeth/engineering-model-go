# Contributing

Thanks for contributing to `engineering-model-go`.

## Development Setup

1. Install Go.
2. Clone the repository.
3. Run tests:

```bash
go test ./...
```

## Code Guidelines

- Keep behavior deterministic.
- Keep model parsing and rendering logic explicit and testable.
- Keep dependency additions minimal and justified.
- Prefer small, focused pull requests.

## Pull Requests

Before opening a PR:

1. Run formatting and tests:

```bash
gofmt -w .
go test ./...
go vet ./...
```

2. Run the full validation gauntlet:

```bash
./scripts/validate-all.sh
```

   This runs the same strict gates that CI enforces:

   - `go build` succeeds.
   - `engdoc` reports 0 errors (including unresolved `TRLC-LINKS`/`ENGMODEL-LINKS` and dangling code trace links).
   - `engtrace` reports 0 dangling code trace links (exit 1 otherwise).
   - artifact-freshness: regenerated `ARCHITECTURE.adoc`, `DECISIONS.adoc`, and `TRACE-MATRIX.json` show no drift under `git diff`.

   The GitHub Actions `validate` job runs this script, so PRs that fail any strict gate will be blocked.

3. Update tests for behavior changes.
4. Update README if API behavior changed.
5. Keep sample outputs reproducible from the example project.

## Reporting Issues

Please include:

- minimal reproducible model input
- expected behavior
- actual behavior and diagnostics
