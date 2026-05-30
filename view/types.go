// ENGMODEL-OWNER-UNIT: FU-VIEW-PROJECTION
package view

// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: EM-VIEW, EM-MODEL
type Node struct {
	ID    string
	Label string
	Kind  string
}

// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: EM-VIEW, EM-AUTHORED-MAPPING
type Edge struct {
	From  string
	To    string
	Type  string
	Label string
}

// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: EM-VIEW
type ProjectedView struct {
	ID    string
	Kind  string
	Title string
	Nodes []Node
	Edges []Edge
}
