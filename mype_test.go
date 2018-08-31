package main

import (
	"fmt"
	"pocket-lang/frontend/pocket"
	"testing"
)

func TestMype(t *testing.T) {
	meFull := pocket.MypeExplicitNewFull()
	meSingle := pocket.MypeExplicitNewSingle(pocket.TY_INT)
	meXsect := meFull.Intersection(meSingle)
	meXsect2 := meSingle.Intersection(meFull)
	fmt.Println("test mype", meXsect, meXsect2)
}
