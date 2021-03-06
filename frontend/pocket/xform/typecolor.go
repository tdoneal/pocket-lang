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
		x.marGenPCCollectionIndex(),
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
		x.marPosFunctionDefs(),
		x.marPosClassDefs(),
		x.marPosRefOp(),
		x.marPosVarAssign(),
		x.marPosPublicParameter(),
		x.marPosPublicClassField(),
		x.marPosObjFieldAccessor(),
		x.marPosObjFieldAccessorDuckProp(),
		x.marPosSelf(),
		x.marPosSysFunc(),
		x.marPosObjInitUser(),
		x.marPosFuncCallUser(),
		x.marPosReturnValue(),
	}
	rv = append(rv, x.marPosOpEvaluateRules()...)
	return rv
}

func (x *XformerPocket) getInitMypeNodFull() Nod {
	return NodNewData(NT_DYPE, NodNew(DYPE_ALL))
}

func (x *XformerPocket) getInitMypeNodEmpty() Nod {
	return NodNewData(NT_DYPE, NodNew(DYPE_EMPTY))
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
		nt == NT_RETURN || nt == NT_FOR_IN || nt == NT_FOR_CLASSIC || nt == NT_WHILE || nt == NT_LOOP ||
		nt == NT_IF
}

func (x *XformerPocket) colorTypes() {
	// generate the "valid" mypes by intersecting the negative with the positive
	// then output a single "type color" for each myped node
	nodes := x.SearchRoot(func(n Nod) bool { return NodHasChild(n, NTR_MYPE_POS) })
	x.generateValidMypes(nodes)
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

func (x *XformerPocket) marPosRefOp() *RewriteRule {
	// ref ops inherit the type of their arg
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_REFERENCEOP {
				arg := NodGetChild(n, NTR_RECEIVERCALL_ARG)
				if argDype := NodGetChildOrNil(arg, NTR_MYPE_POS); argDype != nil {
					return x.RICUnion2(NodGetChild(n, NTR_MYPE_POS), argDype.Data.(Nod))
				}
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
						candMype := NodGetChild(n, NTR_FUNCDEF)
						extMype := NodGetChild(n, NTR_MYPE_POS)
						return x.RICUnion2(extMype, candMype)
					}
				}
			}
			return false
		},
	}
}

// TODO: can merge with the corresponding one for class defs
func (x *XformerPocket) marPosFunctionDefs() *RewriteRule {
	// type of a function def can be itself
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_FUNCDEF {
				extMype := NodGetChild(n, NTR_MYPE_POS)
				candMype := n
				return x.RICUnion2(extMype, candMype)
			}
			return false
		},
	}
}

func (x *XformerPocket) marPosClassDefs() *RewriteRule {
	// type of a classdef is a reflection to itself
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_CLASSDEF {
				extMype := NodGetChild(n, NTR_MYPE_POS)
				if extMype.Data.(Nod).NodeType == DYPE_EMPTY {
					candMype := NodNewChild(NT_REFLECTTYPE, NTR_REFLECTTYPE_CLASSDEF, n)
					return x.RICUnion2(extMype, candMype)
				}
			}
			return false
		},
	}
}

func (x *XformerPocket) copyMypesOf(src Nod, dst Nod) {
	// makes it so that dst effectively has the same dypes as src
	if NodHasChild(dst, NTR_MYPE_POS) {
		NodRemoveChild(dst, NTR_MYPE_POS)
	}
	if NodHasChild(dst, NTR_MYPE_NEG) {
		NodRemoveChild(dst, NTR_MYPE_NEG)
	}
	NodSetChild(dst, NTR_MYPE_POS, NodGetChild(src, NTR_MYPE_POS))
	NodSetChild(dst, NTR_MYPE_NEG, NodGetChild(src, NTR_MYPE_NEG))
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
						x.copyMypesOf(n, varDef)
						return true
					}
					varDefMypePos := NodGetChild(varDef, NTR_MYPE_POS)
					myMypePos := NodGetChild(n, NTR_MYPE_POS)
					if varDefMypePos != myMypePos {
						// if here, inherit the mypes from the extant vardef
						x.copyMypesOf(varDef, n)
						return true
					}
				}
			}
			return false
		},
	}
}

func (x *XformerPocket) NodComputeParentChildIntegrity() bool {
	allNods := x.SearchRoot(func(n Nod) bool { return true })

	for _, nod := range allNods {
		if !x.NodComputeParentChildIntegrityOn(nod) {
			return false
		}
	}
	return true
}

