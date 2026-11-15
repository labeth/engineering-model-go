// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

// Non-code artifacts as implementation evidence. A container build file
// (Dockerfile, Containerfile, *.Dockerfile, *.Containerfile) or a Markdown
// document may carry TRLC-LINKS, ENGMODEL-LINKS and ENGMODEL-OWNER-UNIT markers
// in its comments: `#` lines in a container file, HTML comments starting a line
// outside fenced and indented code blocks in Markdown. Only regular files up to
// MaxArtifactBytes are read. The markers apply to the whole file,
// which becomes one symbol with the file's CODE identifier. Markers are
// optional: an artifact without them yields nothing and no missing-link error.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/validate"
)

// MaxArtifactBytes is the largest container build file or Markdown document
// the scanner reads; a larger one, like a non-regular file (symlink, device,
// FIFO), is skipped, so an untrusted repository cannot make the scan read an
// unbounded or blocking file. It mirrors the MCP server's file index cap.
const MaxArtifactBytes = 512 * 1024

// ReadableArtifact reports whether an artifact file may be read: a regular
// file of at most MaxArtifactBytes. It does not follow symlinks.
// TRLC-LINKS: REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, TB-REPO-WORKSPACE
func ReadableArtifact(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular() && info.Size() <= MaxArtifactBytes
}

// IsArtifactFile reports whether a path names a container build file or a
// Markdown document whose comment markers the scanner reads.
// TRLC-LINKS: REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func IsArtifactFile(path string) bool {
	return isContainerFile(path) || isMarkdownFile(path)
}

// TRLC-LINKS: REQ-EMG-084
func isContainerFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	for _, name := range []string{"dockerfile", "containerfile"} {
		if base == name || strings.HasSuffix(base, "."+name) {
			return true
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-084
func isMarkdownFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".markdown"
}

// artifactComments returns the text of each comment line of an artifact by
// 1-based line number: the text after `#` in a container file; in Markdown the
// text inside an HTML comment that starts a line (indented at most three
// columns, so not in an indented code block), outside fenced code blocks, with
// the comment delimiters removed. The file is split on bytes, so no line length limit
// drops lines.
// TRLC-LINKS: REQ-EMG-084
func artifactComments(path string, src []byte) map[int]string {
	reader := commentReader{out: map[int]string{}, markdown: isMarkdownFile(path)}
	for i, raw := range bytes.Split(src, []byte("\n")) {
		reader.readLine(i+1, strings.TrimSuffix(string(raw), "\r"))
	}
	return reader.out
}

// commentReader reads the comments of an artifact line by line.
type commentReader struct {
	out       map[int]string
	markdown  bool
	inComment bool
	fence     markdownFence
}

// readLine reads one line, without its line break.
// TRLC-LINKS: REQ-EMG-084
func (r *commentReader) readLine(lineNo int, text string) {
	line := strings.TrimSpace(text)
	switch {
	case !r.markdown:
		if comment, ok := strings.CutPrefix(line, "#"); ok {
			r.out[lineNo] = strings.TrimSpace(comment)
		}
	case r.inComment:
		body, closed := strings.CutSuffix(line, "-->")
		if !closed {
			body, _, closed = strings.Cut(line, "-->")
		}
		r.out[lineNo] = strings.TrimSpace(body)
		r.inComment = !closed
	case r.fence.char != 0:
		if r.fence.closedBy(text) {
			r.fence = markdownFence{}
		}
	default:
		r.readMarkdownLine(lineNo, text)
	}
}

// readMarkdownLine reads a Markdown line outside a comment and a fence: a
// fence opening, an indented line, or the start of an HTML comment.
// TRLC-LINKS: REQ-EMG-084
func (r *commentReader) readMarkdownLine(lineNo int, text string) {
	if f, ok := openMarkdownFence(text); ok {
		r.fence = f
		return
	}
	// A line indented four or more columns is an indented code block or
	// a nested list continuation; neither starts a comment.
	if markdownIndent(text) >= 4 {
		return
	}
	if body, ok := strings.CutPrefix(strings.TrimSpace(text), "<!--"); ok {
		comment, _, closed := strings.Cut(body, "-->")
		r.out[lineNo] = strings.TrimSpace(comment)
		r.inComment = !closed
	}
}

// markdownFence is an open CommonMark fenced code block: its fence character
// (` or ~) and length.
type markdownFence struct {
	char byte
	n    int
}

// openMarkdownFence reports the fence a line opens: at most three spaces of
// indentation, then three or more backticks or tildes (a backtick fence's
// info string has no backtick).
// TRLC-LINKS: REQ-EMG-084
func openMarkdownFence(line string) (markdownFence, bool) {
	if markdownIndent(line) >= 4 {
		return markdownFence{}, false
	}
	t := strings.TrimLeft(line, " ")
	if t == "" || !strings.ContainsRune("`~", rune(t[0])) {
		return markdownFence{}, false
	}
	n := len(t) - len(strings.TrimLeft(t, t[:1]))
	if n < 3 || backtickInInfo(t, n) {
		return markdownFence{}, false
	}
	return markdownFence{char: t[0], n: n}, true
}

// backtickInInfo reports whether a backtick fence of n characters has a
// backtick in its info string, which a fence opening may not.
// TRLC-LINKS: REQ-EMG-084
func backtickInInfo(t string, n int) bool {
	return t[0] == '`' && strings.Contains(t[n:], "`")
}

// closedBy reports whether a line closes the fence: at most three spaces of
// indentation, at least as many of the same fence characters, then only
// spaces.
// TRLC-LINKS: REQ-EMG-084
func (f markdownFence) closedBy(line string) bool {
	if markdownIndent(line) >= 4 {
		return false
	}
	t := strings.TrimLeft(line, " ")
	n := len(t) - len(strings.TrimLeft(t, string(f.char)))
	return n >= f.n && strings.TrimSpace(t[n:]) == ""
}

// markdownIndent is the indentation width of a line, a tab counting as four.
// TRLC-LINKS: REQ-EMG-084
func markdownIndent(line string) int {
	width := 0
	for _, r := range line {
		switch r {
		case ' ':
			width++
		case '\t':
			width += 4 - width%4
		default:
			return width
		}
	}
	return width
}

// ArtifactMarker returns the value of the first comment marker (for example
// "ENGMODEL-OWNER-UNIT:") in an artifact and its 1-based line, or "" and 0.
// TRLC-LINKS: REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func ArtifactMarker(path string, src []byte, marker string) (string, int) {
	return artifactMarker(artifactComments(path, src), marker)
}

