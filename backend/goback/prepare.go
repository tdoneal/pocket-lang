package goback

import (
	. "pocket-lang/frontend/pocket"
	. "pocket-lang/parse"
	"pocket-lang/xform"
)

const (
	PNTR_GOIMPORTS = 284000 + iota
	PNT_DUCK_BINOP
)

type Preparer struct {
	*xform.Xformer
}

func (p *Preparer) Prepare(code Nod) {
	p.Root = code
	p.checkForPrintStatements()
	p.createExplicitIndexors()
	p.rewriteDuckedOps()
}

func (p *Preparer) checkForPrintStatements() {
	rcs := p.SearchRoot(func(n Nod) bool {
		return isReceiverCallType(n.NodeType) &&
			NodGetChild(n, NTR_RECEIVERCALL_BASE).Data.(string) == "print"
	})

	if len(rcs) > 0 {
		NodSetChild(p.Root, PNTR_GOIMPORTS, NodNewData(NT_IDENTIFIER, "fmt"))
	}
}

func (p *Preparer) isIndexableType(n Nod) bool {
	if n.NodeType == NT_TYPEBASE {
		bt := n.Data.(int)
		return bt == TY_LIST || bt == TY_MAP
	} else if n.NodeType == NT_TYPEARGED {
		return p.isIndexableType(NodGetChild(n, NTR_TYPEARGED_BASE))
	} else {
		panic("unhandled type of type")
	}
}

func (p *Preparer) createExplicitIndexors() {
	listCalls := p.SearchRoot(func(n Nod) bool {
		if isReceiverCallType(n.NodeType) {
			if funcDef := NodGetChildOrNil(n, NTR_FUNCDEF); funcDef != nil {
				if funcDef.NodeType == NT_VARDEF {
					if p.isIndexableType(NodGetChild(funcDef, NTR_TYPE)) {
						return true
					}
				}
			}
		}
		return false
	})
	for _, listCall := range listCalls {
		varName := NodGetChild(listCall, NTR_RECEIVERCALL_BASE).Data.(string)
		liArgs := []Nod{}
		listGetter := NodNew(NT_VAR_GETTER)
		NodSetChild(listGetter, NTR_VAR_GETTER_NAME, NodNewData(NT_IDENTIFIER, varName))
		varDef := NodGetChild(listCall, NTR_FUNCDEF)
		varType := NodGetChild(varDef, NTR_TYPE)
		NodSetChild(listGetter, NTR_TYPE, varType)
		NodSetChild(listGetter, NTR_VARDEF, varDef)
		liArgs = append(liArgs, listGetter)
		listIndex := p.interpretAsListIndex(NodGetChild(listCall, NTR_RECEIVERCALL_ARG))
		liArgs = append(liArgs, listIndex)
		listNod := NodNewChildList(NT_LIT_LIST, liArgs)

		// copy info into the extant list call
		NodSetChild(listCall, NTR_RECEIVERCALL_BASE, NodNewData(NT_IDENTIFIER, "$li"))
		NodSetChild(listCall, NTR_RECEIVERCALL_ARG, listNod)
	}

}

func (p *Preparer) interpretAsListIndex(arg Nod) Nod {
	if arg.NodeType == NT_LIT_LIST {
		listElements := NodGetChildList(arg)
		if len(listElements) != 1 {
			panic("only one list index dimension supported")
		}
		return listElements[0]
	} else {
		return arg
	}
}

func (p *Preparer) rewriteDuckedOps() {
	// search for: any binary ops with ducked args
	p.SearchReplaceAll(func(n Nod) bool {
		if isBinaryInlineOpType(n.NodeType) {
			if NodGetChild(NodGetChild(n, NTR_BINOP_LEFT), NTR_TYPE).Data.(int) == TY_DUCK ||
				NodGetChild(NodGetChild(n, NTR_BINOP_RIGHT), NTR_TYPE).Data.(int) == TY_DUCK {
				return true
			}
		}
		return false
	}, func(n Nod) Nod {
		rv := NodNew(PNT_DUCK_BINOP)
		rv.Data = n.NodeType
		NodSetChild(rv, NTR_BINOP_LEFT, NodGetChild(n, NTR_BINOP_LEFT))
		NodSetChild(rv, NTR_BINOP_RIGHT, NodGetChild(n, NTR_BINOP_RIGHT))
		return rv
	})

}
