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
	show := func(rel string) ([]byte, error) { return Output(repo, "show", ref+":./"+rel) }
	manifest, err := show("engmod.yml")
	if err != nil {
		return fmt.Errorf("%s has no engmod.yml at %s: %w", repo, ref, err)
	}
	var meta struct {
		Documents map[string]string `yaml:"documents"`
	}
	if err := yaml.Unmarshal(manifest, &meta); err != nil {
		return fmt.Errorf("engmod.yml at %s: %w", ref, err)
	}
	if err := os.MkdirAll(dest, 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dest, "engmod.yml"), StampCurrentVersion(manifest), 0o600); err != nil {
		return err
	}
	keys := make([]string, 0, len(meta.Documents))
	for key := range meta.Documents {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		rel := meta.Documents[key]
		clean := path.Clean(filepath.ToSlash(strings.TrimSpace(rel)))
		if clean == "." || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
			return fmt.Errorf("engmod.yml at %s: documents.%s %q leaves the model directory", ref, key, rel)
		}
		data, err := show(clean)
		if err != nil {
			return fmt.Errorf("documents.%s at %s: %w", key, ref, err)
		}
		if data, err = historicDocument(key, data); err != nil {
			return fmt.Errorf("documents.%s at %s: %w", key, ref, err)
		}
		out := filepath.Join(dest, filepath.FromSlash(clean))
		if err := os.MkdirAll(filepath.Dir(out), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(out, data, 0o600); err != nil {
			return err
		}
	}
	return nil
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
	cmd := exec.Command("git", "-C", repo, "archive", "--format=tar", ref)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		cleanup()
		return "", nil, err
	}
	if err := cmd.Start(); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := extractTar(pipe, tmp); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		cleanup()
		return "", nil, fmt.Errorf("git archive %s: %w", ref, err)
	}
	if err := cmd.Wait(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("git archive %s: %v: %s", ref, err, strings.TrimSpace(stderr.String()))
	}
	if err := WriteModelAtRef(repo, ref, tmp); err != nil {
		cleanup()
		return "", nil, err
	}
	return tmp, cleanup, nil
}

// TRLC-LINKS: REQ-EMG-190
func extractTar(r io.Reader, dest string) error {
	tr := tar.NewReader(r)
	var total int64
	entries := 0
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if entries++; entries > maxCheckoutEntries {
			return fmt.Errorf("more than %d entries", maxCheckoutEntries)
		}
		name := path.Clean(h.Name)
		if name == "." || path.IsAbs(name) || name == ".." || strings.HasPrefix(name, "../") {
			continue
		}
		target := filepath.Join(dest, filepath.FromSlash(name))
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
		case tar.TypeReg:
			if total += h.Size; total > maxCheckoutBytes {
				return fmt.Errorf("more than %d bytes", int64(maxCheckoutBytes))
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, io.LimitReader(tr, h.Size)); err != nil {
				f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		}
	}
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
