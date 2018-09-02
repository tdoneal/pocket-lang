package pocket

import (
	"fmt"
	. "pocket-lang/parse"
	. "pocket-lang/xform"
	"strconv"
)

const (
	TY_BOOL   = 1
	TY_INT    = 2
	TY_FLOAT  = 3
	TY_STRING = 4
	TY_SET    = 5
	TY_MAP    = 6
	TY_LIST   = 7
	TY_OBJECT = 20
	TY_NUMBER = 22
	TY_DUCK   = 30
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
	return nt == NT_LIT_INT || nt == NT_LIT_STRING || nt == NT_LIT_BOOL
}

func getLiteralTypeAnnDataFromNT(nt int) int {
	lut := map[int]int{
		NT_LIT_INT:    TY_INT,
		NT_LIT_STRING: TY_STRING,
		NT_LIT_BOOL:   TY_BOOL,
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

	// general the "valid" mypes by subtracting the converse of the negative from the positive
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

	fmt.Println("after generating valid mypes:", PrettyPrintMypes(nodes))

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

	fmt.Println("Final type assignments:", PrettyPrintMypes(nodes))

}

func (x *XformerPocket) getAllNegativeMARRules() []*RewriteRule {
	return []*RewriteRule{
		marNegUseInOp(),
		marNegDeclaredType(),
	}
}

func (x *XformerPocket) getAllPositiveMARRules() []*RewriteRule {
	rv := []*RewriteRule{
		marPosLiterals(),
		marPosVarAssign(),
	}
	rv = append(rv, marPosOpEvaluateRules()...)
	return rv
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
				if NodHasChild(n, NTR_TYPE_DECL) {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			panic("encountered type decl")
		},
	}
}

type MypeOpEvaluateRule struct {
	operator int
	operand  int
	result   int
}

func marPosGetCompactOpEvaluateRules() []*MypeOpEvaluateRule {
	// define type propagation rules of the form (int + int) -> int
	return []*MypeOpEvaluateRule{
		&MypeOpEvaluateRule{NT_ADDOP, TY_INT, TY_INT},
		&MypeOpEvaluateRule{NT_GTOP, TY_INT, TY_BOOL},
		&MypeOpEvaluateRule{NT_LTOP, TY_INT, TY_BOOL},
	}
}

func marPosOpEvaluateRules() []*RewriteRule {
	oers := marPosGetCompactOpEvaluateRules()
	rv := []*RewriteRule{}
	for _, oer := range oers {
		rv = append(rv, createRewriteRuleFromOpEvaluateRule(oer))
	}
	return rv
}

func createRewriteRuleFromOpEvaluateRule(oer *MypeOpEvaluateRule) *RewriteRule {
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

func marNegUseInOp() *RewriteRule {
	// for now, say that all add ops force all involved mypes (both args and result) to int
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_ADDOP {
				resultMype := NodGetChild(n, NTR_MYPE_NEG).Data.(Mype)
				argMypes := []Mype{
					NodGetChild(NodGetChild(n, NTR_BINOP_LEFT), NTR_MYPE_NEG).Data.(Mype),
					NodGetChild(NodGetChild(n, NTR_BINOP_RIGHT), NTR_MYPE_NEG).Data.(Mype),
				}
				if resultMype.IsPlural() || argMypes[0].IsPlural() ||
					argMypes[1].IsPlural() {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			intMype := MypeExplicitNewSingle(TY_INT)
			resultMypeNod := NodGetChild(n, NTR_MYPE_NEG)
			leftMypeNod := NodGetChild(NodGetChild(n, NTR_BINOP_LEFT), NTR_MYPE_NEG)
			rightMypeNod := NodGetChild(NodGetChild(n, NTR_BINOP_RIGHT), NTR_MYPE_NEG)

			resultMypeNod.Data = intMype.Intersection(resultMypeNod.Data.(Mype))
			leftMypeNod.Data = intMype.Intersection(leftMypeNod.Data.(Mype))
			rightMypeNod.Data = intMype.Intersection(rightMypeNod.Data.(Mype))
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
		if NodHasChild(varRef, NTR_TYPE) {
			NodSetChild(varDef, NTR_TYPE, NodGetChild(varRef, NTR_TYPE))
		}
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
