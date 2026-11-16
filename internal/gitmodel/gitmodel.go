// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package gitmodel

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
	"gopkg.in/yaml.v3"
)

// Output runs git in dir and returns its stdout.
// TRLC-LINKS: REQ-EMG-102, REQ-EMG-103, REQ-EMG-190
func Output(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// CheckRef refuses a ref that git would read as an option and one that names no commit.
// TRLC-LINKS: REQ-EMG-102, REQ-EMG-190
func CheckRef(repo, ref string) error {
	if strings.HasPrefix(ref, "-") || strings.TrimSpace(ref) == "" {
		return fmt.Errorf("invalid git ref %q", ref)
	}
	if _, err := Output(repo, "rev-parse", "--verify", "--quiet", ref+"^{commit}"); err != nil {
		return fmt.Errorf("unknown git ref %q in %s", ref, repo)
	}
	return nil
}

// WriteModelAtRef writes engmod.yml and every document the manifest names, as committed at ref, into dest
// (directories 0700, files 0600), each stamped with the current schemaVersion and with the retired state
// vocabulary stripped. It is the single place where a model of an older schema version is accepted
// (ADR-EMG-077): the model at a ref is only ever read, never validated as a user's model.
// TRLC-LINKS: REQ-EMG-102, REQ-EMG-103, REQ-EMG-186, REQ-EMG-190
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-MODEL-LOADER
func WriteModelAtRef(repo, ref, dest string) error {
	manifest, err := Output(repo, "show", ref+":./engmod.yml")
	if err != nil {
		return fmt.Errorf("%s has no engmod.yml at %s: %w", repo, ref, err)
	}
	var meta struct {
		Documents map[string]string `yaml:"documents"`
	}
	if err := yaml.Unmarshal(manifest, &meta); err != nil {
		return fmt.Errorf("engmod.yml at %s: %w", ref, err)
	}
	if err := writePrivate(filepath.Join(dest, "engmod.yml"), StampCurrentVersion(manifest)); err != nil {
		return err
	}
	rev := revision{repo: repo, ref: ref}
	for _, key := range sortedKeys(meta.Documents) {
		if err := rev.writeDocument(dest, manifestDocument{key: key, rel: meta.Documents[key]}); err != nil {
			return err
		}
	}
	return nil
}

// revision is a repository at a ref.
type revision struct{ repo, ref string }

// manifestDocument is one entry of the manifest's documents map.
type manifestDocument struct{ key, rel string }

// writeDocument copies the document the manifest names, refusing a path that leaves the model directory.
// TRLC-LINKS: REQ-EMG-102, REQ-EMG-103, REQ-EMG-186, REQ-EMG-190
func (v revision) writeDocument(dest string, doc manifestDocument) error {
	clean := path.Clean(filepath.ToSlash(strings.TrimSpace(doc.rel)))
	if !insideTree(clean) {
		return fmt.Errorf("engmod.yml at %s: documents.%s %q leaves the model directory", v.ref, doc.key, doc.rel)
	}
	data, err := Output(v.repo, "show", v.ref+":./"+clean)
	if err != nil {
		return fmt.Errorf("documents.%s at %s: %w", doc.key, v.ref, err)
	}
	if data, err = historicDocument(doc.key, data); err != nil {
		return fmt.Errorf("documents.%s at %s: %w", doc.key, v.ref, err)
	}
	return writePrivate(filepath.Join(dest, filepath.FromSlash(clean)), data)
}

// insideTree reports whether a cleaned slash path names something below the directory it is relative to.
// TRLC-LINKS: REQ-EMG-190
func insideTree(clean string) bool {
	if clean == "." || clean == ".." || path.IsAbs(clean) {
		return false
	}
	return !strings.HasPrefix(clean, "../")
}

// writePrivate writes a file with its directories, readable by the owner only.
// TRLC-LINKS: REQ-EMG-190
func writePrivate(name string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o700); err != nil {
		return err
	}
	return os.WriteFile(name, data, 0o600)
}

// TRLC-LINKS: REQ-EMG-190
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// LoadBundleAtRef loads the model in repo as committed at a git ref, from a private temporary copy that is
// removed afterwards.
// TRLC-LINKS: REQ-EMG-102, REQ-EMG-103, REQ-EMG-186, REQ-EMG-190
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-MODEL-LOADER
func LoadBundleAtRef(repo, ref string) (model.Bundle, error) {
	if err := CheckRef(repo, ref); err != nil {
		return model.Bundle{}, err
	}
	tmp, err := os.MkdirTemp("", "engws-contract-")
	if err != nil {
		return model.Bundle{}, err
	}
	defer os.RemoveAll(tmp)
	if err := WriteModelAtRef(repo, ref, tmp); err != nil {
		return model.Bundle{}, err
	}
	bundle, err := model.LoadBundle(filepath.Join(tmp, "engmod.yml"))
	if err != nil {
		return model.Bundle{}, fmt.Errorf("model at %s: %w", ref, err)
	}
	return bundle, nil
}

