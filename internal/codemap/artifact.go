// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

// Non-code artifacts as implementation evidence. A container build file
// (Dockerfile, Containerfile, *.Dockerfile, *.Containerfile) or a Markdown
// document may carry TRLC-LINKS, ENGMODEL-LINKS and ENGMODEL-OWNER-UNIT markers
// in its comments: `#` lines in a container file, HTML comments starting a line
// outside fenced code blocks in Markdown. The markers apply to the whole file,
// which becomes one symbol with the file's CODE identifier. Markers are
// optional: an artifact without them yields nothing and no missing-link error.

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/labeth/engineering-model-go/internal/validate"
)

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
// text inside an HTML comment that starts a line, outside fenced code blocks,
// with the comment delimiters removed.
// TRLC-LINKS: REQ-EMG-084
func artifactComments(path string, src []byte) map[int]string {
	out := map[int]string{}
	markdown := isMarkdownFile(path)
	inComment := false
	fence := ""
	scanner := bufio.NewScanner(strings.NewReader(string(src)))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if !markdown {
			if text, ok := strings.CutPrefix(line, "#"); ok {
				out[lineNo] = strings.TrimSpace(text)
			}
			continue
		}
		if inComment {
			text, closed := strings.CutSuffix(line, "-->")
			if !closed {
				text, _, closed = strings.Cut(line, "-->")
			}
			out[lineNo] = strings.TrimSpace(text)
			inComment = !closed
			continue
		}
		if fence != "" {
			if strings.HasPrefix(line, fence) {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~") {
			fence = line[:3]
			continue
		}
		if text, ok := strings.CutPrefix(line, "<!--"); ok {
			body, _, closed := strings.Cut(text, "-->")
			out[lineNo] = strings.TrimSpace(body)
			inComment = !closed
		}
	}
	return out
}

// ArtifactMarker returns the value of the first comment marker (for example
// "ENGMODEL-OWNER-UNIT:") in an artifact and its 1-based line, or "" and 0.
// TRLC-LINKS: REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func ArtifactMarker(path string, src []byte, marker string) (string, int) {
	comments := artifactComments(path, src)
	lines := strings.Count(string(src), "\n") + 1
	for lineNo := 1; lineNo <= lines; lineNo++ {
		text, ok := comments[lineNo]
		if !ok {
			continue
		}
		if v, ok := strings.CutPrefix(text, marker); ok {
			if fields := strings.Fields(v); len(fields) > 0 {
				return fields[0], lineNo
			}
		}
	}
	return "", 0
}

// scanArtifact returns the file-level symbol of an artifact with trace
// markers, or none. Invalid requirement IDs are code.invalid_trlc_link errors.
// TRLC-LINKS: REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func scanArtifact(relPath, idPath string, src []byte) ([]Symbol, []validate.Diagnostic) {
	comments := artifactComments(relPath, src)
	var implements, modelLinks []string
	var diags []validate.Diagnostic
	first := 0
	lines := strings.Count(string(src), "\n") + 1
	for lineNo := 1; lineNo <= lines; lineNo++ {
		text, ok := comments[lineNo]
		if !ok {
			continue
		}
		if v, ok := strings.CutPrefix(text, "TRLC-LINKS:"); ok {
			links := splitCSV(v)
			implements = append(implements, links...)
			for _, link := range links {
				if !validRequirementID(link) {
					diags = append(diags, validate.Diagnostic{Code: "code.invalid_trlc_link", Severity: validate.SeverityError,
						Message: fmt.Sprintf("TRLC-LINKS value %q is not a requirement id", link), Path: fmt.Sprintf("%s:%d", relPath, lineNo)})
				}
			}
		} else if v, ok := strings.CutPrefix(text, "ENGMODEL-LINKS:"); ok {
			modelLinks = append(modelLinks, splitCSV(v)...)
		} else {
			continue
		}
		if first == 0 {
			first = lineNo
		}
	}
	if first == 0 {
		return nil, diags
	}
	return []Symbol{{
		TraceID:    FileID(idPath),
		Implements: implements,
		ModelLinks: modelLinks,
		Path:       relPath,
		Line:       first,
		Signature:  filepath.Base(relPath),
	}}, diags
}
