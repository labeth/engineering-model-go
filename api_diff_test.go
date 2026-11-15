// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package engmodel_test

import (
	"os"
	"path/filepath"
	"reflect"
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

// BuildEntityIndex lists the authored entities of this repository with kind, document, parent and fields.
// Mutation: a wrong kind table, a dropped parent or a dropped field of an entity fails this test.
// TRLC-LINKS: REQ-EMG-189, REQ-EMG-196
func TestBuildEntityIndexListsEntitiesWithKindDocumentParentAndFields(t *testing.T) {
	index := selfIndex(t)
	if len(index.Entities) < 400 {
		t.Errorf("%d entities, want at least 400", len(index.Entities))
	}
	cases := []struct {
		id, kind, parent string
		document         string
		field            string
	}{
		{"REQ-EMG-001", "requirement", "", "requirements", "text"},
		{"FU-MODEL-LOADER", "functional_unit", "", "architecture", "prose"},
		{"TS-ARTIFACT-PARSE-ABUSE", "threat_scenario", "", "assurance", ""},
		{"VER-ARTIFACT-FRESHNESS-GATE", "control_verification", "", "assurance", ""},
		{"POAM-CONSISTENCY-BATCH1", "poam_item", "", "assurance", "dueDate"},
		{"COND-CATALOG-REFERENCES-RESOLVE", "catalog_term", "", "catalog", ""},
		{"SM-ARTIFACT-FRESHNESS", "machine", "", "behavior", ""},
		{"STATE-ARTIFACTS-FRESH", "state", "SM-ARTIFACT-FRESHNESS", "behavior", ""},
		{"INV-ARTIFACTS-FRESH-VALIDATED", "invariant", "SM-ARTIFACT-FRESHNESS", "behavior", ""},
		{"build-entity-index", "flow_step", "FLOW-GO-READ-MODEL", "behavior", ""},
		{"ADR-EMG-001", "decision", "", "decisions", ""},
	}
	for _, c := range cases {
		t.Run(c.id, func(t *testing.T) {
			entity := findEntity(t, index, c.id)
			if entity.Kind != c.kind || entity.Parent != c.parent || entity.Title == "" {
				t.Errorf("kind %q parent %q title %q, want kind %q parent %q and a title", entity.Kind, entity.Parent, entity.Title, c.kind, c.parent)
			}
			if !slices.Contains(entity.Documents, c.document) {
				t.Errorf("documents %v lack %s", entity.Documents, c.document)
			}
			if _, ok := entity.Fields[c.field]; c.field != "" && !ok {
				t.Errorf("fields %v lack %q", entity.Fields, c.field)
			}
		})
	}
	if unit := findEntity(t, index, "FU-MODEL-LOADER"); unit.Documents[0] != "architecture" || unit.Documents[len(unit.Documents)-1] != "catalog" {
		t.Errorf("a unit the architecture defines and the catalog names lists documents in order: %v", unit.Documents)
	}
	if machine := findEntity(t, index, "SM-ARTIFACT-FRESHNESS"); machine.Fields["states"] != nil {
		t.Errorf("a nested entity is an entity of its own, not a field of its parent: %v", machine.Fields)
	}
}

// Every reference has two known ends, the four edge types are all reported and REQ-EMG-001 applies to FU-MODEL-LOADER.
// Mutation: dropping an edge type, a reference to an unknown ID or the appliesTo edge fails this test.
// TRLC-LINKS: REQ-EMG-189, REQ-EMG-196
func TestBuildEntityIndexReportsReferencesBetweenKnownEntities(t *testing.T) {
	index := selfIndex(t)
	types := edgeTypes(t, index)
	for _, edge := range []string{engmodel.EntityEdgeField, engmodel.EntityEdgeMention, engmodel.EntityEdgeContains, engmodel.EntityEdgeRelationship} {
		if types[edge] == 0 {
			t.Errorf("no %s reference in %v", edge, types)
		}
	}
	for _, want := range []engmodel.EntityReference{
		{From: "REQ-EMG-001", To: "FU-MODEL-LOADER", Via: "appliesTo", Type: engmodel.EntityEdgeField},
		{From: "SM-ARTIFACT-FRESHNESS", To: "STATE-ARTIFACTS-FRESH", Via: "contains", Type: engmodel.EntityEdgeContains},
	} {
		if !hasReference(index, want) {
			t.Errorf("reference %+v is not reported", want)
		}
	}
}

// edgeTypes counts the references of each type and fails the test for a reference with an unknown end.
func edgeTypes(t *testing.T, index engmodel.EntityIndex) map[string]int {
	t.Helper()
	types := map[string]int{}
	for _, r := range index.References {
		types[r.Type]++
		_, from := index.Find(r.From)
		_, to := index.Find(r.To)
		if !from || !to {
			t.Errorf("reference %+v has an unknown end", r)
		}
	}
	return types
}

// hasReference reports whether the index holds the reference, whatever its description.
func hasReference(index engmodel.EntityIndex, want engmodel.EntityReference) bool {
	for _, r := range index.References {
		r.Description = ""
		if r == want {
			return true
		}
	}
	return false
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
		if err != nil || !reflect.DeepEqual(first, again) {
			t.Fatalf("call %d returned another index (error %v)", i, err)
		}
	}
	if !slices.IsSortedFunc(first.Entities, func(a, b engmodel.Entity) int { return strings.Compare(a.ID, b.ID) }) {
		t.Error("entities are not sorted by ID")
	}
}

