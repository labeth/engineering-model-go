#!/usr/bin/env bash
# Red checks of the Go API requirements REQ-EMG-189 to REQ-EMG-206 (the facade of ADR-EMG-079 to 081).
#
# Every file here is a list of mutations of the engine's own source for red-check.mjs of engineering-flow: the
# requirement the mutation is evidence for, the file, the text to find and the text to put in its place. A mutation is
# RED when the tests named below fail on an assertion with it applied and pass without it; the tree is restored after
# each. The verdicts are written to .engflow/red-checks.jsonl (git-ignored), which `engws status-suggest` reads for the
# tree it runs on, so run this on the tree you want the status of:
#
#   RED_CHECK=path/to/engineering-flow/.github/engflow/red-check.mjs scripts/red-checks/run.sh
#
# It takes about an hour on a loaded machine (one test build per mutation). Exit 0 only when every mutation is RED.
set -u
cd "$(dirname "$0")/../.."
RED_CHECK=${RED_CHECK:-../engineering-flow/.github/engflow/red-check.mjs}
export GONOSUMDB='github.com/labeth/*'
status=0
export GOFLAGS=${GOFLAGS:--p=2}
check() { # check <mutations file> <go test arguments...>
  local file=$1; shift
  echo "=== scripts/red-checks/$file"
  node "$RED_CHECK" --mutations "scripts/red-checks/$file" -- go test -count=1 -p 1 "$@" || status=1
}
# The facade tests of the root package (api_*_test.go, facade_test.go) and the code lookup tests.
check go-api-root.json . -run 'TestBuildEntityIndex|TestDiff|TestLoadBundle|TestBuildTraceMatrix|TestContractItems|TestSortedContract|TestEvaluateRequirementLinks|TestGenerateComposition|TestWorkspaceLinkUsers|TestCatalogUsage|TestValidateBundle|TestFindCodeDefinitions|TestListCodeDeclarations|TestCodeDeclarationsOf|TestFacadeLoadsAndValidates'
# The tests of the internal functions the facade delegates to, which carry REQ-EMG-189 to REQ-EMG-193 too.
check internal-model.json ./internal/model -run 'TestEntityIndex|TestDiffEntityIndexes|TestDiffOf|TestDiffLists'
check internal-trace.json ./internal/trace -run 'TestDeltaOf'
check internal-reqlint.json ./internal/reqlint -run 'TestCatalogUsage'
check internal-codemap.json ./internal/codemap
check internal-engws.json ./internal/engws -run 'TestModelDiff'
exit $status
