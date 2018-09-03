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
	p.createExplicitIndexors()
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

func (p *Preparer) createExplicitIndexors() {
	listCalls := p.SearchRoot(func(n Nod) bool {
		if isReceiverCallType(n.NodeType) {
			if funcDef := NodGetChildOrNil(n, NTR_FUNCDEF); funcDef != nil {
				if funcDef.NodeType == NT_VARDEF {
					varType := NodGetChild(funcDef, NTR_TYPE).Data.(int)
					if varType == TY_LIST || varType == TY_MAP {
						return true
					}
				}
			}
		}
		return false
	})
	for _, listCall := range listCalls {
		varName := NodGetChild(listCall, NTR_RECEIVERCALL_NAME).Data.(string)
		liArgs := []Nod{}
		listGetter := NodNew(NT_VAR_GETTER)
		NodSetChild(listGetter, NTR_VAR_GETTER_NAME, NodNewData(NT_IDENTIFIER, varName))
		varDef := NodGetChild(listCall, NTR_FUNCDEF)
		varType := NodGetChild(varDef, NTR_TYPE)
		NodSetChild(listGetter, NTR_TYPE, varType)
		NodSetChild(listGetter, NTR_VARDEF, varDef)
		liArgs = append(liArgs, listGetter)
		liArgs = append(liArgs, NodGetChild(listCall, NTR_RECEIVERCALL_VALUE))
		listNod := NodNewChildList(NT_LIT_LIST, liArgs)

		// copy info into the extant list call
		NodSetChild(listCall, NTR_RECEIVERCALL_NAME, NodNewData(NT_IDENTIFIER, "$li"))
		NodSetChild(listCall, NTR_RECEIVERCALL_VALUE, listNod)
	}

}
