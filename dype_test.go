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
	testSymm(DypeXSect, NodNew(DYPE_ALL), NodNewData(NT_TYPEBASE, TY_INT),
		NodNewData(NT_TYPEBASE, TY_INT))
	testSymm(DypeXSect, NodNew(DYPE_EMPTY), NodNewData(NT_TYPEBASE, TY_INT),
		NodNew(DYPE_EMPTY))
	testSymm(DypeXSect, NodNew(DYPE_ALL), NodNew(DYPE_ALL),
		NodNew(DYPE_ALL))
	testSymm(DypeXSect, NodNew(DYPE_ALL), NodNew(DYPE_EMPTY),
		NodNew(DYPE_EMPTY))
	testSymm(DypeXSect, NodNew(DYPE_EMPTY), NodNew(DYPE_EMPTY),
		NodNew(DYPE_EMPTY))
	testSymm(DypeXSect, NodNewData(NT_TYPEBASE, TY_INT), NodNewData(NT_TYPEBASE, TY_INT),
		NodNewData(NT_TYPEBASE, TY_INT))
}

func testUnionAll() {
	testSymm(DypeUnion, NodNew(DYPE_ALL), NodNewData(NT_TYPEBASE, TY_INT),
		NodNew(DYPE_ALL))
	testSymm(DypeUnion, NodNew(DYPE_EMPTY), NodNewData(NT_TYPEBASE, TY_INT),
		NodNewData(NT_TYPEBASE, TY_INT))
	testSymm(DypeUnion, NodNew(DYPE_ALL), NodNew(DYPE_ALL),
		NodNew(DYPE_ALL))
	testSymm(DypeUnion, NodNew(DYPE_ALL), NodNew(DYPE_EMPTY),
		NodNew(DYPE_ALL))
	testSymm(DypeUnion, NodNew(DYPE_EMPTY), NodNew(DYPE_EMPTY),
		NodNew(DYPE_EMPTY))
	testSymm(DypeUnion, NodNewData(NT_TYPEBASE, TY_INT), NodNewData(NT_TYPEBASE, TY_INT),
		NodNewData(NT_TYPEBASE, TY_INT))
}

func MakeInt() Nod {
	return NodNewData(NT_TYPEBASE, TY_INT)
}

func MakeFloat() Nod {
	return NodNewData(NT_TYPEBASE, TY_FLOAT)
}

func MakeBool() Nod {
	return NodNewData(NT_TYPEBASE, TY_BOOL)
}

func MakeEmpty() Nod {
	return NodNew(DYPE_EMPTY)
}

func MakeFull() Nod {
	return NodNew(DYPE_ALL)
}

func MakeUnion(nods ...Nod) Nod {
	return NodNewChildList(DYPE_UNION, nods)
}

func MakeXSect(nods ...Nod) Nod {
	return NodNewChildList(DYPE_XSECT, nods)
}

func testSimpShal(input Nod, expected Nod) {
	got := DypeSimplifyShallow(input)
	if !DypeDeepForwardsEqual(got, expected) {
		panic("failed")
	}
}

func testSimpDeep(input Nod, expected Nod) {
	got := DypeSimplifyDeep(input)
	if !DypeDeepForwardsEqual(got, expected) {
		panic("failed")
	}
}

func testSimplifyCases() {
	testSimplifyCasesShal()
	testSimplifyCasesDeep()
}

func testSimplifyCasesShal() {
	testSimpShal(NodNewChildList(DYPE_UNION, []Nod{MakeInt()}), MakeInt())
	testSimpShal(NodNewChildList(DYPE_XSECT, []Nod{MakeInt()}), MakeInt())
	testSimpShal(NodNewChildList(DYPE_UNION, []Nod{}), MakeEmpty())

	// 11/23/2018: cases where unions should be able to "get rid of baggage"
	testSimpShal(NodNewChildList(DYPE_UNION, []Nod{MakeInt(), MakeEmpty()}), MakeInt())
	testSimpShal(NodNewChildList(DYPE_UNION, []Nod{MakeEmpty()}), MakeEmpty())
	testSimpShal(NodNewChildList(DYPE_UNION, []Nod{MakeInt(), MakeFull()}), MakeFull())

	// Simp(Union(Union(int, float),int)) -> Union(int, float)
	u := DypeUnion(NodNewData(NT_TYPEBASE, TY_INT), NodNewData(NT_TYPEBASE, TY_FLOAT))
	fmt.Println("u", PrettyPrint(u))
	u2 := DypeUnion(u, NodNewData(NT_TYPEBASE, TY_INT))
	fmt.Println("u2", PrettyPrint(u2))
	testSimpShal(u2, NodNewChildList(DYPE_UNION,
		[]Nod{NodNewData(NT_TYPEBASE, TY_INT), NodNewData(NT_TYPEBASE, TY_FLOAT)}))

}

