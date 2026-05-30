// ENGMODEL-OWNER-UNIT: FU-VIEW-PROJECTION
package view

// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
type Node struct {
	ID    string
	Label string
	Kind  string
}

// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
type Edge struct {
	From  string
	To    string
	Type  string
	Label string
}

// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
type ProjectedView struct {
	ID    string
	Kind  string
	Title string
	Nodes []Node
	Edges []Edge
}
