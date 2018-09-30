package xform

import (
	"fmt"
	. "pocket-lang/frontend/pocket/common"
	. "pocket-lang/parse"
)

func (x *XformerPocket) getAllSolveTypeRules() []*RewriteRule {
	generalRules := x.getAllGeneralMARRules()
	positiveRules := x.getAllPositiveMARRules()
	negativeRules := x.getAllNegativeMARRules()
	rv := append(generalRules, positiveRules...)
	rv = append(rv, negativeRules...)
	return rv
}

func (x *XformerPocket) getAllGeneralMARRules() []*RewriteRule {
	// get type annotation rules that don't apply specifically to
	// positive or negative mypes
	return []*RewriteRule{
		x.marGenLinkVarRefsToVarDef(),
		x.marRemoveMypesFromDotopQualifiers(),
	}
}

func (x *XformerPocket) getAllNegativeMARRules() []*RewriteRule {
	rv := []*RewriteRule{
		x.marNegDeclaredType(),
		x.marNegVarAssign(),
	}
	rv = append(rv, x.marNegOpRestrictRules()...)
	return rv
}

func (x *XformerPocket) getAllPositiveMARRules() []*RewriteRule {
	rv := []*RewriteRule{
		x.marPosPrimitiveLiterals(),
		x.marPosCollectionLiterals(),
		x.marPosFunctionRefs(),
		x.marPosRefOps(),
		x.marPosVarAssign(),
		x.marPosPublicParameter(),
		x.marPosPublicClassField(),
		x.marPosSelf(),
		x.marPosSysFunc(),
		x.marPosVarFunc(),
		x.marPosObjInitUser(),
		x.marPosFuncCallUser(),
		x.marPosReturnValue(),
	}
	rv = append(rv, x.marPosOpEvaluateRules()...)
	return rv
}

func (x *XformerPocket) getInitMypeNodFull() Nod {
	return NodNew(DYPE_ALL)
}

func (x *XformerPocket) getInitMypeNodEmpty() Nod {
	return NodNew(DYPE_EMPTY)
}

func (x *XformerPocket) initializePosNegMypes(n Nod) {
	NodSetChild(n, NTR_MYPE_NEG, x.getInitMypeNodFull())
	NodSetChild(n, NTR_MYPE_POS, x.getInitMypeNodEmpty())
}

func isMypedValueType(nt int) bool {
	return isLiteralNodeType(nt) || isBinaryOpType(nt) || isUnaryOpType(nt) ||
		isRValVarReferenceNT(nt) || isCallType(nt)
}

func isImperativeType(nt int) bool {
	return nt == NT_IMPERATIVE || nt == NT_RECEIVERCALL_CMD ||
		nt == NT_RETURN || nt == NT_FOR || nt == NT_WHILE || nt == NT_LOOP ||
		nt == NT_IF
}

func (x *XformerPocket) colorTypes() {
	// generate the "valid" mypes by intersecting the negative with the positive
	// then output a single "type color" for each myped node
	nodes := x.SearchRoot(func(n Nod) bool { return NodHasChild(n, NTR_MYPE_POS) })
	x.generateValidMypes(nodes)
	fmt.Println("after generating valid mypes:", PrettyPrintMypes(nodes))
	x.convertValidMypesToFinalTypes()
	fmt.Println("Final type assignments:", PrettyPrintMypes(nodes))

}

func (x *XformerPocket) marRemoveMypesFromDotopQualifiers() *RewriteRule {
	// there is no inherent type of the right side of a dot expression
	// e.g.  in obj.x, the ".x" should be typeless
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_DOTOP_QUALIFIER {
				if NodHasChild(n, NTR_MYPE_POS) {
					NodRemoveChild(n, NTR_MYPE_POS)
					NodRemoveChild(n, NTR_MYPE_NEG)
					return true
				}
			}
			return false
		},
	}
}

