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

func NodGetChild(n Nod, edgeType int) Nod {
	return n.Out[edgeType].Out
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
