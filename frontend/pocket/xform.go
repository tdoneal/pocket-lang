package pocket

import (
	"fmt"
	. "pocket-lang/parse"
	. "pocket-lang/xform"
	"strconv"
)

const (
	TY_VOID   = 1
	TY_BOOL   = 2
	TY_INT    = 3
	TY_FLOAT  = 4
	TY_STRING = 5
	TY_SET    = 6
	TY_MAP    = 7
	TY_LIST   = 8
	TY_OBJECT = 20
	TY_NUMBER = 22
	TY_DUCK   = 30
)

const (
	VSCOPE_FUNCLOCAL = 1
	VSCOPE_FUNCPARAM = 2
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
	x.parseInlineOpStreams()
	x.rewriteForInLoops()
	fmt.Println("after rewriting for in loops:", PrettyPrint(x.Root))

	x.annotateDotScopes()
	x.buildFuncDefTables()
	x.buildVarDefTables()
	// now do second pass of resolving functions, specifically those that may refer to variables in a local scope
	x.linkCallsToVariableFuncdefs()
	x.checkAllCallsResolved()

	fmt.Println("after building var def tables:", PrettyPrint(x.Root))

	x.solveTypes()
}

func (x *XformerPocket) rewriteForInLoops() {
	forLoops := x.SearchRoot(func(n Nod) bool { return n.NodeType == NT_FOR })
	for _, forLoop := range forLoops {
		x.Replace(forLoop, x.rewriteForInLoop(forLoop))
	}
}

func (x *XformerPocket) getTempVarName() string {
	rv := "__pkx" + strconv.Itoa(x.tempVarCounter) + "__"
	x.tempVarCounter++
	return rv
}

func (x *XformerPocket) rewriteForInLoop(forLoop Nod) Nod {
	// rewrites the for <var> in <list> : body syntax
	// to a lower level form involving a while loop and an index variable
	rvSeq := []Nod{}
	loopOver := NodGetChild(forLoop, NTR_FOR_ITEROVER)
	declaredElementVarName := NodGetChild(forLoop, NTR_FOR_ITERVAR).Data.(string)
	ndxVarName := x.getTempVarName()
	iterOverVarName := x.getTempVarName()
	// generate the effective code: __ndx_var__ : 0
	ndxVarInitializer := NodNew(NT_VARASSIGN)
	ndxVarIdentifier := NodNewData(NT_IDENTIFIER, ndxVarName)
	ndxVarInitValue := NodNewData(NT_LIT_INT, 0)
	NodSetChild(ndxVarInitializer, NTR_VAR_NAME, ndxVarIdentifier)
	NodSetChild(ndxVarInitializer, NTR_VARASSIGN_VALUE, ndxVarInitValue)

	// __iterover_var__: <iterover>
	iterOverVarInitializer := NodNew(NT_VARASSIGN)
	iterOverVarIdentifier := NodNewData(NT_IDENTIFIER, iterOverVarName)
	iterOverVarInitValue := loopOver
	NodSetChild(iterOverVarInitializer, NTR_VAR_NAME, iterOverVarIdentifier)
	NodSetChild(iterOverVarInitializer, NTR_VARASSIGN_VALUE, iterOverVarInitValue)

	loopBodySeq := []Nod{}
	// while __ndx__ < seq.len
	termCond := NodNew(NT_LTOP)
	termCondNdxVarGetter := NodNewChild(NT_VAR_GETTER, NTR_VAR_GETTER_NAME,
		NodNewData(NT_IDENTIFIER, ndxVarName))
	termCondLenGetter := NodNew(NT_DOTOP)
	NodSetChild(termCondLenGetter, NTR_BINOP_LEFT,
		NodNewChild(NT_VAR_GETTER, NTR_VAR_GETTER_NAME,
			NodNewData(NT_IDENTIFIER, iterOverVarName)))
	NodSetChild(termCondLenGetter, NTR_BINOP_RIGHT, NodNewData(NT_IDENTIFIER, "len"))
	NodSetChild(termCond, NTR_BINOP_LEFT, termCondNdxVarGetter)
	NodSetChild(termCond, NTR_BINOP_RIGHT, termCondLenGetter)

	// <itervar>: __iterover_var__[__ndx__]
	iterVarAssigner := NodNew(NT_VARASSIGN)
	NodSetChild(iterVarAssigner, NTR_VAR_NAME, NodNewData(NT_IDENTIFIER, declaredElementVarName))
	iterVarAssignerValue := NodNew(NT_RECEIVERCALL)
	// generate the list indexor as a receiver call
	NodSetChild(iterVarAssignerValue, NTR_RECEIVERCALL_NAME, NodNewData(NT_IDENTIFIER, iterOverVarName))
	NodSetChild(iterVarAssignerValue, NTR_RECEIVERCALL_VALUE,
		NodNewChild(NT_VAR_GETTER, NTR_VAR_GETTER_NAME,
			NodNewData(NT_IDENTIFIER, ndxVarName)))
	NodSetChild(iterVarAssigner, NTR_VARASSIGN_VALUE, iterVarAssignerValue)
	loopBodySeq = append(loopBodySeq, iterVarAssigner)

	// actual user body
	loopBodySeq = append(loopBodySeq, NodGetChild(forLoop, NTR_FOR_BODY))

	// __ndx__++
	ndxVarIncrementor := NodNew(NT_VARASSIGN)
	NodSetChild(ndxVarIncrementor, NTR_VAR_NAME, NodNewData(NT_IDENTIFIER, ndxVarName))
	ndxVarIncrementorValue := NodNew(NT_ADDOP)
	NodSetChild(ndxVarIncrementorValue, NTR_BINOP_LEFT, NodNewChild(
		NT_VAR_GETTER, NTR_VAR_GETTER_NAME, NodNewData(NT_IDENTIFIER, ndxVarName)))
	NodSetChild(ndxVarIncrementorValue, NTR_BINOP_RIGHT, NodNewData(
		NT_LIT_INT, 1))
	NodSetChild(ndxVarIncrementor, NTR_VARASSIGN_VALUE, ndxVarIncrementorValue)
	loopBodySeq = append(loopBodySeq, ndxVarIncrementor)

	// put it all together and return
	whileLoop := NodNew(NT_WHILE)
	NodSetChild(whileLoop, NTR_WHILE_COND, termCond)
	NodSetChild(whileLoop, NTR_WHILE_BODY, NodNewChildList(NT_IMPERATIVE, loopBodySeq))

	rvSeq = append(rvSeq, ndxVarInitializer)
	rvSeq = append(rvSeq, iterOverVarInitializer)
	rvSeq = append(rvSeq, whileLoop)
	rv := NodNewChildList(NT_IMPERATIVE, rvSeq)
	return rv

}

