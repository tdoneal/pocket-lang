package main

import (
	"fmt"
	"pocket-lang/frontend/pocket"
	"testing"
)

func TestMypeArged(t *testing.T) {
	meFull := pocket.MypeArgedNewFull()
	meSingle := pocket.MypeArgedNewSingleBase(pocket.TY_INT)
	// meSingle2 := pocket.MypeArgedNewSingleBase(pocket.TY_FLOAT)

	fmt.Println("test mype", pocket.PrettyPrint(meFull.Node))
	fmt.Println("test mype", pocket.PrettyPrint(meSingle.Node))
	// fmt.Println("test mype", pocket.PrettyPrint((meSingle.Union(meFull)).Node))
	// fmt.Println("test mype", pocket.PrettyPrint((meFull.Union(meSingle)).Node))
	// fmt.Println("test mype", pocket.PrettyPrint((meSingle2.Union(meSingle)).Node))

}
