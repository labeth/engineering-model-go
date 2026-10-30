// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION

// Package engmodel is the public Go API of engineering-model-go. Every other
// package of the module is internal; the types and functions listed here are
// the supported surface.
//
// The API areas are:
//
//   - Model: LoadBundle reads an engmod.yml manifest and its domain documents
//     into a Bundle (Requirement, FunctionalUnit, Interface, Decision,
//     CatalogEntry and the other document and entity types are aliases of the
//     loader's types). LoadCanonicalBundle, NewCanonicalBundle and
//     ProjectSemanticModel give the canonical semantic model;
//     BuildAuthoringContract describes how to author it. LoadWorkspace,
//     FindWorkspace and FindWorkspaceWith resolve engmod.work.yml workspaces.
//   - Validation: ValidateBundle returns Diagnostic values; HasErrors,
//     SortDiagnostics, DedupeDiagnostics and AuthoringRules work on them.
//   - Composition: ResolveCompositionProjection and the Generate*Composition*
//     functions resolve and render systems of systems across model modules.
//   - Views and documents: Generate and GenerateFromFile project one view to
//     Mermaid; GenerateAsciiDoc* renders the architecture document.
//   - Trace: BuildTraceMatrixFromFiles links requirements, code and tests.
//   - Exports: Structurizr DSL, SysML v2, NAF, TRLC, LOBSTER, Threat Dragon
//     and Open Threat Model, Gemara, OSCAL and airborne assurance packages
//     (the Generate*, Export* and Validate* functions named after each format).
//
// The implementation lives in internal/engine; the files of this package are
// type aliases, constants and thin wrappers over it, grouped by area
// (model_api.go, validate_api.go, api_<area>.go).
//
// The command-line tools under cmd/ are thin wrappers over this API.
package engmodel
