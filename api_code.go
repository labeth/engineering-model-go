// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package engmodel

import (
	"path"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codemap"
	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/reqlint"
)

// CatalogTermUsage is where one catalog term is used: the requirements whose text names it and the other
// entities that hold its ID.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
type CatalogTermUsage = reqlint.TermUsage

// CatalogUsage lists every catalog term with the requirements that name it (by name or alias, on word boundaries)
// and the entities that reference it, for tools that show a term in context.
// TRLC-LINKS: REQ-EMG-193
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func CatalogUsage(bundle Bundle) ([]CatalogTermUsage, error) { return reqlint.CatalogUsage(bundle) }

// CodeDeclaration is one named declaration of a source file (function, method, type, var, const, class, ...),
// whether or not a trace marker is attached to it.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
type CodeDeclaration = codemap.Declaration

// ListCodeDeclarations lists every named declaration of the model's code scope under root, in path and line order,
// for tools that jump from a use to its definition. It reads files only. The code scope is the one the check and
// the trace use: git-ignored files, inferenceHints.exclude paths and nested modules are left out.
// TRLC-LINKS: REQ-EMG-192
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func ListCodeDeclarations(bundle Bundle, root string) ([]CodeDeclaration, error) {
	return codescan.ListDeclarations(bundle, root)
}

// CodeDeclarationsOf lists the declarations of one source file given its root-relative path and content.
// TRLC-LINKS: REQ-EMG-192
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func CodeDeclarationsOf(relPath string, src []byte) ([]CodeDeclaration, error) {
	return codemap.DeclarationsOf(relPath, src)
}

// FindCodeDefinitions returns the declarations a use of name in the file fromPath may refer to, best first: the
// same file, then the same directory (the package), then the same language elsewhere. Declarations of test-only
// code rank after production code unless the use is itself in test-only code. It matches the name exactly and
// does not resolve scopes, imports or overloads, so a result is a candidate, not a proof.
// TRLC-LINKS: REQ-EMG-192
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func FindCodeDefinitions(declarations []CodeDeclaration, fromPath, name string) []CodeDeclaration {
	use := codeUse{path: fromPath, dir: path.Dir(fromPath), inTest: inTestOnlyCode(declarations, fromPath)}
	use.language, _ = codemap.LanguageForPath(fromPath)
	found := use.candidates(declarations, name)
	sort.SliceStable(found, func(i, j int) bool { return found[i].before(found[j]) })
	out := make([]CodeDeclaration, len(found))
	for i, r := range found {
		out[i] = r.decl
	}
	return out
}

// inTestOnlyCode reports whether the file declares test-only code, which is what a use in it is.
// TRLC-LINKS: REQ-EMG-192
func inTestOnlyCode(declarations []CodeDeclaration, file string) bool {
	for _, d := range declarations {
		if d.Path == file && d.TestOnly {
			return true
		}
	}
	return false
}

// candidates are the declarations of the name in the language of the use, each with its rank.
// TRLC-LINKS: REQ-EMG-192
func (u codeUse) candidates(declarations []CodeDeclaration, name string) []rankedDeclaration {
	var found []rankedDeclaration
	for _, d := range declarations {
		if d.Name == name && (u.language == "" || d.Language == u.language) {
			found = append(found, rankedDeclaration{decl: d, rank: u.rank(d)})
		}
	}
	return found
}

// codeUse is where a name is used.
type codeUse struct {
	path, dir, language string
	inTest              bool
}

// rank is 0 for the same file, 1 for the same directory, 2 for a directory above or below, 3 elsewhere, and 4 more
// for test-only code when the use is not in test-only code.
// TRLC-LINKS: REQ-EMG-192
func (u codeUse) rank(d CodeDeclaration) int {
	rank := 3
	declDir := path.Dir(d.Path)
	switch {
	case d.Path == u.path:
		rank = 0
	case declDir == u.dir:
		rank = 1
	case strings.HasPrefix(u.dir, declDir+"/") || strings.HasPrefix(declDir, u.dir+"/"):
		rank = 2
	}
	if d.TestOnly && !u.inTest {
		rank += 4
	}
	return rank
}

type rankedDeclaration struct {
	decl CodeDeclaration
	rank int
}

// before orders by rank, then path, then line.
// TRLC-LINKS: REQ-EMG-192
func (r rankedDeclaration) before(o rankedDeclaration) bool {
	if r.rank != o.rank {
		return r.rank < o.rank
	}
	if r.decl.Path != o.decl.Path {
		return r.decl.Path < o.decl.Path
	}
	return r.decl.Line < o.decl.Line
}