func (x *XformerPocket) marPosRefOps() *RewriteRule {
	// type of a ref op depends on what it refers to
	// currently, we only support function refs
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_REFERENCEOP {
				// TODO: maybe remove below if unimportant (dont remember now)
				// lval := NodGetChild(n, NTR_RECEIVERCALL_ARG)
				// lvalMype := NodGetChild(lval, NTR_MYPE_POS)
				funcMype := NodNewData(NT_TYPEBASE, TY_FUNC)
				extMype := NodGetChild(n, NTR_MYPE_POS)
				return x.RICUnion(extMype, funcMype)
			}
			return false
		},
	}
}

func (x *XformerPocket) marPosFunctionRefs() *RewriteRule {
	// type of a function ref is a func
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_IDENTIFIER_RESOLVED {
				if NodHasChild(n, NTR_MYPE_POS) {
					if NodHasChild(n, NTR_FUNCDEF) {
						candMype := NodNewData(NT_TYPEBASE, TY_FUNC)
						extMype := NodGetChild(n, NTR_MYPE_POS)
						return x.RICUnion(extMype, candMype)
					}
				}
			}
			return false
		},
	}
}

func (x *XformerPocket) marGenLinkVarRefsToVarDef() *RewriteRule {
	// link up mypes of variable references to point to the same
	// mypes as their definitions
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_VAR_GETTER || n.NodeType == NT_VARASSIGN ||
				n.NodeType == NT_PARAMETER {
				if varDef := NodGetChildOrNil(n, NTR_VARDEF); varDef != nil {
					if !NodHasChild(varDef, NTR_MYPE_POS) {
						// handle the case if the vardef isn't initialized mype-wise
						NodSetChild(varDef, NTR_MYPE_POS, NodGetChild(n, NTR_MYPE_POS))
						NodSetChild(varDef, NTR_MYPE_NEG, NodGetChild(n, NTR_MYPE_NEG))
						return true
					}
					varDefMypePos := NodGetChild(varDef, NTR_MYPE_POS)
					myMypePos := NodGetChild(n, NTR_MYPE_POS)
					if varDefMypePos != myMypePos {
						// if here, inherit the mypes from the extant vardef
						NodSetChild(n, NTR_MYPE_POS, NodGetChild(varDef, NTR_MYPE_POS))
						NodSetChild(n, NTR_MYPE_NEG, NodGetChild(varDef, NTR_MYPE_NEG))
						return true
					}
				}
			}
			return false
		},
	}
}

func (x *XformerPocket) generateValidMypes(nodes []Nod) {
	for _, node := range nodes {
		posMype := NodGetChild(node, NTR_MYPE_POS).Data.(Mype)
		negMype := NodGetChild(node, NTR_MYPE_NEG).Data.(Mype)
		validMype := posMype.Intersection(negMype)

		if validMype.IsEmpty() {
			if node.NodeType == NT_RECEIVERCALL_CMD || node.NodeType == NT_FUNCDEF_RV_PLACEHOLDER ||
				node.NodeType == NT_RECEIVERCALL_METHOD {
				// this is acceptable for these node types
			} else {
				panic("couldn't find a valid type for node: " + PrettyPrintMype(node))
			}
		}
		NodSetChild(node, NTR_MYPE_VALID, NodNewData(NT_MYPE, validMype))
		NodRemoveChild(node, NTR_MYPE_POS)
		NodRemoveChild(node, NTR_MYPE_NEG)
	}
}

func (x *XformerPocket) convertValidMypesToFinalTypes() {
	// explicitly convert mypes to types (and remove the mypes in the process)
	// these final types are explicitly for the generator's sake
	x.applyRewriteOnGraph(&RewriteRule{
		condition: func(n Nod) bool {
			return NodHasChild(n, NTR_MYPE_VALID)
		},
		action: func(n Nod) {
			mype := NodGetChild(n, NTR_MYPE_VALID).Data.(Mype)
			NodSetChild(n, NTR_TYPE, XMypeToFinalType(mype))
			NodRemoveChild(n, NTR_MYPE_VALID)
		},
	})
}

