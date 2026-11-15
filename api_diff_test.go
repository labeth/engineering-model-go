// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package engmodel_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

// selfBundle loads the model of this repository through the facade.
func selfBundle(t *testing.T) engmodel.Bundle {
	t.Helper()
	isolatedModel(t)
	bundle, err := engmodel.LoadBundle("engmod.yml")
	if err != nil {
		t.Fatalf("LoadBundle: %v", err)
	}
	return bundle
}

// selfIndex is the entity index of this repository's model.
func selfIndex(t *testing.T) engmodel.EntityIndex {
	t.Helper()
	index, err := engmodel.BuildEntityIndex(selfBundle(t))
	if err != nil {
		t.Fatalf("BuildEntityIndex: %v", err)
	}
	return index
}

// findEntity returns the entity with the ID, or fails the test.
func findEntity(t *testing.T, index engmodel.EntityIndex, id string) engmodel.Entity {
	t.Helper()
	entity, ok := index.Find(id)
	if !ok {
		t.Fatalf("%s is not in the index", id)
	}
	return entity
}

// entityWant is what an entity of this repository's model must be: its kind, parent, a document that defines it and a field.
type entityWant struct{ kind, parent, document, field string }

// BuildEntityIndex lists the authored entities of this repository with kind, document, parent and fields.
// Mutation: a wrong kind table, a dropped parent or a dropped field of an entity fails this test.
// TRLC-LINKS: REQ-EMG-189, REQ-EMG-196
func TestBuildEntityIndexListsEntitiesWithKindDocumentParentAndFields(t *testing.T) {
	index := selfIndex(t)
	expect(t, len(index.Entities) >= 400, "%d entities, want at least 400", len(index.Entities))
	cases := map[string]entityWant{
		"REQ-EMG-001":                     {"requirement", "", "requirements", "text"},
		"FU-MODEL-LOADER":                 {"functional_unit", "", "architecture", "prose"},
		"TS-ARTIFACT-PARSE-ABUSE":         {"threat_scenario", "", "assurance", ""},
		"VER-ARTIFACT-FRESHNESS-GATE":     {"control_verification", "", "assurance", ""},
		"POAM-CONSISTENCY-BATCH1":         {"poam_item", "", "assurance", "dueDate"},
		"COND-CATALOG-REFERENCES-RESOLVE": {"catalog_term", "", "catalog", ""},
		"SM-ARTIFACT-FRESHNESS":           {"machine", "", "behavior", ""},
		"STATE-ARTIFACTS-FRESH":           {"state", "SM-ARTIFACT-FRESHNESS", "behavior", ""},
		"INV-ARTIFACTS-FRESH-VALIDATED":   {"invariant", "SM-ARTIFACT-FRESHNESS", "behavior", ""},
		"build-entity-index":              {"flow_step", "FLOW-GO-READ-MODEL", "behavior", ""},
		"ADR-EMG-001":                     {"decision", "", "decisions", ""},
	}
	for id, want := range cases {
		e := findEntity(t, index, id)
		check{t}.equal(id+" kind and parent", [2]string{e.Kind, e.Parent}, [2]string{want.kind, want.parent})
		expect(t, e.Title != "" && slices.Contains(e.Documents, want.document), "%s: title %q, documents %v lack %s", id, e.Title, e.Documents, want.document)
		_, hasField := e.Fields[want.field]
		expect(t, hasField || want.field == "", "%s: fields %v lack %q", id, e.Fields, want.field)
	}
	unit := findEntity(t, index, "FU-MODEL-LOADER")
	check{t}.equal("first and last document of a unit the architecture defines and the catalog names", []string{unit.Documents[0], unit.Documents[len(unit.Documents)-1]}, []string{"architecture", "catalog"})
	_, nested := findEntity(t, index, "SM-ARTIFACT-FRESHNESS").Fields["states"]
	expect(t, !nested, "a nested entity is an entity of its own, not a field of its parent")
}

// edgeTypes counts the references of each type and fails the test for a reference with an unknown end.
func edgeTypes(t *testing.T, index engmodel.EntityIndex) map[string]int {
	t.Helper()
	types := map[string]int{}
	for _, r := range index.References {
		types[r.Type]++
		_, from := index.Find(r.From)
		_, to := index.Find(r.To)
		expect(t, from && to, "reference %+v has an unknown end", r)
	}
	return types
}

// hasReference reports whether the index holds the reference, whatever its description.
func hasReference(index engmodel.EntityIndex, want engmodel.EntityReference) bool {
	return slices.ContainsFunc(index.References, func(r engmodel.EntityReference) bool {
		r.Description = ""
		return r == want
	})
}