func (x *XformerPocket) annotateDotScopes() {
	allDotOps := x.SearchRoot(func(n Nod) bool {
		return n.NodeType == NT_DOTOP
	})

	// rewrite the right side of dot ops to be simple NT_IDENTIFIERs
	for _, dotOp := range allDotOps {
		rightArg := NodGetChild(dotOp, NTR_BINOP_RIGHT)
		if rightArg.NodeType == NT_VAR_GETTER {
			varName := NodGetChild(rightArg, NTR_VAR_GETTER_NAME).Data.(string)
			newNode := NodNewData(NT_IDENTIFIER, varName)
			x.Replace(rightArg, newNode)
		} else if rightArg.NodeType == NT_IDENTIFIER {
			// pass, everything looks good already
		} else {
			panic("illegal expression on right side of dot")
		}
	}

}

func isSystemCall(n Nod) bool {
	if isReceiverCallType(n.NodeType) {
		callName := NodGetChild(n, NTR_RECEIVERCALL_NAME).Data.(string)
		return isSystemFuncName(callName)
	}
	return false
}

func (x *XformerPocket) linkCallsToVariableFuncdefs() {
	// find all unresolved calls
	unresCalls := x.SearchRoot(func(n Nod) bool {
		if isReceiverCallType(n.NodeType) && !isSystemCall(n) {
			if !NodHasChild(n, NTR_FUNCDEF) {
				return true
			}
		}
		return false
	})

	for _, call := range unresCalls {
		varTable := x.getEnclosingVarTable(call)
		varDefs := NodGetChildList(varTable)
		callName := NodGetChild(call, NTR_RECEIVERCALL_NAME).Data.(string)
		var matchedVarDef Nod
		for _, varDef := range varDefs {
			varName := NodGetChild(varDef, NTR_VARDEF_NAME).Data.(string)
			if callName == varName {
				matchedVarDef = varDef
				break
			}
		}
		if matchedVarDef != nil {
			NodSetChild(call, NTR_FUNCDEF, matchedVarDef)
		}
	}
}

