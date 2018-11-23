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
	PNT_DUCK_METHOD_CALL

	PNT_WRAP_OBJ_INIT

	// duck annotation flags
	PNTR_TYPE_INDEXABLE
)

type Preparer struct {
	*xform.Xformer
}

func (p *Preparer) Prepare(code Nod) {
	p.Root = code
	p.checkForPrintStatements()
	p.createExplicitIndexors()
	p.rewriteDuckedOps()
	p.serializeKeywordArgs()
	p.createObjInitWrappers()
}

func (p *Preparer) createObjInitWrappers() {
	// replace all object initializers with "wrapped" ones specific to the go backend
	// this makes it much easier for the generator to generate the correct object
	// initialization code

	objInits := p.SearchRoot(func(n Nod) bool {
		return n.NodeType == NT_OBJINIT
	})

	for _, objInit := range objInits {
		base := NodGetChild(objInit, NTR_RECEIVERCALL_BASE)
		if base.NodeType == NT_CLASSDEF {
			p.Replace2(objInit, func(n Nod) Nod { return p.createObjInitWrapped(n) })
		}
	}
}

func (p *Preparer) createObjInitWrapped(objInit Nod) Nod {
	rv := objInit
	clsDef := NodGetChild(objInit, NTR_RECEIVERCALL_BASE)
	cfgArg := NodGetChildOrNil(objInit, NTR_RECEIVERCALL_CFG_ARG)
	if cfgArg != nil {
		rv = p.wrapObjInit(rv, clsDef, cfgArg, true)
		NodRemoveChild(objInit, NTR_RECEIVERCALL_CFG_ARG)
	}
	initArg := NodGetChildOrNil(objInit, NTR_RECEIVERCALL_ARG)
	if initArg != nil {
		// optimization: don't wrap if empty arg list
		if initArg.NodeType != NT_EMPTYARGLIST {
			rv = p.wrapObjInit(rv, clsDef, initArg, false)
		}
		NodRemoveChild(objInit, NTR_RECEIVERCALL_ARG)
	}

	return rv
}

func (p *Preparer) wrapObjInit(objInit Nod, clsDef Nod, arg Nod, isConfig bool) Nod {
	nn := NodNew(PNT_WRAP_OBJ_INIT)
	NodSetChild(nn, NTR_RECEIVERCALL_BASE, objInit)
	NodSetChild(nn, NTR_RECEIVERCALL_ARG, arg)
	NodSetChild(nn, NTR_CLASSDEF, clsDef)
	NodSetChild(nn, NTR_PRAGMAPAINT, NodNewData(NT_PRAGMAPAINT, isConfig))
	return nn
}

func (p *Preparer) serializeKeywordArgs() {
	kwargCalls := p.SearchRoot(func(n Nod) bool {
		if isReceiverCallType(n.NodeType) {
			arg := NodGetChild(n, NTR_RECEIVERCALL_ARG)
			if arg.NodeType == NT_KWARGS {
				return true
			}
		}
		return false
	})

	for _, call := range kwargCalls {
		arg := NodGetChild(call, NTR_RECEIVERCALL_ARG)
		kwargs := NodGetChildList(arg)
		if len(kwargs) > 1 {
			panic("multi-kwargs not yet supported")
		}
		// serialArgs := []Nod{}
		var newArg Nod
		if len(kwargs) == 1 {
			newArg = NodGetChild(kwargs[0], NTR_VARASSIGN_VALUE)
		} else {
			newArg = NodNew(NT_EMPTYARGLIST)
		}
		p.Replace(arg, newArg)
	}
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
		// TODO: remove this path
		return p.isIndexableType(NodGetChild(n, NTR_TYPEARGED_BASE))
	} else if n.NodeType == NT_CLASSDEF || n.NodeType == NT_FUNCDEF {
		return false
	} else if n.NodeType == NT_TYPECALL {
		return p.isIndexableType(NodGetChild(n, NTR_RECEIVERCALL_BASE))
	} else if n.NodeType == DYPE_UNION {
		return p.isIndexableTypeUnion(n)
	} else if n.NodeType == DYPE_ALL || n.NodeType == DYPE_EMPTY {
		return false
	} else {
		panic("unhandled type:" + PrettyPrint(n))
	}
}

