// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/yamledit"
)

// defaultReqLinksPath is the requirement link file engws link add creates in
// a model that has none.
const defaultReqLinksPath = "model/req-links.yml"

// linkFiles collects the file edits of one command in writes: the manifest
// is the engmod.yml of the model, read returns an edit already collected
// before the file on disk.
type linkFiles struct {
	manifest string
	read     func(string) ([]byte, error)
	writes   map[string][]byte
}

// newLinkFiles starts an empty edit of the model whose manifest is manifest.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func newLinkFiles(manifest string) *linkFiles {
	f := &linkFiles{manifest: manifest, writes: map[string][]byte{}}
	f.read = func(path string) ([]byte, error) {
		if data, ok := f.writes[path]; ok {
			return data, nil
		}
		return os.ReadFile(path)
	}
	return f
}

// appendLinks adds links to the requirement link file of b in the collected
// edits: appended to the existing file, line-preserving, or a new
// model/req-links.yml with documents.reqLinks added to engmod.yml.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func (f *linkFiles) appendLinks(b model.Bundle, links []model.ReqLink) error {
	rel, err := f.linksDocument(b)
	if err != nil {
		return err
	}
	blocks := make([]string, 0, len(links))
	for _, l := range links {
		blocks = append(blocks, linkBlock(l))
	}
	return f.appendBlocks(filepath.Join(filepath.Dir(f.manifest), filepath.FromSlash(rel)), blocks)
}

// linksDocument returns the link file path of b relative to the manifest,
// adding documents.reqLinks to the manifest edit when b names none.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func (f *linkFiles) linksDocument(b model.Bundle) (string, error) {
	if rel := strings.TrimSpace(b.Manifest.Documents.ReqLinks); rel != "" {
		return rel, nil
	}
	manifest, err := f.read(f.manifest)
	if err != nil {
		return "", err
	}
	manifest, err = yamledit.SetKey(manifest, yamledit.FindPath("documents"), "reqLinks", yamledit.Scalar(defaultReqLinksPath))
	if err != nil {
		return "", fmt.Errorf("%s: %w", f.manifest, err)
	}
	f.writes[f.manifest] = manifest
	return defaultReqLinksPath, nil
}

// appendBlocks appends the link blocks to the file at path, or creates the
// file with them when it does not exist.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func (f *linkFiles) appendBlocks(path string, blocks []string) error {
	existing, err := f.read(path)
	if os.IsNotExist(err) {
		f.writes[path] = []byte(model.SchemaVersionLine + "\nreqLinks:\n" + strings.Join(blocks, ""))
		return nil
	}
	if err != nil {
		return err
	}
	if existing, err = yamledit.AppendSequenceItems(existing, yamledit.FindPath(), "reqLinks", blocks); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	f.writes[path] = existing
	return nil
}

// fileSnapshot is a file as it was before a command wrote it.
type fileSnapshot struct {
	data    []byte
	existed bool
}

// snapshotFiles reads each of paths; a file that does not exist is recorded
// as such.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func snapshotFiles(paths []string) (map[string]fileSnapshot, error) {
	snapshots := map[string]fileSnapshot{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		switch {
		case err == nil:
			snapshots[path] = fileSnapshot{data: data, existed: true}
		case os.IsNotExist(err):
			snapshots[path] = fileSnapshot{}
		default:
			return nil, err
		}
	}
	return snapshots, nil
}

// restoreFile puts a file back as snap recorded it: its old content, or
// removed when it did not exist.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func restoreFile(path string, snap fileSnapshot) error {
	if snap.existed {
		return writeFileAtomic(path, snap.data)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// restoreFiles restores each of done and returns a description of every file
// it could not restore.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func restoreFiles(done []string, snapshots map[string]fileSnapshot) []string {
	var failed []string
	for _, path := range done {
		if err := restoreFile(path, snapshots[path]); err != nil {
			failed = append(failed, fmt.Sprintf("%s (%v)", path, err))
		}
	}
	return failed
}

// writeAllOrNothing writes every file of paths with its content in writes.
// It snapshots each target first; when a write fails, every file already
// written is restored to its snapshot and a file that did not exist is
// removed, so the model is never left half written.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func writeAllOrNothing(paths []string, writes map[string][]byte) error {
	snapshots, err := snapshotFiles(paths)
	if err != nil {
		return err
	}
	for i, path := range paths {
		err := writeFile(path, writes[path])
		if err == nil {
			continue
		}
		if failed := restoreFiles(paths[:i], snapshots); len(failed) > 0 {
			return fmt.Errorf("write %s: %w; restoring also failed for %s", path, err, strings.Join(failed, ", "))
		}
		return fmt.Errorf("write %s: %w; every file restored, nothing written", path, err)
	}
	return nil
}

// cleanLinkPart upper-cases s and replaces each run-breaking character with a
// dash, trimmed of leading and trailing dashes.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func cleanLinkPart(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if isUpperASCII(r) || isDigitASCII(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// isUpperASCII reports whether r is A to Z.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func isUpperASCII(r rune) bool { return r >= 'A' && r <= 'Z' }

// isDigitASCII reports whether r is 0 to 9.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func isDigitASCII(r rune) bool { return r >= '0' && r <= '9' }

// linkID is LNK-<requirement without REQ->-<ALIAS>, upper case.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func linkID(requirement, alias string) string {
	req := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(requirement)), "REQ-")
	return "LNK-" + cleanLinkPart(req) + "-" + cleanLinkPart(alias)
}

// uniqueLinkID returns id, or id-2, id-3 ... when taken, and marks it used.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func uniqueLinkID(id string, used map[string]bool) string {
	candidate := id
	for n := 2; used[candidate]; n++ {
		candidate = fmt.Sprintf("%s-%d", id, n)
	}
	used[candidate] = true
	return candidate
}

// linkBlock renders a link as a block sequence item at column 0.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func linkBlock(l model.ReqLink) string {
	var b strings.Builder
	fmt.Fprintf(&b, "- id: %s\n", yamledit.Scalar(l.ID))
	if l.Parent != "" {
		fmt.Fprintf(&b, "  parent: %s\n", yamledit.Scalar(l.Parent))
	}
	fmt.Fprintf(&b, "  child: %s\n", yamledit.Scalar(l.Child))
	if len(l.Evidence) > 0 {
		b.WriteString("  evidence:\n")
		for _, e := range l.Evidence {
			fmt.Fprintf(&b, "  - %s\n", yamledit.Scalar(e))
		}
	}
	b.WriteString("  pinned:\n")
	fmt.Fprintf(&b, "    version: %s\n", yamledit.Scalar(l.Pinned.Version))
	fmt.Fprintf(&b, "  status: %s\n", l.Status)
	if l.Rationale != "" {
		fmt.Fprintf(&b, "  rationale: %s\n", yamledit.Scalar(l.Rationale))
	}
	return b.String()
}
