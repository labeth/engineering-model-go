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

// ModelDiffRequest says which two states of the model in a repository DiffModels compares.
type ModelDiffRequest struct {
	Repo string
	From string
	// To is the ref compared against; empty means the working tree.
	To string
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

// DiffModels compares the model of req.Repo as committed at req.From with its state at req.To, or with the working
// tree when req.To is empty. With req.Trace it also reports how the traceability of the requirements moved: it
// extracts each committed tree into a private temporary directory, scans its code and builds the trace matrix, so a
// requirement that gained or lost code or tests shows. A trace that cannot be built is recorded in TraceError and
// does not fail the entity diff.
//
// TRLC-LINKS: REQ-EMG-190, REQ-EMG-191
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-ALLOCATION-TRACE
func DiffModels(req ModelDiffRequest) (ModelDiff, error) {
	out := ModelDiff{Repo: req.Repo, From: req.From, To: req.To}
	before, err := entityIndexAt(req.Repo, req.From)
	if err != nil {
		return out, err
	}
	after, err := entityIndexAt(req.Repo, req.To)
	if err != nil {
		return out, err
	}
	out.Entities = DiffEntityIndexes(before, after)
	if req.Trace {
		out.Trace, out.TraceError = traceDelta(req)
	}
	return out, nil
}

// entityIndexAt indexes the model as committed at ref, or the working tree when ref is empty.
// TRLC-LINKS: REQ-EMG-190
func entityIndexAt(repo, ref string) (EntityIndex, error) {
	var b Bundle
	var err error
	if ref == "" {
		b, err = model.LoadBundle(filepath.Join(repo, "engmod.yml"))
	} else {
		b, err = LoadBundleAtRef(repo, ref)
	}
	if err != nil {
		return EntityIndex{}, err
	}
	return BuildEntityIndex(b)
}

// traceDelta compares the trace of the two states; on failure it returns no delta and the reason.
// TRLC-LINKS: REQ-EMG-191
func traceDelta(req ModelDiffRequest) (*TraceDelta, string) {
	before, err := traceAt(req.Repo, req.From)
	if err != nil {
		return nil, fmt.Sprintf("trace at %s: %v", labelOf(req.From), err)
	}
	after, err := traceAt(req.Repo, req.To)
	if err != nil {
		return nil, fmt.Sprintf("trace at %s: %v", labelOf(req.To), err)
	}
	delta := DiffTraceMatrices(before, after)
	return &delta, ""
}

// traceAt is the trace matrix of the committed tree at ref, or of the working tree when ref is empty.
// TRLC-LINKS: REQ-EMG-191
func traceAt(repo, ref string) (TraceMatrix, error) {
	if ref == "" {
		return traceOfDir(repo)
	}
	return traceAtRef(repo, ref)
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