// Limits of CheckoutAtRef: a checkout is a convenience for reading one revision, not a way to copy a repository.
const (
	maxCheckoutBytes   = 1 << 30
	maxCheckoutEntries = 300_000
)

// CheckoutAtRef extracts the committed tree at ref into a private temporary directory (0700) and rewrites
// its model files like WriteModelAtRef, so the model can be loaded and the code scanned as it was. Only
// regular files and directories are written: a link, a device or a name that leaves the directory is skipped.
// The caller removes the directory with the returned function.
// TRLC-LINKS: REQ-EMG-190, REQ-EMG-191
// ENGMODEL-LINKS: FU-MODEL-LOADER
func CheckoutAtRef(repo, ref string) (dir string, cleanup func(), err error) {
	if err := CheckRef(repo, ref); err != nil {
		return "", nil, err
	}
	tmp, err := os.MkdirTemp("", "engws-checkout-")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { os.RemoveAll(tmp) }
	if err := archiveInto(repo, ref, tmp); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := WriteModelAtRef(repo, ref, tmp); err != nil {
		cleanup()
		return "", nil, err
	}
	return tmp, cleanup, nil
}

// archiveInto streams git archive of ref into dest.
// TRLC-LINKS: REQ-EMG-190
func archiveInto(repo, ref, dest string) error {
	cmd := exec.Command("git", "-C", repo, "archive", "--format=tar", ref)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := extractTar(pipe, dest); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("git archive %s: %w", ref, err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("git archive %s: %v: %s", ref, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// extractTar writes the directories and regular files of a tar stream below dest, within the limits of a checkout.
// TRLC-LINKS: REQ-EMG-190
func extractTar(r io.Reader, dest string) error {
	tr := tar.NewReader(r)
	var total int64
	for entries := 1; ; entries++ {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if entries > maxCheckoutEntries {
			return fmt.Errorf("more than %d entries", maxCheckoutEntries)
		}
		if total += h.Size; total > maxCheckoutBytes {
			return fmt.Errorf("more than %d bytes", int64(maxCheckoutBytes))
		}
		if err := extractEntry(tr, h, dest); err != nil {
			return err
		}
	}
}

// extractEntry writes one tar entry if it is a directory or a regular file whose name stays below dest.
// TRLC-LINKS: REQ-EMG-190
func extractEntry(tr *tar.Reader, h *tar.Header, dest string) error {
	name := path.Clean(h.Name)
	if !insideTree(name) {
		return nil
	}
	target := filepath.Join(dest, filepath.FromSlash(name))
	switch h.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(target, 0o700)
	case tar.TypeReg:
		return copyEntry(tr, h.Size, target)
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-190
func copyEntry(r io.Reader, size int64, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, io.LimitReader(r, size)); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// topLevelSchemaVersion matches the top-level schemaVersion line of a document: the key at the start of a
// line, so a nested key is never matched.
var topLevelSchemaVersion = regexp.MustCompile(`(?m)^schemaVersion:[ \t]*[0-9]+`)

// historicDocument prepares one document read at an older ref for the loader: the copy is stamped with the
// current version, and a behavior document is stripped of the retired state vocabulary.
// TRLC-LINKS: REQ-EMG-102, REQ-EMG-186
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-MODEL-LOADER
func historicDocument(key string, data []byte) ([]byte, error) {
	data = StampCurrentVersion(data)
	if key != "behavior" {
		return data, nil
	}
	return model.StripLegacyStateVocabulary(data)
}

// StampCurrentVersion replaces the value of the top-level schemaVersion line of a document with
// model.CurrentSchemaVersion and leaves every other byte, including comments and the order of keys, as it was.
// A document without the line is returned unchanged.
// TRLC-LINKS: REQ-EMG-186
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func StampCurrentVersion(data []byte) []byte {
	loc := topLevelSchemaVersion.FindIndex(data)
	if loc == nil {
		return data
	}
	stamped := append([]byte{}, data[:loc[0]]...)
	stamped = append(stamped, model.SchemaVersionLine...)
	return append(stamped, data[loc[1]:]...)
}
