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
		marNegDeclaredType(),
		marNegVarAssign(),
	}
	rv = append(rv, marNegOpRestrictRules()...)
	return rv
}

func (x *XformerPocket) getAllPositiveMARRules() []*RewriteRule {
	rv := []*RewriteRule{
		marPosPrimitiveLiterals(),
		marPosCollectionLiterals(),
		marPosVarAssign(),
		marPosPublicParameter(),
		marPosPublicClassField(),
		x.marPosSelf(),
		marPosSysFunc(),
		marPosVarFunc(),
		marPosObjInitPrim(),
		marPosObjInitUser(),
		// marPosOwnField(),
		marPosFuncCallUser(),
		marPosReturnValue(),
	}
	rv = append(rv, marPosOpEvaluateRules()...)
	return rv
}

func (x *XformerPocket) getInitMypeNodFull() Nod {
	md := MypeArgedNewFull()
	return NodNewData(NT_MYPE, md)
}

func (x *XformerPocket) getInitMypeNodEmpty() Nod {
	md := MypeArgedNewEmpty()
	return NodNewData(NT_MYPE, md)
}

func (x *XformerPocket) initializePosNegMypes(n Nod) {
	NodSetChild(n, NTR_MYPE_NEG, x.getInitMypeNodFull())
	NodSetChild(n, NTR_MYPE_POS, x.getInitMypeNodEmpty())
}

func isMypedValueType(nt int) bool {
	return isLiteralNodeType(nt) || isBinaryOpType(nt) ||
		isRValVarReferenceNT(nt) || isCallType(nt)
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
		condition: func(n Nod) bool {
			if n.NodeType == NT_DOTOP_QUALIFIER {
				if NodHasChild(n, NTR_MYPE_POS) {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			NodRemoveChild(n, NTR_MYPE_POS)
			NodRemoveChild(n, NTR_MYPE_NEG)
		},
	}
}

func (x *XformerPocket) marGenLinkVarRefsToVarDef() *RewriteRule {
	// link up mypes of variable references to point to the same
	// mypes as their definitions
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_VAR_GETTER || n.NodeType == NT_VARASSIGN ||
				n.NodeType == NT_PARAMETER {
				if varDef := NodGetChildOrNil(n, NTR_VARDEF); varDef != nil {
					if !NodHasChild(varDef, NTR_MYPE_POS) {
						return true // handle the case if the vardef isn't initialized mype-wise
					}
					varDefMypePos := NodGetChild(varDef, NTR_MYPE_POS)
					myMypePos := NodGetChild(n, NTR_MYPE_POS)
					if varDefMypePos != myMypePos {
						return true
					}
				}
			}
			return false
		},
		action: func(n Nod) {
			varDef := NodGetChild(n, NTR_VARDEF)
			if !NodHasChild(varDef, NTR_MYPE_POS) {
				// intialize the mypes of the var def
				// optimization: initialize them to our defs
				NodSetChild(varDef, NTR_MYPE_POS, NodGetChild(n, NTR_MYPE_POS))
				NodSetChild(varDef, NTR_MYPE_NEG, NodGetChild(n, NTR_MYPE_NEG))
				return
			}

			// if here, inherit the mypes from the extant vardef
			NodSetChild(n, NTR_MYPE_POS, NodGetChild(varDef, NTR_MYPE_POS))
			NodSetChild(n, NTR_MYPE_NEG, NodGetChild(varDef, NTR_MYPE_NEG))
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

func marPosFuncCallUser() *RewriteRule {
	// calls to user functions should link to the funcdef's return type
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_RECEIVERCALL || n.NodeType == NT_RECEIVERCALL_CMD ||
				n.NodeType == NT_RECEIVERCALL_METHOD {
				if NodHasChild(n, NTR_FUNCDEF) {
					fDef := NodGetChild(n, NTR_FUNCDEF)
					myPosMype := NodGetChild(n, NTR_MYPE_POS)
					fDefRVPosMype := NodGetChild(NodGetChild(fDef, NTR_RETURNVAL_PLACEHOLDER), NTR_MYPE_POS)
					if myPosMype != fDefRVPosMype {
						return true
					}
				}
			}
			return false
		},
		action: func(n Nod) {
			// link ourselves to the func def's return type
			fDef := NodGetChild(n, NTR_FUNCDEF)
			rvPlaceholder := NodGetChild(fDef, NTR_RETURNVAL_PLACEHOLDER)
			NodSetChild(n, NTR_MYPE_POS, NodGetChild(rvPlaceholder, NTR_MYPE_POS))
			NodSetChild(n, NTR_MYPE_NEG, NodGetChild(rvPlaceholder, NTR_MYPE_NEG))

		},
	}
}

