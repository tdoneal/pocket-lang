package main

import (
	"fmt"
	. "pocket-lang/frontend/pocket/common"
	. "pocket-lang/parse"

	"testing"
)

func testSymm(op func(Nod, Nod) Nod, a Nod, b Nod, expected Nod) {
	order0Result := op(a, b)
	fmt.Println("order0Result", PrettyPrint(order0Result), "expected", PrettyPrint(expected))
	if !DypeDeepForwardsEqual(order0Result, expected) {
		panic("failed")
	}
	order1Result := op(b, a)
	fmt.Println("order1Result", PrettyPrint(order1Result))
	if !DypeDeepForwardsEqual(order1Result, expected) {
		panic("failed")
	}
}

func testXSectAll() {
	testSymm(DypeIntersectProg, NodNew(DYPE_ALL), NodNewData(NT_TYPEBASE, TY_INT),
		NodNewData(NT_TYPEBASE, TY_INT))
	testSymm(DypeIntersectProg, NodNew(DYPE_EMPTY), NodNewData(NT_TYPEBASE, TY_INT),
		NodNew(DYPE_EMPTY))
	testSymm(DypeIntersectProg, NodNew(DYPE_ALL), NodNew(DYPE_ALL),
		NodNew(DYPE_ALL))
	testSymm(DypeIntersectProg, NodNew(DYPE_ALL), NodNew(DYPE_EMPTY),
		NodNew(DYPE_EMPTY))
	testSymm(DypeIntersectProg, NodNew(DYPE_EMPTY), NodNew(DYPE_EMPTY),
		NodNew(DYPE_EMPTY))
}

func testUnionAll() {
	testSymm(DypeUnionProg, NodNew(DYPE_ALL), NodNewData(NT_TYPEBASE, TY_INT),
		NodNew(DYPE_ALL))
	testSymm(DypeUnionProg, NodNew(DYPE_EMPTY), NodNewData(NT_TYPEBASE, TY_INT),
		NodNewData(NT_TYPEBASE, TY_INT))
	testSymm(DypeUnionProg, NodNew(DYPE_ALL), NodNew(DYPE_ALL),
		NodNew(DYPE_ALL))
	testSymm(DypeUnionProg, NodNew(DYPE_ALL), NodNew(DYPE_EMPTY),
		NodNew(DYPE_ALL))
	testSymm(DypeUnionProg, NodNew(DYPE_EMPTY), NodNew(DYPE_EMPTY),
		NodNew(DYPE_EMPTY))
}

func TestDype(t *testing.T) {
	testXSectAll()
	testUnionAll()

	// meFull := pocket.MypeArgedNewFull()
	// meSingle := pocket.MypeArgedNewSingleBase(pocket.TY_INT)
	// // meSingle2 := pocket.MypeArgedNewSingleBase(pocket.TY_FLOAT)

	// fmt.Println("test mype", pocket.PrettyPrint(meFull.Node))
	// fmt.Println("test mype", pocket.PrettyPrint(meSingle.Node))
	// fmt.Println("test mype", pocket.PrettyPrint((meSingle.Union(meFull)).Node))
	// fmt.Println("test mype", pocket.PrettyPrint((meFull.Union(meSingle)).Node))
	// fmt.Println("test mype", pocket.PrettyPrint((meSingle2.Union(meSingle)).Node))

}
