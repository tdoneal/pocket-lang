package pocket

import (
	"fmt"
	. "pocket-lang/parse"
)

// Represents an algebra over the universe of sets of arged types

const (
	MATYPE_ALL   = 300000 + iota // represents the enumeration of all arged types
	MATYPE_UNION                 // union of its children.  MUST have at least 2 non-empty, non-full children
	MATYPE_SINGLE_BASE
	MATYPE_EMPTY
	MATYPE_SINGLE_ARGED
	MATYPER_BASE
	MATYPER_ARG
)

// Represents a set of arged types
type MypeArged struct {
	*Node // use the same graph structure as used by the parser
}

var _ Mype = &MypeArged{}

func MypeArgedNewSingleBase(t int) *MypeArged {
	return &MypeArged{Node: NodNewData(MATYPE_SINGLE_BASE, t)}
}

func MypeArgedNewSingleArged(baseT int, arg Mype) *MypeArged {
	rvNod := NodNew(MATYPE_SINGLE_ARGED)
	baseMype := MypeArgedNewSingleBase(baseT)
	argNod := arg.(*MypeArged).Node
	NodSetChild(rvNod, MATYPER_BASE, baseMype.Node)
	NodSetChild(rvNod, MATYPER_ARG, argNod)
	return &MypeArged{Node: rvNod}
}

func MypeArgedNewFull() *MypeArged {
	return &MypeArged{Node: NodNew(MATYPE_ALL)}
}

func MypeArgedNewEmpty() *MypeArged {
	return &MypeArged{Node: NodNew(MATYPE_EMPTY)}
}

func (ma *MypeArged) Union(other Mype) Mype {
	if otherMa, ok := other.(*MypeArged); ok {
		// check for degenerate case
		if ma.NodeType == MATYPE_ALL || otherMa.NodeType == MATYPE_ALL {
			return MypeArgedNewFull()
		}
		if ma.NodeType == MATYPE_EMPTY {
			return otherMa
		}
		if otherMa.NodeType == MATYPE_EMPTY {
			return ma
		}
		if ma.NodeType == MATYPE_SINGLE_BASE && otherMa.NodeType == MATYPE_SINGLE_BASE {
			// optimization: if both singles are the same, just return that
			if ma.Data.(int) == otherMa.Data.(int) {
				return ma
			}
			return &MypeArged{NodNewChildList(MATYPE_UNION, []Nod{ma.Node, otherMa.Node})}
		}
		// from here on refer to big and small rather than ma and otherMa
		big := ma
		small := otherMa
		if (big.NodeType == MATYPE_SINGLE_BASE && small.NodeType == MATYPE_UNION) ||
			(big.NodeType == MATYPE_SINGLE_BASE && small.NodeType == MATYPE_SINGLE_ARGED) {
			small, big = big, small
		}
		if big.NodeType == MATYPE_UNION && small.NodeType == MATYPE_SINGLE_BASE {
			newNods := append(NodGetChildList(big.Node), small.Node)
			return &MypeArged{NodNewChildList(MATYPE_UNION, newNods)}
		}
		if big.NodeType == MATYPE_SINGLE_ARGED && small.NodeType == MATYPE_SINGLE_BASE {
			newNods := []Nod{
				big.Node,
				small.Node,
			}
			return &MypeArged{NodNewChildList(MATYPE_UNION, newNods)}
		}
		fmt.Println("couldnt union", PrettyPrintNodes([]Nod{ma.Node, otherMa.Node}))
		panic("couldnt union it")
	} else {
		panic("must union with mypearged")
	}
}
func (ma *MypeArged) Intersection(other Mype) Mype {
	if otherMa, ok := other.(*MypeArged); ok {
		// check for degenerate case
		if ma.NodeType == MATYPE_EMPTY || otherMa.NodeType == MATYPE_EMPTY {
			return MypeArgedNewEmpty()
		}
		if ma.NodeType == MATYPE_ALL {
			return otherMa
		}
		if otherMa.NodeType == MATYPE_ALL {
			return ma
		}
		if ma.NodeType == MATYPE_SINGLE_BASE && otherMa.NodeType == MATYPE_SINGLE_BASE {
			mySingle := ma.Data.(int)
			otherSingle := otherMa.Data.(int)
			if mySingle == otherSingle {
				return ma
			} else {
				return MypeArgedNewEmpty()
			}
		}
		// from here on refer to big and small rather than ma and otherMa
		big := ma
		small := otherMa
		if (big.NodeType == MATYPE_SINGLE_BASE && small.NodeType == MATYPE_UNION) ||
			(big.NodeType == MATYPE_SINGLE_BASE && small.NodeType == MATYPE_SINGLE_ARGED) {
			small, big = big, small
		}
		if big.NodeType == MATYPE_UNION && small.NodeType == MATYPE_SINGLE_BASE {
			newNods := append(NodGetChildList(big.Node), small.Node)
			return &MypeArged{NodNewChildList(MATYPE_UNION, newNods)}
		}
		if big.NodeType == MATYPE_SINGLE_ARGED && small.NodeType == MATYPE_SINGLE_BASE {
			// these are incompatible
			return MypeArgedNewEmpty()
		}
		if big.NodeType == MATYPE_UNION && small.NodeType == MATYPE_UNION {
			return big.intersectionUU(small)
		}
		fmt.Println("couldnt intersect", PrettyPrintNodes([]Nod{ma.Node, otherMa.Node}))
		panic("couldnt intersect it")
	} else {
		panic("must intersect with mypearged")
	}
}

