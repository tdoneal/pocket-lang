package goback

import (
	. "pocket-lang/parse"
	"pocket-lang/xform"
)

const (
	PN_GOIMPORTS = 200000
)

type Preparer struct {
	xformer *xform.Xformer
}

func (p *Preparer) Prepare(code Nod) {
	panic("preparing code")

}