// func marPosOwnField() *RewriteRule {
// 	// refernces to internal class variables -> all (for now)
// 	// TODO: support stricter typing
// 	return &RewriteRule{
// 		condition: func(n Nod) bool {
// 			if n.NodeType == NT_VAR_GETTER && NodHasChild(n, NTR_VARDEF) {
// 				varDef := NodGetChild(n, NTR_VARDEF)
// 				if scope := NodGetChildOrNil(varDef, NTR_VARDEF_SCOPE); scope != nil {
// 					if scope.Data.(int) == VSCOPE_CLASSFIELD {

// 						// we don't properly support typed fields yet, so
// 						// just say it could be anything
// 						return !NodGetChild(n, NTR_MYPE_POS).Data.(Mype).IsFull()
// 					}
// 				}

// 			}
// 			return false
// 		},
// 		action: func(n Nod) {
// 			NodGetChild(n, NTR_MYPE_POS).Data = XMypeNewFull()
// 		},
// 	}
// }

func marPosObjInitUser() *RewriteRule {
	// Type.new(x) or Type(x) returns type Type for user-defined classes
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_OBJINIT {
				base := NodGetChild(n, NTR_RECEIVERCALL_BASE)
				if base.NodeType == NT_CLASSDEF {
					candMype := XMypeNewSingleClassDef(base)
					return NodGetChild(n, NTR_MYPE_POS).Data.(Mype).WouldChangeFromUnionWith(candMype)
				}
			}
			return false
		},
		action: func(n Nod) {
			base := NodGetChild(n, NTR_RECEIVERCALL_BASE)
			NodGetChild(n, NTR_MYPE_POS).Data = NodGetChild(n, NTR_MYPE_POS).Data.(Mype).Union(
				XMypeNewSingleClassDef(base))
		},
	}
}

func marPosObjInitPrim() *RewriteRule {
	// Type.new(x) or Type(x) returns type Type for primitive types
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_OBJINIT {
				base := NodGetChild(n, NTR_RECEIVERCALL_BASE)
				if base.NodeType == NT_TYPEBASE {
					candMype := XMypeNewSingleBase(base.Data.(int))
					return NodGetChild(n, NTR_MYPE_POS).Data.(Mype).WouldChangeFromUnionWith(candMype)
				}
			}
			return false
		},
		action: func(n Nod) {
			baseTy := NodGetChild(n, NTR_RECEIVERCALL_BASE).Data.(int)
			NodGetChild(n, NTR_MYPE_POS).Data = XMypeNewSingleBase(baseTy).Union(
				NodGetChild(n, NTR_MYPE_POS).Data.(Mype))
		},
	}
}

func marPosCollectionGetArgedCand(elements []Nod) Mype {
	accum := XMypeNewEmpty()
	for _, element := range elements {
		elementPosMype := NodGetChild(element, NTR_MYPE_POS).Data.(Mype)
		accum = accum.Union(elementPosMype)
	}
	if accum.IsEmpty() {
		return XMypeNewEmpty()
	} else {
		return XMypeNewSingleArged(TY_LIST, accum)
	}
}

func marPosCollectionEvaluateMype(n Nod) Mype {
	candMype := marPosCollectionGetArgedCand(NodGetChildList(n))
	untypedMype := XMypeNewSingleBase(TY_LIST)
	allMypes := candMype.Union(untypedMype)
	return allMypes
}