func (x *XformerPocket) NodComputeParentChildIntegrityOn(n Nod) bool {
	for _, edge := range n.Out {
		child := edge.Out
		cInIndex := NodGetInEdgeNdx(child, edge)
		if cInIndex == -1 {
			fmt.Println("missing", PrettyPrint(n), "for child", PrettyPrint(child))
			return false
		}
	}
	return true
}

func (x *XformerPocket) NodCheckParentChildIntegrity() {
	integrity := x.NodComputeParentChildIntegrity()
	if !integrity {
		panic("integrity lost")
	}
}

func (x *XformerPocket) generateValidMypes(nodes []Nod) {

	x.NodCheckParentChildIntegrity()

	for _, node := range nodes {
		posMype := NodGetChild(node, NTR_MYPE_POS).Data.(Nod)
		negMype := NodGetChild(node, NTR_MYPE_NEG).Data.(Nod)
		validMype := DypeSimplifyDeep(DypeXSect(posMype, negMype))
		validMype = x.postProcessDype(validMype)

		if validMype.NodeType == DYPE_EMPTY {
			if node.NodeType == NT_RECEIVERCALL_CMD || node.NodeType == NT_FUNCDEF_RV_PLACEHOLDER ||
				node.NodeType == NT_RECEIVERCALL_METHOD {
				// this is acceptable for these node types (can safely ignore)
			} else {
				panic("couldn't find a valid type for node: " + PrettyPrint(node))
			}
		}
		NodSetChild(node, NTR_TYPE, validMype)
		NodRemoveChild(node, NTR_MYPE_POS)
		NodRemoveChild(node, NTR_MYPE_NEG)
	}
}

func (x *XformerPocket) postProcessDype(dype Nod) Nod {
	// returns a postprocessed dype for the final type coloring
	// allows for a layer of postprocessing after the solver has concluded
	// to determine what type is "assigned"
	// currently all this does is remove lists with empty element specifications
	if dype.NodeType == DYPE_UNION {
		args := NodGetChildList(dype)
		nargs := []Nod{}
		for _, arg := range args {
			shouldAdd := true
			if arg.NodeType == NT_TYPEBASE {
				if arg.Data.(int) == TY_LIST {
					shouldAdd = false
				}
			}
			if shouldAdd {
				nargs = append(nargs, arg)
			}
		}

		// simplify and recurse
		simplified := DypeSimplifyDeep(NodNewChildList(DYPE_UNION, nargs))
		// if simplified.NodeType == DYPE_UNION {
		// 	fmt.Println("offending type:", PrettyPrint(simplified))
		// 	panic("Couldn't simplify post processed type enough")
		// }
		return simplified
	}
	return dype
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
					return x.RICUnion2(NodGetChild(n, NTR_MYPE_POS), base)
				}
			}
			return false
		},
	}
}

func marPosCollectionGetArgedCand(elements []Nod) Nod {
	accum := NodNew(DYPE_EMPTY)
	for _, element := range elements {
		elementPosMype := NodGetChild(element, NTR_MYPE_POS).Data.(Nod)
		accum = DypeDeduplicate(DypeUnion(accum, elementPosMype))
	}
	accum = DypeSimplifyShallow(accum)
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
	allMypes := DypeSimplifyShallow(DypeUnion(candMype, untypedMype))
	return allMypes
}

func (x *XformerPocket) RICUnionOld(old Nod, new Nod) bool {
	// fmt.Println("ricunion 0, old", PrettyPrint(old), "new", PrettyPrint(new))
	if DypeWouldChangeUnion(old, new) {
		// fmt.Println("ricunion 01")
		unioned := DypeSimplifyShallowComplex(DypeUnion(old, new))
		// fmt.Println("ricunion 1, before replace: old:", PrettyPrint(old),
		// 	"\nnew:", PrettyPrint(unioned))
		fmt.Println("ric, unioned:", PrettyPrint(unioned))
		x.Replace(old, unioned)
		fmt.Println("ricunion 2, after replace: old:", PrettyPrint(old),
			"\nnew:", PrettyPrint(unioned))
		return true
	}
	return false
}

func (x *XformerPocket) RICUnion2(oldContainer Nod, newDype Nod) bool {
	// fmt.Println("ricunion 0, old", PrettyPrint(old), "new", PrettyPrint(new))
	oldDype := oldContainer.Data.(Nod)
	if DypeWouldChangeUnion(oldDype, newDype) {
		unioned := DypeSimplifyShallowComplex(DypeUnion(oldDype, newDype))
		oldContainer.Data = unioned
		return true
	}
	return false
}

