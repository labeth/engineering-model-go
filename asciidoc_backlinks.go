// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

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

func applyReferenceBacklinks(document string, ref asciidocReferenceIndex) asciidocReferenceIndex {
	refAnchors := map[string]bool{}
	addRefAnchors := func(entries []asciidocReferenceEntry) {
		for _, e := range entries {
			if strings.TrimSpace(e.Anchor) == "" {
				continue
			}
			refAnchors[strings.TrimSpace(e.Anchor)] = true
		}
	}
	addRefAnchors(ref.Authored)
	addRefAnchors(ref.Catalog)
	addRefAnchors(ref.Runtime)
	addRefAnchors(ref.Code)
	addRefAnchors(ref.Verification)

	backlinksByRef := map[string][]string{}
	seen := map[string]bool{}
	addBacklink := func(refAnchor, sectionAnchor, sectionLabel string) {
		refAnchor = strings.TrimSpace(refAnchor)
		sectionAnchor = strings.TrimSpace(sectionAnchor)
		sectionLabel = strings.TrimSpace(sectionLabel)
		if refAnchor == "" || sectionAnchor == "" || sectionLabel == "" {
			return
		}
		key := refAnchor + "|" + sectionAnchor + "|" + sectionLabel
		if seen[key] {
			return
		}
		seen[key] = true
		backlinksByRef[refAnchor] = append(backlinksByRef[refAnchor], fmt.Sprintf("<<%s,%s>>", sectionAnchor, sectionLabel))
	}

	currentSectionAnchor := ""
	currentSectionLabel := ""
	pendingAnchor := ""
	lines := strings.Split(document, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if trimmed == "== Reference Index" {
			break
		}
		if match := anchorLinePattern.FindStringSubmatch(trimmed); len(match) == 2 {
			pendingAnchor = strings.TrimSpace(match[1])
			continue
		}
		if match := headingLinePattern.FindStringSubmatch(trimmed); len(match) == 3 {
			if strings.TrimSpace(pendingAnchor) != "" {
				currentSectionAnchor = strings.TrimSpace(pendingAnchor)
				currentSectionLabel = strings.TrimSpace(match[2])
				pendingAnchor = ""
			}
			continue
		}
		if currentSectionAnchor == "" || currentSectionLabel == "" {
			continue
		}
		for _, linkMatch := range sectionLinkPattern.FindAllStringSubmatch(trimmed, -1) {
			if len(linkMatch) != 2 {
				continue
			}
			target := strings.TrimSpace(linkMatch[1])
			if !refAnchors[target] {
				continue
			}
			if target == currentSectionAnchor {
				continue
			}
			addBacklink(target, currentSectionAnchor, currentSectionLabel)
		}
	}

	apply := func(entries []asciidocReferenceEntry) []asciidocReferenceEntry {
		out := make([]asciidocReferenceEntry, len(entries))
		copy(out, entries)
		for i := range out {
			out[i].Backlinks = backlinksByRef[out[i].Anchor]
		}
		return out
	}

	ref.Authored = apply(ref.Authored)
	ref.Catalog = apply(ref.Catalog)
	ref.Runtime = apply(ref.Runtime)
	ref.Code = apply(ref.Code)
	ref.Verification = apply(ref.Verification)
	return ref
}