func (x *XformerPocket) marPosFuncCallUser() *RewriteRule {
	// calls to user functions should link to the funcdef's return type
	// TODO: this logic isn't the best; we don't want the use of a function to affect
	// it's potential return type
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_RECEIVERCALL || n.NodeType == NT_RECEIVERCALL_CMD ||
				n.NodeType == NT_RECEIVERCALL_METHOD {
				if NodHasChild(n, NTR_FUNCDEF) {
					fDef := NodGetChild(n, NTR_FUNCDEF)
					myPosMype := NodGetChild(n, NTR_MYPE_POS)
					fDefRVPosMype := NodGetChild(NodGetChild(fDef, NTR_RETURNVAL_PLACEHOLDER), NTR_MYPE_POS)
					if myPosMype != fDefRVPosMype {
						rvPlaceholder := NodGetChild(fDef, NTR_RETURNVAL_PLACEHOLDER)
						NodSetChild(n, NTR_MYPE_POS, NodGetChild(rvPlaceholder, NTR_MYPE_POS))
						NodSetChild(n, NTR_MYPE_NEG, NodGetChild(rvPlaceholder, NTR_MYPE_NEG))
						return true
					}
				}
			}
			return false
		},
	}
}

func (x *XformerPocket) marPosObjInitUser() *RewriteRule {
	// Type.new(x) or Type(x) returns type Type for user-defined classes
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_OBJINIT {
				base := NodGetChild(n, NTR_RECEIVERCALL_BASE)
				if base.NodeType == NT_CLASSDEF || base.NodeType == NT_TYPEBASE {
					return x.RICUnion(NodGetChild(n, NTR_MYPE_POS), base)
				}
			}
			return false
		},
	}
}

func marPosCollectionGetArgedCand(elements []Nod) Nod {
	accum := NodNew(DYPE_EMPTY)
	for _, element := range elements {
		elementPosMype := NodGetChild(element, NTR_MYPE_POS)
		accum = DypeDeduplicate(DypeUnion(accum, elementPosMype))
	}
	accum = DypeSimplify(accum)
	if accum.NodeType == DYPE_EMPTY {
		return accum
	} else {
		rv := NodNew(NT_TYPECALL)
		NodSetChild(rv, NTR_RECEIVERCALL_BASE, NodNewData(NT_TYPEBASE, TY_LIST))
		NodSetChild(rv, NTR_RECEIVERCALL_ARG, accum)
		return rv
	}
}

func marPosCollectionEvaluateMype(n Nod) Nod {
	candMype := marPosCollectionGetArgedCand(NodGetChildList(n))
	untypedMype := NodNewData(NT_TYPEBASE, TY_LIST)
	allMypes := DypeSimplify(DypeUnion(candMype, untypedMype))
	return allMypes
}

func (x *XformerPocket) RICUnion(old Nod, new Nod) bool {
	if DypeWouldChangeUnion(old, new) {
		x.Replace(old, DypeUnion(old, new))
		return true
	}
	return false
}

func (x *XformerPocket) RICXSect(old Nod, new Nod) bool {
	if DypeWouldChangeXSect(old, new) {
		x.Replace(old, DypeXSect(old, new))
		return true
	}
	return false
}

func (x *XformerPocket) marPosCollectionLiterals() *RewriteRule {
	// [3, 4, 5] -+> {list, list<int>}
	return &RewriteRule{
		condaction: func(n Nod) bool {
			// TODO: support maps and sets
			if n.NodeType == NT_LIT_LIST {
				// evaluating this can be expensive, there might be a need
				// to optimize this at some point
				return x.RICUnion(NodGetChild(n, NTR_MYPE_POS), marPosCollectionEvaluateMype(n))
			}
			return false
		},
	}
}

func XMypeToFinalType(arg Mype) Nod {
	if ma, ok := arg.(*MypeArged); ok {
		return XMypeArgedNodToFinalType(ma.Node)
	} else {
		panic("only MypeArged supported")
	}
}