func testSimplifyCasesDeep() {
	cmplx := MakeUnion(MakeUnion())
	testSimpDeep(cmplx, MakeEmpty())

	xsect := MakeXSect(MakeInt(), MakeFloat())
	testSimpDeep(xsect, MakeEmpty())

	testSimpDeep(MakeXSect(MakeInt(), MakeInt()), MakeInt())

	cmplxXSect := MakeXSect(MakeUnion(MakeInt(), MakeFloat()), MakeFloat())
	testSimpDeep(cmplxXSect, MakeFloat())

	cmplxXSect = MakeXSect(MakeFloat(), MakeUnion(MakeInt(), MakeFloat()))
	testSimpDeep(cmplxXSect, MakeFloat())

	cmplxXSect = MakeXSect(MakeUnion(MakeInt(), MakeFloat()), MakeUnion(MakeInt(), MakeFloat()))
	testSimpDeep(cmplxXSect, MakeUnion(MakeInt(), MakeFloat()))

	cmplxXSect = MakeXSect(MakeUnion(MakeInt(), MakeFloat()), MakeUnion(MakeFloat(), MakeInt()))
	testSimpDeep(cmplxXSect, MakeUnion(MakeInt(), MakeFloat()))

	cmplxXSect = MakeXSect(MakeUnion(MakeFloat(), MakeBool()), MakeUnion(MakeBool(), MakeInt()))
	testSimpDeep(cmplxXSect, MakeBool())

	// complex structure comparison in union
	arg0 := NodNewChildList(DYPE_UNION, []Nod{
		NodNewChild(NT_TYPECALL, NTR_RECEIVERCALL_BASE, MakeInt()),
		MakeInt(),
	})
	arg1 := NodNewChild(NT_TYPECALL, NTR_RECEIVERCALL_BASE, MakeInt())
	cmplxXSect = MakeXSect(arg0, arg1)
	testSimpDeep(cmplxXSect, NodNewChild(NT_TYPECALL, NTR_RECEIVERCALL_BASE, MakeInt()))

}

func exploreUnion() {

	u := DypeUnion(NodNewData(NT_TYPEBASE, TY_INT), NodNewData(NT_TYPEBASE, TY_FLOAT))
	fmt.Println("u", PrettyPrint(u))
	u2 := DypeUnion(u, NodNewData(NT_TYPEBASE, TY_INT))
	fmt.Println("u2", PrettyPrint(u2))
	u3 := DypeSimplifyShallow(u2)
	fmt.Println("u3", PrettyPrint(u3))
}

func testSubset(container Nod, sub Nod, expected bool) {
	got := DypeIsSubset(container, sub)
	fmt.Println("container", PrettyPrint(container), "sub", PrettyPrint(sub), "got", got)
	if got != expected {
		panic("failed")
	}
}

func testSubsetCases() {
	testSubset(MakeFull(), MakeFull(), true)
	testSubset(MakeFull(), MakeEmpty(), true)
	testSubset(MakeEmpty(), MakeFull(), false)
	testSubset(MakeEmpty(), MakeEmpty(), true)

	testSubset(MakeFull(), MakeInt(), true)
	testSubset(MakeEmpty(), MakeInt(), false)
	testSubset(MakeInt(), MakeFull(), false)
	testSubset(MakeInt(), MakeEmpty(), true)

	testSubset(MakeInt(), MakeInt(), true)

	u0 := MakeUnion(MakeInt(), MakeFloat())
	testSubset(u0, MakeInt(), true)
	testSubset(u0, MakeBool(), false)

	u1 := MakeUnion(MakeFloat(), MakeInt())
	testSubset(u0, u1, true)

	u2 := MakeUnion(MakeBool(), MakeFloat())
	testSubset(u0, u2, false)

	ubig := MakeUnion(MakeInt(), MakeFloat(), MakeBool())
	testSubset(ubig, u2, true)
	testSubset(ubig, u1, true)
	testSubset(u1, ubig, false)
	testSubset(u2, ubig, false)

	testSubset(MakeInt(), ubig, false)

}

func exploreIsSubset() {
	fmt.Println(DypeIsSubset(MakeFull(), MakeEmpty()))
}

func TestDype(t *testing.T) {
	testSubsetCases()
	testSimplifyCases()
	testXSectAll()
	testUnionAll()

}
