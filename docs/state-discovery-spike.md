# State-like code spike results (ADR-EMG-058)

Throwaway measurement; code in internal/health/spike, binary /tmp/statespike. Classification is by reading the declaration and its uses.

Rubric: REAL STATE = value that changes over an object lifetime and drives behavior; ENUMERATION = closed set of kinds/options/results that does not change over time.

## Summary

- Go all (>=2 members): candidates 20, real 1; decl+switch 1/18 real; decl+assign-to-field 1/7 real; both 1/7 real; name signal 1/2 real; assign+name signal 1/2 real; transition-like 0/3 real
- Go >=3 members: candidates 13, real 1; decl+switch 1/13 real; decl+assign-to-field 1/5 real; both 1/5 real; name signal 1/2 real; assign+name signal 1/2 real; transition-like 0/2 real
- TypeScript third-party sample (60 of 89 unique, seed 58): candidates 60, real 1; decl+switch 0/30 real; decl+assign-to-field 1/33 real; both 0/22 real; name signal 1/2 real; assign+name signal 1/2 real; transition-like 0/0 real

## Table

| # | candidate | file:line | lang | members | heuristic hits (switch / typed field / assign-to-field / transition-like) | marker near decl | classification | reason |
|---|---|---|---|---|---|---|---|---|
| 1 | Impact (go-const) | internal/engws/contractdiff.go:22 | go | 3 | 1 / 2 / 4 / 0 | no | ENUMERATION | semver impact level of a contract change; computed result, no transitions |
| 2 | Change (go-const) | internal/engws/contractdiff.go:45 | go | 4 | 1 / 1 / 1 / 0 | no | ENUMERATION | kind of change between two versions of an item; computed result |
| 3 | Publication (go-iota) | internal/engws/registry.go:18 | go | 3 | 2 / 0 / 0 / 0 | no | ENUMERATION | registry answer for one module version; returned value, not stored |
| 4 | Mode (go-const) | internal/export/quint/check/check.go:25 | go | 3 | 1 / 1 / 1 / 1 | no | ENUMERATION | CLI option run/verify/both; the only write defaults it (read then write is defaulting, not a transition) |
| 5 | Format (go-const) | internal/export/threatmodel/threat_model_export.go:18 | go | 2 | 1 / 1 / 0 / 0 | yes | ENUMERATION | export format option |
| 6 | Severity (go-const) | internal/health/types.go:17 | go | 2 | 2 / 4 / 2 / 0 | no | ENUMERATION | diagnostic/finding severity; set from tier rows, never moves over an object lifetime |
| 7 | MachineExprKind (go-const) | internal/model/machine_expr.go:13 | go | 2 | 2 / 0 / 0 / 0 | no | ENUMERATION | kind of a machine expression (boolean/effect) |
| 8 | machineTokenKind (go-iota) | internal/model/machine_expr.go:89 | go | 7 | 4 / 1 / 0 / 0 | yes | ENUMERATION | lexer token kind |
| 9 | MachineType (go-const) | internal/model/machine_expr_ops.go:13 | go | 6 | 6 / 2 / 0 / 0 | no | ENUMERATION | variable type of the machine expression language |
| 10 | MachineOperatorForm (go-const) | internal/model/machine_expr_ops.go:29 | go | 5 | 3 / 1 / 0 / 0 | no | ENUMERATION | operator syntax form |
| 11 | SemanticElementKind (go-const) | internal/model/semantic.go:18 | go | 52 | 5 / 1 / 0 / 0 | no | ENUMERATION | 52 metamodel element kinds |
| 12 | SemanticRelationshipKind (go-const) | internal/model/semantic.go:76 | go | 20 | 5 / 1 / 0 / 0 | no | ENUMERATION | metamodel relationship kinds |
| 13 | SemanticSeverity (go-const) | internal/model/semantic.go:101 | go | 2 | 0 / 1 / 0 / 0 | no | ENUMERATION | diagnostic severity |
| 14 | SemanticFeatureKind (go-const) | internal/model/semantic.go:206 | go | 3 | 3 / 1 / 0 / 0 | no | ENUMERATION | metamodel feature kinds |
| 15 | SpecDeltaOperation (go-const) | internal/reqdelta/spec_delta.go:15 | go | 2 | 0 / 1 / 0 / 0 | no | ENUMERATION | OpenSpec section selector option |
| 16 | State (go-const) | internal/reqlinks/evaluate.go:15 | go | 6 | 5 / 1 / 8 / 0 | no | REAL STATE | evaluated state of a requirement link (unchanged/broken/version_mismatch/...); drives review and reports in 5 packages; the model has SM-LINK-LIFECYCLE for it |
| 17 | Reason (go-const) | internal/reqlinks/evaluate.go:40 | go | 6 | 2 / 1 / 4 / 4 | no | ENUMERATION | cause annotation attached to a link status, not a state |
| 18 | Kind (go-const) | internal/reqlinks/reqlinks.go:25 | go | 2 | 1 / 1 / 0 / 0 | no | ENUMERATION | capability/assumption item kind |
| 19 | VerificationOutcome (go-iota) | internal/validate/assurance.go:145 | go | 6 | 2 / 0 / 0 / 0 | yes | ENUMERATION | result of classifying a verification status |
| 20 | Severity (go-const) | internal/validate/diagnostic.go:13 | go | 2 | 1 / 3 / 6 / 1 | yes | ENUMERATION | diagnostic/finding severity; set from tier rows, never moves over an object lifetime |
| 21 | LanguageModelV2FinishReason (union) | [dep] @ai-sdk/provider/src/language-model/v2/language-model-v2-finish-reason.ts:13 | typescript | 7 | 0 / 2 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 22 | .unified (inline-union) | [dep] @ai-sdk/provider/src/language-model/v3/language-model-v3-finish-reason.ts:20 | typescript | 6 | 0 / 1 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 23 | .strategy (inline-union) | [dep] effect/src/Channel.ts:452 | typescript | 3 | 2 / 1 / 8 / 0 | no | ENUMERATION | queue/channel strategy option fixed at construction |
| 24 | UnitSingular (union) | [dep] effect/src/DateTime.ts:133 | typescript | 8 | 3 / 0 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 25 | UnitPlural (union) | [dep] effect/src/DateTime.ts:150 | typescript | 8 | 1 / 0 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 26 | Disambiguation (union) | [dep] effect/src/DateTime.ts:398 | typescript | 4 | 0 / 10 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 27 | Unit (union) | [dep] effect/src/Duration.ts:115 | typescript | 16 | 3 / 0 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 28 | OpenFlag (union) | [dep] effect/src/FileSystem.ts:684 | typescript | 10 | 0 / 4 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 29 | Type (union) | [dep] effect/src/FileSystem.ts:1157 | typescript | 8 | 7 / 13 / 8 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 30 | MermaidNodeShape (union) | [dep] effect/src/Graph.ts:2839 | typescript | 8 | 1 / 0 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 31 | MermaidDirection (union) | [dep] effect/src/Graph.ts:2884 | typescript | 5 | 0 / 1 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 32 | Dialect (union) | [dep] effect/src/JsonSchema.ts:55 | typescript | 4 | 0 / 2 / 1 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 33 | Type (union) | [dep] effect/src/JsonSchema.ts:67 | typescript | 7 | 14 / 13 / 8 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 34 | LogLevel (union) | [dep] effect/src/LogLevel.ts:60 | typescript | 8 | 2 / 3 / 2 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 35 | Severity (union) | [dep] effect/src/LogLevel.ts:80 | typescript | 6 | 2 / 16 / 1 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 36 | .mode (inline-union) | [dep] effect/src/Logger.ts:1041 | typescript | 3 | 0 / 1 / 2 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 37 | Type (union) | [dep] effect/src/Metric.ts:1052 | typescript | 5 | 8 / 13 / 8 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 38 | ._tag (inline-union) | [dep] effect/src/Optic.ts:1253 | typescript | 4 | 166 / 1 / 9 / 0 | no | ENUMERATION | ADT discriminant tag, a kind of node |
| 39 | Enqueue.strategy (inline-union) | [dep] effect/src/Queue.ts:166 | typescript | 3 | 2 / 1 / 8 / 0 | no | ENUMERATION | queue/channel strategy option fixed at construction |
| 40 | Type (union) | [dep] effect/src/SchemaAST.ts:2323 | typescript | 10 | 17 / 13 / 8 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 41 | SpanKind (union) | [dep] effect/src/Tracer.ts:290 | typescript | 5 | 0 / 5 / 1 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 42 | TxPubSub.strategy (inline-union) | [dep] effect/src/TxPubSub.ts:57 | typescript | 4 | 2 / 1 / 8 / 0 | no | ENUMERATION | queue/channel strategy option fixed at construction |
| 43 | TxQueueState.strategy (inline-union) | [dep] effect/src/TxQueue.ts:137 | typescript | 4 | 2 / 1 / 8 / 0 | no | ENUMERATION | queue/channel strategy option fixed at construction |
| 44 | Node._tag (inline-union) | [dep] effect/src/internal/hashMap.ts:96 | typescript | 5 | 165 / 1 / 9 / 0 | no | ENUMERATION | ADT discriminant tag, a kind of node |
| 45 | WellKnownSystem (union) | [dep] effect/src/unstable/ai/Telemetry.ts:206 | typescript | 14 | 0 / 1 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 46 | Shell (union) | [dep] effect/src/unstable/cli/Completions.ts:19 | typescript | 3 | 1 / 0 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 47 | Param._tag (inline-union) | [dep] effect/src/unstable/cli/Param.ts:45 | typescript | 5 | 165 / 1 / 9 / 0 | no | ENUMERATION | ADT discriminant tag, a kind of node |
| 48 | EntityManagerState.status (inline-union) | [dep] effect/src/unstable/cluster/Sharding.ts:215 | typescript | 3 | 0 / 1 / 6 / 0 | no | REAL STATE | status alive/closing/closed of an entity manager; assigned at 1300/1324, compared with === in 6 places (no switch) |
| 49 | EventLogSessionAuthError.reason (inline-union) | [dep] effect/src/unstable/eventlog/EventLogSessionAuth.ts:109 | typescript | 8 | 1 / 1 / 1 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 50 | Cookie.priority (inline-union) | [dep] effect/src/unstable/http/Cookies.ts:116 | typescript | 3 | 2 / 1 / 3 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 51 | Cookie.sameSite (inline-union) | [dep] effect/src/unstable/http/Cookies.ts:120 | typescript | 3 | 2 / 1 / 3 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 52 | .retryOn (inline-union) | [dep] effect/src/unstable/http/HttpClient.ts:1224 | typescript | 3 | 0 / 1 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 53 | HttpMethod (union) | [dep] effect/src/unstable/http/HttpMethod.ts:18 | typescript | 8 | 0 / 3 / 1 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 54 | MultipartErrorReason._tag (inline-union) | [dep] effect/src/unstable/http/Multipart.ts:198 | typescript | 6 | 165 / 1 / 9 / 0 | no | ENUMERATION | ADT discriminant tag, a kind of node |
| 55 | ._tag (inline-union) | [dep] effect/src/unstable/httpapi/HttpApiBuilder.ts:552 | typescript | 4 | 168 / 1 / 9 / 0 | no | ENUMERATION | ADT discriminant tag, a kind of node |
| 56 | HttpApiSchemaError.kind (inline-union) | [dep] effect/src/unstable/httpapi/HttpApiError.ts:421 | typescript | 5 | 0 / 1 / 1 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 57 | ScalarThemeId (union) | [dep] effect/src/unstable/httpapi/HttpApiScalar.ts:27 | typescript | 12 | 0 / 1 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 58 | ApiKey.in (inline-union) | [dep] effect/src/unstable/httpapi/HttpApiSecurity.ts:76 | typescript | 3 | 0 / 1 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 59 | .in (inline-union) | [dep] effect/src/unstable/httpapi/HttpApiSecurity.ts:179 | typescript | 3 | 0 / 1 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 60 | Signal (union) | [dep] effect/src/unstable/observability/internal/otlpEnv.ts:5 | typescript | 3 | 0 / 1 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 61 | PipeFromOption (union) | [dep] effect/src/unstable/process/ChildProcess.ts:88 | typescript | 3 | 0 / 1 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 62 | CommandInput (union) | [dep] effect/src/unstable/process/ChildProcess.ts:150 | typescript | 4 | 0 / 2 / 1 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 63 | Signal (union) | [dep] effect/src/unstable/process/ChildProcess.ts:176 | typescript | 37 | 0 / 1 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 64 | Encoding (union) | [dep] effect/src/unstable/process/ChildProcess.ts:221 | typescript | 12 | 0 / 4 / 3 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 65 | .encodeInitialAs (inline-union) | [dep] effect/src/unstable/reactivity/Hydration.ts:65 | typescript | 3 | 0 / 1 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 66 | MigrationError.kind (inline-union) | [dep] effect/src/unstable/sql/Migrator.ts:82 | typescript | 5 | 0 / 1 / 1 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 67 | Dialect (union) | [dep] effect/src/unstable/sql/Statement.ts:60 | typescript | 5 | 0 / 2 / 1 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 68 | PrimitiveKind (union) | [dep] effect/src/unstable/sql/Statement.ts:396 | typescript | 8 | 9 / 0 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 69 | FailureReason (union) | [dep] multipasta/src/HeadersParser.ts:3 | typescript | 4 | 0 / 1 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 70 | .limit (inline-union) | [dep] multipasta/src/index.ts:27 | typescript | 4 | 0 / 1 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 71 | StringValidation (union) | [dep] zod/src/v3/ZodError.ts:88 | typescript | 18 | 0 / 1 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 72 | ZodTooSmallIssue.type (inline-union) | [dep] zod/src/v3/ZodError.ts:121 | typescript | 6 | 6 / 1 / 21 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 73 | ZodTooBigIssue.type (inline-union) | [dep] zod/src/v3/ZodError.ts:129 | typescript | 6 | 6 / 1 / 21 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 74 | $ZodTypeDef.type (inline-union) | [dep] zod/src/v4/core/schemas.ts:44 | typescript | 40 | 49 / 1 / 21 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 75 | JSONSchemaGeneratorParams.target (inline-union) | [dep] zod/src/v4/core/to-json-schema.ts:16 | typescript | 4 | 0 / 1 / 1 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 76 | ParsedTypes (union) | [dep] zod/src/v4/core/util.ts:66 | typescript | 16 | 49 / 0 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 77 | BuiltInValidationStrategy (union) | [dep] @eslint/object-schema/dist/esm/types.ts:8 | typescript | 7 | 0 / 2 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 78 | $ZodStringFormats (union) | [dep] zod/src/v4/core/checks.ts:689 | typescript | 32 | 2 / 1 / 4 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 79 | SupportedStringFormat (union) | [dep] zod/src/v4/core/compile.ts:737 | typescript | 6 | 1 / 0 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |
| 80 | JSONSchemaGeneratorParams.target (inline-union) | [dep] zod/src/v4/core/to-json-schema.ts:49 | typescript | 4 | 0 / 1 / 0 / 0 | no | ENUMERATION | option/kind/format/type-name set; fixed, not a lifecycle |