// ArtifactMarkers returns the values and 1-based lines of the
// ENGMODEL-OWNER-UNIT, TRLC-LINKS and ENGMODEL-LINKS markers of an artifact,
// parsing it once; an absent marker has value "" and line 0.
// TRLC-LINKS: REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func ArtifactMarkers(path string, src []byte) (owner string, ownerLine, linksLine, modelLinksLine int) {
	comments := artifactComments(path, src)
	owner, ownerLine = artifactMarker(comments, "ENGMODEL-OWNER-UNIT:")
	_, linksLine = artifactMarker(comments, "TRLC-LINKS:")
	_, modelLinksLine = artifactMarker(comments, "ENGMODEL-LINKS:")
	return owner, ownerLine, linksLine, modelLinksLine
}

// artifactMarker returns the first value of a marker in parsed comments.
// TRLC-LINKS: REQ-EMG-084
func artifactMarker(comments map[int]string, marker string) (string, int) {
	for _, lineNo := range sortedLines(comments) {
		if v, ok := strings.CutPrefix(comments[lineNo], marker); ok && len(strings.Fields(v)) > 0 {
			return strings.Fields(v)[0], lineNo
		}
	}
	return "", 0
}

// sortedLines returns the line numbers of parsed comments in order.
// TRLC-LINKS: REQ-EMG-084
func sortedLines(comments map[int]string) []int {
	lines := make([]int, 0, len(comments))
	for lineNo := range comments {
		lines = append(lines, lineNo)
	}
	sort.Ints(lines)
	return lines
}

// scanArtifact returns the file-level symbol of an artifact with trace
// markers, or none. Invalid requirement IDs are code.invalid_trlc_link errors.
// TRLC-LINKS: REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func scanArtifact(relPath, idPath string, src []byte) ([]Symbol, []validate.Diagnostic) {
	comments := artifactComments(relPath, src)
	var links artifactLinks
	for _, lineNo := range sortedLines(comments) {
		links.note(relPath, lineNo, comments[lineNo])
	}
	if links.first == 0 {
		return nil, links.diags
	}
	return []Symbol{{
		TraceID:    FileID(idPath),
		Implements: links.implements,
		ModelLinks: links.modelLinks,
		Path:       relPath,
		Line:       links.first,
		Signature:  filepath.Base(relPath),
	}}, links.diags
}

// artifactLinks collects the TRLC-LINKS and ENGMODEL-LINKS values of an
// artifact's comments and the line of the first.
type artifactLinks struct {
	implements []string
	modelLinks []string
	diags      []validate.Diagnostic
	first      int
}

// note reads one comment line.
// TRLC-LINKS: REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func (a *artifactLinks) note(relPath string, lineNo int, text string) {
	if v, ok := strings.CutPrefix(text, "TRLC-LINKS:"); ok {
		a.noteRequirements(relPath, lineNo, v)
	} else if v, ok := strings.CutPrefix(text, "ENGMODEL-LINKS:"); ok {
		a.modelLinks = append(a.modelLinks, splitCSV(v)...)
	} else {
		return
	}
	if a.first == 0 {
		a.first = lineNo
	}
}

// noteRequirements collects the requirement ids of a TRLC-LINKS value and
// reports each that is not one.
// TRLC-LINKS: REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func (a *artifactLinks) noteRequirements(relPath string, lineNo int, value string) {
	links := splitCSV(value)
	a.implements = append(a.implements, links...)
	for _, link := range links {
		if !validRequirementID(link) {
			a.diags = append(a.diags, validate.Diagnostic{Code: "code.invalid_trlc_link", Severity: validate.SeverityError,
				Message: fmt.Sprintf("TRLC-LINKS value %q is not a requirement id", link), Path: fmt.Sprintf("%s:%d", relPath, lineNo)})
		}
	}
}
