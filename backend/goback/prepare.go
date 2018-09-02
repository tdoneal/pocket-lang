package goback

import (
	. "pocket-lang/frontend/pocket"
	. "pocket-lang/parse"
	"pocket-lang/xform"
)

const (
	PNR_GOIMPORTS = 200000
)

type Preparer struct {
	*xform.Xformer
}

func (p *Preparer) Prepare(code Nod) {
	p.Root = code
	p.checkForPrintStatements()
}

func (p *Preparer) checkForPrintStatements() {
	rcs := p.SearchRoot(func(n Nod) bool {
		return isReceiverCallType(n.NodeType) &&
			NodGetChild(n, NTR_RECEIVERCALL_NAME).Data.(string) == "print"
	})

	if len(rcs) > 0 {
		NodSetChild(p.Root, PNR_GOIMPORTS, NodNewData(NT_IDENTIFIER, "fmt"))
	}
}