// DiffEntityIndexes names what was added, removed and changed by ID and the references that moved with it.
// Mutation: swapping the direction of the comparison or listing an untouched entity fails this test.
// TRLC-LINKS: REQ-EMG-190, REQ-EMG-197
func TestDiffEntityIndexesThroughTheFacade(t *testing.T) {
	entity := func(id, text string) engmodel.Entity {
		return engmodel.Entity{ID: id, Kind: "requirement", Documents: []string{"requirements"}, Title: id, Fields: map[string]any{"text": text}}
	}
	from := engmodel.EntityIndex{
		Entities:   []engmodel.Entity{entity("REQ-1", "a"), entity("REQ-2", "b"), entity("REQ-3", "c")},
		References: []engmodel.EntityReference{{From: "REQ-1", To: "REQ-3", Via: "parent", Type: engmodel.EntityEdgeField}},
	}
	to := engmodel.EntityIndex{
		Entities:   []engmodel.Entity{entity("REQ-1", "a"), entity("REQ-2", "changed"), entity("REQ-6", "f"), entity("REQ-4", "d")},
		References: []engmodel.EntityReference{{From: "REQ-2", To: "REQ-4", Via: "parent", Type: engmodel.EntityEdgeField}},
	}
	diff := engmodel.DiffEntityIndexes(from, to)
	if got := ids(diff.Added); !reflect.DeepEqual(got, []string{"REQ-4", "REQ-6"}) {
		t.Errorf("added %v, want REQ-4 and REQ-6 in order", got)
	}
	if len(diff.ReferencesRemoved) != 0 {
		t.Errorf("a link of a removed entity goes with it: %+v", diff.ReferencesRemoved)
	}
	if got := ids(diff.Removed); !reflect.DeepEqual(got, []string{"REQ-3"}) {
		t.Errorf("removed %v, want REQ-3", got)
	}
	if len(diff.Changed) != 1 || diff.Changed[0].ID != "REQ-2" || !reflect.DeepEqual(diff.Changed[0].Fields, []string{"text"}) {
		t.Errorf("changed %+v, want REQ-2 in text only", diff.Changed)
	}
	if len(diff.ReferencesAdded) != 1 || diff.ReferencesAdded[0].To != "REQ-4" {
		t.Errorf("references added %+v", diff.ReferencesAdded)
	}
	if empty := engmodel.DiffEntityIndexes(from, from); len(empty.Added)+len(empty.Removed)+len(empty.Changed)+len(empty.ReferencesAdded)+len(empty.ReferencesRemoved) != 0 {
		t.Errorf("an index differs from itself: %+v", empty)
	}
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
	if diff.From != "v1.0.0" || diff.To != "v1.1.0" || diff.Repo != dir {
		t.Errorf("request echoed as repo %q from %q to %q", diff.Repo, diff.From, diff.To)
	}
	if got := ids(diff.Entities.Added); !reflect.DeepEqual(got, []string{"REQ-A-005"}) {
		t.Errorf("added %v, want REQ-A-005", got)
	}
	if got := ids(diff.Entities.Removed); !reflect.DeepEqual(got, []string{"REQ-A-004"}) {
		t.Errorf("removed %v, want REQ-A-004", got)
	}
	changed := diff.Entities.Changed
	if len(changed) != 1 || changed[0].ID != "REQ-A-002" || !reflect.DeepEqual(changed[0].Fields, []string{"verificationCriteria"}) {
		t.Errorf("changed %+v, want REQ-A-002 in verificationCriteria only", changed)
	}
	if diff.Trace != nil || diff.TraceError != "" {
		t.Errorf("a diff without the trace flag carries no trace: %+v %q", diff.Trace, diff.TraceError)
	}
}