func XMypeArgedNodToFinalType(n Nod) Nod {
	if n.NodeType == MATYPE_SINGLE_BASE {
		return NodNewData(NT_TYPEBASE, n.Data.(int))
	} else if n.NodeType == NT_CLASSDEF {
		return n
	} else if n.NodeType == MATYPE_SINGLE_ARGED {
		base := NodNewData(NT_TYPEBASE, NodGetChild(n, MATYPER_BASE).Data.(int))
		arg := XMypeArgedNodToFinalType(NodGetChild(n, MATYPER_ARG))
		rv := NodNew(NT_TYPEARGED)
		NodSetChild(rv, NTR_TYPEARGED_BASE, base)
		NodSetChild(rv, NTR_TYPEARGED_ARG, arg)
		return rv
	} else if n.NodeType == MATYPE_ALL {
		return NodNewData(NT_TYPEBASE, TY_DUCK)
	} else if n.NodeType == MATYPE_UNION {
		unionNods := NodGetChildList(n)
		fmt.Println("union nodes are", PrettyPrintNodes(unionNods))
		commonStem := MANodComputeGreatestCommonStem(NodGetChildList(n))
		if commonStem != nil {
			return XMypeArgedNodToFinalType(commonStem)
		} else {
			return NodNewData(NT_TYPEBASE, TY_DUCK)
		}
	} else if n.NodeType == MATYPE_EMPTY {
		return NodNewData(NT_TYPEBASE, TY_VOID)
	} else {
		panic("unhandled")
	}
}

func (x *XformerPocket) marPosVarFunc() *RewriteRule {
	// for now, all calls to a variable can return anything
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if isReceiverCallType(n.NodeType) {
				// alternate syntax: vargetter is the base of the call
				// TODO: possibly remove in favor of the FUNCDEF syntax
				if NodGetChild(n, NTR_RECEIVERCALL_BASE).NodeType == NT_VAR_GETTER {
					return x.RICUnion(NodGetChild(n, NTR_MYPE_POS), NodNew(DYPE_ALL))
				}
			}
			return false
		},
	}
}

func (x *XformerPocket) marPosSysFunc() *RewriteRule {
	// all sys funcs can return anything
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if isReceiverCallType(n.NodeType) {
				baseNod := NodGetChild(n, NTR_RECEIVERCALL_BASE)
				if baseNod.NodeType == NT_IDENTIFIER_FUNC_NOSCOPE {
					rcName := baseNod.Data.(string)
					if isSystemFuncName(rcName) {
						return x.RICUnion(NodGetChild(n, NTR_MYPE_POS), NodNew(DYPE_ALL))
					}
				}
			}
			return false
		},
	}
}

func (x *XformerPocket) marPosPublicParameter() *RewriteRule {
	// assume that assignments to this parameter are in fact called
	// with every allowable type
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_PARAMETER {
				// return x.RICUnion(NodGetChild(n, NTR_MYPE_POS), NodNew(DYPE_ALL))
				typeDecl := NodGetChildOrNil(n, NTR_TYPE_DECL)
				var allowedMype Nod
				if typeDecl == nil {
					allowedMype = NodNew(DYPE_ALL)
				} else {
					allowedMype = typeDecl
				}
				return x.RICUnion(NodGetChild(n, NTR_MYPE_POS), allowedMype)
			}
			return false
		},
	}
}

func marPosPublicClassFieldGetCandMype(classField Nod) Nod {
	// get the assumed maximal type from a class field using its type decl
	typeDecl := NodGetChildOrNil(classField, NTR_TYPE_DECL)
	if typeDecl == nil {
		return NodNew(DYPE_ALL)
	}
	if typeDecl.NodeType == NT_CLASSDEF || typeDecl.NodeType == NT_TYPEBASE {
		return typeDecl
	}
	return nil // means we can't deduce anything now
}