// Every reference has two known ends, the four edge types are all reported and REQ-EMG-001 applies to FU-MODEL-LOADER.
// Mutation: dropping an edge type, a reference to an unknown ID or the appliesTo edge fails this test.
// TRLC-LINKS: REQ-EMG-189, REQ-EMG-196
func TestBuildEntityIndexReportsReferencesBetweenKnownEntities(t *testing.T) {
	index := selfIndex(t)
	types := edgeTypes(t, index)
	for _, edge := range []string{engmodel.EntityEdgeField, engmodel.EntityEdgeMention, engmodel.EntityEdgeContains, engmodel.EntityEdgeRelationship} {
		expect(t, types[edge] > 0, "no %s reference in %v", edge, types)
	}
	for _, want := range []engmodel.EntityReference{
		{From: "REQ-EMG-001", To: "FU-MODEL-LOADER", Via: "appliesTo", Type: engmodel.EntityEdgeField},
		{From: "SM-ARTIFACT-FRESHNESS", To: "STATE-ARTIFACTS-FRESH", Via: "contains", Type: engmodel.EntityEdgeContains},
	} {
		expect(t, hasReference(index, want), "reference %+v is not reported", want)
	}
}

// Two calls with the same bundle return equal indexes, sorted by ID.
// Mutation: an unsorted result or one that depends on map order fails this test.
// TRLC-LINKS: REQ-EMG-189, REQ-EMG-196
func TestBuildEntityIndexIsDeterministic(t *testing.T) {
	bundle := selfBundle(t)
	first, err := engmodel.BuildEntityIndex(bundle)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		again, err := engmodel.BuildEntityIndex(bundle)
		expect(t, err == nil, "call %d: %v", i, err)
		check{t}.equal("an index of the same bundle", again, first)
	}
	expect(t, slices.IsSortedFunc(first.Entities, func(a, b engmodel.Entity) int { return strings.Compare(a.ID, b.ID) }), "entities are not sorted by ID")
}

// A typed relationship is a reference only when both of its ends are entities of the index.
// Mutation: keeping a relationship with an unknown end, or dropping the known one, fails this test.
// TRLC-LINKS: REQ-EMG-189, REQ-EMG-196
func TestBuildEntityIndexKeepsARelationshipOnlyBetweenKnownEntities(t *testing.T) {
	bundle := linksModel(t, "child-a")
	bundle.Behavior.Behavior.Relationships = append(slices.Clone(bundle.Behavior.Behavior.Relationships),
		engmodel.Mapping{Type: "depends_on", From: "FU-ENCODER", To: "FU-GHOST"},
		engmodel.Mapping{Type: "depends_on", From: "FG-ENCODER", To: "FU-ENCODER"},
	)
	index, err := engmodel.BuildEntityIndex(bundle)
	if err != nil {
		t.Fatal(err)
	}
	known := engmodel.EntityReference{From: "FG-ENCODER", To: "FU-ENCODER", Via: "depends_on", Type: engmodel.EntityEdgeRelationship}
	expect(t, hasReference(index, known), "the relationship between known entities is missing")
	ghost := slices.ContainsFunc(index.References, func(r engmodel.EntityReference) bool { return r.To == "FU-GHOST" || r.From == "FU-GHOST" })
	expect(t, !ghost, "a relationship to an unknown entity is reported")
}

// requirementEntity is a requirement entity with a text.
func requirementEntity(id, text string) engmodel.Entity {
	return engmodel.Entity{ID: id, Kind: "requirement", Documents: []string{"requirements"}, Title: id, Fields: map[string]any{"text": text}}
}

// sampleIndexes are two indexes of one model: REQ-3 was removed, REQ-2 edited and REQ-4 and REQ-6 added.
func sampleIndexes() (from, to engmodel.EntityIndex) {
	from = engmodel.EntityIndex{
		Entities:   []engmodel.Entity{requirementEntity("REQ-1", "a"), requirementEntity("REQ-2", "b"), requirementEntity("REQ-3", "c")},
		References: []engmodel.EntityReference{{From: "REQ-1", To: "REQ-3", Via: "parent", Type: engmodel.EntityEdgeField}},
	}
	to = engmodel.EntityIndex{
		Entities:   []engmodel.Entity{requirementEntity("REQ-1", "a"), requirementEntity("REQ-2", "changed"), requirementEntity("REQ-6", "f"), requirementEntity("REQ-4", "d")},
		References: []engmodel.EntityReference{{From: "REQ-2", To: "REQ-4", Via: "parent", Type: engmodel.EntityEdgeField}},
	}
	return from, to
}

