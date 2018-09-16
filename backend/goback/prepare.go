package goback

import (
	. "pocket-lang/frontend/pocket/common"
	. "pocket-lang/parse"
	"pocket-lang/xform"
)

const (
	PNTR_GOIMPORTS = 284000 + iota
	PNT_DUCK_BINOP
	PNT_DUCK_FIELD_READ
	PNT_DUCK_FIELD_WRITE
	PNTR_DUCK_FIELD_WRITE_OBJ
	PNTR_DUCK_FIELD_WRITE_NAME
	PNTR_DUCK_FIELD_WRITE_VAL
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
	printCalls := p.SearchRoot(func(n Nod) bool {
		if isReceiverCallType(n.NodeType) {
			base := NodGetChild(n, NTR_RECEIVERCALL_BASE)
			if base.NodeType == NT_IDENTIFIER || base.NodeType == NT_IDENTIFIER_RESOLVED ||
				base.NodeType == NT_IDENTIFIER_FUNC_NOSCOPE {
				if base.Data.(string) == "print" {
					return true
				}
			}
		}
		return false
	})

	for _, printCall := range printCalls {
		NodGetChild(printCall, NTR_RECEIVERCALL_BASE).Data = "fmt.Println"
	}

	if len(printCalls) > 0 {
		NodSetChild(p.Root, PNTR_GOIMPORTS, NodNewData(NT_IDENTIFIER, "fmt"))
	}
}

func (p *Preparer) isIndexableType(n Nod) bool {
	if n.NodeType == NT_TYPEBASE {
		bt := n.Data.(int)
		return bt == TY_LIST || bt == TY_MAP
	} else if n.NodeType == NT_TYPEARGED {
		return p.isIndexableType(NodGetChild(n, NTR_TYPEARGED_BASE))
	} else if n.NodeType == NT_CLASSDEF {
		return false
	} else {
		panic("unhandled type")
	}
}

func (p *Preparer) createExplicitIndexors() {
	listCalls := p.SearchRoot(func(n Nod) bool {
		if isReceiverCallType(n.NodeType) {
			base := NodGetChild(n, NTR_RECEIVERCALL_BASE)
			if base.NodeType == NT_VAR_GETTER {
				vDef := NodGetChild(base, NTR_VARDEF)
				if p.isIndexableType(NodGetChild(vDef, NTR_TYPE)) {
					return true
				}
			}
		}
		return false
	})
	for _, listCall := range listCalls {
		listCall.NodeType = NT_COLLECTION_INDEXOR

		// convert [i] syntax into (i) syntax if applicable
		arg := NodGetChild(listCall, NTR_RECEIVERCALL_ARG)
		if arg.NodeType == NT_LIT_LIST {
			listEles := NodGetChildList(arg)
			if len(listEles) == 1 {
				p.Replace(arg, listEles[0])
			}
		}
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
	p.rewriteDuckedOpsObjFieldWrite()
	p.rewriteDuckedOpsObjFieldRead()
	p.rewriteDuckedOpsBinary()
}

func (p *Preparer) rewriteDuckedOpsObjFieldRead() {
	// rewrite obj.x -> P__duck_field_read(obj, "x")
	p.SearchReplaceAll(func(n Nod) bool {
		if n.NodeType == NT_DOTOP {
			leftType := NodGetChild(NodGetChild(n, NTR_BINOP_LEFT), NTR_TYPE)
			if leftType.NodeType == NT_TYPEBASE && leftType.Data.(int) == TY_DUCK {
				return true
			}
		}
		return false
	}, func(n Nod) Nod {
		rv := NodNew(PNT_DUCK_FIELD_READ)
		NodSetChild(rv, NTR_BINOP_LEFT, NodGetChild(n, NTR_BINOP_LEFT))
		NodSetChild(rv, NTR_BINOP_RIGHT, NodGetChild(n, NTR_BINOP_RIGHT))
		return rv
	})
}

func (p *Preparer) rewriteDuckedOpsObjFieldWrite() {
	// rewrite obj.x : val -> P__duck_field_write(obj, "x", val)
	duckAssigns := p.SearchRoot(func(n Nod) bool {
		if n.NodeType == NT_VARASSIGN {
			lhs := NodGetChild(n, NTR_VAR_NAME)
			if lhs.NodeType == NT_DOTOP {
				obj := NodGetChild(lhs, NTR_BINOP_LEFT)
				objType := NodGetChild(obj, NTR_TYPE)
				if objType.NodeType == NT_TYPEBASE && objType.Data.(int) == TY_DUCK {
					return true
				}
			}
		}
		return false
	})

	for _, duckAssign := range duckAssigns {
		dotOp := NodGetChild(duckAssign, NTR_VAR_NAME)
		obj := NodGetChild(dotOp, NTR_BINOP_LEFT)
		qual := NodGetChild(dotOp, NTR_BINOP_RIGHT)
		val := NodGetChild(duckAssign, NTR_VARASSIGN_VALUE)
		duckWrite := NodNew(PNT_DUCK_FIELD_WRITE)
		NodSetChild(duckWrite, PNTR_DUCK_FIELD_WRITE_OBJ, obj)
		NodSetChild(duckWrite, PNTR_DUCK_FIELD_WRITE_NAME, qual)
		NodSetChild(duckWrite, PNTR_DUCK_FIELD_WRITE_VAL, val)
		p.Replace(duckAssign, duckWrite)
	}
}

func (p *Preparer) rewriteDuckedOpsBinary() {
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
