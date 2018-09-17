package xform

import (
	"fmt"
	. "pocket-lang/frontend/pocket/common"
	. "pocket-lang/parse"
	. "pocket-lang/xform"
	"strconv"
)

type XformerPocket struct {
	*Xformer
	tempVarCounter int
}

func Xform(root Nod) Nod {
	fmt.Println("starting Xform()")

	xformer := &XformerPocket{&Xformer{}, 0}

	xformer.Root = root
	xformer.Xform()
	return root
}

func (x *XformerPocket) Xform() {
	x.prepare()
	x.desugar()

	fmt.Println("after desugaring:", PrettyPrint(x.Root))

	x.solve()
	fmt.Println("after solving", PrettyPrint(x.Root))
	x.colorTypes()
	x.checkAllCallsResolved()

	fmt.Println("after xform:", PrettyPrint(x.Root))
}

func (x *XformerPocket) solve() {
	// resolves identifiers and types to the best of our ability
	rules := x.getAllSolveRules()
	nodes := x.getSolvableNodes()
	x.initializeSolvableNodes(nodes)
	// fmt.Println("after initializing solveable nodes", PrettyPrint(x.Root))

	x.applyRewritesUntilStable(nodes, rules)
}

func (x *XformerPocket) initializeSolvableNodes(ns []Nod) {
	for _, n := range ns {
		x.initializeSolvableNode(n)
	}
}

func (x *XformerPocket) initializeSolvableNode(n Nod) {
	nt := n.NodeType
	if isMypedValueType(nt) {
		x.initializePosNegMypes(n)
	} else if nt == NT_FUNCDEF {
		NodSetChild(n, NTR_VARTABLE, NodNew(NT_VARTABLE))
		rvPlaceholder := NodNew(NT_FUNCDEF_RV_PLACEHOLDER)
		NodSetChild(n, NTR_RETURNVAL_PLACEHOLDER, rvPlaceholder)
		x.initializePosNegMypes(rvPlaceholder)
	} else if nt == NT_CLASSDEF {
		x.buildClassVardefTable(n)
		x.buildClassFuncdefTable(n)
		cVarDefs := NodGetChildList(NodGetChild(n, NTR_VARTABLE))
		for _, cVarDef := range cVarDefs {
			x.initializePosNegMypes(cVarDef)
		}
	} else if nt == NT_TOPLEVEL {
		x.buildClassTable()
		x.buildRootFuncTable()
	} else if nt == NT_IDENTIFIER || nt == NT_RETURN || nt == NT_CLASSFIELD {
		// purposeful pass
	} else {
		panic("couldn't initialize solvable node")
	}
}

func (x *XformerPocket) getSolvableNodes() []Nod {
	return x.SearchRoot(func(n Nod) bool {
		return x.isSolvableNode(n)
	})
}

func (x *XformerPocket) isSolvableNode(n Nod) bool {
	nt := n.NodeType
	return isMypedValueType(nt) ||
		nt == NT_FUNCDEF || nt == NT_CLASSDEF || nt == NT_CLASSFIELD ||
		nt == NT_TOPLEVEL || nt == NT_IDENTIFIER || nt == NT_RETURN
}

func (x *XformerPocket) getAllSolveRules() []*RewriteRule {
	typeRules := x.getAllSolveTypeRules()
	idRules := x.getIdentifierRewriteRules()
	return append(typeRules, idRules...)
}

func (x *XformerPocket) getContainingNodOrNil(start Nod, condition func(Nod) bool) Nod {
	return x.SearchOneFrom(start, condition, func(n Nod) []Nod {
		return x.AllInNodes(n)
	})
}

func (x *XformerPocket) SearchOneFrom(start Nod, condition func(Nod) bool, direction func(Nod) []Nod) Nod {
	allNodes := x.SearchFrom(start, condition, direction, func(ns []Nod) bool { return len(ns) >= 1 })
	if len(allNodes) == 0 {
		return nil
	}
	if len(allNodes) == 1 {
		return allNodes[0]
	}
	panic("state error")
}

