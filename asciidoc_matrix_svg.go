// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

import (
	"fmt"
	"html"
	"strings"
)

// TRLC-LINKS: REQ-EMG-003
// publicationMatrixSVG keeps an overview atomic in paginated publications.
func publicationMatrixSVG(rows [][]string, manhattan bool) string {
	if len(rows) == 0 || len(rows[0]) == 0 {
		return ""
	}
	const width, height = 180, 30
	var out strings.Builder
	fmt.Fprintf(&out, "++++\n<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 %d %d\" role=\"img\" aria-label=\"Complete matrix overview\" style=\"display:block;width:100%%;height:auto;max-height:650px;break-inside:avoid;page-break-inside:avoid;\">\n", len(rows[0])*width, len(rows)*height)
	for r, row := range rows {
		for c, label := range row {
			if manhattan && label == "" {
				continue
			}
			fill, stroke := "#ffffff", "#cccccc"
			if manhattan {
				fill, stroke = "#e3f2fd", "#0d47a1"
			}
			if (manhattan && r == len(rows)-1) || (!manhattan && (r == 0 || c == 0)) {
				fill, stroke = "#e8f5e9", "#1b5e20"
			}
			fmt.Fprintf(&out, "<rect x=\"%d\" y=\"%d\" width=\"%d\" height=\"%d\" fill=\"%s\" stroke=\"%s\"/>\n", c*width, r*height, width, height, fill, stroke)
			if label == "" {
				continue
			}
			fit := ""
			if len([]rune(label)) > 23 {
				fit = ` textLength="170" lengthAdjust="spacingAndGlyphs"`
			}
			fmt.Fprintf(&out, "<text x=\"%d\" y=\"%d\" text-anchor=\"middle\" font-family=\"sans-serif\" font-size=\"12\"%s>%s</text>\n", c*width+width/2, r*height+19, fit, html.EscapeString(label))
		}
	}
	out.WriteString("</svg>\n++++")
	return out.String()
}