func (x *XformerPocket) getEnclosingVarTable(n Nod) Nod {
	funcDef := x.findTopLevelFuncDef(n)
	return NodGetChild(funcDef, NTR_VARTABLE)
}

func isReceiverCallType(nt int) bool {
	return nt == NT_RECEIVERCALL || nt == NT_RECEIVERCALL_CMD
}

func (x *XformerPocket) checkAllCallsResolved() {
	calls := x.SearchRoot(func(n Nod) bool {
		return isReceiverCallType(n.NodeType)
	})
	for _, call := range calls {
		funcName := NodGetChild(call, NTR_RECEIVERCALL_NAME).Data.(string)
		if isSystemFuncName(funcName) {
			continue // don't check these
		}
		if !NodHasChild(call, NTR_FUNCDEF) {
			panic("unknown function '" + funcName + "'")
		}
	}
}

func isSystemFuncName(name string) bool {
	return name == "print" || name == "$li"
}

func (x *XformerPocket) buildFuncDefTables() {
	funcDefs := x.SearchRoot(func(n Nod) bool { return n.NodeType == NT_FUNCDEF })
	funcTable := NodNewChildList(NT_FUNCTABLE, funcDefs)
	NodSetChild(x.Root, NTR_FUNCTABLE, funcTable)

	// link functional calls to their associated def (if found)
	calls := x.SearchRoot(func(n Nod) bool { return n.NodeType == NT_RECEIVERCALL })
	for _, call := range calls {
		callName := NodGetChild(call, NTR_RECEIVERCALL_NAME).Data.(string)
		if isSystemFuncName(callName) {
			// continue, don't worry about linking system funcs
			// as by definition there is nothing to point them to
			continue
		}
		fmt.Println("call name", callName)
		// lookup function in func table (naive linear search for now)
		var matchedFuncDef Nod
		for _, funcDef := range funcDefs {
			funcDefName := NodGetChild(funcDef, NTR_FUNCDEF_NAME).Data.(string)
			if callName == funcDefName {
				matchedFuncDef = funcDef
				break
			}
		}
		if matchedFuncDef != nil {
			NodSetChild(call, NTR_FUNCDEF, matchedFuncDef)
		}
	}
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

func (x *XformerPocket) getInitMypeNodFull() Nod {
	md := MypeExplicitNewFull()
	return NodNewData(NT_MYPE, md)
}

func (x *XformerPocket) getInitMypeNodEmpty() Nod {
	md := MypeExplicitNewEmpty()
	return NodNewData(NT_MYPE, md)
}

func isLiteralNodeType(nt int) bool {
	return nt == NT_LIT_INT || nt == NT_LIT_STRING ||
		nt == NT_LIT_BOOL || nt == NT_LIT_FLOAT ||
		nt == NT_LIT_MAP || nt == NT_LIT_LIST ||
		nt == NT_LIT_SET
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
		nt == NT_DOTOP
}

func isVarReferenceNT(nt int) bool {
	return nt == NT_VAR_GETTER || nt == NT_VARASSIGN || nt == NT_PARAMETER
}

func isCallType(nt int) bool {
	return nt == NT_RECEIVERCALL
}

func (x *XformerPocket) initializeMypes(n Nod) {
	NodSetChild(n, NTR_MYPE_NEG, x.getInitMypeNodFull())
	NodSetChild(n, NTR_MYPE_POS, x.getInitMypeNodEmpty())
}

func isMypedValueType(nt int) bool {
	return isLiteralNodeType(nt) || isBinaryOpType(nt) ||
		isVarReferenceNT(nt) || isCallType(nt)
}

func (x *XformerPocket) initializeAllMypes() []Nod {
	// initializes all mypes and returns a list of myped nodes

	// first: initialize all positive and negative in the vartable
	varDefs := x.SearchRoot(func(n Nod) bool { return n.NodeType == NT_VARDEF })
	for _, value := range varDefs {
		x.initializeMypes(value)
	}

	// next gather all value nodes
	// TODO: multi-function support
	values := x.SearchRoot(func(n Nod) bool {
		return isMypedValueType(n.NodeType)
	})

	// assign an initial mype to all value nodes
	for _, value := range values {
		// special case of variables: we don't want a mype per *instance* of a variable accessor,
		// we want a single mype per *definition* of a variable, so we point it as such
		if isVarReferenceNT(value.NodeType) {
			// we implement this by pointing the type to the unified variable table
			varDef := NodGetChild(value, NTR_VARDEF)
			NodSetChild(value, NTR_MYPE_NEG, NodGetChild(varDef, NTR_MYPE_NEG))
			NodSetChild(value, NTR_MYPE_POS, NodGetChild(varDef, NTR_MYPE_POS))
		} else {
			x.initializeMypes(value)
		}
	}

	return append(values, varDefs...)
}

func (x *XformerPocket) solveTypes() {
	// // assign a concrete type to every node
	nodes := x.initializeAllMypes()
	fmt.Println("after initial mype assignments:", PrettyPrintMypes(nodes))
	// apply repeated solve rules until convergence (for system 1 semantics)
	// apply positive rules
	positiveRules := x.getAllPositiveMARRules()
	x.applyRewritesUntilStable(nodes, positiveRules)
	// apply negative rules
	negativeRules := x.getAllNegativeMARRules()
	x.applyRewritesUntilStable(nodes, negativeRules)
	fmt.Println("after positive and negative rules:", PrettyPrintMypes(nodes))
	// generate the "valid" mypes by subtracting the converse of the negative from the positive
	x.generateValidMypes(nodes)
	fmt.Println("after generating valid mypes:", PrettyPrintMypes(nodes))
	x.convertValidMypesToFinalTypes()
	fmt.Println("Final type assignments:", PrettyPrintMypes(nodes))
}

func (x *XformerPocket) generateValidMypes(nodes []Nod) {
	for _, node := range nodes {
		posMype := NodGetChild(node, NTR_MYPE_POS).Data.(Mype)
		negMype := NodGetChild(node, NTR_MYPE_NEG).Data.(Mype)
		validMype := posMype.Subtract(negMype.Converse())
		if validMype.IsEmpty() {
			panic("couldn't find a valid type for node: " + PrettyPrintMype(node))
		}
		NodSetChild(node, NTR_MYPE_VALID, NodNewData(NT_MYPE, validMype))
		NodRemoveChild(node, NTR_MYPE_POS)
		NodRemoveChild(node, NTR_MYPE_NEG)
	}
}

func (x *XformerPocket) convertValidMypesToFinalTypes() {
	// explicitly convert mypes to types (and remove the mypes in the process)
	x.applyRewriteOnGraph(&RewriteRule{
		condition: func(n Nod) bool {
			return NodHasChild(n, NTR_MYPE_VALID)
		},
		action: func(n Nod) {
			mype := NodGetChild(n, NTR_MYPE_VALID).Data.(Mype)
			var assignType Nod
			if mype.IsSingle() {
				assignType = NodNewData(NT_TYPE, mype.GetSingleType())
			} else if mype.IsPlural() {
				assignType = NodNewData(NT_TYPE, TY_DUCK)
			} else {
				panic("should never be here")
			}
			NodSetChild(n, NTR_TYPE, assignType)
			NodRemoveChild(n, NTR_MYPE_VALID)
		},
	})
}

func (x *XformerPocket) getAllNegativeMARRules() []*RewriteRule {
	rv := []*RewriteRule{
		marNegDeclaredType(),
	}
	rv = append(rv, marNegOpRestrictRules()...)
	return rv
}

func (x *XformerPocket) getAllPositiveMARRules() []*RewriteRule {
	rv := []*RewriteRule{
		marPosLiterals(),
		marPosVarAssign(),
		marPosPublicParameter(),
		marPosSysFunc(),
		marPosVarFunc(),
	}
	rv = append(rv, marPosOpEvaluateRules()...)
	rv = append(rv, x.marPosUserFuncEvaluateRules()...)
	return rv
}

func marPosVarFunc() *RewriteRule {
	// all calls to a variable can return anything
	return &RewriteRule{
		condition: func(n Nod) bool {
			if isReceiverCallType(n.NodeType) {
				if funcDef := NodGetChildOrNil(n, NTR_FUNCDEF); funcDef != nil {
					if funcDef.NodeType == NT_VARDEF {
						return !NodGetChild(n, NTR_MYPE_POS).Data.(Mype).IsFull()
					}
				}
			}
			return false
		},
		action: func(n Nod) {
			NodGetChild(n, NTR_MYPE_POS).Data = MypeExplicitNewFull()
		},
	}
}

func marPosSysFunc() *RewriteRule {
	// all sys funcs can return anything
	return &RewriteRule{
		condition: func(n Nod) bool {
			if isReceiverCallType(n.NodeType) {
				rcName := NodGetChild(n, NTR_RECEIVERCALL_NAME).Data.(string)
				if isSystemFuncName(rcName) {
					pMype := NodGetChild(n, NTR_MYPE_POS).Data.(Mype)
					return !pMype.IsFull()
				}
			}
			return false
		},
		action: func(n Nod) {
			NodGetChild(n, NTR_MYPE_POS).Data = MypeExplicitNewFull()
		},
	}
}

func (x *XformerPocket) marPosUserFuncEvaluateRules() []*RewriteRule {
	funcTableNod := NodGetChild(x.Root, NTR_FUNCTABLE)
	funcDefs := NodGetChildList(funcTableNod)
	rv := []*RewriteRule{}
	for _, funcDef := range funcDefs {
		rule := x.marPosGenFuncEvaluateRule(funcDef)
		if rule != nil {
			rv = append(rv, rule)
		}
	}
	fmt.Println("generated evaluation rules for ", len(rv), "funcdefs")
	return rv
}

func marPosGFERCondition(call Nod, funcDef Nod) bool {
	callMype := NodGetChild(call, NTR_MYPE_POS).Data.(Mype)
	declaredDefType := NodGetChild(funcDef, NTR_FUNCDEF_OUTTYPE)
	funcOutMype := MypeExplicitNewSingle(declaredDefType.Data.(int))
	if callMype.WouldChangeFromUnionWith(funcOutMype) {
		return true
	}
	return false
}

func marPosGFERAction(call Nod) {
	funcDef := NodGetChild(call, NTR_FUNCDEF)
	declaredDefType := NodGetChild(funcDef, NTR_FUNCDEF_OUTTYPE)
	funcOutMype := MypeExplicitNewSingle(declaredDefType.Data.(int))
	NodGetChild(call, NTR_MYPE_POS).Data = funcOutMype
}

func (x *XformerPocket) marPosGenFuncEvaluateRule(funcDef Nod) *RewriteRule {
	// generates the rule that replaces a function call with its return type
	if outType := NodGetChildOrNil(funcDef, NTR_FUNCDEF_OUTTYPE); outType != nil {
		return &RewriteRule{
			condition: func(n Nod) bool {
				if n.NodeType == NT_RECEIVERCALL {
					if callDef := NodGetChildOrNil(n, NTR_FUNCDEF); callDef != nil {
						if callDef == funcDef {
							return marPosGFERCondition(n, callDef)
						}
					}

				}
				return false
			},
			action: marPosGFERAction,
		}
	} else {
		return nil
	}
}

func marPosPublicParameter() *RewriteRule {
	// assume that assignments to this parameter can be of any possible type
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_PARAMETER {
				paramMype := NodGetChild(n, NTR_MYPE_POS).Data.(Mype)
				if paramMype.IsEmpty() {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			NodGetChild(n, NTR_MYPE_POS).Data = MypeExplicitNewFull()
		},
	}
}

func marPosVarAssign() *RewriteRule {
	// propagate var assign values from rhs -> lhs
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_VARASSIGN {
				mypeLHS := NodGetChild(n, NTR_MYPE_POS).Data.(Mype)
				mypeRHS := NodGetChild(NodGetChild(n, NTR_VARASSIGN_VALUE), NTR_MYPE_POS).Data.(Mype)
				if mypeLHS.WouldChangeFromUnionWith(mypeRHS) {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			mypeLHS := NodGetChild(n, NTR_MYPE_POS).Data.(Mype)
			mypeRHS := NodGetChild(NodGetChild(n, NTR_VARASSIGN_VALUE), NTR_MYPE_POS).Data.(Mype)
			newLHS := mypeLHS.Union(mypeRHS)
			NodGetChild(n, NTR_MYPE_POS).Data = newLHS
		},
	}
}

func marNegDeclaredType() *RewriteRule {
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_PARAMETER || n.NodeType == NT_VARASSIGN {
				if typeDeclNod := NodGetChildOrNil(n, NTR_TYPE_DECL); typeDeclNod != nil {
					negMype := NodGetChild(n, NTR_MYPE_NEG).Data.(Mype)
					declMype := MypeExplicitNewSingle(typeDeclNod.Data.(int))
					if negMype.WouldChangeFromIntersectionWith(declMype) {
						return true
					}
				}
			}
			return false
		},
		action: func(n Nod) {
			typeDeclNod := NodGetChild(n, NTR_TYPE_DECL)
			declMype := MypeExplicitNewSingle(typeDeclNod.Data.(int))
			currNegMypeNod := NodGetChild(n, NTR_MYPE_NEG)
			currNegMypeNod.Data = declMype.Intersection(currNegMypeNod.Data.(Mype))
		},
	}
}

