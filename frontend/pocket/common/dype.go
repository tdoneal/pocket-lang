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

func DypeXSect(a Nod, b Nod) Nod {
	// returns the intersection of two nodes
	// does some very basic simplifications, if can't simplify then
	// just returns a representation of the intersection
	if a.NodeType == DYPE_EMPTY || b.NodeType == DYPE_EMPTY {
		return NodNew(DYPE_EMPTY)
	}
	if a.NodeType == DYPE_ALL {
		return b
	}
	if b.NodeType == DYPE_ALL {
		return a
	}
	if DypeDeepForwardsEqual(a, b) {
		return a
	}
	rv := NodNew(DYPE_XSECT)
	NodSetOutList(rv, []Nod{a, b})
	return rv
}

func DypeUnion(a Nod, b Nod) Nod {
	if a.NodeType == DYPE_ALL || b.NodeType == DYPE_ALL {
		return NodNew(DYPE_ALL)
	}
	if a.NodeType == DYPE_EMPTY {
		return b
	}
	if b.NodeType == DYPE_EMPTY {
		return a
	}
	if DypeDeepForwardsEqual(a, b) {
		return a
	}
	rv := NodNew(DYPE_UNION)
	NodSetOutList(rv, []Nod{a, b})
	return rv
}

func DypeIsSubset(a Nod, b Nod) bool {
	// assumes no nested operators

	if a.NodeType == DYPE_ALL {
		return true
	}
	if b.NodeType == DYPE_EMPTY {
		return true
	}
	if b.NodeType == DYPE_ALL {
		return a.NodeType == DYPE_ALL
	}
	if a.NodeType == DYPE_EMPTY {
		return b.NodeType == DYPE_EMPTY
	}

	if !DypeIsMeta(a.NodeType) && !DypeIsMeta(b.NodeType) {
		return DypeDeepForwardsEqual(a, b)
	}

	if DypeDeepForwardsEqual(a, b) {
		return true
	}

	asimp := DypeSimplifyShallow(a)
	bsimp := DypeSimplifyShallow(b)

	if asimp.NodeType == DYPE_UNION && !DypeIsMeta(bsimp.NodeType) {
		DypeCheckNoNestedOps(asimp)
		rv := DypeListContains(NodGetChildList(asimp), bsimp)
		return rv
	}
	if bsimp.NodeType == DYPE_UNION && !DypeIsMeta(asimp.NodeType) {
		return false
	}
	if asimp.NodeType == DYPE_UNION && bsimp.NodeType == DYPE_UNION {
		return DypeIsSubsetUnionUnion(asimp, bsimp)
	}

	fmt.Println("Couldn't determine subsetness: container (simplified):",
		PrettyPrint(asimp), "\nsub:", PrettyPrint(bsimp))
	panic("Couldn't determine subsetness")
}

func DypeListContains(nods []Nod, e Nod) bool {
	for _, cnod := range nods {
		if DypeDeepForwardsEqual(cnod, e) {
			return true
		}
	}
	return false
}

func DypeIsSubsetUnionUnion(ua Nod, ub Nod) bool {

	DypeCheckNoNestedOps(ua)
	DypeCheckNoNestedOps(ub)

	// iterate through other's elements
	myNods := NodGetChildList(ua)
	otherNods := NodGetChildList(ub)

	for _, otherNod := range otherNods {
		contained := DypeListContains(myNods, otherNod)
		if !contained {
			return false
		}
	}
	return true
}

func DypeCheckNoNestedOps(n Nod) {
	if !DypeIsOperator(n.NodeType) {
		return
	}
	myNods := NodGetChildList(n)
	for _, nod := range myNods {
		if DypeIsOperator(nod.NodeType) {
			panic("no nested ops allowed")
		}
	}
}

func DypeWouldChangeUnion(a Nod, b Nod) bool {
	return !DypeIsSubset(a, b)
}

func DypeWouldChangeXSect(a Nod, b Nod) bool {
	return !DypeIsSubset(b, a)
}

func DypeSimplifyShallow(n Nod) Nod {
	// performs quick simplifications (non-recursive),
	// returns original Nod if no simplifications made (Nod-Idem)
	n = DypeCollapseShallow(n)
	n = DypeRemoveMonoArgs(n)
	n = DypeDeassociate(n)
	n = DypeDeduplicate(n)
	return n
}

func DypeCollapseShallow(n Nod) Nod {
	// performs simplifications of the form:
	// Union[m, n, empty] -> Union[m, n]
	// Union[m, n, all] -> all
	// Union[m, empty] -> Union[m]
	if n.NodeType == DYPE_UNION {
		args := NodGetChildList(n)

		// first check if anything can be done
		// avoid allocating new output array if nothing can be done
		anyEmpty := false
		for _, arg := range args {
			if arg.NodeType == DYPE_ALL {
				return NodNew(DYPE_ALL)
			} else if arg.NodeType == DYPE_EMPTY {
				anyEmpty = true
				break
			}
		}
		if !anyEmpty {
			// nothing else to do
			return n
		}

		nargs := []Nod{}
		// filter, removing all EMPTY nodes
		for _, arg := range args {
			if arg.NodeType == DYPE_EMPTY {
				// purposeful pass
			} else {
				nargs = append(nargs, arg)
			}
		}
		return NodNewChildList(DYPE_UNION, nargs)
	}
	return n
}

func DypeRemoveMonoArgs(n Nod) Nod {
	// Union[] -> EMPTY
	// Union[n] -> n
	if DypeIsOperator(n.NodeType) {
		args := NodGetChildList(n)
		if len(args) == 0 {
			return NodNew(DYPE_EMPTY)
		} else if len(args) == 1 {
			return args[0]
		}
	}
	return n
}