func (ma *MypeArged) intersectionUU(other *MypeArged) *MypeArged {
	// returns intersection of two unions
	ma.CheckNoNestedUnions()
	other.CheckNoNestedUnions()

	rvNods := []Nod{}

	myNods := NodGetChildList(ma.Node)
	otherNods := NodGetChildList(other.Node)

	for _, myNod := range myNods {
		if MANodListContains(otherNods, myNod) {
			rvNods = append(rvNods, myNod)
		}
	}

	return &MypeArged{Node: NodNewChildList(MATYPE_UNION, rvNods)}
}

func (ma *MypeArged) IsPlural() bool { panic("unimplemented") }
func (ma *MypeArged) IsSingle() bool {
	if ma.NodeType == MATYPE_SINGLE_ARGED || ma.NodeType == MATYPE_SINGLE_BASE {
		return true
	}
	if ma.NodeType == MATYPE_EMPTY || ma.NodeType == MATYPE_ALL {
		return false
	}
	fmt.Println("Failed: IsSingle", PrettyPrint(ma.Node))
	panic("couldn't figure out")
}
func (ma *MypeArged) IsSingleType(int) bool { panic("unimplemented") }
func (ma *MypeArged) GetSingleType() int    { panic("unimplemented") }
func (ma *MypeArged) IsEmpty() bool {
	if ma.NodeType == MATYPE_EMPTY {
		return true
	}
	if ma.NodeType == MATYPE_SINGLE_BASE || ma.NodeType == MATYPE_ALL ||
		ma.NodeType == MATYPE_SINGLE_ARGED {
		return false
	}
	if ma.NodeType == MATYPE_UNION {
		return false
	}
	fmt.Println("Failed: IsEmpty", PrettyPrint(ma.Node))
	panic("couldn't determine if empty")
}
func (ma *MypeArged) IsFull() bool {
	if ma.NodeType == MATYPE_ALL {
		return true
	}
	if ma.NodeType == MATYPE_EMPTY || ma.NodeType == MATYPE_SINGLE_BASE ||
		ma.NodeType == MATYPE_SINGLE_ARGED {
		return false
	}
	if ma.NodeType == MATYPE_UNION {
		// the only way this could ever be true is if all concrete base types were
		// explicitly enumerated with a type arg of all
		// for now, we'll just assume that isn't the case
		return false
	}
	fmt.Println("Failed: IsFull", PrettyPrint(ma.Node))
	panic("couldn't figure out if full")
}

func (ma *MypeArged) WouldChangeFromUnionWith(other Mype) bool {
	if otherMa, ok := other.(*MypeArged); ok {
		if ma.IsFull() {
			return false
		}
		if otherMa.IsEmpty() {
			return false
		}
		if ma.IsEmpty() && !otherMa.IsEmpty() {
			return true
		}
		if ma.NodeType == MATYPE_SINGLE_BASE && otherMa.NodeType == MATYPE_SINGLE_BASE {
			return ma.Data.(int) != otherMa.Data.(int)
		}
		if ma.NodeType == MATYPE_SINGLE_ARGED && otherMa.NodeType == MATYPE_SINGLE_ARGED {
			isEq := ma.ExactDeepEqual(otherMa)
			if isEq {
				return false
			}
		}
		if ma.NodeType == MATYPE_UNION && otherMa.NodeType == MATYPE_UNION {
			return ma.WouldChangeFromUnionWithUU(otherMa)
		}
		fmt.Println("Failed: WCFUW", PrettyPrintNodes([]Nod{ma.Node, otherMa.Node}))
		panic("couldnt figure it out")
	} else {
		panic("must union with mypearged (got " + fmt.Sprint(other))
	}
}

func (ma *MypeArged) CheckNoNestedUnions() {
	myNods := NodGetChildList(ma.Node)
	for _, nod := range myNods {
		if nod.NodeType == MATYPE_SINGLE_ARGED || nod.NodeType == MATYPE_SINGLE_BASE {
			// fine
		} else {
			panic("invalid union structure: nesting detected")
		}
	}
}