type MypeOpEvaluateRule struct {
	operator int
	operand  int
	result   int
}

func marGetCompactOpEvaluateRules() []*MypeOpEvaluateRule {
	// define type propagation rules of the form (int + int) -> int
	return []*MypeOpEvaluateRule{
		&MypeOpEvaluateRule{NT_ADDOP, TY_INT, TY_INT},
		&MypeOpEvaluateRule{NT_ADDOP, TY_FLOAT, TY_FLOAT},
		&MypeOpEvaluateRule{NT_ADDOP, TY_STRING, TY_STRING},
		&MypeOpEvaluateRule{NT_SUBOP, TY_INT, TY_INT},
		&MypeOpEvaluateRule{NT_SUBOP, TY_FLOAT, TY_FLOAT},
		&MypeOpEvaluateRule{NT_MULOP, TY_INT, TY_INT},
		&MypeOpEvaluateRule{NT_MULOP, TY_FLOAT, TY_FLOAT},
		&MypeOpEvaluateRule{NT_DIVOP, TY_INT, TY_INT},
		&MypeOpEvaluateRule{NT_DIVOP, TY_FLOAT, TY_FLOAT},
		&MypeOpEvaluateRule{NT_MODOP, TY_INT, TY_INT},
		&MypeOpEvaluateRule{NT_MODOP, TY_FLOAT, TY_FLOAT},
		&MypeOpEvaluateRule{NT_GTOP, TY_INT, TY_BOOL},
		&MypeOpEvaluateRule{NT_GTOP, TY_FLOAT, TY_BOOL},
		&MypeOpEvaluateRule{NT_LTOP, TY_INT, TY_BOOL},
		&MypeOpEvaluateRule{NT_LTOP, TY_FLOAT, TY_BOOL},
		&MypeOpEvaluateRule{NT_GTEQOP, TY_INT, TY_BOOL},
		&MypeOpEvaluateRule{NT_GTEQOP, TY_FLOAT, TY_BOOL},
		&MypeOpEvaluateRule{NT_LTEQOP, TY_INT, TY_BOOL},
		&MypeOpEvaluateRule{NT_LTEQOP, TY_FLOAT, TY_BOOL},
		&MypeOpEvaluateRule{NT_EQOP, TY_INT, TY_BOOL},
		&MypeOpEvaluateRule{NT_EQOP, TY_FLOAT, TY_BOOL},
		&MypeOpEvaluateRule{NT_EQOP, TY_STRING, TY_BOOL},
		&MypeOpEvaluateRule{NT_EQOP, TY_BOOL, TY_BOOL},
		&MypeOpEvaluateRule{NT_OROP, TY_BOOL, TY_BOOL},
		&MypeOpEvaluateRule{NT_ANDOP, TY_BOOL, TY_BOOL},
	}
}