func (x *XformerPocket) RICXSect2(oldContainer Nod, newDype Nod) bool {
	oldDype := oldContainer.Data.(Nod)
	if DypeWouldChangeXSect(oldDype, newDype) {
		oldContainer.Data = DypeSimplifyShallowComplex(DypeXSect(oldDype, newDype))
		return true
	}
	return false
}

func (x *XformerPocket) RICXSectOld(old Nod, new Nod) bool {
	if DypeWouldChangeXSect(old, new) {
		x.Replace(old, DypeSimplifyShallowComplex(DypeXSect(old, new)))
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
				return x.RICUnion2(NodGetChild(n, NTR_MYPE_POS), marPosCollectionEvaluateMype(n))
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
						return x.RICUnion2(NodGetChild(n, NTR_MYPE_POS), NodNew(DYPE_ALL))
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
				extMype := NodGetChild(n, NTR_MYPE_POS)
				rv := x.RICUnion2(extMype, allowedMype)
				return rv
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
					return x.RICUnion2(NodGetChild(varDef, NTR_MYPE_POS), candMype)
				}
			}
			return false
		},
	}
}

func (x *XformerPocket) marPosObjFieldAccessor() *RewriteRule {
	// lookup types of both object instance accesses and static class accesses

	// TODO: somehow merge with the above marPosPublicClassField(), they seem to be doing similar things
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_OBJFIELD_ACCESSOR {
				posDype := NodGetChild(n, NTR_MYPE_POS).Data.(Nod)
				if posDype.NodeType == DYPE_EMPTY {
					object := NodGetChild(n, NTR_RECEIVERCALL_BASE)
					objPosDype := NodGetChild(object, NTR_MYPE_POS).Data.(Nod)
					// TODO: support more types of dypes, not just single classdef
					// for example, might want to scan through user classdefs consistent with the dype
					var clsDef Nod = nil
					var isStatic bool = false
					if objPosDype.NodeType == NT_REFLECTTYPE {
						clsDef = NodGetChild(objPosDype, NTR_REFLECTTYPE_CLASSDEF)
						isStatic = true
					} else if objPosDype.NodeType == NT_CLASSDEF {
						clsDef = objPosDype
						isStatic = false
					}
					if clsDef != nil {
						var classVarTable Nod = nil
						if isStatic {
							classVarTable = NodGetChild(NodGetChild(clsDef, NTR_CLASSDEF_STATICZONE), NTR_VARTABLE)
						} else {
							classVarTable = NodGetChild(clsDef, NTR_VARTABLE)
						}
						varName := NodGetChild(n, NTR_OBJFIELD_ACCESSOR_NAME).Data.(string)
						luResult := x.varTableLookup(classVarTable, varName)
						if luResult != nil {
							x.RICUnion2(NodGetChild(n, NTR_MYPE_POS), NodGetChild(luResult, NTR_MYPE_POS).Data.(Nod))
							return true
						}
					}
				}
			}
			return false
		},
	}
}