func (x *XformerPocket) marPosPublicClassField() *RewriteRule {
	// assume that assignments to this field are maximal
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_CLASSFIELD {
				varDef := NodGetChild(n, NTR_VARDEF)
				candMype := marPosPublicClassFieldGetCandMype(n)

				if candMype != nil {
					return x.RICUnion(NodGetChild(varDef, NTR_MYPE_POS), candMype)
				}
			}
			return false
		},
	}
}

func (x *XformerPocket) marPosSelf() *RewriteRule {
	// type of 'self' is the current class
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_VARDEF {
				varName := NodGetChild(n, NTR_VARDEF_NAME).Data.(string)
				if varName == "self" {
					cCls := x.getContainingClassDef(n)
					if cCls != nil {
						return x.RICUnion(NodGetChild(n, NTR_MYPE_POS), cCls)
					}
				}
			}
			return false
		},
	}
}

func (x *XformerPocket) marPosVarAssign() *RewriteRule {
	// propagate var assign values from rhs -> lhs
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_VARASSIGN {
				dypeLHS := NodGetChild(n, NTR_MYPE_POS)
				dypeRHS := NodGetChild(NodGetChild(n, NTR_VARASSIGN_VALUE), NTR_MYPE_POS)
				return x.RICUnion(dypeLHS, dypeRHS)
			}
			return false
		},
	}
}

func (x *XformerPocket) marPosReturnValue() *RewriteRule {
	// propagate return values into the placeholder
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_RETURN {
				if lhs := NodGetChildOrNil(n, NTR_RETURNVAL_PLACEHOLDER); lhs != nil {
					mypeLHS := NodGetChild(lhs, NTR_MYPE_POS)
					mypeRHS := NodGetChild(NodGetChild(n, NTR_RETURN_VALUE), NTR_MYPE_POS)
					return x.RICUnion(mypeLHS, mypeRHS)
				}
			}
			return false
		},
	}
}

func (x *XformerPocket) marNegVarAssign() *RewriteRule {
	// propagate var type restrictions from lhs -> rhs
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_VARASSIGN {
				mypeLHS := NodGetChild(n, NTR_MYPE_NEG)
				mypeRHS := NodGetChild(NodGetChild(n, NTR_VARASSIGN_VALUE), NTR_MYPE_NEG)
				return x.RICXSect(mypeLHS, mypeRHS)
			}
			return false
		},
	}
}

func (x *XformerPocket) marNegDeclaredType() *RewriteRule {
	// propagate type declarations to the base dypes of parameters, var assignment, and var defs
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_PARAMETER || n.NodeType == NT_VARASSIGN ||
				n.NodeType == NT_VARDEF {
				if typeDeclNod := NodGetChildOrNil(n, NTR_TYPE_DECL); typeDeclNod != nil {
					if typeDeclNod.NodeType == NT_TYPEBASE || typeDeclNod.NodeType == NT_CLASSDEF {
						negMype := NodGetChild(n, NTR_MYPE_NEG)
						declMype := typeDeclNod
						return x.RICXSect(negMype, declMype)
					} else {
						fmt.Println("interesting situation: got type decl but wasn't supported:",
							PrettyPrint(typeDeclNod))
					}
				}
			}
			return false
		},
	}
}

// "low" and "high" refer to the canonical order of types (to avoid duplication issues with commutativity)
type MypeOpEvaluateRule struct {
	operator    int
	operandLow  int
	operandHigh int
	result      int
}