func marPosOpEvaluateRules() []*RewriteRule {
	oers := marGetCompactOpEvaluateRules()
	rv := []*RewriteRule{}
	for _, oer := range oers {
		rv = append(rv, createPosRewriteRuleFromOpEvaluateRule(oer))
	}

	rv = append(rv, marPosOpCollectionLenRule())

	return rv
}

func getLengthableTypes() []int {
	return []int{TY_LIST, TY_MAP, TY_SET}
}

func marPosOpCollectionLenRule() *RewriteRule {
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_DOTOP {
				qualName := NodGetChild(n, NTR_BINOP_RIGHT).Data.(string)
				resulPosMype := NodGetChild(n, NTR_MYPE_POS).Data.(Mype)
				if qualName == "len" {
					leftArgPosMype := NodGetChild(NodGetChild(n, NTR_BINOP_LEFT), NTR_MYPE_POS).Data.(Mype)
					intMype := MypeExplicitNewSingle(TY_INT)
					fmt.Println("left arg", PrettyPrintMype(NodGetChild(n, NTR_BINOP_LEFT)))
					return leftArgPosMype.ContainsAnyType(getLengthableTypes()) &&
						resulPosMype.WouldChangeFromUnionWith(intMype)
				}
			}
			return false
		},
		action: func(n Nod) {
			NodGetChild(n, NTR_MYPE_POS).Data = (NodGetChild(n,
				NTR_MYPE_POS).Data.(Mype).Union(MypeExplicitNewSingle(TY_INT)))
		},
	}
}

