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
}

func Xform(root Nod) Nod {
	fmt.Println("starting Xform()")

	xformer := &XformerPocket{
		&Xformer{},
	}

	xformer.Root = root
	xformer.Xform()
	return root
}

func (x *XformerPocket) Xform() {
	x.parseInlineOpStreams()

	x.buildVarDefTables()

	fmt.Println("after building var def tables:", PrettyPrint(x.Root))

	x.solveTypes()
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
	return nt == NT_LIT_INT || nt == NT_LIT_STRING || nt == NT_LIT_BOOL || nt == NT_LIT_FLOAT
}

func getLiteralTypeAnnDataFromNT(nt int) int {
	lut := map[int]int{
		NT_LIT_INT:    TY_INT,
		NT_LIT_STRING: TY_STRING,
		NT_LIT_BOOL:   TY_BOOL,
		NT_LIT_FLOAT:  TY_FLOAT,
	}
	if rv, ok := lut[nt]; ok {
		return rv
	} else {
		panic("unknown literal type: " + strconv.Itoa(nt))
	}
}

func isBinaryOpType(nt int) bool {
	return nt == NT_ADDOP || nt == NT_GTOP || nt == NT_LTOP
}

func isVarReferenceNT(nt int) bool {
	return nt == NT_VAR_GETTER || nt == NT_VARASSIGN || nt == NT_PARAMETER
}

func (x *XformerPocket) initializeMypes(n Nod) {
	NodSetChild(n, NTR_MYPE_NEG, x.getInitMypeNodFull())
	NodSetChild(n, NTR_MYPE_POS, x.getInitMypeNodEmpty())
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
		nt := n.NodeType
		return isLiteralNodeType(nt) || isBinaryOpType(nt) || isVarReferenceNT(nt)
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
	}
	rv = append(rv, marPosOpEvaluateRules()...)
	return rv
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
			if n.NodeType == NT_VARASSIGN {
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
		&MypeOpEvaluateRule{NT_GTOP, TY_INT, TY_BOOL},
		&MypeOpEvaluateRule{NT_LTOP, TY_INT, TY_BOOL},
		&MypeOpEvaluateRule{NT_GTEQOP, TY_INT, TY_BOOL},
		&MypeOpEvaluateRule{NT_LTEQOP, TY_INT, TY_BOOL},
	}
}

func marPosOpEvaluateRules() []*RewriteRule {
	oers := marGetCompactOpEvaluateRules()
	rv := []*RewriteRule{}
	for _, oer := range oers {
		rv = append(rv, createPosRewriteRuleFromOpEvaluateRule(oer))
	}
	return rv
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
				if resultMype.IsEmpty() && argMypes[0].IsSingleType(oer.operand) &&
					argMypes[1].IsSingleType(oer.operand) {
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
		tlImper := x.findTopLevelImperative(ele)
		if tlImper == nil {
			panic("failed to find top level imperative")
		}
		impToVarRef[tlImper] = append(impToVarRef[tlImper], ele)
	}

	// next, generate the vartables for each imperative
	for imper, varRefs := range impToVarRef {
		varTable := x.generateVarTableFromVarRefs(varRefs)
		NodSetChild(imper, NTR_VARTABLE, varTable)
	}

	fmt.Println("right before lvtovt", PrettyPrint(x.Root))

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
		tlImper := x.findTopLevelImperative(varRef)
		if tlImper == nil {
			panic("failed to find top level imperative")
		}
		varTable := NodGetChildOrNil(tlImper, NTR_VARTABLE)
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

func (x *XformerPocket) findTopLevelImperative(n Nod) Nod {
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
	opStreamNodes := NodGetChildList(opStream)
	streamNodes := make([]Nod, len(opStreamNodes))
	copy(streamNodes, opStreamNodes)
	output := streamNodes[0]
	i := 1
	for {
		if i+1 >= len(streamNodes) {
			break
		}
		op := streamNodes[i]
		right := streamNodes[i+1]
		opCopy := NodNew(op.NodeType)
		opCopy.Data = op.Data
		NodSetChild(opCopy, NTR_BINOP_LEFT, output)
		NodSetChild(opCopy, NTR_BINOP_RIGHT, right)
		output = opCopy
		i += 2
	}
	return output
}
