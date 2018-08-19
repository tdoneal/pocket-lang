package xform

import (
	"fmt"
	"pocket-lang/parse"
)

type Xformer struct {
}

func Xform(root parse.Nod) parse.Nod {
	fmt.Println("xforming node")
	return root
}