func marNegOpRestrictRules() []*RewriteRule {
	oers := marGetCompactOpEvaluateRules()

	opToallowableResult := map[int]Mype{}
	for _, oer := range oers {
		var allowableResult Mype
		if _, ok := opToallowableResult[oer.operator]; !ok {
			allowableResult = MypeExplicitNewEmpty()
			opToallowableResult[oer.operator] = allowableResult
		} else {
			allowableResult = opToallowableResult[oer.operator]
		}
		allowableResult = allowableResult.Union(MypeExplicitNewSingle(oer.result))
		opToallowableResult[oer.operator] = allowableResult
	}
	rv := []*RewriteRule{} // one rewrite rule per operator
	for operatorType, allowableResult := range opToallowableResult {
		rv = append(rv, createNegOpResultRestrictRule(operatorType, allowableResult))
	}
	return rv
}

func createNegOpResultRestrictRule(operatorType int, allowableResult Mype) *RewriteRule {
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == operatorType {
				resultMype := NodGetChild(n, NTR_MYPE_NEG).Data.(Mype)
				if resultMype.WouldChangeFromIntersectionWith(allowableResult) {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			NodGetChild(n, NTR_MYPE_NEG).Data = NodGetChild(n, NTR_MYPE_NEG).Data.(Mype).Intersection(allowableResult)
		},
	}
}

