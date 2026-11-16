// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package gitmodel

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

type tarItem struct {
	name, body, link string
	kind             byte
}

// header is the tar header of the item; an item without a kind is a regular file.
func (it tarItem) header() *tar.Header {
	if it.kind == 0 {
		it.kind = tar.TypeReg
	}
	h := &tar.Header{Name: it.name, Typeflag: it.kind, Mode: 0o644, Linkname: it.link}
	if it.kind == tar.TypeReg {
		h.Size = int64(len(it.body))
	}
	return h
}

func writeTarItem(t *testing.T, tw *tar.Writer, it tarItem) {
	t.Helper()
	h := it.header()
	if err := tw.WriteHeader(h); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(it.body[:h.Size])); err != nil {
		t.Fatal(err)
	}
}

func tarOf(t *testing.T, items ...tarItem) *bytes.Reader {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, it := range items {
		writeTarItem(t, tw, it)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}

func extractSample(t *testing.T) (root, dest string) {
	t.Helper()
	root = t.TempDir()
	dest = filepath.Join(root, "dest")
	err := extractTar(tarOf(t,
		tarItem{name: "src/", kind: tar.TypeDir},
		tarItem{name: "src/a.go", body: "package a"},
		tarItem{name: "../escape.txt", body: "x"},
		tarItem{name: "/abs.txt", body: "x"},
		tarItem{name: "link", kind: tar.TypeSymlink, link: "/etc/passwd"},
		tarItem{name: "dev", kind: tar.TypeChar},
	), dest)
	if err != nil {
		t.Fatal(err)
	}
	return root, dest
}

// A checkout writes regular files below the destination with 0600.
// TRLC-LINKS: REQ-EMG-190
func TestExtractTarWritesRegularFiles(t *testing.T) {
	_, dest := extractSample(t)
	name := filepath.Join(dest, "src", "a.go")
	body, err := os.ReadFile(name)
	info, _ := os.Stat(name)
	if err != nil || string(body) != "package a" || info.Mode().Perm() != 0o600 {
		t.Errorf("src/a.go: %q %v %v", body, err, info)
	}
}

// Names that leave the destination, absolute names, links and devices are skipped.
// Mutation: following a link or joining an unchecked name writes outside dest and fails this test.
// TRLC-LINKS: REQ-EMG-190
func TestExtractTarSkipsEverythingElse(t *testing.T) {
	root, dest := extractSample(t)
	for _, p := range []string{filepath.Join(root, "escape.txt"), filepath.Join(dest, "escape.txt"), filepath.Join(dest, "abs.txt"), filepath.Join(dest, "link"), filepath.Join(dest, "dev"), "/abs.txt"} {
		if _, err := os.Lstat(p); err == nil {
			t.Errorf("%s was written", p)
		}
	}
}

// A refused ref and an unknown one are errors that name the ref; a ref that git would read as an option never
// reaches git.
// TRLC-LINKS: REQ-EMG-102, REQ-EMG-190
func TestCheckRefRefusesOptionsAndUnknownRefs(t *testing.T) {
	repo := t.TempDir()
	for _, ref := range []string{"--output=x", "-h", "", "  ", "no-such-ref"} {
		if err := CheckRef(repo, ref); err == nil {
			t.Errorf("%q was accepted", ref)
		}
	}
}

// A manifest that names a document outside the model directory is refused before anything is read from git.
// TRLC-LINKS: REQ-EMG-190
func TestInsideTreeRefusesNamesThatLeave(t *testing.T) {
	for name, want := range map[string]bool{"model/a.yml": true, "a.yml": true, ".": false, "..": false, "../a": false, "/a": false} {
		if got := insideTree(name); got != want {
			t.Errorf("insideTree(%q) = %v, want %v", name, got, want)
		}
	}
}
