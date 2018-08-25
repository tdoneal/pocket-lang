package parse

// TODO: write pretty printer for this int-based graph
//  will require way of going from int->human readable edge names

type NodeType struct {
	id   int
	name string
}

const (
	NT_IMPERATIVE           = 10
	NT_VARINIT              = 20
	NTR_VARINIT_NAME        = 21
	NTR_VARINIT_VALUE       = 22
	NT_VARDEF               = 25
	NTR_VARDEF_NAME         = 26
	NT_RECEIVERCALL         = 30
	NTR_RECEIVERCALL_NAME   = 31
	NTR_RECEIVERCALL_VALUE  = 32
	NT_IDENTIFIER           = 40
	NT_VARTABLE             = 50
	NTR_VARTABLE            = 51
	NT_TOPLEVEL             = 60
	NTR_TOPLEVEL_IMPERATIVE = 61
	NT_FUNCDEF              = 65
	NTR_FUNCDEF_NAME        = 66
	NTR_FUNCDEF_CODE        = 68
	NT_ADDOP                = 100
	NT_INLINEOPSTREAM       = 150
	NTR_LIT_VALUE           = 201
	NT_LIT_INT              = 210
	NT_VAR_GETTER           = 250
	NTR_VAR_GETTER_NAME     = 251
	NTR_LIST_0              = 100000 // 100000<->0th element, 100001<->1st element, etc
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