func marGetCompactOpEvaluateRules() []*MypeOpEvaluateRule {
	// define type propagation rules of the form (int + int) -> int
	return []*MypeOpEvaluateRule{
		// TODO: potentially compress this table
		&MypeOpEvaluateRule{NT_ADDOP, TY_INT, TY_INT, TY_INT},
		&MypeOpEvaluateRule{NT_ADDOP, TY_INT, TY_FLOAT, TY_FLOAT},
		&MypeOpEvaluateRule{NT_ADDOP, TY_FLOAT, TY_FLOAT, TY_FLOAT},
		&MypeOpEvaluateRule{NT_ADDOP, TY_STRING, TY_STRING, TY_STRING},

		&MypeOpEvaluateRule{NT_SUBOP, TY_INT, TY_INT, TY_INT},
		&MypeOpEvaluateRule{NT_SUBOP, TY_FLOAT, TY_FLOAT, TY_FLOAT},
		&MypeOpEvaluateRule{NT_SUBOP, TY_INT, TY_FLOAT, TY_FLOAT},

		&MypeOpEvaluateRule{NT_MULOP, TY_INT, TY_INT, TY_INT},
		&MypeOpEvaluateRule{NT_MULOP, TY_FLOAT, TY_FLOAT, TY_FLOAT},
		&MypeOpEvaluateRule{NT_MULOP, TY_INT, TY_FLOAT, TY_FLOAT},

		&MypeOpEvaluateRule{NT_DIVOP, TY_INT, TY_INT, TY_INT},
		&MypeOpEvaluateRule{NT_DIVOP, TY_FLOAT, TY_FLOAT, TY_FLOAT},
		&MypeOpEvaluateRule{NT_DIVOP, TY_INT, TY_FLOAT, TY_FLOAT},

		&MypeOpEvaluateRule{NT_MODOP, TY_INT, TY_INT, TY_INT},
		&MypeOpEvaluateRule{NT_MODOP, TY_FLOAT, TY_FLOAT, TY_FLOAT},
		&MypeOpEvaluateRule{NT_MODOP, TY_INT, TY_FLOAT, TY_FLOAT},

		&MypeOpEvaluateRule{NT_GTOP, TY_INT, TY_INT, TY_BOOL},
		&MypeOpEvaluateRule{NT_GTOP, TY_FLOAT, TY_FLOAT, TY_BOOL},

		&MypeOpEvaluateRule{NT_LTOP, TY_INT, TY_INT, TY_BOOL},
		&MypeOpEvaluateRule{NT_LTOP, TY_FLOAT, TY_FLOAT, TY_BOOL},

		&MypeOpEvaluateRule{NT_GTEQOP, TY_INT, TY_INT, TY_BOOL},
		&MypeOpEvaluateRule{NT_GTEQOP, TY_FLOAT, TY_FLOAT, TY_BOOL},

		&MypeOpEvaluateRule{NT_LTEQOP, TY_INT, TY_INT, TY_BOOL},
		&MypeOpEvaluateRule{NT_LTEQOP, TY_FLOAT, TY_FLOAT, TY_BOOL},

		&MypeOpEvaluateRule{NT_EQOP, TY_INT, TY_INT, TY_BOOL},
		&MypeOpEvaluateRule{NT_EQOP, TY_FLOAT, TY_FLOAT, TY_BOOL},
		&MypeOpEvaluateRule{NT_EQOP, TY_STRING, TY_STRING, TY_BOOL},
		&MypeOpEvaluateRule{NT_EQOP, TY_BOOL, TY_BOOL, TY_BOOL},

		&MypeOpEvaluateRule{NT_OROP, TY_BOOL, TY_BOOL, TY_BOOL},

		&MypeOpEvaluateRule{NT_ANDOP, TY_BOOL, TY_BOOL, TY_BOOL},
	}
}

func (x *XformerPocket) marPosOpEvaluateRules() []*RewriteRule {
	oers := marGetCompactOpEvaluateRules()
	rv := []*RewriteRule{}
	for _, oer := range oers {
		rv = append(rv, x.createPosRewriteRuleFromOpEvaluateRule(oer))
	}

	rv = append(rv, x.marPosOpCollectionLenRule())

	return rv
}

func getLengthableTypes() []int {
	return []int{TY_LIST, TY_MAP, TY_SET}
}

func getLengthableDype() Nod {
	ncs := []Nod{}
	ltys := getLengthableTypes()
	for _, lty := range ltys {
		ncs = append(ncs, NodNewData(NT_TYPEBASE, lty))
	}
	return NodNewChildList(DYPE_UNION, ncs)
}

