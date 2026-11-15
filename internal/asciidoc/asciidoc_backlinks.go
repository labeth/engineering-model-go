// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	anchorLinePattern  = regexp.MustCompile(`^\[\[([^\]]+)\]\]\s*$`)
	headingLinePattern = regexp.MustCompile(`^(=+)\s+(.+)$`)
	sectionLinkPattern = regexp.MustCompile(`<<([^>,]+)(?:,[^>]+)?>>`)
)

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-014
func applyReferenceBacklinks(document string, ref asciidocReferenceIndex) asciidocReferenceIndex {
	scan := &backlinkScan{refAnchors: referenceAnchorSet(ref), backlinks: map[string][]string{}, seen: map[string]bool{}}
	scan.scan(document)
	ref.Authored = withBacklinks(ref.Authored, scan.backlinks)
	ref.Catalog = withBacklinks(ref.Catalog, scan.backlinks)
	ref.Runtime = withBacklinks(ref.Runtime, scan.backlinks)
	ref.Code = withBacklinks(ref.Code, scan.backlinks)
	ref.Verification = withBacklinks(ref.Verification, scan.backlinks)
	return ref
}

// backlinkScan walks a rendered document and collects, for every reference
// anchor, the sections that link to it.
type backlinkScan struct {
	refAnchors    map[string]bool
	backlinks     map[string][]string
	seen          map[string]bool
	sectionAnchor string
	sectionLabel  string
	pendingAnchor string
}

// referenceAnchorSet is the set of the anchors of every reference entry.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-014
func referenceAnchorSet(ref asciidocReferenceIndex) map[string]bool {
	anchors := map[string]bool{}
	for _, entries := range [][]asciidocReferenceEntry{ref.Authored, ref.Catalog, ref.Runtime, ref.Code, ref.Verification} {
		for _, e := range entries {
			if anchor := strings.TrimSpace(e.Anchor); anchor != "" {
				anchors[anchor] = true
			}
		}
	}
	return anchors
}

// scan reads the document down to the Reference Index.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-014
func (s *backlinkScan) scan(document string) {
	for _, line := range strings.Split(document, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if trimmed == "== Reference Index" {
			return
		}
		s.consume(trimmed)
	}
}

// consume handles one non-blank line: an anchor, a heading or text with links.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-014
func (s *backlinkScan) consume(line string) {
	if match := anchorLinePattern.FindStringSubmatch(line); len(match) == 2 {
		s.pendingAnchor = strings.TrimSpace(match[1])
		return
	}
	if match := headingLinePattern.FindStringSubmatch(line); len(match) == 3 {
		s.enterSection(match[2])
		return
	}
	if s.sectionAnchor != "" && s.sectionLabel != "" {
		s.collectLinks(line)
	}
}

// enterSection makes the heading the current section when an anchor precedes it.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-014
func (s *backlinkScan) enterSection(heading string) {
	if strings.TrimSpace(s.pendingAnchor) == "" {
		return
	}
	s.sectionAnchor = strings.TrimSpace(s.pendingAnchor)
	s.sectionLabel = strings.TrimSpace(heading)
	s.pendingAnchor = ""
}

// collectLinks records the links of a line to reference anchors, other than
// the current section itself.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-014
func (s *backlinkScan) collectLinks(line string) {
	for _, linkMatch := range sectionLinkPattern.FindAllStringSubmatch(line, -1) {
		if len(linkMatch) != 2 {
			continue
		}
		if target := strings.TrimSpace(linkMatch[1]); s.refAnchors[target] && target != s.sectionAnchor {
			s.addBacklink(target)
		}
	}
}

// addBacklink records the current section as a backlink of the anchor, once.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-014
func (s *backlinkScan) addBacklink(refAnchor string) {
	key := refAnchor + "|" + s.sectionAnchor + "|" + s.sectionLabel
	if s.seen[key] {
		return
	}
	s.seen[key] = true
	s.backlinks[refAnchor] = append(s.backlinks[refAnchor], fmt.Sprintf("<<%s,%s>>", s.sectionAnchor, s.sectionLabel))
}

// withBacklinks copies the entries with the backlinks of their anchors.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-014
func withBacklinks(entries []asciidocReferenceEntry, backlinks map[string][]string) []asciidocReferenceEntry {
	out := make([]asciidocReferenceEntry, len(entries))
	copy(out, entries)
	for i := range out {
		out[i].Backlinks = backlinks[out[i].Anchor]
	}
	return out
}
