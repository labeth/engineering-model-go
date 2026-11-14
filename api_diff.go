// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package engmodel

import (
	"fmt"
	"path/filepath"

	"github.com/labeth/engineering-model-go/internal/gitmodel"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/trace"
)

// Entity is one authored thing with a stable ID: a requirement, a functional unit, a threat, a decision, a
// state of a machine, a step of a flow.
// ENGMODEL-LINKS: FU-MODEL-LOADER
type Entity = model.Entity

// EntityReference is a directed link the model states between two entities.
// ENGMODEL-LINKS: FU-MODEL-LOADER
type EntityReference = model.EntityReference

// EntityIndex is every authored entity of a bundle and every link between them, in a deterministic order.
// ENGMODEL-LINKS: FU-MODEL-LOADER
type EntityIndex = model.EntityIndex

// EntityChange is one entity that differs between two states of a model.
// ENGMODEL-LINKS: FU-MODEL-LOADER
type EntityChange = model.EntityChange

// EntityDiff is what changed between two entity indexes.
// ENGMODEL-LINKS: FU-MODEL-LOADER
type EntityDiff = model.EntityDiff

// TraceDelta is how the traceability of requirements moved between two trace matrices.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type TraceDelta = trace.Delta

// TraceStatusChange is a requirement whose trace status differs between two matrices.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type TraceStatusChange = trace.StatusChange

// The edge types of an EntityReference.
const (
	EntityEdgeField        = model.EdgeField
	EntityEdgeMention      = model.EdgeMention
	EntityEdgeContains     = model.EdgeContains
	EntityEdgeRelationship = model.EdgeRelationship
)

// BuildEntityIndex reads every authored document of the bundle into entities and the references between them.
// TRLC-LINKS: REQ-EMG-189
// ENGMODEL-LINKS: FU-MODEL-LOADER
func BuildEntityIndex(bundle Bundle) (EntityIndex, error) { return model.BuildEntityIndex(bundle) }

// DiffEntityIndexes compares two entity indexes of one model by stable ID.
// TRLC-LINKS: REQ-EMG-190
// ENGMODEL-LINKS: FU-MODEL-LOADER
func DiffEntityIndexes(from, to EntityIndex) EntityDiff { return model.DiffEntityIndexes(from, to) }

// DiffTraceMatrices compares two trace matrices of one model by requirement ID.
// TRLC-LINKS: REQ-EMG-191
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func DiffTraceMatrices(from, to TraceMatrix) TraceDelta { return trace.DeltaOf(from, to) }

// LoadBundleAtRef loads the model in repo as committed at a git ref. It is the one place that accepts a model of
// an older schema version (it stamps a private copy), and it is meant for reading history, never for validating a
// user's model.
// TRLC-LINKS: REQ-EMG-190
// ENGMODEL-LINKS: FU-MODEL-LOADER
func LoadBundleAtRef(repo, ref string) (Bundle, error) { return gitmodel.LoadBundleAtRef(repo, ref) }

// ModelDiffOptions selects what DiffModels computes.
type ModelDiffOptions struct {
	// Trace also builds the trace matrix of both states (it scans the code of each) and reports how traceability moved.
	Trace bool
}

// ModelDiff is the difference of the model in one repository between two states.
type ModelDiff struct {
	Repo string `json:"repo"`
	From string `json:"from"`
	// To is the ref compared against; empty means the working tree.
	To       string     `json:"to"`
	Entities EntityDiff `json:"entities"`
	// Trace is nil when it was not asked for or could not be built on one side; TraceError says why.
	Trace      *TraceDelta `json:"trace,omitempty"`
	TraceError string      `json:"traceError,omitempty"`
}

// DiffModels compares the model of repo as committed at fromRef with its state at toRef, or with the working tree
// when toRef is empty. With opts.Trace it also reports how the traceability of the requirements moved: it extracts
// each committed tree into a private temporary directory, scans its code and builds the trace matrix, so a
// requirement that gained or lost code or tests shows.
//
// TRLC-LINKS: REQ-EMG-190, REQ-EMG-191
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-ALLOCATION-TRACE
func DiffModels(repo, fromRef, toRef string, opts ModelDiffOptions) (ModelDiff, error) {
	out := ModelDiff{Repo: repo, From: fromRef, To: toRef}
	from, err := LoadBundleAtRef(repo, fromRef)
	if err != nil {
		return out, err
	}
	var to Bundle
	if toRef == "" {
		to, err = model.LoadBundle(filepath.Join(repo, "engmod.yml"))
	} else {
		to, err = LoadBundleAtRef(repo, toRef)
	}
	if err != nil {
		return out, err
	}
	a, err := BuildEntityIndex(from)
	if err != nil {
		return out, err
	}
	b, err := BuildEntityIndex(to)
	if err != nil {
		return out, err
	}
	out.Entities = DiffEntityIndexes(a, b)
	if !opts.Trace {
		return out, nil
	}
	before, err := traceAtRef(repo, fromRef)
	if err != nil {
		out.TraceError = fmt.Sprintf("trace at %s: %v", fromRef, err)
		return out, nil
	}
	var after TraceMatrix
	if toRef == "" {
		after, err = traceOfDir(repo)
	} else {
		after, err = traceAtRef(repo, toRef)
	}
	if err != nil {
		out.TraceError = fmt.Sprintf("trace at %s: %v", labelOf(toRef), err)
		return out, nil
	}
	delta := DiffTraceMatrices(before, after)
	out.Trace = &delta
	return out, nil
}

// TRLC-LINKS: REQ-EMG-190
func labelOf(ref string) string {
	if ref == "" {
		return "the working tree"
	}
	return ref
}

// TRLC-LINKS: REQ-EMG-190
func traceAtRef(repo, ref string) (TraceMatrix, error) {
	dir, cleanup, err := gitmodel.CheckoutAtRef(repo, ref)
	if err != nil {
		return TraceMatrix{}, err
	}
	defer cleanup()
	return traceOfDir(dir)
}

// traceOfDir is the trace matrix of the module checked out in dir, the requirements document being the one the
// manifest names.
// TRLC-LINKS: REQ-EMG-190
func traceOfDir(dir string) (TraceMatrix, error) {
	manifest := filepath.Join(dir, "engmod.yml")
	b, err := LoadBundle(manifest)
	if err != nil {
		return TraceMatrix{}, err
	}
	m, _, err := BuildTraceMatrixFromFiles(manifest, b.RequirementsPath, dir)
	return m, err
}