func (x *XformerPocket) prepare() {
	x.parseInlineOpStreams()
	x.prepareDotOps()
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

func isSystemCall(n Nod) bool {
	if isReceiverCallType(n.NodeType) {
		base := NodGetChild(n, NTR_RECEIVERCALL_BASE)
		if callName, ok := base.Data.(string); ok {
			return isSystemFuncName(callName)
		}
	}
	return false
}

func isReceiverCallType(nt int) bool {
	return nt == NT_RECEIVERCALL || nt == NT_RECEIVERCALL_CMD
}

func (x *XformerPocket) checkAllCallsResolved() {
	calls := x.SearchRoot(func(n Nod) bool {
		return isReceiverCallType(n.NodeType)
	})
	for _, call := range calls {
		base := NodGetChild(call, NTR_RECEIVERCALL_BASE)
		if isSystemCall(call) || base.NodeType == NT_DOTOP || base.NodeType == NT_VAR_GETTER {
			continue // don't check these
		}

		if !NodHasChild(call, NTR_FUNCDEF) {
			panic("unknown function '" + PrettyPrint(base) + "'")
		}
	}
}

func isSystemFuncName(name string) bool {
	return name == "print" || name == "$li"
}

func (x *XformerPocket) parseInlineOpStreams() {
	x.applyRewriteOnGraph(&RewriteRule{
		condition: func(n Nod) bool {
			return n.NodeType == NT_INLINEOPSTREAM
		},
		action: func(n Nod) {
			x.Replace(n, x.parseInlineOpStream(n))
		},
	})
}

type RewriteRule struct {
	condition func(n Nod) bool
	action    func(n Nod)
}

func isLiteralNodeType(nt int) bool {
	return isPrimitiveLiteralNodeType(nt) ||
		isCollectionLiteralNodeType(nt)

}

func isCollectionLiteralNodeType(nt int) bool {
	return nt == NT_LIT_MAP || nt == NT_LIT_LIST ||
		nt == NT_LIT_SET
}

func isPrimitiveLiteralNodeType(nt int) bool {
	return nt == NT_LIT_INT || nt == NT_LIT_STRING ||
		nt == NT_LIT_BOOL || nt == NT_LIT_FLOAT
}

func getLiteralTypeAnnDataFromNT(nt int) int {
	lut := map[int]int{
		NT_LIT_INT:    TY_INT,
		NT_LIT_STRING: TY_STRING,
		NT_LIT_BOOL:   TY_BOOL,
		NT_LIT_FLOAT:  TY_FLOAT,
		NT_LIT_LIST:   TY_LIST,
		NT_LIT_SET:    TY_SET,
		NT_LIT_MAP:    TY_MAP,
	}
	if rv, ok := lut[nt]; ok {
		return rv
	} else {
		panic("unknown literal type: " + strconv.Itoa(nt))
	}
}

func isBinaryOpType(nt int) bool {
	return nt == NT_ADDOP || nt == NT_GTOP || nt == NT_LTOP ||
		nt == NT_GTEQOP || nt == NT_LTEQOP || nt == NT_EQOP ||
		nt == NT_SUBOP || nt == NT_DIVOP || nt == NT_MULOP ||
		nt == NT_OROP || nt == NT_ANDOP || nt == NT_MODOP ||
		nt == NT_DOTOP || nt == NT_DOTPIPEOP
}

func isRValVarReferenceNT(nt int) bool {
	return nt == NT_VAR_GETTER || nt == NT_VARASSIGN || nt == NT_PARAMETER ||
		nt == NT_IDENTIFIER_RVAL
}

func isCallType(nt int) bool {
	return nt == NT_RECEIVERCALL || nt == NT_RECEIVERCALL_CMD ||
		nt == NT_RECEIVERCALL_METHOD || nt == NT_OBJINIT
}

func (x *XformerPocket) applyRewritesUntilStable(nods []Nod, rules []*RewriteRule) {
	nPasses := 0
	for {
		maxApplied := 0
		for _, rule := range rules {
			nApplied := x.applyRewriteRuleOnJust(nods, rule)
			if nPasses > 20 && nApplied > 0 { // we should never need more than 20 passes
				fmt.Println("Warning: 20 passes exceed, likely cycle detected")
				rule.action(nil) // try to break it to get a good debug trace
				panic("too many passes, could not solve")
			}
			if nApplied > maxApplied {
				maxApplied = nApplied
			}
		}
		if maxApplied == 0 {
			break
		}
		nPasses++
		fmt.Println("nPasses", nPasses)
	}
	fmt.Println("exiting applyrwus")
}

func (x *XformerPocket) applyRewriteOnGraph(rule *RewriteRule) int {
	nApplied := 0
	nods := x.SearchRoot(rule.condition)
	for _, ele := range nods {
		rule.action(ele)
		nApplied++
	}
	return nApplied
}

func (x *XformerPocket) applyRewriteRuleOnJust(nods []Nod, rule *RewriteRule) int {
	nApplied := 0
	for _, ele := range nods {
		if rule.condition(ele) {
			rule.action(ele)
			nApplied++
		}
	}
	return nApplied
}

func (x *XformerPocket) parseInlineOpStream(opStream Nod) Nod {
	// converts an inline op stream to a proper prioritized tree representation
	// for now assume all elements are same priority and group left to right
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

func (x *XformerPocket) removeNodListAt(nods []Nod, removeAt int) []Nod {
	return append(nods[:removeAt], nods[removeAt+1:]...)
}