// Comparing in the other direction swaps what was added and what was removed.
// Mutation: comparing only one direction fails this test.
// TRLC-LINKS: REQ-EMG-190, REQ-EMG-197
func TestDiffModelsInTheOtherDirectionSwapsAddedAndRemoved(t *testing.T) {
	back, err := engmodel.DiffModels(engmodel.ModelDiffRequest{Repo: childAReleased(t), From: "v1.1.0", To: "v1.0.0"})
	if err != nil || !reflect.DeepEqual(ids(back.Entities.Added), []string{"REQ-A-004"}) || !reflect.DeepEqual(ids(back.Entities.Removed), []string{"REQ-A-005"}) {
		t.Errorf("the reverse comparison swaps added and removed: %+v (%v)", back.Entities, err)
	}
}

// With no second revision the working tree is compared, so an uncommitted edit shows.
// Mutation: reading the first revision instead of the working tree, or failing on an empty ref, fails this test.
// TRLC-LINKS: REQ-EMG-190, REQ-EMG-197
func TestDiffModelsWithoutASecondRevisionComparesTheWorkingTree(t *testing.T) {
	dir := childAReleased(t)
	reqs := filepath.Join(dir, "model", "requirements.yml")
	replaceInFile(t, reqs, "verificationCriteria: One sentence is published per elapsed period.", "verificationCriteria: Exactly one sentence is published per elapsed period.")
	diff, err := engmodel.DiffModels(engmodel.ModelDiffRequest{Repo: dir, From: "v1.1.0"})
	if err != nil {
		t.Fatal(err)
	}
	if diff.To != "" || len(diff.Entities.Added)+len(diff.Entities.Removed) != 0 {
		t.Errorf("only an edit is uncommitted: to %q, %+v", diff.To, diff.Entities)
	}
	if len(diff.Entities.Changed) != 1 || diff.Entities.Changed[0].ID != "REQ-A-001" || !reflect.DeepEqual(diff.Entities.Changed[0].Fields, []string{"verificationCriteria"}) {
		t.Errorf("changed %+v, want REQ-A-001 in verificationCriteria", diff.Entities.Changed)
	}
}

// With the trace flag the change of traceability is returned: REQ-A-001 gained code, a test and stopped being an orphan
// and REQ-A-005 is a new orphan.
// Mutation: reading the trace of one revision twice, or dropping the delta, fails this test.
// TRLC-LINKS: REQ-EMG-191, REQ-EMG-197
func TestDiffModelsWithTheTraceFlagReportsHowTraceabilityMoved(t *testing.T) {
	dir := childAReleased(t)
	diff, err := engmodel.DiffModels(engmodel.ModelDiffRequest{Repo: dir, From: "v1.0.0", To: "v1.1.0", Trace: true})
	if err != nil {
		t.Fatal(err)
	}
	if diff.Trace == nil {
		t.Fatalf("no trace delta: %s", diff.TraceError)
	}
	for name, list := range map[string][]string{"gained code": diff.Trace.Gained, "gained a test": diff.Trace.TestsGained, "resolved orphan": diff.Trace.ResolvedOrphans} {
		if !slices.Contains(list, "REQ-A-001") {
			t.Errorf("REQ-A-001 is not among the requirements that %s: %v", name, list)
		}
	}
	if !slices.Contains(diff.Trace.NewOrphans, "REQ-A-005") || slices.Contains(diff.Trace.NewOrphans, "REQ-A-001") {
		t.Errorf("new orphans %v, want REQ-A-005 and not REQ-A-001", diff.Trace.NewOrphans)
	}
	if len(diff.Trace.Lost) != 0 {
		t.Errorf("no requirement lost its trace: %v", diff.Trace.Lost)
	}
}

