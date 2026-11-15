// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import (
	"fmt"
	"html"
	"strings"
)

const (
	matrixCellWidth  = 180
	matrixCellHeight = 30
)

// matrixLayout is a matrix of labels and how it is drawn: the plain matrix
// has a header row and column, the Manhattan matrix a label row at the bottom
// and no cells for empty labels.
type matrixLayout struct {
	rows      [][]string
	manhattan bool
}

// TRLC-LINKS: REQ-EMG-003
// publicationMatrixSVG keeps an overview atomic in paginated publications.
func publicationMatrixSVG(rows [][]string, manhattan bool) string {
	if len(rows) == 0 || len(rows[0]) == 0 {
		return ""
	}
	m := matrixLayout{rows: rows, manhattan: manhattan}
	var out strings.Builder
	fmt.Fprintf(&out, "++++\n<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 %d %d\" role=\"img\" aria-label=\"Complete matrix overview\" style=\"display:block;width:100%%;height:auto;max-height:650px;break-inside:avoid;page-break-inside:avoid;\">\n", len(rows[0])*matrixCellWidth, len(rows)*matrixCellHeight)
	for r, row := range rows {
		for c := range row {
			m.writeCell(&out, r, c)
		}
	}
	out.WriteString("</svg>\n++++")
	return out.String()
}

// isHeader reports whether the cell is part of the header.
// TRLC-LINKS: REQ-EMG-003
func (m matrixLayout) isHeader(r, c int) bool {
	if m.manhattan {
		return r == len(m.rows)-1
	}
	return r == 0 || c == 0
}

// colors are the fill and stroke of a cell.
// TRLC-LINKS: REQ-EMG-003
func (m matrixLayout) colors(r, c int) (fill, stroke string) {
	switch {
	case m.isHeader(r, c):
		return "#e8f5e9", "#1b5e20"
	case m.manhattan:
		return "#e3f2fd", "#0d47a1"
	}
	return "#ffffff", "#cccccc"
}

// writeCell writes the rectangle and the text of a cell.
// TRLC-LINKS: REQ-EMG-003
func (m matrixLayout) writeCell(out *strings.Builder, r, c int) {
	label := m.rows[r][c]
	if m.manhattan && label == "" {
		return
	}
	fill, stroke := m.colors(r, c)
	fmt.Fprintf(out, "<rect x=\"%d\" y=\"%d\" width=\"%d\" height=\"%d\" fill=\"%s\" stroke=\"%s\"/>\n", c*matrixCellWidth, r*matrixCellHeight, matrixCellWidth, matrixCellHeight, fill, stroke)
	if label == "" {
		return
	}
	fit := ""
	if len([]rune(label)) > 23 {
		fit = ` textLength="170" lengthAdjust="spacingAndGlyphs"`
	}
	fmt.Fprintf(out, "<text x=\"%d\" y=\"%d\" text-anchor=\"middle\" font-family=\"sans-serif\" font-size=\"12\"%s>%s</text>\n", c*matrixCellWidth+matrixCellWidth/2, r*matrixCellHeight+19, fit, html.EscapeString(label))
}