func marPosCollectionLiterals() *RewriteRule {
	// [3, 4, 5] -+> {list, list<int>}
	return &RewriteRule{
		condition: func(n Nod) bool {
			// TODO: support maps and sets
			if n.NodeType == NT_LIT_LIST {
				// evaluating this can be expensive, there might be a need
				// to optimize this at some point
				extantMype := NodGetChild(n, NTR_MYPE_POS).Data.(Mype)
				candMype := marPosCollectionEvaluateMype(n)
				if extantMype.WouldChangeFromUnionWith(candMype) {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			candMype := marPosCollectionEvaluateMype(n)
			NodGetChild(n, NTR_MYPE_POS).Data = candMype.Union(NodGetChild(n, NTR_MYPE_POS).Data.(Mype))
		},
	}
}

// provides a way to abstract away the specific mype algebra from the
// type propagation rules
func XMypeNewFull() Mype {
	return MypeArgedNewFull()
}

func XMypeNewEmpty() Mype {
	return MypeArgedNewEmpty()
}

func XMypeNewSingle(n Nod) Mype {
	if n.NodeType == NT_TYPEBASE {
		ty := n.Data.(int)
		return XMypeNewSingleBase(ty)
	} else if n.NodeType == NT_TYPEARGED {
		base := NodGetChild(n, NTR_TYPEARGED_BASE)
		arg := NodGetChild(n, NTR_TYPEARGED_ARG)
		return MypeArgedNewSingleArged(base.Data.(int), XMypeNewSingle(arg))
	} else if n.NodeType == NT_CLASSDEF {
		return XMypeNewSingleClassDef(n)
	} else {
		fmt.Println("FAIL: XMypeNewSingle on", PrettyPrint(n))
		panic("cannot generate single mype")
	}
}

func XMypeNewSingleClassDef(classDef Nod) Mype {
	return MypeArgedNewNod(classDef)
}

func XMypeNewSingleBase(ty int) Mype {
	return MypeArgedNewSingleBase(ty)
}

func XMypeNewSingleArged(baseTy int, arg Mype) Mype {
	return MypeArgedNewSingleArged(baseTy, arg)
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

func marPosVarFunc() *RewriteRule {
	// all calls to a variable can return anything
	return &RewriteRule{
		condition: func(n Nod) bool {
			if isReceiverCallType(n.NodeType) {
				// alternate syntax: vargetter is the base of the call
				// TODO: possibly remove in favor of the FUNCDEF syntax
				if NodGetChild(n, NTR_RECEIVERCALL_BASE).NodeType == NT_VAR_GETTER {
					return !NodGetChild(n, NTR_MYPE_POS).Data.(Mype).IsFull()
				}
			}
			return false
		},
		action: func(n Nod) {
			NodGetChild(n, NTR_MYPE_POS).Data = XMypeNewFull()
		},
	}
}

func marPosSysFunc() *RewriteRule {
	// all sys funcs can return anything
	return &RewriteRule{
		condition: func(n Nod) bool {
			if isReceiverCallType(n.NodeType) {
				baseNod := NodGetChild(n, NTR_RECEIVERCALL_BASE)
				if baseNod.NodeType == NT_IDENTIFIER_FUNC_NOSCOPE {
					rcName := baseNod.Data.(string)
					if isSystemFuncName(rcName) {
						pMype := NodGetChild(n, NTR_MYPE_POS).Data.(Mype)
						return !pMype.IsFull()
					}
				}
			}
			return false
		},
		action: func(n Nod) {
			NodGetChild(n, NTR_MYPE_POS).Data = XMypeNewFull()
		},
	}
}

func marPosPublicParameter() *RewriteRule {
	// assume that assignments to this parameter are in fact called
	// with every allowable type
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
			typeDecl := NodGetChildOrNil(n, NTR_TYPE_DECL)
			var allowedMype Mype
			if typeDecl == nil {
				allowedMype = XMypeNewFull()
			} else {
				allowedMype = XMypeNewSingle(typeDecl)
			}
			NodGetChild(n, NTR_MYPE_POS).Data = allowedMype
		},
	}
}

func marPosPublicClassFieldGetCandMype(classField Nod) Mype {
	// get the assumed maximal type from a class field using its type decl
	typeDecl := NodGetChildOrNil(classField, NTR_TYPE_DECL)
	if typeDecl == nil {
		return XMypeNewFull()
	}
	if typeDecl.NodeType == NT_CLASSDEF || typeDecl.NodeType == NT_TYPEBASE {
		return XMypeNewSingle(typeDecl)
	}
	return nil // means we can't deduce anything now
}

func marPosPublicClassField() *RewriteRule {
	// assume that assignments to this field are maximal
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_CLASSFIELD {
				varDef := NodGetChild(n, NTR_VARDEF)
				curPosMype := NodGetChild(varDef, NTR_MYPE_POS).Data.(Mype)
				candMype := marPosPublicClassFieldGetCandMype(n)
				if candMype != nil && curPosMype.WouldChangeFromUnionWith(candMype) {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			varDef := NodGetChild(n, NTR_VARDEF)
			curPosMype := NodGetChild(varDef, NTR_MYPE_POS).Data.(Mype)
			candMype := marPosPublicClassFieldGetCandMype(n)
			NodGetChild(varDef, NTR_MYPE_POS).Data = curPosMype.Union(candMype)
		},
	}
}

func (x *XformerPocket) marPosSelf() *RewriteRule {
	// type of 'self' is the current class
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_VARDEF {
				varName := NodGetChild(n, NTR_VARDEF_NAME).Data.(string)
				if varName == "self" {
					cCls := x.getContainingClassDef(n)
					if cCls != nil {
						curPosMype := NodGetChild(n, NTR_MYPE_POS).Data.(Mype)
						candMype := XMypeNewSingleClassDef(cCls)
						if curPosMype.WouldChangeFromUnionWith(candMype) {
							return true
						}
					}
				}
			}
			return false
		},
		action: func(n Nod) {
			cCls := x.getContainingClassDef(n)
			curPosMype := NodGetChild(n, NTR_MYPE_POS).Data.(Mype)
			candMype := XMypeNewSingleClassDef(cCls)
			NodGetChild(n, NTR_MYPE_POS).Data = curPosMype.Union(candMype)
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

func marPosReturnValue() *RewriteRule {
	// propagate return values into the placeholder
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_RETURN {
				if lhs := NodGetChildOrNil(n, NTR_RETURNVAL_PLACEHOLDER); lhs != nil {
					mypeLHS := NodGetChild(lhs, NTR_MYPE_POS).Data.(Mype)
					mypeRHS := NodGetChild(NodGetChild(n, NTR_RETURN_VALUE), NTR_MYPE_POS).Data.(Mype)
					if mypeLHS.WouldChangeFromUnionWith(mypeRHS) {
						return true
					}
				}
			}
			return false
		},
		action: func(n Nod) {
			lhs := NodGetChild(n, NTR_RETURNVAL_PLACEHOLDER)
			mypeLHS := NodGetChild(lhs, NTR_MYPE_POS).Data.(Mype)
			mypeRHS := NodGetChild(NodGetChild(n, NTR_RETURN_VALUE), NTR_MYPE_POS).Data.(Mype)
			newLHS := mypeLHS.Union(mypeRHS)
			NodGetChild(lhs, NTR_MYPE_POS).Data = newLHS
		},
	}
}