## Real states found and the repo model

- reqlinks.State (internal/reqlinks/evaluate.go:15): modeled at the concept level by SM-LINK-LIFECYCLE (owner FU-ALLOCATION-TRACE, same unit as the code), but no code member carries a STATE-LINK-* ID or TR-LINK-* marker, and the code has 6 values (unchanged, broken, version_mismatch, child_unpublished, unresolved, local) against 5 model states (ABSENT, CLAIMED, ACCEPTED, SUSPECT, BROKEN). Four code values have no model state; the model states CLAIMED/ACCEPTED live in the untyped string LinkStatus.Status (evaluate.go:84), which the pass cannot see.
- EntityManagerState.status (third-party effect library, not a project model): real, found by assignment, missed by the switch heuristic (compared with === only).

## Misses of the pass (not detected, noted while checking)

- SM-MCP-SESSION: held as a fingerprint string in internal/mcp/server.go:49, no enum.
- SM-REQUIREMENTS-DELTA: no enum; the written flag and phases are implicit control flow.
- internal/engws/plan.go:31 Entry.State string with untyped constants StateDirty, StatePinned, StatePublished, StateUnknown (plan.go:18): a derived classification, not a lifecycle; invisible to a typed-constant pass.
- effect TxQueue State (_tag Open/Closing/Done): a tagged union of object types, invisible to a string-literal union pass.