// A trace that cannot be built is reported by its reason and does not fail the entity diff. The second revision names
// one functional unit twice: the entity index reads it, the canonical projection that the trace needs refuses it.
// Mutation: failing the whole diff, or an empty reason, fails this test.
// TRLC-LINKS: REQ-EMG-191, REQ-EMG-197
func TestDiffModelsReportsWhyATraceCouldNotBeBuilt(t *testing.T) {
	dir := childARepo(t)
	replaceInFile(t, filepath.Join(dir, "model", "architecture.yml"), "  functionalUnits:\n", "  functionalUnits:\n  - id: FU-ENCODER\n    name: Duplicate Encoder\n    group: FG-ENCODER\n    prose: The same ID again.\n")
	commitAll(t, dir, "v1.1.0")
	diff, err := engmodel.DiffModels(engmodel.ModelDiffRequest{Repo: dir, From: "v1.0.0", To: "v1.1.0", Trace: true})
	if err != nil {
		t.Fatalf("the entity diff must not fail when the trace cannot be built: %v", err)
	}
	if diff.Trace != nil || !strings.HasPrefix(diff.TraceError, "trace at v1.1.0: ") {
		t.Errorf("trace %+v error %q, want no delta and the reason that names the revision", diff.Trace, diff.TraceError)
	}
	back, err := engmodel.DiffModels(engmodel.ModelDiffRequest{Repo: dir, From: "v1.1.0", To: "v1.0.0", Trace: true})
	if err != nil || back.Trace != nil || !strings.HasPrefix(back.TraceError, "trace at v1.1.0: ") {
		t.Errorf("the first revision cannot be traced either: trace %+v error %q (%v)", back.Trace, back.TraceError, err)
	}
}

// An unknown revision and a revision that git would read as an option are errors and write nothing.
// Mutation: passing the revision to git unchecked, or ignoring the error, fails this test.
// TRLC-LINKS: REQ-EMG-190, REQ-EMG-197
func TestDiffModelsRefusesAnUnknownOrOptionLikeRevision(t *testing.T) {
	dir := childAReleased(t)
	pwned := filepath.Join(dir, "pwned")
	for _, c := range []struct{ name, from, to string }{
		{"unknown first", "v9.9.9", "v1.1.0"},
		{"unknown second", "v1.0.0", "v9.9.9"},
		{"option first", "--output=" + pwned, "v1.1.0"},
		{"option second", "v1.0.0", "--output=" + pwned},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, err := engmodel.DiffModels(engmodel.ModelDiffRequest{Repo: dir, From: c.from, To: c.to}); err == nil {
				t.Error("the revision was accepted")
			}
		})
	}
	if _, err := os.Stat(pwned); err == nil {
		t.Fatal("an option-like revision made git write a file")
	}
}