func DypeDeassociate(n Nod) Nod {
	// flattens nested similar operators in arguments
	if DypeIsOperator(n.NodeType) {
		args := NodGetChildList(n)
		couldSimp := false
		for _, arg := range args {
			if arg.NodeType == n.NodeType {
				couldSimp = true
				break
			}
		}
		if !couldSimp {
			return n
		}
		newArgs := []Nod{}
		for _, arg := range args {
			if arg.NodeType == n.NodeType {
				newArgs = append(newArgs, NodGetChildList(arg)...)
			} else {
				newArgs = append(newArgs, arg)
			}
		}
		newNod := NodNewChildList(n.NodeType, newArgs)
		return DypeDeassociate(newNod)
	}
	return n
}

func DypeDeduplicate(n Nod) Nod {
	// removes duplicate args in operators
	// works best if the arg list has already been flattened (using DypeDeassociate or the like)
	if DypeIsOperator(n.NodeType) {
		args := NodGetChildList(n)
		var toRm []bool
		for i := 0; i < len(args); i++ {
			a0 := args[i]
			for j := i + 1; j < len(args); j++ {
				a1 := args[j]
				if DypeDeepForwardsEqual(a0, a1) {
					if toRm == nil {
						toRm = make([]bool, len(args))
					}
					toRm[j] = true
				}
			}
		}
		if toRm == nil {
			return n
		}
		newArgs := []Nod{}
		for i := 0; i < len(args); i++ {
			if !toRm[i] {
				newArgs = append(newArgs, args[i])
			}
		}
		newNod := NodNewChildList(n.NodeType, newArgs)
		return newNod
	}
	return n
}

func DypeSimplifyDeep(n Nod) Nod {
	// performs recursive simplifications. (Nod-Idem)
	if !DypeIsMeta(n.NodeType) {
		return n
	}
	if DypeIsOperator(n.NodeType) {
		newN := DypeSimplifyChildren(n)
		return DypeSimplifyShallowComplex(newN)
	}
	return n
}

func DypeSimplifyShallowComplex(n Nod) Nod {
	if n.NodeType == DYPE_XSECT {
		return DypeSimplifyShallowComplex(DypeEvaluateXSect(n))
	}
	return DypeSimplifyShallow(n)
}

func DypeEvaluateXSect(n Nod) Nod {
	// tries performing shallow progress on xsects (aiming to be union, prim only)
	args := NodGetChildList(n)

	// TODO:!
	filter := NodNew(DYPE_ALL)
	// gather the list of "all elements"
	// an "element" can be either a standalone non-meta child of xsect, or an element of
	// a direct union child
	for _, arg := range args {
		filter = DypeEvaluateXSectBinary(filter, arg)
	}

	return filter
}

func DypeEvaluateXSectBinary(a Nod, b Nod) Nod {
	// evaluates the intersection of two non-xsect types
	// return value is not an XSECT node, rather actual progress is made
	if a.NodeType == DYPE_ALL {
		return b
	}
	if b.NodeType == DYPE_ALL {
		return a
	}
	if !DypeIsMeta(a.NodeType) && !DypeIsMeta(b.NodeType) {
		if DypeDeepForwardsEqual(a, b) {
			return a
		}
		return NodNew(DYPE_EMPTY)
	}
	if a.NodeType == DYPE_UNION && !DypeIsMeta(b.NodeType) {
		return DypeEvaluateXSectBinaryUnionNonUnion(a, b)
	}
	if b.NodeType == DYPE_UNION && !DypeIsMeta(a.NodeType) {
		return DypeEvaluateXSectBinaryUnionNonUnion(b, a)
	}
	if a.NodeType == DYPE_UNION && b.NodeType == DYPE_UNION {
		return DypeEvaluateXSectBinaryUnionUnion(a, b)
	}
	panic("invalid preconditions")
}

func DypeEvaluateXSectBinaryUnionUnion(a Nod, b Nod) Nod {
	// find the things each union has in common
	aArgs := NodGetChildList(a)
	bArgs := NodGetChildList(b)

	commonArgs := []Nod{}
	for _, aArg := range aArgs {
		if DypeListContains(bArgs, aArg) {
			commonArgs = append(commonArgs, aArg)
		}
	}

	if len(commonArgs) == len(aArgs) {
		// nothing changed, avoid new object creation
		return a
	}
	if len(commonArgs) == 1 {
		return commonArgs[0]
	}
	return NodNewChildList(DYPE_UNION, commonArgs)
}

func DypeEvaluateXSectBinaryUnionNonUnion(union Nod, nonunion Nod) Nod {
	unionArgs := NodGetChildList(union)
	if DypeListContains(unionArgs, nonunion) {
		return nonunion
	}
	return NodNew(DYPE_EMPTY)
}

func DypeSimplifyChildren(n Nod) Nod {
	args := NodGetChildList(n)
	simpArgs := []Nod{}
	allSame := true
	for _, arg := range args {
		argSimped := DypeSimplifyDeep(arg)
		if argSimped != arg {
			allSame = false
		}
		simpArgs = append(simpArgs, argSimped)
	}
	if allSame {
		return n
	}
	newN := NodNewChildList(n.NodeType, simpArgs)
	return newN
}

func DypeIsOperator(nt int) bool {
	return nt == DYPE_UNION || nt == DYPE_XSECT
}

func DypeIsMeta(nt int) bool {
	return nt == DYPE_ALL || nt == DYPE_EMPTY ||
		nt == DYPE_UNION || nt == DYPE_XSECT
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
	if n0nod, ok0 := n0.Data.(Nod); ok0 {
		if n1nod, ok1 := n1.Data.(Nod); ok1 {
			return DypeDeepForwardsEqual(n0nod, n1nod)
		}
		return false
	}
	panic("undetermined")
}
