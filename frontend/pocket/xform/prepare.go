package xform

import (
	"fmt"
	. "pocket-lang/frontend/pocket/common"
	. "pocket-lang/parse"
)

func (x *XformerPocket) prepare() {
	x.parseMolecules()
	x.parseInlineOpStreams()
	x.prepareDotOps()
	x.addImplicitSelvesToMethods()
	x.annotateKeywordArgs()

}

func (x *XformerPocket) parseMolecules() {
	x.SearchReplaceAll(func(n Nod) bool {
		return n.NodeType == NT_VALUE_MOLECULE
	}, func(n Nod) Nod { return x.parseMolecule(n) })

}

func (x *XformerPocket) parseInlineOpStreams() {
	// TODO: make slightly more concise using SearchReplaceAll()
	x.applyRewriteOnGraph(&RewriteRule{
		condition: func(n Nod) bool {
			return n.NodeType == NT_INLINEOPSTREAM
		},
		action: func(n Nod) {
			x.Replace(n, x.parseInlineOpStream(n))
		},
	})
}

func (x *XformerPocket) annotateKeywordArgs() {
	candCalls := x.SearchRoot(func(n Nod) bool {
		if n.NodeType == NT_RECEIVERCALL || n.NodeType == NT_RECEIVERCALL_CMD {
			arg := NodGetChild(n, NTR_RECEIVERCALL_ARG)
			if arg.NodeType == NT_LIT_MAP {
				allSeemKWArg := true
				kvpairs := NodGetChildList(arg)
				for _, kvpair := range kvpairs {
					key := NodGetChild(kvpair, NTR_KVPAIR_KEY)
					if key.NodeType == NT_IDENTIFIER_RVAL {
						//pass
					} else {
						allSeemKWArg = false
						break
					}
				}
				if allSeemKWArg {
					return true
				}
			}
		}
		return false
	})

	for _, call := range candCalls {
		arg := NodGetChild(call, NTR_RECEIVERCALL_ARG)
		arg.NodeType = NT_KWARGS
		kvpairs := NodGetChildList(arg)
		for _, kvpair := range kvpairs {
			key := NodGetChild(kvpair, NTR_KVPAIR_KEY)
			val := NodGetChild(kvpair, NTR_KVPAIR_VAL)
			kwarg := NodNew(NT_KWARG)
			NodSetChild(kwarg, NTR_VAR_NAME, key)
			key.NodeType = NT_IDENTIFIER_KWARG
			NodSetChild(kwarg, NTR_VARASSIGN_VALUE, val)
			x.Replace(kvpair, kwarg)
		}
	}
}

func (x *XformerPocket) addImplicitSelvesToMethods() {
	methods := x.SearchRoot(func(n Nod) bool {
		if n.NodeType == NT_FUNCDEF {
			cCls := x.getContainingClassDef(n)
			if cCls != nil {
				return true
			}
		}
		return false
	})

	for _, method := range methods {
		cCls := x.getContainingClassDef(method)
		selfDef := NodNew(NT_VARDEF)
		NodSetChild(selfDef, NTR_VARDEF_NAME, NodNewData(NT_IDENTIFIER_RESOLVED, "self"))
		NodSetChild(selfDef, NTR_VARDEF_SCOPE, NodNewData(NT_VARDEF_SCOPE, VSCOPE_FUNCPARAM))
		NodSetChild(selfDef, NTR_TYPE_DECL, cCls)
		NodSetChild(method, NTR_METHOD_SELFDEF, selfDef)
	}
}