func (x *XformerPocket) marPosObjFieldAccessorDuckProp() *RewriteRule {
	// if the type of base is ALL, destructively union upwards

	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_OBJFIELD_ACCESSOR {
				dype := NodGetChild(n, NTR_MYPE_POS)
				base := NodGetChild(n, NTR_RECEIVERCALL_BASE)
				baseDype := NodGetChild(base, NTR_MYPE_POS).Data.(Nod)
				if baseDype.NodeType == DYPE_ALL {
					return x.RICUnion2(dype, baseDype)
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
						return x.RICUnion2(NodGetChild(n, NTR_MYPE_POS), cCls)
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
				dypeRHS := NodGetChild(NodGetChild(n, NTR_VARASSIGN_VALUE), NTR_MYPE_POS).Data.(Nod)
				return x.RICUnion2(dypeLHS, dypeRHS)
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
					mypeRHS := NodGetChild(NodGetChild(n, NTR_RETURN_VALUE), NTR_MYPE_POS).Data.(Nod)
					return x.RICUnion2(mypeLHS, mypeRHS)
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
				mypeLHSData := NodGetChild(n, NTR_MYPE_NEG).Data.(Nod)
				mypeRHS := NodGetChild(NodGetChild(n, NTR_VARASSIGN_VALUE), NTR_MYPE_NEG)
				return x.RICXSect2(mypeRHS, mypeLHSData)
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
					negMype := NodGetChild(n, NTR_MYPE_NEG)
					declMype := typeDeclNod
					return x.RICXSect2(negMype, declMype)
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
		&MypeOpEvaluateRule{NT_ADDOP, TY_LIST, TY_LIST, TY_LIST},

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
	// <collection>.len -> int

	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_OBJFIELD_ACCESSOR {
				if qualName, ok := NodGetChild(n, NTR_OBJFIELD_ACCESSOR_NAME).Data.(string); ok {
					if qualName == "len" {
						resulPosMype := NodGetChild(n, NTR_MYPE_POS)
						leftArgPosMype := NodGetChild(NodGetChild(n, NTR_RECEIVERCALL_BASE), NTR_MYPE_POS)
						isLengthable := DypeSimplifyShallow(DypeUnion(
							leftArgPosMype, getLengthableDype())).NodeType != DYPE_EMPTY
						if isLengthable {
							fmt.Println("applied collection len rule")
							intMype := NodNewData(NT_TYPEBASE, TY_INT)
							return x.RICUnion2(resulPosMype, intMype)
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
				return x.RICXSect2(resultMype, allowableResult)
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
					NodGetChild(NodGetChild(n, NTR_BINOP_LEFT), NTR_MYPE_POS).Data.(Nod),
					NodGetChild(NodGetChild(n, NTR_BINOP_RIGHT), NTR_MYPE_POS).Data.(Nod),
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
					candMype := NodNewData(NT_TYPEBASE, oer.result)
					rv := x.RICUnion2(resultMype, candMype)
					return rv
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
				if extDype.Data.(Nod).NodeType == DYPE_EMPTY {
					ty := getLiteralTypeAnnDataFromNT(n.NodeType)
					return x.RICUnion2(extDype, NodNewData(NT_TYPEBASE, ty))
				}
			}
			return false
		},
	}
}

// specifies type evaluation rules for expressions such as
// <list>(<index>) -> <list>.eletype

func (x *XformerPocket) marGenPCCollectionIndex() *RewriteRule {
	// <list>(<index>) -> <list>.eletype
	// syntax of <list>(<index>) at this stage is a receivercall with an
	// indexable base
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_RECEIVERCALL {
				base := NodGetChild(n, NTR_RECEIVERCALL_BASE)
				if NodHasChild(base, NTR_MYPE_POS) {
					basePosDype := NodGetChild(base, NTR_MYPE_POS)
					baseNegDype := NodGetChild(base, NTR_MYPE_NEG)
					// heuristic: only apply the indexing rule if something was explicitly
					// mentioned as indexable
					posEleDype := x.getIndexableElementDype(basePosDype.Data.(Nod))
					negEleDype := x.getIndexableElementDype(baseNegDype.Data.(Nod))

					// fmt.Println("basePosDype", PrettyPrint(basePosDype))
					// fmt.Println("baseNegDype", PrettyPrint(baseNegDype))
					// fmt.Println("posEleDype", PrettyPrint(posEleDype))
					// fmt.Println("negEleDype", PrettyPrint(negEleDype))

					// heuristic: chop the pos dype by the neg dype
					chopPosEleType := DypeSimplifyDeep(DypeXSect(posEleDype, negEleDype))

					// fmt.Println("chopPosEleType", PrettyPrint(chopPosEleType))

					changedPos := x.RICUnion2(NodGetChild(n, NTR_MYPE_POS), chopPosEleType)
					changedNeg := x.RICXSect2(NodGetChild(n, NTR_MYPE_NEG), negEleDype)

					return changedPos || changedNeg
				}
			}
			return false
		},
	}
}

func (x *XformerPocket) getIndexableElementDype(dype Nod) Nod {
	// returns the element dype of the input dype, if it's a collection
	// given e.g. list ~int~ returns int
	// given e.g. Union(list~int~, list) returns Union(int, empty) = int
	// given a non-collection dype returns EMPTY
	// given ALL returns ALL
	if dype.NodeType == DYPE_ALL {
		return dype
	} else if dype.NodeType == NT_TYPEBASE {
		whichType := dype.Data.(int)
		if whichType == TY_LIST {
			return NodNew(DYPE_EMPTY)
		}
	} else if dype.NodeType == NT_TYPECALL {
		baseType := NodGetChild(dype, NTR_RECEIVERCALL_BASE)
		if baseType.NodeType == NT_TYPEBASE {
			whichBaseType := baseType.Data.(int)
			if whichBaseType == TY_LIST {
				eleType := NodGetChild(dype, NTR_RECEIVERCALL_ARG)
				return eleType
			}
		}
	} else if dype.NodeType == DYPE_UNION {
		subNodes := NodGetChildList(dype)
		subElementTypes := []Nod{}
		for _, subNode := range subNodes {
			subElementTypes = append(subElementTypes, x.getIndexableElementDype(subNode))
		}
		newUnion := NodNewChildList(DYPE_UNION, subElementTypes)
		return DypeSimplifyShallow(newUnion)
	}

	return NodNew(DYPE_EMPTY)
}