// LoadBundleAtRef returns the model as committed at a revision, not the working tree, and an error for an unknown revision.
// Mutation: loading the working tree, or accepting an unknown revision, fails this test.
// TRLC-LINKS: REQ-EMG-190, REQ-EMG-194
func TestLoadBundleAtRefReadsTheCommittedModel(t *testing.T) {
	dir := childAReleased(t)
	replaceInFile(t, filepath.Join(dir, "model", "requirements.yml"), "- id: REQ-A-005", "- id: REQ-A-006")
	cases := []struct {
		ref  string
		want []string
	}{
		{"v1.0.0", []string{"REQ-A-001", "REQ-A-002", "REQ-A-003", "REQ-A-004"}},
		{"v1.1.0", []string{"REQ-A-001", "REQ-A-002", "REQ-A-003", "REQ-A-005"}},
	}
	for _, c := range cases {
		t.Run(c.ref, func(t *testing.T) {
			bundle, err := engmodel.LoadBundleAtRef(dir, c.ref)
			if err != nil {
				t.Fatal(err)
			}
			got := []string{}
			for _, r := range bundle.Requirements.Requirements {
				got = append(got, r.ID)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("requirements at %s: %v, want %v", c.ref, got, c.want)
			}
			if bundle.Manifest.Module.ModelID != "REQLINKS-CHILD-A" || len(bundle.Architecture.Contract.Provides) != 2 {
				t.Errorf("the other documents are loaded too: model %q, %d capabilities", bundle.Manifest.Module.ModelID, len(bundle.Architecture.Contract.Provides))
			}
		})
	}
	for _, ref := range []string{"v9.9.9", "--output=x"} {
		if _, err := engmodel.LoadBundleAtRef(dir, ref); err == nil {
			t.Errorf("the revision %q was accepted", ref)
		}
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
	if len(bundle.Requirements.Requirements) != 4 || len(bundle.Catalog.Catalog.Systems) == 0 || len(bundle.Architecture.AuthoredArchitecture.FunctionalUnits) != 1 || bundle.Manifest.Module.ModelID != "REQLINKS-CHILD-A" {
		t.Errorf("manifest %q: %d requirements, %d catalog systems, %d functional units", bundle.Manifest.Module.ModelID, len(bundle.Requirements.Requirements), len(bundle.Catalog.Catalog.Systems), len(bundle.Architecture.AuthoredArchitecture.FunctionalUnits))
	}
	renamed := filepath.Join(dir, "manifest.yml")
	if err := os.Rename(filepath.Join(dir, "engmod.yml"), renamed); err != nil {
		t.Fatal(err)
	}
	if _, err := engmodel.LoadBundle(renamed); err == nil {
		t.Error("a manifest that is not named engmod.yml was accepted")
	}
}

// A manifest that names a document that does not exist, or one that breaks the schema, is an error and the bundle has
// no requirements.
// Mutation: returning the partly loaded bundle with the error, or no error, fails this test.
// TRLC-LINKS: REQ-EMG-206
func TestLoadBundleFailureReturnsAnErrorAndAnEmptyBundle(t *testing.T) {
	isolatedModel(t)
	cases := []struct {
		name string
		edit func(t *testing.T, dir string)
	}{
		{"missing requirements", func(t *testing.T, dir string) { removeFile(t, filepath.Join(dir, "model", "requirements.yml")) }},
		{"missing architecture", func(t *testing.T, dir string) { removeFile(t, filepath.Join(dir, "model", "architecture.yml")) }},
		{"missing catalog", func(t *testing.T, dir string) { removeFile(t, filepath.Join(dir, "model", "catalog.yml")) }},
		{"requirements break the schema", func(t *testing.T, dir string) {
			replaceInFile(t, filepath.Join(dir, "model", "requirements.yml"), "requirements:\n", "requirements: not-a-list\nignored:\n")
		}},
		{"unknown field", func(t *testing.T, dir string) {
			replaceInFile(t, filepath.Join(dir, "model", "architecture.yml"), "architecture:\n", "architecture:\n  nonsense: true\n")
		}},
		{"manifest breaks the schema", func(t *testing.T, dir string) {
			replaceInFile(t, filepath.Join(dir, "engmod.yml"), "kind: system", "kind: [not, a, kind]")
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "child-a")
			copyModel(t, childA, dir)
			c.edit(t, dir)
			bundle, err := engmodel.LoadBundle(filepath.Join(dir, "engmod.yml"))
			if err == nil {
				t.Fatal("LoadBundle returned no error")
			}
			if len(bundle.Requirements.Requirements) != 0 {
				t.Errorf("the failed load returned %d requirements", len(bundle.Requirements.Requirements))
			}
		})
	}
}

// removeFile deletes a file of a fixture.
func removeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
}

// A typed relationship is a reference only when both of its ends are entities of the index.
// Mutation: keeping a relationship with an unknown end, or dropping the known one, fails this test.
// TRLC-LINKS: REQ-EMG-189, REQ-EMG-196
func TestBuildEntityIndexKeepsARelationshipOnlyBetweenKnownEntities(t *testing.T) {
	bundle := linksModel(t, "child-a")
	bundle.Behavior.Behavior.Relationships = append([]engmodel.Mapping(nil), bundle.Behavior.Behavior.Relationships...)
	bundle.Behavior.Behavior.Relationships = append(bundle.Behavior.Behavior.Relationships,
		engmodel.Mapping{Type: "depends_on", From: "FU-ENCODER", To: "FU-GHOST"},
		engmodel.Mapping{Type: "depends_on", From: "FG-ENCODER", To: "FU-ENCODER"},
	)
	index, err := engmodel.BuildEntityIndex(bundle)
	if err != nil {
		t.Fatal(err)
	}
	known, ghost := 0, 0
	for _, r := range index.References {
		if r.Type == engmodel.EntityEdgeRelationship && r.Via == "depends_on" && r.To == "FU-ENCODER" {
			known++
		}
		if r.To == "FU-GHOST" || r.From == "FU-GHOST" {
			ghost++
		}
	}
	if known != 1 || ghost != 0 {
		t.Errorf("%d relationships between known entities (want 1), %d to an unknown one (want 0)", known, ghost)
	}
}