func marNegVarAssign() *RewriteRule {
	// propagate var type restrictions from lhs -> rhs
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_VARASSIGN {
				mypeLHS := NodGetChild(n, NTR_MYPE_NEG).Data.(Mype)
				mypeRHS := NodGetChild(NodGetChild(n, NTR_VARASSIGN_VALUE), NTR_MYPE_NEG).Data.(Mype)

				if mypeRHS.WouldChangeFromIntersectionWith(mypeLHS) {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			mypeLHS := NodGetChild(n, NTR_MYPE_NEG).Data.(Mype)
			mypeRHS := NodGetChild(NodGetChild(n, NTR_VARASSIGN_VALUE), NTR_MYPE_NEG).Data.(Mype)
			newRHS := mypeLHS.Intersection(mypeRHS)
			NodGetChild(NodGetChild(n, NTR_VARASSIGN_VALUE), NTR_MYPE_NEG).Data = newRHS
		},
	}
}

func marNegDeclaredType() *RewriteRule {
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_PARAMETER || n.NodeType == NT_VARASSIGN ||
				n.NodeType == NT_VARDEF {
				if typeDeclNod := NodGetChildOrNil(n, NTR_TYPE_DECL); typeDeclNod != nil {
					if typeDeclNod.NodeType == NT_TYPEBASE || typeDeclNod.NodeType == NT_CLASSDEF {
						negMype := NodGetChild(n, NTR_MYPE_NEG).Data.(Mype)
						declMype := XMypeNewSingle(typeDeclNod)
						if negMype.WouldChangeFromIntersectionWith(declMype) {
							return true
						}
					} else {
						fmt.Println("interesting situation: got type decl but wasn't supported:",
							PrettyPrint(typeDeclNod))
					}
				}
			}
			return false
		},
		action: func(n Nod) {
			typeDeclNod := NodGetChild(n, NTR_TYPE_DECL)
			declMype := XMypeNewSingle(typeDeclNod)
			currNegMypeNod := NodGetChild(n, NTR_MYPE_NEG)
			currNegMypeNod.Data = declMype.Intersection(currNegMypeNod.Data.(Mype))
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
				if qualName, ok := NodGetChild(n, NTR_BINOP_RIGHT).Data.(string); ok {
					if qualName == "len" {
						resulPosMype := NodGetChild(n, NTR_MYPE_POS).Data.(Mype)
						leftArgPosMype := NodGetChild(NodGetChild(n, NTR_BINOP_LEFT), NTR_MYPE_POS).Data.(Mype)
						intMype := XMypeNewSingleBase(TY_INT)
						return leftArgPosMype.ContainsAnyType(getLengthableTypes()) &&
							resulPosMype.WouldChangeFromUnionWith(intMype)
					}
				}
			}
			return false
		},
		action: func(n Nod) {
			NodGetChild(n, NTR_MYPE_POS).Data = (NodGetChild(n,
				NTR_MYPE_POS).Data.(Mype).Union(XMypeNewSingleBase(TY_INT)))
			fmt.Println("applied collection len rule, new mype is", PrettyPrintMype(n))
		},
	}
}

