package common

import (
	"fmt"
	. "pocket-lang/parse"
)

// Represents an algebra over the universe of dypes, a minimal extension
// to "regular nodes" that includes unions and intersections.
// The concrete representation of a dype is Nod.

const (
	DYPE_ALL   = 382322 + iota // represents the enumeration of all nodes
	DYPE_EMPTY                 // represents no nod, or void
	DYPE_UNION                 // union of its children
	DYPE_XSECT                 // intersection of its children
)

func DypeIntersectProg(a Nod, b Nod) Nod {
	if a.NodeType == DYPE_EMPTY || b.NodeType == DYPE_EMPTY {
		return NodNew(DYPE_EMPTY)
	}
	if a.NodeType == DYPE_ALL {
		return b
	}
	if b.NodeType == DYPE_ALL {
		return a
	}
	fmt.Println("couldn't make progress on xsect: args:", PrettyPrint(a), PrettyPrint(b))
	panic("couldn't make progress")
}

func DypeUnionProg(a Nod, b Nod) Nod {
	if a.NodeType == DYPE_ALL || b.NodeType == DYPE_ALL {
		return NodNew(DYPE_ALL)
	}
	if a.NodeType == DYPE_EMPTY {
		return b
	}
	if b.NodeType == DYPE_EMPTY {
		return a
	}
	fmt.Println("couldn't make progress on union: args:", PrettyPrint(a), PrettyPrint(b))
	panic("couldn't make progress")
}

func DypeDeepForwardsEqual(n0 Nod, n1 Nod) bool {
	if n0 == n1 {
		return true
	}
	if n0.NodeType != n1.NodeType {
		return false
	}
	if !NodDatasEqual(n0, n1) {
		return false
	}
	for _, childEdge := range n0.Out {
		myChild := childEdge.Out
		otherChild := NodGetChildOrNil(n1, childEdge.EdgeType)
		if otherChild == nil {
			return false
		}
		childsEq := DypeDeepForwardsEqual(myChild, otherChild)
		if !childsEq {
			return false
		}
	}
	return true
}

func NodDatasEqual(n0 Nod, n1 Nod) bool {
	if n0.Data == nil && n1.Data == nil {
		return true
	}
	if n0str, ok0 := n0.Data.(string); ok0 {
		if n1str, ok1 := n1.Data.(string); ok1 {
			return n0str == n1str
		}
		return false
	}
	if n0int, ok0 := n0.Data.(int); ok0 {
		if n1int, ok1 := n1.Data.(int); ok1 {
			return n0int == n1int
		}
		return false
	}
	panic("undetermined")
}
