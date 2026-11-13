# Code health dogfood: the scan on two real repositories

This is the calibration evidence for the limits of ADR-EMG-047, ADR-EMG-057 and ADR-EMG-059. The owner
decides whether the limits stay; this page only reports what they do. The scores are this project's own
(ADR-EMG-046), not CodeScene scores.

There have been two runs. The **first** used the limits proposed in ADR-EMG-057 and scoring version 1
(no cap on the tail). The **second** (this page) uses the calibrated limits of ADR-EMG-059 (complex_method
8 and 15, large_method 50 and 100, deep_nesting 3 and 4, bumpy_road 2 and 3, complex_conditional 3 and 5,
excess_arguments 4 and 6, large_assertion_block 5 and 8, the same at product and operational) and scoring
version 2 (the functions beyond the second count at most 2.0 points together).

What was scanned, with the `enghealth` of this branch (engine `treesitter` 2):

- `engineering-model-go` (Go only): 230 production files with 54,430 logical lines and 211 test files. One
  generated file, `internal/model/sysml_metamodel_generated.go`, is above the 4 MiB limit and is listed as a
  `health.file_skipped` diagnostic. The first run saw 224 production files; the code has grown by the files
  of this feature since.
- `engineering-flow` (its own `engmod.yml`): 40 production files with 11,712 logical lines (31 JavaScript,
  9 Go) and 25 test files. Dot directories are not scanned by convention. No TypeScript, TSX or Python
  source outside `node_modules` exists there, so those languages were only exercised by the engine tests.

```
enghealth scan --model <engmod.yml> --tier operational --format json   # product uses the same limits
enghealth scan --model <engmod.yml> --tier mvp         --format json   # the default limits
```

## Score distribution

Production files per band and the project score (the LOC-weighted mean). Median is of the file scores.

| Repository | Limits | Run | Project | Green | Yellow | Red | Median | Functions with a finding | Findings |
|---|---|---|---|---|---|---|---|---|---|
| model-go | default (mvp) | first | 4.1 yellow | 98 | 86 | 40 | 7.45 | 28% | 1,660 warnings |
| model-go | default (mvp) | **second** | 4.2 yellow | 104 | 85 | 41 | 7.5 | 27% | 1,723 warnings |
| model-go | strict (operational) | first | 2.0 red | 56 | 43 | 125 | 2.6 | 55% | 4,486 errors |
| model-go | strict (operational) | **second** | 2.8 red | 83 | 57 | 90 | 6.0 | 35% | 2,705 errors |
| engineering-flow | default (mvp) | first | 3.3 red | 16 | 13 | 11 | 6.45 | 28% | 391 warnings |
| engineering-flow | default (mvp) | **second** | 3.6 red | 16 | 13 | 11 | 6.45 | 28% | 391 warnings |
| engineering-flow | strict (operational) | first | 1.6 red | 7 | 8 | 25 | 1.0 | 54% | 1,160 errors |
| engineering-flow | strict (operational) | **second** | 2.6 red | 14 | 9 | 17 | 4.95 | 37% | 652 errors |

The test files are scored apart. At the strict limits the tests of model-go went from 3.0 red (51, 71, 82)
to 5.0 yellow (92, 81, 38) and those of engineering-flow from 3.2 red (6, 8, 11) to 5.0 yellow (9, 11, 5);
at the default limits they are unchanged (6.6 and 6.7 yellow).

Findings by code at the strict limits, production code only (the error counts in the table above include test files), production code only. model-go: complex_method 528, deep_nesting
327, bumpy_road 261 (was 889), large_method 188, excess_arguments 150, brain_method 125,
complex_conditional 113 (was 288), primitive_obsession 56. engineering-flow: complex_method 161,
deep_nesting 90, bumpy_road 63 (was 245), complex_conditional 38, excess_arguments 37, large_method 36,
brain_method 21, primitive_obsession 7.

## What each change did

**The calibrated limits did almost all of the work.** Flagged production functions at the strict limits fell
from 55% to 35% in model-go and from 54% to 37% in engineering-flow, red files from 56% to 39% and from 63%
to 43%, and the number of errors from 4,486 to 2,705 and from 1,160 to 652. `bumpy_road` at 2 instead of 1
removed about two thirds of its findings (889 to 261 and 245 to 63). The default limits are unchanged, so
the mvp rows move only through the cap.