// DiffEntityIndexes names what was added, removed and changed by ID and the references that moved with it.
// Mutation: swapping the direction of the comparison or listing an untouched entity fails this test.
// TRLC-LINKS: REQ-EMG-190, REQ-EMG-197
func TestDiffEntityIndexesThroughTheFacade(t *testing.T) {
	from, to := sampleIndexes()
	diff := engmodel.DiffEntityIndexes(from, to)
	check{t}.equal("added, in order", ids(diff.Added), []string{"REQ-4", "REQ-6"})
	check{t}.equal("removed", ids(diff.Removed), []string{"REQ-3"})
	check{t}.equal("changed", diff.Changed, []engmodel.EntityChange{{ID: "REQ-2", Kind: "requirement", Title: "REQ-2", Document: "requirements", Fields: []string{"text"}}})
	check{t}.equal("references added", diff.ReferencesAdded, to.References)
	check{t}.equal("references removed (a link of a removed entity goes with it)", len(diff.ReferencesRemoved), 0)
}

// An index does not differ from itself.
// TRLC-LINKS: REQ-EMG-190, REQ-EMG-197
func TestDiffEntityIndexesOfAnIndexWithItselfIsEmpty(t *testing.T) {
	from, _ := sampleIndexes()
	diff := engmodel.DiffEntityIndexes(from, from)
	check{t}.equal("changes", len(diff.Added)+len(diff.Removed)+len(diff.Changed)+len(diff.ReferencesAdded)+len(diff.ReferencesRemoved), 0)
}

// Between two tags an added requirement is added, a removed one removed, one whose verification criteria changed is
// changed with that field named, and an untouched requirement is not listed.
// Mutation: comparing only one direction, reading the head twice or listing every entity fails this test.
// TRLC-LINKS: REQ-EMG-190, REQ-EMG-197
func TestDiffModelsBetweenTagsListsAddedRemovedAndChangedEntities(t *testing.T) {
	dir := childAReleased(t)
	diff, err := engmodel.DiffModels(engmodel.ModelDiffRequest{Repo: dir, From: "v1.0.0", To: "v1.1.0"})
	if err != nil {
		t.Fatal(err)
	}
	check{t}.equal("repo, from and to", [3]string{diff.Repo, diff.From, diff.To}, [3]string{dir, "v1.0.0", "v1.1.0"})
	check{t}.equal("added", ids(diff.Entities.Added), []string{"REQ-A-005"})
	check{t}.equal("removed", ids(diff.Entities.Removed), []string{"REQ-A-004"})
	check{t}.equal("changed", ids(diff.Entities.Changed), []string{"REQ-A-002"})
	check{t}.equal("the field that changed", diff.Entities.Changed[0].Fields, []string{"verificationCriteria"})
	check{t}.equal("trace of a diff without the trace flag", [2]any{diff.Trace, diff.TraceError}, [2]any{(*engmodel.TraceDelta)(nil), ""})
}

