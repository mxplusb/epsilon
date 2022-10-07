package geometry

type Edge struct {
	next *Edge
	reverse *Edge
	targetVertex int
}

//func (e *Edge) GetSourceVertex() *Edge {
//}

func (e Edge) GetTargetVertex() int {
	return e.targetVertex
}

func (e *Edge) GetNextEdgeOfVertex() *Edge {
	return e.next
}

func (e *Edge) GetNextEdgeOfFace() *Edge {
	return (*e.reverse).GetNextEdgeOfVertex()
}