func (p *Preparer) isIndexableTypeUnion(n Nod) bool {
	// assumption: simplified, non-nested union
	args := NodGetChildList(n)
	for _, arg := range args {
		if !p.isIndexableType(arg) {
			return false
		}
	}
	return true
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
		// i.e., replace a list with its only element
		arg := NodGetChild(listCall, NTR_RECEIVERCALL_ARG)
		if arg.NodeType == NT_LIT_LIST {
			listEles := NodGetChildList(arg)
			if len(listEles) == 1 {
				p.Replace(arg, listEles[0])
			}
		}

		// annotate the type as explicitly indexable
		base := NodGetChild(listCall, NTR_RECEIVERCALL_BASE)
		NodSetChild(NodGetChild(base, NTR_TYPE), PNTR_TYPE_INDEXABLE, NodNew(NT_EMPTYARGLIST))
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
	p.rewriteDuckedOpsObjMethodCall()
	p.rewriteDuckedOpsBinary()
}

func (p *Preparer) isDuckType(n Nod) bool {
	if n == nil {
		return true
	}
	return n.NodeType == DYPE_ALL || n.NodeType == DYPE_UNION
}

func (p *Preparer) rewriteDuckedOpsObjMethodCall() {
	// rewrite obj.x(arg) -> P__duck_method_call(obj, "x", arg)
	duckCalls := p.SearchRoot(func(n Nod) bool {
		if n.NodeType == NT_RECEIVERCALL_METHOD {
			baseType := NodGetChild(NodGetChild(n, NTR_RECEIVERCALL_BASE), NTR_TYPE)
			if p.isDuckType(baseType) {
				return true
			}
		}
		return false
	})

	for _, duckCall := range duckCalls {
		duckCall.NodeType = PNT_DUCK_METHOD_CALL
	}
}

func (p *Preparer) rewriteDuckedOpsObjFieldRead() {
	// rewrite obj.x -> P__duck_field_read(obj, "x")
	p.SearchReplaceAll(func(n Nod) bool {
		if n.NodeType == NT_OBJFIELD_ACCESSOR {
			leftType := NodGetChildOrNil(NodGetChild(n, NTR_RECEIVERCALL_BASE), NTR_TYPE)
			if p.isDuckType(leftType) {
				return true
			}
		}
		return false
	}, func(n Nod) Nod {
		rv := NodNew(PNT_DUCK_FIELD_READ)
		NodSetChild(rv, NTR_RECEIVERCALL_BASE, NodGetChild(n, NTR_RECEIVERCALL_BASE))
		NodSetChild(rv, NTR_OBJFIELD_ACCESSOR_NAME, NodGetChild(n, NTR_OBJFIELD_ACCESSOR_NAME))
		return rv
	})
}

func (p *Preparer) rewriteDuckedOpsObjFieldWrite() {
	// rewrite obj.x : val -> P__duck_field_write(obj, "x", val)
	duckAssigns := p.SearchRoot(func(n Nod) bool {
		if n.NodeType == NT_VARASSIGN {
			lhs := NodGetChild(n, NTR_VAR_NAME)
			if lhs.NodeType == NT_OBJFIELD_ACCESSOR {
				obj := NodGetChild(lhs, NTR_RECEIVERCALL_BASE)
				objType := NodGetChildOrNil(obj, NTR_TYPE)
				if p.isDuckType(objType) {
					return true
				}
			}
		}
		return false
	})

	for _, duckAssign := range duckAssigns {
		objAccessor := NodGetChild(duckAssign, NTR_VAR_NAME)
		obj := NodGetChild(objAccessor, NTR_RECEIVERCALL_BASE)
		qual := NodGetChild(objAccessor, NTR_OBJFIELD_ACCESSOR_NAME)
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
			if NodGetChild(NodGetChild(n, NTR_BINOP_LEFT), NTR_TYPE).NodeType == DYPE_ALL ||
				NodGetChild(NodGetChild(n, NTR_BINOP_RIGHT), NTR_TYPE).NodeType == DYPE_ALL {
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
