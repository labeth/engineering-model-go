package view

type Node struct {
	ID    string
	Label string
	Kind  string
}

type Edge struct {
	From  string
	To    string
	Type  string
	Label string
}

type ProjectedView struct {
	ID    string
	Kind  string
	Title string
	Nodes []Node
	Edges []Edge
}
