package main

import (
	"fmt"
	"pocket-lang/frontend/pocket/common"
	"testing"
)

func TestMype(t *testing.T) {
	meFull := common.MypeExplicitNewFull()
	meSingle := common.MypeExplicitNewSingle(common.TY_INT)
	meXsect := meFull.Intersection(meSingle)
	meXsect2 := meSingle.Intersection(meFull)
	fmt.Println("test mype", meXsect, meXsect2)
}