func createPosRewriteRuleFromOpEvaluateRule(oer *MypeOpEvaluateRule) *RewriteRule {
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == oer.operator {
				resultMype := NodGetChild(n, NTR_MYPE_POS).Data.(Mype)
				argMypes := []Mype{
					NodGetChild(NodGetChild(n, NTR_BINOP_LEFT), NTR_MYPE_POS).Data.(Mype),
					NodGetChild(NodGetChild(n, NTR_BINOP_RIGHT), NTR_MYPE_POS).Data.(Mype),
				}
				if resultMype.IsEmpty() && argMypes[0].ContainsSingleType(oer.operand) &&
					argMypes[1].ContainsSingleType(oer.operand) {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			NodGetChild(n, NTR_MYPE_POS).Data = MypeExplicitNewSingle(oer.result)
		},
	}
}

func writeTypeAndData(dst Nod, src Nod) {
	dst.NodeType = src.NodeType
	dst.Data = src.Data
}

func marPosLiterals() *RewriteRule {
	return &RewriteRule{
		condition: func(n Nod) bool {
			if isLiteralNodeType(n.NodeType) {
				if NodGetChild(n, NTR_MYPE_POS).Data.(Mype).IsEmpty() {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			ty := getLiteralTypeAnnDataFromNT(n.NodeType)
			NodGetChild(n, NTR_MYPE_POS).Data = MypeExplicitNewSingle(ty)
		},
	}
}

func isNodOfTypeString(n Nod, cmp string) bool {
	if nt := NodGetChildOrNil(n, NTR_TYPE); nt != nil {
		if tn, ok := nt.Data.(string); ok {
			return tn == cmp
		}
	}
	return false
}

func (x *XformerPocket) applyRewritesUntilStable(nods []Nod, rules []*RewriteRule) {
	for {
		maxApplied := 0
		for _, rule := range rules {
			nApplied := x.applyRewriteRuleOnJust(nods, rule)
			if nApplied > maxApplied {
				maxApplied = nApplied
			}
		}
		if maxApplied == 0 {
			break
		}
	}
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

func (x *XformerPocket) buildVarDefTables() {
	// find all variable assignments and construct
	// the canonical union of variables

	// TODO: support function parameters as well as just varassigns
	// determine which (top-level) imperatives have which vars
	varReferences := x.SearchRoot(func(n Nod) bool {
		return n.NodeType == NT_VARASSIGN || n.NodeType == NT_PARAMETER
	})
	impToVarRef := make(map[Nod][]Nod)
	for _, ele := range varReferences {
		funcDef := x.findTopLevelFuncDef(ele)
		if funcDef == nil {
			panic("failed to find top level imperative")
		}
		impToVarRef[funcDef] = append(impToVarRef[funcDef], ele)
	}

	// next, generate the vartables for each imperative
	for imper, varRefs := range impToVarRef {
		varTable := x.generateVarTableFromVarRefs(varRefs)
		NodSetChild(imper, NTR_VARTABLE, varTable)
	}

	x.linkVarsToVarTables()
	x.computeVariableScopes()

}

func (x *XformerPocket) linkVarsToVarTables() {

	// link up all var assignments and var getters to refer to this unified table
	varRefs := x.SearchRoot(func(n Nod) bool {
		return isVarReferenceNT(n.NodeType)
	})
	for _, varRef := range varRefs {
		// get the var table associated with this var reference
		funcDef := x.findTopLevelFuncDef(varRef)
		if funcDef == nil {
			panic("failed to find top level imperative")
		}
		varTable := NodGetChildOrNil(funcDef, NTR_VARTABLE)
		if varTable == nil {
			panic("failed to find vartable for top level imperative")
		}

		// get the var name as a string, which serves as the lookup key in the var table
		varName := x.getVarNameFromVarRef(varRef)

		// search the varTable for the name (naive linear search but should always be small list)
		varDefs := NodGetChildList(varTable)
		var matchedVarDef Nod
		for _, varDef := range varDefs {
			varDefVarName := NodGetChild(varDef, NTR_VARDEF_NAME).Data.(string)
			if varDefVarName == varName {
				matchedVarDef = varDef
				break
			}
		}
		if matchedVarDef == nil {
			panic("unknown var: '" + varName + "'")
		}
		// finally, store a reference to the definition
		NodSetChild(varRef, NTR_VARDEF, matchedVarDef)
	}

}

func (x *XformerPocket) computeVariableScopes() {
	varTables := x.SearchRoot(func(n Nod) bool { return n.NodeType == NT_VARTABLE })
	for _, varTable := range varTables {
		varDefs := NodGetChildList(varTable)
		for _, varDef := range varDefs {
			varScope := VSCOPE_FUNCLOCAL
			incomingNods := varDef.In
			for _, inEdge := range incomingNods {
				inNod := inEdge.In
				if inNod.NodeType == NT_PARAMETER {
					varScope = VSCOPE_FUNCPARAM
					break
				}
			}
			// scope has been computed, now save it
			NodSetChild(varDef, NTR_VARDEF_SCOPE, NodNewData(NT_VARDEF_SCOPE, varScope))
		}
	}
}

func (x *XformerPocket) getVarNameFromVarRef(varRef Nod) string {
	if varRef.NodeType == NT_PARAMETER {
		return NodGetChild(varRef, NTR_VARDEF_NAME).Data.(string)
	} else if varRef.NodeType == NT_VAR_GETTER {
		return NodGetChild(varRef, NTR_VAR_GETTER_NAME).Data.(string)
	} else if varRef.NodeType == NT_VARASSIGN {
		return NodGetChild(varRef, NTR_VAR_NAME).Data.(string)
	} else {
		panic("unhandled var ref type")
	}
}

func (x *XformerPocket) generateVarTableFromVarRefs(varRefs []Nod) Nod {
	varDefsByName := make(map[string]Nod)
	for _, varRef := range varRefs {
		varName := x.getVarNameFromVarRef(varRef)
		varDef := NodNew(NT_VARDEF)
		NodSetChild(varDef, NTR_VARDEF_NAME, NodNewData(NT_IDENTIFIER, varName))
		varDefsByName[varName] = varDef
	}

	varDefsList := make([]Nod, 0)
	for _, varDef := range varDefsByName {
		varDefsList = append(varDefsList, varDef)
	}
	return NodNewChildList(NT_VARTABLE, varDefsList)
}

func (x *XformerPocket) findTopLevelFuncDef(n Nod) Nod {
	found := x.SearchFor(n, func(n Nod) bool {
		return n.NodeType == NT_FUNCDEF
	}, func(n Nod) []Nod {
		return x.AllInNodes(n)
	})

	return found[0]
}

func (x *XformerPocket) parseInlineOpStream(opStream Nod) Nod {
	// converts an inline op stream to a proper prioritized tree representation
	// for now assume all elements are same priority and group left to right
	priGroups := [][]int{
		[]int{NT_DOTOP},
		[]int{NT_MULOP, NT_DIVOP},
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