func (ma *MypeArged) WouldChangeFromUnionWithUU(other *MypeArged) bool {
	// iterate through other's elements
	myNods := NodGetChildList(ma.Node)
	otherNods := NodGetChildList(other.Node)

	ma.CheckNoNestedUnions()
	other.CheckNoNestedUnions()

	// assume no nested unions
	fmt.Println("otherNods", PrettyPrintNodes(otherNods))
	for _, otherNod := range otherNods {
		contained := false
		for _, myNod := range myNods {
			if MANodDeepEqual(myNod, otherNod) {
				contained = true
				break
			}
		}
		if !contained {
			return true
		}
	}
	return false
}

func (ma *MypeArged) WouldChangeFromIntersectionWith(other Mype) bool {
	if otherMa, ok := other.(*MypeArged); ok {
		if otherMa.IsFull() {
			return false
		}
		if ma.IsEmpty() {
			return false
		}
		if otherMa.IsEmpty() && !ma.IsEmpty() {
			return true
		}
		if ma.IsFull() && !otherMa.IsFull() {
			return true
		}
		if ma.NodeType == MATYPE_SINGLE_BASE && otherMa.NodeType == MATYPE_SINGLE_BASE {
			return ma.Data.(int) != otherMa.Data.(int)
		}
		if ma.NodeType == MATYPE_SINGLE_ARGED && otherMa.NodeType == MATYPE_SINGLE_ARGED {
			isEq := ma.ExactDeepEqual(otherMa)
			if isEq {
				return false
			}
		}
		if ma.NodeType == MATYPE_UNION && otherMa.NodeType == MATYPE_UNION {
			return ma.WouldChangeFromXSectWithUU(otherMa)
		}
		fmt.Println("Failed: WCFIW", PrettyPrintNodes([]Nod{ma.Node, otherMa.Node}))
		panic("couldnt figure it out")
	} else {
		panic("must union with mypearged (got " + fmt.Sprint(other))
	}
}

func (ma *MypeArged) WouldChangeFromXSectWithUU(other *MypeArged) bool {
	// iterate through other's elements
	// note: this may return incorrect results for nested unions

	ma.CheckNoNestedUnions()
	other.CheckNoNestedUnions()

	myNods := NodGetChildList(ma.Node)
	otherNods := NodGetChildList(other.Node)
	fmt.Println("otherNods", PrettyPrintNodes(otherNods))
	// for each other element, if it's missing from us, return true
	for _, otherNod := range otherNods {
		if !MANodListContains(myNods, otherNod) {
			return true
		}
	}
	return false
}

func MANodListContains(nods []Nod, e Nod) bool {
	for _, cnod := range nods {
		if MANodDeepEqual(cnod, e) {
			return true
		}
	}
	return false
}

func (ma *MypeArged) Subtract(Mype) Mype { panic("unimplemented") }
func (ma *MypeArged) Converse() Mype     { panic("unimplemented") }
func (ma *MypeArged) ContainsSingleType(ty int) bool {
	return MANodContainsSingleType(ma.Node, ty)
}

func MANodContainsSingleType(n Nod, ty int) bool {
	if n.NodeType == MATYPE_ALL {
		return true
	}
	if n.NodeType == MATYPE_EMPTY {
		return false
	}
	if n.NodeType == MATYPE_SINGLE_BASE {
		return n.Data.(int) == ty
	}
	if n.NodeType == MATYPE_SINGLE_ARGED {
		return NodGetChild(n, MATYPER_BASE).Data.(int) == ty
	}
	if n.NodeType == MATYPE_UNION {
		unionNods := NodGetChildList(n)
		for _, unod := range unionNods {
			if MANodContainsSingleType(unod, ty) {
				return true
			}
		}
		return false
	}
	fmt.Println("Failed: CST", PrettyPrint(n))
	panic("couldnt figure it out")
}

func (ma *MypeArged) ContainsAnyType([]int) bool { panic("unimplemented") }
func (ma *MypeArged) ToType() int                { panic("unimplemented") }
func (ma *MypeArged) ExactDeepEqual(other *MypeArged) bool {
	if ma.NodeType != other.NodeType {
		return false
	}
	for _, childEdge := range ma.Out {
		myChild := childEdge.Out
		otherChild := NodGetChildOrNil(other.Node, childEdge.EdgeType)
		if otherChild == nil {
			return false
		}
		childsEq := MANodDeepEqual(myChild, otherChild)
		if !childsEq {
			return false
		}
	}
	return true
}

func MANodDeepEqual(n0 Nod, n1 Nod) bool {
	if n0.NodeType != n1.NodeType {
		return false
	}
	for _, childEdge := range n0.Out {
		myChild := childEdge.Out
		otherChild := NodGetChildOrNil(n1, childEdge.EdgeType)
		if otherChild == nil {
			return false
		}
		childsEq := MANodDeepEqual(myChild, otherChild)
		if !childsEq {
			return false
		}
	}
	return true
}
