package parse

// TODO: write pretty printer for this int-based graph
//  will require way of going from int->human readable edge names

type NodeType struct {
	id   int
	name string
}

const (
	NT_IMPERATIVE          = 10
	NT_VARINIT             = 20
	NTR_VARINIT_NAME       = 21
	NTR_VARINIT_VALUE      = 22
	NT_RECEIVERCALL        = 30
	NTR_RECEIVERCALL_NAME  = 31
	NTR_RECEIVERCALL_VALUE = 32
	NT_IDENTIFIER          = 40
	NT_ADDOP               = 100
	NT_INLINEOPSTREAM      = 150
	NTR_LIT_VALUE          = 201
	NT_LIT_INT             = 210
	NT_VAR_GETTER          = 250
	NTR_VAR_GETTER_NAME    = 251
	NTR_LIST_0             = 100000 // 100000<->0th element, 100001<->1st element, etc
)

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

func NodeNew(nodeType int) Nod {
	rv := &Node{
		NodeType: nodeType,
		In:       make([]*Edge, 0),
		Out:      make(map[int]*Edge),
	}
	return rv
}

func NodeNewData(nodeType int, data interface{}) Nod {
	rv := NodeNew(nodeType)
	rv.Data = data
	return rv
}

func NodeNewChild(nodeType int, edgeType int, child Nod) Nod {
	rv := (*Node)(NodeNew(nodeType))
	rv.setChild(edgeType, child)
	return rv
}

func NodeNewChildList(nodeType int, children []Nod) Nod {
	rv := (*Node)(NodeNew(nodeType))
	rv.setOutList(children)
	return rv
}

func (n *Node) setOutList(children []Nod) {
	for i := 0; i < len(children); i++ {
		child := children[i]
		n.setChild(NTR_LIST_0+i, child)
	}
}

func (n *Node) setChild(edgeType int, child Nod) {
	// for now assume child doesn't already exist, so skip check
	newEdge := &Edge{
		EdgeType: edgeType,
		In:       n,
		Out:      child,
	}
	n.Out[edgeType] = newEdge
	child.In = append(child.In, newEdge)
}