**Scoring version 2 did very little, and that corrects an assumption of ADR-EMG-059.** Recomputing every
file of the second run under version 1 (checked: the version 2 recomputation reproduces the reports) shows
that the cap changes the score of 2 production files at the strict limits of model-go and none in
engineering-flow, and of 4 and 3 files at the default limits, with no band change. At the default limits
the project score moves by +0.1 (4.1 to 4.2) in model-go and +0.3 (3.3 to 3.6) in engineering-flow. The
reason is that almost every file that is red is red because of its two worst functions: 87 of the 90 red
production files of model-go and 15 of the 17 of engineering-flow (9 of 11 at the default limits) would
still be red if the tail counted nothing at all. File size alone decides the band of few files. The cap
is still correct and cheap protection (a file of many mildly bad functions can no longer fall below its
worst two functions plus 2.0 points), but it is not what makes the code red.

**The code written for this feature**, `internal/health`, `cmd/enghealth` and `api_health.go`, now has 32
production files, 5,115 logical lines and 261 functions (the first run: 27 files, 203 functions). At the
strict limits it scores 5.2 yellow with 10 green, 14 yellow and 8 red files (first run: 2.6 red; 3, 7, 17), and
81 of the 261 functions (31%) carry a finding (first run: 110 of 203, 54%). At the default limits it scores
7.3 yellow with 15 green, 16 yellow and 1 red file. The eight red files are code of the earlier phases (the
engine's language modules, `git.go`, `delta.go`, `import.go`, `sarif.go`, `rollup.go`, `scan.go`); the
files of this phase (`check.go` of the library, of the pipeline and of the CLI, `scoring.go`) were
refactored until they scan clean at product, which took a few small helper functions each and shows that
the limits are reachable for new code.

## What remains red, and why

At the strict limits 90 of 230 production files (39%) in model-go and 17 of 40 (43%) in engineering-flow
are red, and both projects score red as a whole (2.8 and 2.6). Reasons, in order of weight:

1. **The worst functions are real design problems.** Of the five worst functions of model-go,
   `InferVerificationChecks` (cyclomatic 64, 299 lines), `main` of `cmd/engoscal` (44, 171),
   `buildThreatDragonV2` (37, 261), `buildRequirementCoverageMermaid` (40, 155) and
   `buildTraceMatrixWithIDs` (32, 125) each carry six high findings; in engineering-flow
   `evaluateRules` (52, 109), `run` of `engmod-check` (44, 169) and `stageFindings` (48, 99) carry four or five. The score
   calls them red at any limit, and a limit that did not would be wrong.
2. **A red file needs only two such functions**, since the second counts half, and the validators and
   generators of the model packages are long functions with many branches. Raising the limits further
   would not make these functions pass; it would only hide the many smaller ones.
3. **35% of the functions still carry a finding** at 8/15, 50/100 and 3/4 nesting. That is the strict bar
   the owner asked for, applied to code that was not written for it; it is debt, not noise. The limits
   most likely to be disputed are `complex_method` at 8 (528 findings) and `deep_nesting` at 3 (327).
4. **Tests are yellow, not red**, at 5.0, and the large_assertion_block limit of 5 produced 33 findings.

What lets a project with this much debt adopt product or operational is not a different limit but the
other parts of ADR-EMG-059: `enghealth check` fails on a new or worsened finding of severity high (exit 1 when that
regression is an error at the tier) while existing debt stands; a new function at warn level is an error
in `enghealth scan` at the two highest tiers, not in `check`, and the committed baseline with an
accepted decision turns the existing findings into a list that can only shrink. Generated files belong in
`health.exclude`; the 4 MiB limit caught one, smaller generated files would be scored.

**The owner confirms or moves the limits.** Every number on this page and in ADR-EMG-059 is an OWNER item;
the limits above are the starting point of the tier ladder, not a decision.

## The five worst functions at the strict limits

Ranked by the points their findings deduct (weight times severity factor, before the combination across
functions). A trailing `!` is a high finding.

| Points | Function | cc | loc | Findings |
|---|---|---|---|---|
| 12.3 | `internal/codescan/inferred_verification.go:60 InferVerificationChecks` | 64 | 299 | complex_method!, large_method!, deep_nesting!, bumpy_road!, complex_conditional!, excess_arguments, brain_method! |
| 11.8 | `cmd/engoscal/main.go:18 main` | 44 | 171 | complex_method!, large_method!, deep_nesting!, bumpy_road!, complex_conditional!, brain_method! |
| 11.8 | `internal/export/threatmodel/threat_model_export.go:151 buildThreatDragonV2` | 37 | 261 | complex_method!, large_method!, deep_nesting!, bumpy_road!, complex_conditional!, brain_method! |
| 11.6 | `internal/asciidoc/asciidoc_diagrams_core.go:598 buildRequirementCoverageMermaid` | 40 | 155 | complex_method!, large_method!, deep_nesting!, bumpy_road!, excess_arguments!, brain_method! |
| 11.6 | `internal/trace/trace_matrix.go:651 buildTraceMatrixWithIDs` | 32 | 125 | complex_method!, large_method!, deep_nesting!, bumpy_road!, excess_arguments!, brain_method! |

engineering-flow: `cicd/engflow-pr-policy.mjs:267 evaluateRules` 11.2 (cc 52, 109 lines),
`cicd/engmod-check/main.go:30 run` 10.2 (44, 169), `cicd/engflow/constants.mjs:119 constantMutations`,
`cicd/engflow/stage-check.mjs:122 stageFindings` and `cicd/engmod-check/consistency.go:150
scenarioDriftDiagnostics` 9.8 each.

## Runtime

Wall time and peak memory of one process, on this machine, cold file cache not controlled:

| Command | Files | Time | Peak memory |
|---|---|---|---|
| `scan`, model-go, whole repository | 441 | 5.0 to 5.4 s | 305 MB |
| `scan`, engineering-flow, whole repository | 65 | 1.2 s | 110 MB |
| `check`, model-go, one changed file, warm cache | 1 | 0.2 to 0.4 s | 80 MB |
| `check`, model-go, nothing changed in the path (model load and git only) | 0 | 0.3 to 0.4 s | 80 MB |
| `check`, model-go, the whole uncommitted change set, warm cache | 59 | 0.3 s | 82 MB |
| `check`, model-go, the same, `--no-cache` | 59 | 0.65 s | 90 MB |

`scan` stays at the cost of the first run: the roll-ups need the code map (the owner marker of each file and
the `TRLC-LINKS` symbols come from `codescan.InferCodeItems`, a tree-sitter pass over the whole module of
about 3.7 s), so even a one-file scan costs about 3.3 s. `check` needs no roll-ups, so it never makes that
pass: it loads the model, asks git for the changed files and analyses only those, from the result cache
when it can (the cache key includes the content, the limits and the scoring version). The time is
dominated by loading the model and starting git, not by the analysis: a check of a path with no change
costs as much as one of a changed file, and 59 changed files cost no more than one when warm. The target
of ADR-EMG-054 was 300 ms for a single file when warm; the measured 0.2 to 0.4 s is at that bound, under
the 1 s that was asked for this phase. Two runs on the same tree produce byte-identical output.

## Update after the review fixes (engine version 2, scoring version 2)

Re-measured after the independent reviews (ADR-EMG-064 and ADR-EMG-065): nested named units own their lines, the
calibrated limits of ADR-EMG-059 apply at product and operational, and the cache is off (`--no-cache`). Production files
only; "functions with a finding" counts production functions with at least one finding; "findings" is the summary total
including test files.

| Repository | Tier | Project | Green / yellow / red | Median file | Functions with a finding | Findings | Time, peak memory |
|---|---|---|---|---|---|---|---|
| engineering-model-go (232 files, 55,358 LOC) | mvp | 4.3 yellow | 105 / 87 / 40 | 7.5 | 27% (588 of 2,169) | 1,782 warnings | 5.4 s, 296 MB |
| engineering-model-go | operational | 2.8 red | 84 / 60 / 88 | 6.0 | 35% (762) | 2,800 errors | 5.3 s, 308 MB |
| engineering-flow (40 files, 11,712 LOC) | mvp | 3.7 red | 16 / 13 / 11 | 6.45 | 28% (165 of 595) | 385 warnings | 1.2 s, 108 MB |
| engineering-flow | operational | 2.6 red | 14 / 9 / 17 | 4.95 | 37% (218) | 646 errors | 1.2 s, 105 MB |

The only diagnostic in either repository is the skipped generated file `internal/model/sysml_metamodel_generated.go`.
Test files score 6.5 yellow at mvp and 4.9 yellow at operational in engineering-model-go, and 6.7 and 5.1 in
engineering-flow. The LOC fix changed few numbers (engineering-flow `large_method` 36 to 34); the calibrated limits
remain the lever. Roughly a third of the production functions of both repositories carry an error at operational, so a
project of this size adopts the tier with a baseline and an accepted decision (ADR-EMG-059), and new code is written to the
limits from the start. The owner confirms or moves the limits before the release.