func (x *XformerPocket) marPosOpCollectionLenRule() *RewriteRule {
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_DOTOP {
				if qualName, ok := NodGetChild(n, NTR_BINOP_RIGHT).Data.(string); ok {
					if qualName == "len" {
						resulPosMype := NodGetChild(n, NTR_MYPE_POS)
						leftArgPosMype := NodGetChild(NodGetChild(n, NTR_BINOP_LEFT), NTR_MYPE_POS)
						intMype := NodNewData(NT_TYPEBASE, TY_INT)
						isLengthable := DypeSimplify(DypeUnion(leftArgPosMype, getLengthableDype())).NodeType != DYPE_EMPTY
						if isLengthable {
							fmt.Println("applied collection len rule")
							return x.RICUnion(resulPosMype, intMype)
						}
					}
				}
			}
			return false
		},
	}
}

func (x *XformerPocket) marNegOpRestrictRules() []*RewriteRule {
	oers := marGetCompactOpEvaluateRules()

	opToallowableResult := map[int]Nod{}
	for _, oer := range oers {
		var allowableResult Nod
		if _, ok := opToallowableResult[oer.operator]; !ok {
			allowableResult = NodNew(DYPE_EMPTY)
			opToallowableResult[oer.operator] = allowableResult
		} else {
			allowableResult = opToallowableResult[oer.operator]
		}
		allowableResult = DypeUnion(allowableResult, NodNewData(NT_TYPEBASE, oer.result))
		opToallowableResult[oer.operator] = allowableResult
	}
	rv := []*RewriteRule{} // one rewrite rule per operator
	for operatorType, allowableResult := range opToallowableResult {
		rv = append(rv, x.createNegOpResultRestrictRule(operatorType, allowableResult))
	}
	return rv
}

func (x *XformerPocket) createNegOpResultRestrictRule(operatorType int, allowableResult Nod) *RewriteRule {
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == operatorType {
				resultMype := NodGetChild(n, NTR_MYPE_NEG)
				return x.RICXSect(resultMype, allowableResult)
			}
			return false
		},
	}
}

func (x *XformerPocket) createPosRewriteRuleFromOpEvaluateRule(oer *MypeOpEvaluateRule) *RewriteRule {
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == oer.operator {
				resultMype := NodGetChild(n, NTR_MYPE_POS)
				argMypes := []Nod{
					NodGetChild(NodGetChild(n, NTR_BINOP_LEFT), NTR_MYPE_POS),
					NodGetChild(NodGetChild(n, NTR_BINOP_RIGHT), NTR_MYPE_POS),
				}

				lowDype := NodNewData(NT_TYPEBASE, oer.operandLow)
				highDype := NodNewData(NT_TYPEBASE, oer.operandHigh)
				matchLowHigh := DypeIsSubset(argMypes[0], lowDype) &&
					DypeIsSubset(argMypes[1], highDype)
				matchHighLow := DypeIsSubset(argMypes[1], lowDype) &&
					DypeIsSubset(argMypes[0], highDype)

				// fmt.Println("oer.operator", oer.operator, "opHigh", oer.operandHigh,
				// 	"opLow", oer.operandLow, "matchLowHigh", matchLowHigh, "matchHighLow", matchHighLow)

				// todo: ensure we have precise semantics for arged types
				if matchLowHigh || matchHighLow {
					return x.RICUnion(resultMype, NodNewData(NT_TYPEBASE, oer.result))
				}
			}
			return false
		},
	}
}

func writeTypeAndData(dst Nod, src Nod) {
	dst.NodeType = src.NodeType
	dst.Data = src.Data
}

func (x *XformerPocket) marPosPrimitiveLiterals() *RewriteRule {
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if isPrimitiveLiteralNodeType(n.NodeType) {
				extDype := NodGetChild(n, NTR_MYPE_POS)
				if extDype.NodeType == DYPE_EMPTY {
					ty := getLiteralTypeAnnDataFromNT(n.NodeType)
					return x.RICUnion(extDype, NodNewData(NT_TYPEBASE, ty))
				}
			}
			return false
		},
	}
}