func marNegOpRestrictRules() []*RewriteRule {
	oers := marGetCompactOpEvaluateRules()

	opToallowableResult := map[int]Mype{}
	for _, oer := range oers {
		var allowableResult Mype
		if _, ok := opToallowableResult[oer.operator]; !ok {
			allowableResult = XMypeNewEmpty()
			opToallowableResult[oer.operator] = allowableResult
		} else {
			allowableResult = opToallowableResult[oer.operator]
		}
		allowableResult = allowableResult.Union(XMypeNewSingleBase(oer.result))
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

				matchLowHigh := (argMypes[0].ContainsSingleType(oer.operandLow) &&
					argMypes[1].ContainsSingleType(oer.operandHigh))
				matchHighLow := (argMypes[1].ContainsSingleType(oer.operandLow) &&
					argMypes[0].ContainsSingleType(oer.operandHigh))

				fmt.Println("oer.operator", oer.operator, "opHigh", oer.operandHigh,
					"opLow", oer.operandLow, "matchLowHigh", matchLowHigh, "matchHighLow", matchHighLow)

				// todo: ensure we have precise semantics for arged types
				if matchLowHigh || matchHighLow {
					if !resultMype.ContainsSingleType(oer.result) {
						return true
					}
				}
			}
			return false
		},
		action: func(n Nod) {
			NodGetChild(n, NTR_MYPE_POS).Data = NodGetChild(n, NTR_MYPE_POS).Data.(Mype).Union(XMypeNewSingleBase(oer.result))
		},
	}
}

func writeTypeAndData(dst Nod, src Nod) {
	dst.NodeType = src.NodeType
	dst.Data = src.Data
}

func marPosPrimitiveLiterals() *RewriteRule {
	return &RewriteRule{
		condition: func(n Nod) bool {
			if isPrimitiveLiteralNodeType(n.NodeType) {
				if NodGetChild(n, NTR_MYPE_POS).Data.(Mype).IsEmpty() {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			ty := getLiteralTypeAnnDataFromNT(n.NodeType)
			NodGetChild(n, NTR_MYPE_POS).Data = XMypeNewSingleBase(ty)
		},
	}
}