func (x *XformerPocket) prepareDotOps() {
	allDotOps := x.SearchRoot(func(n Nod) bool {
		return n.NodeType == NT_DOTOP
	})

	// rewrite the right side of dot ops to be simple NT_IDENTIFIERs
	for _, dotOp := range allDotOps {
		rightArg := NodGetChild(dotOp, NTR_BINOP_RIGHT)
		if rightArg.NodeType == NT_VAR_GETTER {
			varName := NodGetChild(rightArg, NTR_VAR_NAME).Data.(string)
			newNode := NodNewData(NT_DOTOP_QUALIFIER, varName)
			x.Replace(rightArg, newNode)
			panic("unsupported")
		} else if rightArg.NodeType == NT_IDENTIFIER || rightArg.NodeType == NT_IDENTIFIER_RVAL {
			// rewrite as object field accessor
			fieldAccess := NodNew(NT_OBJFIELD_ACCESSOR)
			obj := NodGetChild(dotOp, NTR_BINOP_LEFT)
			fieldName := NodGetChild(dotOp, NTR_BINOP_RIGHT)
			NodSetChild(fieldAccess, NTR_RECEIVERCALL_BASE, obj)
			NodSetChild(fieldAccess, NTR_OBJFIELD_ACCESSOR_NAME, fieldName)
			x.Replace(dotOp, fieldAccess)
		} else if rightArg.NodeType == NT_RECEIVERCALL {
			rcBase := NodGetChild(rightArg, NTR_RECEIVERCALL_BASE)
			if !(rcBase.NodeType == NT_IDENTIFIER) {
				panic("illegal call on right side of dot")
			}
			methArg := NodGetChild(rightArg, NTR_RECEIVERCALL_ARG)
			methBase := NodGetChild(dotOp, NTR_BINOP_LEFT)
			// rewrite as method call
			methCall := NodNew(NT_RECEIVERCALL_METHOD)
			NodSetChild(methCall, NTR_RECEIVERCALL_BASE, methBase)
			NodSetChild(methCall, NTR_RECEIVERCALL_METHOD_NAME, rcBase)
			NodSetChild(methCall, NTR_RECEIVERCALL_ARG, methArg)
			x.Replace(dotOp, methCall)

		} else {
			panic("illegal expression on right side of dot")
		}
	}

	// rewrite certain forms of callcmd to be method calls
	callCmds := x.SearchRoot(func(n Nod) bool {
		if n.NodeType == NT_RECEIVERCALL_CMD {
			base := NodGetChild(n, NTR_RECEIVERCALL_BASE)
			if base.NodeType == NT_DOTOP {
				panic("state error, these should be rewritten by now")
			}
			if base.NodeType == NT_OBJFIELD_ACCESSOR {
				return true
			}
		}
		return false
	})
	for _, callCmd := range callCmds {
		fieldAccessor := NodGetChild(callCmd, NTR_RECEIVERCALL_BASE)
		methArg := NodGetChild(callCmd, NTR_RECEIVERCALL_ARG)
		methBase := NodGetChild(fieldAccessor, NTR_RECEIVERCALL_BASE)
		methName := NodGetChild(fieldAccessor, NTR_OBJFIELD_ACCESSOR_NAME)
		// rewrite as method call
		methCall := NodNew(NT_RECEIVERCALL_METHOD)
		NodSetChild(methCall, NTR_RECEIVERCALL_BASE, methBase)
		NodSetChild(methCall, NTR_RECEIVERCALL_METHOD_NAME, methName)
		NodSetChild(methCall, NTR_RECEIVERCALL_ARG, methArg)
		x.Replace(callCmd, methCall)
	}

}

func (x *XformerPocket) parseMolecule(molecule Nod) Nod {
	// converts an instance of NT_VALUE_MOLECULE
	// to a proper tree representation of the constituent ops

	// for now assume priority is determined by left-right ordering
	streamNods := NodGetChildList(molecule)
	// our strategy: reduce streamNods until it no longer directly contains any stub operators
	atomNdx := -1 // contains the known index of the atom within streamNods
	// first find the atom
	for i, nod := range streamNods {
		if !isPrefixOpType(nod.NodeType) {
			atomNdx = i
			break
		}
	}
	if atomNdx == -1 {
		panic("state error")
	}
	for len(streamNods) > 1 {
		if firstNod := streamNods[0]; isPrefixOpType(firstNod.NodeType) {
			pop := streamNods[atomNdx-1]
			atom := streamNods[atomNdx]
			NodSetChild(pop, NTR_RECEIVERCALL_ARG, atom)
			streamNods = x.removeNodListAt(streamNods, atomNdx)
			atomNdx--
		} else if lastNod := streamNods[len(streamNods)-1]; isSuffixOpType(lastNod.NodeType) {
			panic("ah suffix")
		} else {
			panic("invalid molecule stream")
		}
	}
	rv := streamNods[0]
	return rv
}

func (x *XformerPocket) parseInlineOpStream(opStream Nod) Nod {
	// converts an inline op stream to a proper prioritized tree representation
	priGroups := [][]int{
		[]int{NT_DOTOP, NT_DOTPIPEOP},
		[]int{NT_MULOP, NT_DIVOP, NT_MODOP},
		[]int{NT_ADDOP, NT_SUBOP},
		[]int{NT_LTOP, NT_LTEQOP, NT_GTOP, NT_GTEQOP, NT_EQOP},
		[]int{NT_OROP, NT_ANDOP},
	}
	opStreamNods := NodGetChildList(opStream)
	operands := []Nod{}
	operators := []Nod{}
	for i := 0; i < len(opStreamNods); i += 2 {
		operands = append(operands, opStreamNods[i])
	}
	for i := 1; i < len(opStreamNods); i += 2 {
		operators = append(operators, opStreamNods[i])
	}
	fmt.Println("operands", PrettyPrintNodes(operands))
	fmt.Println("operators", PrettyPrintNodes(operators))
	for _, priGroup := range priGroups {
		for _, currOp := range priGroup {
			for i := 0; i < len(operators); i++ {
				op := operators[i].NodeType
				if currOp == op {
					groupedOp := NodNew(op)
					NodSetChild(groupedOp, NTR_BINOP_LEFT, operands[i])
					NodSetChild(groupedOp, NTR_BINOP_RIGHT, operands[i+1])
					// replace 2 operands with single group
					operands = x.removeNodListAt(operands, i)
					operands[i] = groupedOp
					// remove operator
					operators = x.removeNodListAt(operators, i)
					i--
				}
			}
		}
	}

	if len(operands) > 1 {
		panic("couldn't fully parse inline op stream")
	} else if len(operands) == 0 {
		panic("weird state error")
	}

	return operands[0]
}