// Comparing in the other direction swaps what was added and what was removed.
// Mutation: comparing only one direction fails this test.
// TRLC-LINKS: REQ-EMG-190, REQ-EMG-197
func TestDiffModelsInTheOtherDirectionSwapsAddedAndRemoved(t *testing.T) {
	back, err := engmodel.DiffModels(engmodel.ModelDiffRequest{Repo: childAReleased(t), From: "v1.1.0", To: "v1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	check{t}.equal("added and removed", [2][]string{ids(back.Entities.Added), ids(back.Entities.Removed)}, [2][]string{{"REQ-A-004"}, {"REQ-A-005"}})
}

// With no second revision the working tree is compared, so an uncommitted edit shows.
// Mutation: reading the first revision instead of the working tree, or failing on an empty ref, fails this test.
// TRLC-LINKS: REQ-EMG-190, REQ-EMG-197
func TestDiffModelsWithoutASecondRevisionComparesTheWorkingTree(t *testing.T) {
	dir := childAReleased(t)
	inFile(filepath.Join(dir, "model", "requirements.yml")).replace(t, "verificationCriteria: One sentence is published per elapsed period.", "verificationCriteria: Exactly one sentence is published per elapsed period.")
	diff, err := engmodel.DiffModels(engmodel.ModelDiffRequest{Repo: dir, From: "v1.1.0"})
	if err != nil {
		t.Fatal(err)
	}
	check{t}.equal("to", diff.To, "")
	check{t}.equal("added and removed", len(diff.Entities.Added)+len(diff.Entities.Removed), 0)
	check{t}.equal("changed", ids(diff.Entities.Changed), []string{"REQ-A-001"})
	check{t}.equal("the field that changed", diff.Entities.Changed[0].Fields, []string{"verificationCriteria"})
}

// With the trace flag the change of traceability is returned: REQ-A-001 gained code, a test and stopped being an orphan
// and REQ-A-005 is a new orphan.
// Mutation: reading the trace of one revision twice, or dropping the delta, fails this test.
// TRLC-LINKS: REQ-EMG-191, REQ-EMG-197
func TestDiffModelsWithTheTraceFlagReportsHowTraceabilityMoved(t *testing.T) {
	diff, err := engmodel.DiffModels(engmodel.ModelDiffRequest{Repo: childAReleased(t), From: "v1.0.0", To: "v1.1.0", Trace: true})
	if err != nil {
		t.Fatal(err)
	}
	if diff.Trace == nil {
		t.Fatalf("no trace delta: %s", diff.TraceError)
	}
	check{t}.equal("gained code", diff.Trace.Gained, []string{"REQ-A-001"})
	check{t}.equal("gained a test", diff.Trace.TestsGained, []string{"REQ-A-001"})
	check{t}.equal("resolved orphans", diff.Trace.ResolvedOrphans, []string{"REQ-A-001"})
	check{t}.equal("new orphans", diff.Trace.NewOrphans, []string{"REQ-A-005"})
	check{t}.equal("lost", diff.Trace.Lost, []string{})
}

// A trace that cannot be built is reported by its reason and does not fail the entity diff. The second revision names
// one functional unit twice: the entity index reads it, the canonical projection that the trace needs refuses it.
// Mutation: failing the whole diff, or an empty reason, fails this test.
// TRLC-LINKS: REQ-EMG-191, REQ-EMG-197
func TestDiffModelsReportsWhyATraceCouldNotBeBuilt(t *testing.T) {
	dir := childARepo(t)
	inFile(filepath.Join(dir, "model", "architecture.yml")).replace(t, "  functionalUnits:\n", "  functionalUnits:\n  - id: FU-ENCODER\n    name: Duplicate Encoder\n    group: FG-ENCODER\n    prose: The same ID again.\n")
	commitAll(t, dir, "v1.1.0")
	for _, from := range []string{"v1.0.0", "v1.1.0"} {
		to := map[string]string{"v1.0.0": "v1.1.0", "v1.1.0": "v1.0.0"}[from]
		diff, err := engmodel.DiffModels(engmodel.ModelDiffRequest{Repo: dir, From: from, To: to, Trace: true})
		expect(t, err == nil, "the entity diff must not fail when the trace cannot be built: %v", err)
		expect(t, diff.Trace == nil && strings.HasPrefix(diff.TraceError, "trace at v1.1.0: "), "from %s: trace %+v, error %q, want no delta and the reason that names v1.1.0", from, diff.Trace, diff.TraceError)
	}
}

// An unknown revision and a revision that git would read as an option are errors and write nothing.
// Mutation: passing the revision to git unchecked, or ignoring the error, fails this test.
// TRLC-LINKS: REQ-EMG-190, REQ-EMG-197
func TestDiffModelsRefusesAnUnknownOrOptionLikeRevision(t *testing.T) {
	dir := childAReleased(t)
	pwned := filepath.Join(dir, "pwned")
	cases := map[string]engmodel.ModelDiffRequest{
		"unknown first":  {From: "v9.9.9", To: "v1.1.0"},
		"unknown second": {From: "v1.0.0", To: "v9.9.9"},
		"option first":   {From: "--output=" + pwned, To: "v1.1.0"},
		"option second":  {From: "v1.0.0", To: "--output=" + pwned},
	}
	for name, req := range cases {
		req.Repo = dir
		_, err := engmodel.DiffModels(req)
		expect(t, err != nil, "%s: the revision was accepted", name)
	}
	_, err := os.Stat(pwned)
	expect(t, err != nil, "an option-like revision made git write a file")
}

// requirementIDs lists the IDs of the requirements of a bundle.
func requirementIDs(bundle engmodel.Bundle) []string {
	out := []string{}
	for _, r := range bundle.Requirements.Requirements {
		out = append(out, r.ID)
	}
	return out
}

// LoadBundleAtRef returns the model as committed at a revision, not the working tree, and an error for an unknown revision.
// Mutation: loading the working tree, or accepting an unknown revision, fails this test.
// TRLC-LINKS: REQ-EMG-190, REQ-EMG-194
func TestLoadBundleAtRefReadsTheCommittedModel(t *testing.T) {
	dir := childAReleased(t)
	inFile(filepath.Join(dir, "model", "requirements.yml")).replace(t, "- id: REQ-A-005", "- id: REQ-A-006")
	cases := map[string][]string{
		"v1.0.0": {"REQ-A-001", "REQ-A-002", "REQ-A-003", "REQ-A-004"},
		"v1.1.0": {"REQ-A-001", "REQ-A-002", "REQ-A-003", "REQ-A-005"},
	}
	for ref, want := range cases {
		bundle, err := engmodel.LoadBundleAtRef(dir, ref)
		expect(t, err == nil, "%s: %v", ref, err)
		check{t}.equal("requirements at "+ref, requirementIDs(bundle), want)
		check{t}.equal("model ID and capabilities of the other documents at "+ref, [2]any{bundle.Manifest.Module.ModelID, len(bundle.Architecture.Contract.Provides)}, [2]any{"REQLINKS-CHILD-A", 2})
	}
	for _, ref := range []string{"v9.9.9", "--output=x"} {
		_, err := engmodel.LoadBundleAtRef(dir, ref)
		expect(t, err != nil, "the revision %q was accepted", ref)
	}
}

// LoadBundle returns the manifest and every document the manifest names, and refuses a path that is not an engmod.yml.
// Mutation: skipping a document or accepting another file name fails this test.
// TRLC-LINKS: REQ-EMG-194
func TestLoadBundleReturnsEveryDocumentAndRefusesAnotherFileName(t *testing.T) {
	isolatedModel(t)
	dir := filepath.Join(t.TempDir(), "child-a")
	copyModel(t, childA, dir)
	bundle, err := engmodel.LoadBundle(filepath.Join(dir, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	check{t}.equal("model ID and the number of requirements, catalog systems and functional units", [4]any{bundle.Manifest.Module.ModelID, len(bundle.Requirements.Requirements), len(bundle.Catalog.Catalog.Systems), len(bundle.Architecture.AuthoredArchitecture.FunctionalUnits)}, [4]any{"REQLINKS-CHILD-A", 4, 1, 1})
	renamed := filepath.Join(dir, "manifest.yml")
	if err := os.Rename(filepath.Join(dir, "engmod.yml"), renamed); err != nil {
		t.Fatal(err)
	}
	_, err = engmodel.LoadBundle(renamed)
	expect(t, err != nil, "a manifest that is not named engmod.yml was accepted")
}

// removeFile deletes a file of a fixture.
func removeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
}

// brokenModels are the ways a manifest can name a document that cannot be read.
var brokenModels = map[string]func(t *testing.T, dir string){
	"missing requirements": func(t *testing.T, dir string) { removeFile(t, filepath.Join(dir, "model", "requirements.yml")) },
	"missing architecture": func(t *testing.T, dir string) { removeFile(t, filepath.Join(dir, "model", "architecture.yml")) },
	"missing catalog":      func(t *testing.T, dir string) { removeFile(t, filepath.Join(dir, "model", "catalog.yml")) },
	"requirements break the schema": func(t *testing.T, dir string) {
		inFile(filepath.Join(dir, "model", "requirements.yml")).replace(t, "requirements:\n", "requirements: not-a-list\nignored:\n")
	},
	"unknown field": func(t *testing.T, dir string) {
		inFile(filepath.Join(dir, "model", "architecture.yml")).replace(t, "architecture:\n", "architecture:\n  nonsense: true\n")
	},
	"manifest breaks the schema": func(t *testing.T, dir string) {
		inFile(filepath.Join(dir, "engmod.yml")).replace(t, "kind: system", "kind: [not, a, kind]")
	},
}

// A manifest that names a document that does not exist, or one that breaks the schema, is an error and the bundle has
// no requirements.
// Mutation: returning the partly loaded bundle with the error, or no error, fails this test.
// TRLC-LINKS: REQ-EMG-206
func TestLoadBundleFailureReturnsAnErrorAndAnEmptyBundle(t *testing.T) {
	isolatedModel(t)
	for name, break_ := range brokenModels {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "child-a")
			copyModel(t, childA, dir)
			break_(t, dir)
			bundle, err := engmodel.LoadBundle(filepath.Join(dir, "engmod.yml"))
			expect(t, err != nil, "LoadBundle returned no error")
			check{t}.equal("requirements of the failed load", len(bundle.Requirements.Requirements), 0)
		})
	}
}
