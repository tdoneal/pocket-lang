package parse

const (
	NTR_LIST_0   = 100000 // 100000<->0th element, 100001<->1st element, etc
	NTR_LIST_MAX = 200000
)

type NodeType struct {
	id   int
	name string
}

type Edge struct {
	In       *Node
	EdgeType int
	Out      *Node
}

type Node struct {
	NodeType int
	In       []*Edge
	Out      map[int]*Edge
	Data     interface{}
}

type Nod *Node

func NodNew(nodeType int) Nod {
	rv := &Node{
		NodeType: nodeType,
		In:       make([]*Edge, 0),
		Out:      make(map[int]*Edge),
	}
	return rv
}

func NodNewData(nodeType int, data interface{}) Nod {
	rv := NodNew(nodeType)
	rv.Data = data
	return rv
}

func NodNewChild(nodeType int, edgeType int, child Nod) Nod {
	rv := (*Node)(NodNew(nodeType))
	NodSetChild(rv, edgeType, child)
	return rv
}

func NodNewChildList(nodeType int, children []Nod) Nod {
	rv := (*Node)(NodNew(nodeType))
	NodSetOutList(rv, children)
	return rv
}

func NodSetOutList(n Nod, children []Nod) {
	for i := 0; i < len(children); i++ {
		child := children[i]
		NodSetChild(n, NTR_LIST_0+i, child)
	}
}

func NodRemoveOutList(n Nod) {
	li := NTR_LIST_0
	for {
		_, hasIt := n.Out[li]
		if hasIt {
			NodRemoveChild(n, li)
			li++
		} else {
			break
		}
	}
}

func NodReplaceOutList(n Nod, children []Nod) {
	// sets the out list, making sure to properly clear any existing list nodes first
	NodRemoveOutList(n)
	NodSetOutList(n, children)
}

func NodGetChild(n Nod, edgeType int) Nod {
	return n.Out[edgeType].Out
}

func NodGetParent(n Nod, edgeType int) Nod {
	for _, inEdge := range n.In {
		if inEdge.EdgeType == edgeType {
			return inEdge.In
		}
	}
	panic("parent not found")
}

func NodGetParentByOrNil(n Nod, cond func(Nod) bool) Nod {
	for _, inEdge := range n.In {
		if cond(inEdge.In) {
			return inEdge.In
		}
	}
	return nil
}

func NodGetParentOrNil(n Nod, edgeType int) Nod {
	for _, inEdge := range n.In {
		if inEdge.EdgeType == edgeType {
			return inEdge.In
		}
	}
	return nil
}

func NodGetChildOrNil(n Nod, edgeType int) Nod {
	rvEdge, ok := n.Out[edgeType]
	if !ok {
		return nil
	}
	return rvEdge.Out
}

func NodHasChild(n Nod, edgeType int) bool {
	_, ok := n.Out[edgeType]
	return ok
}

func NodGetChildList(n Nod) []Nod {
	rv := make([]Nod, 0)
	li := NTR_LIST_0
	for {
		val, hasIt := n.Out[li]
		if hasIt {
			rv = append(rv, val.Out)
			li++
		} else {
			break
		}
	}
	return rv
}

func NodSetChild(n Nod, edgeType int, child Nod) {
	// for now assume child doesn't already exist, so skip check
	newEdge := &Edge{
		EdgeType: edgeType,
		In:       n,
		Out:      child,
	}
	n.Out[edgeType] = newEdge
	child.In = append(child.In, newEdge)
}

func NodRemoveChild(n Nod, edgeType int) {
	// completely detatches a node from its parent
	// removes a child from parent, and also removes the child's old reference to the parent
	edge := n.Out[edgeType]
	child := edge.Out
	childInEdgeNdx := NodGetInEdgeNdx(child, edge)
	if childInEdgeNdx == -1 {
		panic("parent edge of child not found")
	}
	delete(n.Out, edgeType)
	child.In = slicePEdgeRemove(child.In, childInEdgeNdx)
}

func slicePEdgeRemove(s []*Edge, i int) []*Edge {
	s[len(s)-1], s[i] = s[i], s[len(s)-1]
	return s[:len(s)-1]
}

func NodGetInEdgeNdx(n Nod, inEdge *Edge) int {
	return sliceIndex(len(n.In), func(i int) bool { return n.In[i] == inEdge })
}

func NodDeepCopyDownwards(n Nod) Nod {
	rv := NodNew(n.NodeType)
	rv.Data = n.Data
	for edgeType, edge := range n.Out {
		NodSetChild(rv, edgeType, NodDeepCopyDownwards(edge.Out))
	}
	return rv
}

func sliceIndex(limit int, predicate func(int) bool) int {
	for i := 0; i < limit; i++ {
		if predicate(i) {
			return i
		}
	}
	return -1
}
