package xform

import (
	"fmt"
	. "pocket-lang/frontend/pocket/common"
	. "pocket-lang/parse"
)

func (x *XformerPocket) solveIdentifiers() {
	allNods := x.SearchRoot(func(n Nod) bool { return true })
	rules := x.getIdentifierRewriteRules()
	x.applyRewritesUntilStable(allNods, rules)

}

func (x *XformerPocket) getIdentifierRewriteRules() []*RewriteRule {
	rv := []*RewriteRule{
		x.IRRBaseIdentifiers(),
		x.IRRBaseCallIdentifiers(),
		x.IRRParametersToLocals(),
		x.IRRNoscopesClass(),
		x.IRRNoscopesLocals(),
		x.IRRNoscopesFuncGlobal(),
		x.IRRNoscopesFuncObjInit(),
		x.IRRNoscopesFuncLocalVar(),
		// TODO: somehow re-use the var lookup framework to resolve certain Noscope Funcs
		x.IRRSimpleVarWrites(),
		x.IRRPlainObjInit(),
		x.IRRArgedObjInit(),
		x.IRRReturnToPlaceholder(),
		x.IRRResolveMethodCalls(),
	}
	return rv
}

func (x *XformerPocket) IRRReturnToPlaceholder() *RewriteRule {
	// link return statements directly to the placeholder for the return value
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_RETURN {
				if !NodHasChild(n, NTR_RETURNVAL_PLACEHOLDER) {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			cFunc := x.getContainingFuncDef(n)
			ph := NodGetChild(cFunc, NTR_RETURNVAL_PLACEHOLDER)
			NodSetChild(n, NTR_RETURNVAL_PLACEHOLDER, ph)
		},
	}
}

func (x *XformerPocket) IRRParametersToLocals() *RewriteRule {
	// all parameters link to variables in the local var table
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_PARAMETER {
				if !NodHasChild(n, NTR_VARDEF) {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			idtext := NodGetChild(n, NTR_VARDEF_NAME).Data.(string)
			cFunc := x.getContainingFuncDef(n)
			fTable := NodGetChild(cFunc, NTR_VARTABLE)
			vDef := x.varTableLookup(fTable, idtext)
			if vDef != nil {
				// link to existing
				NodSetChild(n, NTR_VARDEF, vDef)
				return
			}
			// if we got here, it's a new local variable for sure
			x.linkNewVarDefWithName(n, fTable, idtext, VSCOPE_FUNCPARAM)
		},
	}
}

func (x *XformerPocket) IRRPlainObjInit() *RewriteRule {
	// look up single-word references to classes, treat as object initializers
	// eg Point
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_IDENTIFIER_RVAL_NOSCOPE {
				idtext := n.Data.(string)
				cDef := x.globalClassDefLookup(idtext)
				if cDef != nil {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			cDef := x.globalClassDefLookup(n.Data.(string))
			// rewrite the node in-place
			n.NodeType = NT_OBJINIT
			NodSetChild(n, NTR_RECEIVERCALL_BASE, cDef)
			NodSetChild(n, NTR_RECEIVERCALL_ARG, NodNew(NT_EMPTYARGLIST))
		},
	}
}
func (x *XformerPocket) IRRArgedObjInit() *RewriteRule {
	// look up single-word class references in call bases, treat as object initializers
	// e.g. Point()
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_RECEIVERCALL {
				if callBase := NodGetChildOrNil(n, NTR_RECEIVERCALL_BASE); callBase != nil {
					if callBase.NodeType == NT_IDENTIFIER {
						idtext := callBase.Data.(string)
						cDef := x.globalClassDefLookup(idtext)
						if cDef != nil {
							return true
						}
					}
				}
			}
			return false
		},
		action: func(n Nod) {
			cDef := x.globalClassDefLookup(NodGetChild(n, NTR_RECEIVERCALL_BASE).Data.(string))
			// rewrite the node in-place
			n.NodeType = NT_OBJINIT
			NodSetChild(n, NTR_RECEIVERCALL_BASE, cDef)
		},
	}
}

func (x *XformerPocket) IRRBaseIdentifiers() *RewriteRule {
	// simple rvalue identifiers are either dotop qualifiers or noscope
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_IDENTIFIER_RVAL {
				return true
			}
			return false
		},
		action: func(n Nod) {
			if parentBinOp := NodGetParentOrNil(n, NTR_BINOP_RIGHT); parentBinOp != nil {
				if parentBinOp.NodeType == NT_DOTOP {
					n.NodeType = NT_DOTOP_QUALIFIER
					return
				}
			}
			n.NodeType = NT_IDENTIFIER_RVAL_NOSCOPE
		},
	}
}

func (x *XformerPocket) IRRResolveMethodCalls() *RewriteRule {
	// try linking known method calls to their static definition
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_RECEIVERCALL_METHOD {
				if !NodHasChild(n, NTR_TYPECOND_DEFS) {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			fmt.Println("apply irrresolvmeth on", PrettyPrint(n))
			methName := NodGetChild(n, NTR_RECEIVERCALL_METHOD_NAME).Data.(string)
			// look up method in func table of all known classes
			clss := x.SearchRoot(func(n Nod) bool { return n.NodeType == NT_CLASSDEF })
			possMeths := []Nod{}
			for _, cls := range clss {
				methTable := NodGetChild(cls, NTR_FUNCTABLE)
				methDef := x.funcTableLookup(methTable, methName)
				if methDef != nil {
					possMeths = append(possMeths, methDef)
				}
			}
			condDef := NodNewChildList(NT_TYPECOND_DEFS, possMeths)
			NodSetChild(n, NTR_TYPECOND_DEFS, condDef)
		},
	}
}

func (x *XformerPocket) IRRBaseCallIdentifiers() *RewriteRule {
	// simple words on the base of calls can be specified as such
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_RECEIVERCALL || n.NodeType == NT_RECEIVERCALL_CMD {
				baseNod := NodGetChild(n, NTR_RECEIVERCALL_BASE)
				if baseNod.NodeType == NT_IDENTIFIER {
					if !x.OneParentIs(n, func(n Nod) bool { return n.NodeType == NT_DOTOP }) {
						return true
					}
				}
			}
			return false
		},
		action: func(n Nod) {
			NodGetChild(n, NTR_RECEIVERCALL_BASE).NodeType = NT_IDENTIFIER_FUNC_NOSCOPE
		},
	}
}

func (x *XformerPocket) IRRNoscopesClass() *RewriteRule {
	// make progress towards resolving NT_IDENTIFIER_RVAL_NOSCOPEs
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_IDENTIFIER_RVAL_NOSCOPE {
				idtext := n.Data.(string)
				cDef := x.getContainingClassDef(n)
				if cDef != nil {
					cTable := NodGetChild(cDef, NTR_VARTABLE)
					cVarDef := x.varTableLookup(cTable, idtext)
					if cVarDef != nil {
						return true
					}
				}
			}
			return false
		},
		action: func(n Nod) {
			idtext := n.Data.(string)
			cDef := x.getContainingClassDef(n)
			cTable := NodGetChild(cDef, NTR_VARTABLE)
			cVarDef := x.varTableLookup(cTable, idtext)
			n.NodeType = NT_VAR_GETTER
			varName := NodNewData(NT_IDENTIFIER_RESOLVED, idtext)
			NodSetChild(n, NTR_VAR_NAME, varName)
			NodSetChild(n, NTR_VARDEF, cVarDef)
			n.Data = nil
		},
	}
}

func (x *XformerPocket) IRRNoscopesFuncGlobal() *RewriteRule {
	// make progress towards resolving NT_IDENTIFIER_FUNC_NOSCOPE: lookup in global func table
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_IDENTIFIER_FUNC_NOSCOPE {
				idtext := n.Data.(string)
				fTable := NodGetChild(x.Root, NTR_FUNCTABLE)
				fDef := x.funcTableLookup(fTable, idtext)
				if fDef != nil {
					return true // no need to wait on anything else, can perform lookup now
				}
			}
			return false
		},
		action: func(n Nod) {
			idtext := n.Data.(string)
			fTable := NodGetChild(x.Root, NTR_FUNCTABLE)
			fDef := x.funcTableLookup(fTable, idtext)
			n.NodeType = NT_IDENTIFIER_RESOLVED
			parentCall := NodGetParent(n, NTR_RECEIVERCALL_BASE)
			NodSetChild(parentCall, NTR_FUNCDEF, fDef)
		},
	}
}

func (x *XformerPocket) IRRNoscopesFuncObjInit() *RewriteRule {
	// make progress towards resolving NT_IDENTIFIER_FUNC_NOSCOPE: lookup object initializer in class table
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_IDENTIFIER_FUNC_NOSCOPE {
				idtext := n.Data.(string)
				cTable := NodGetChild(x.Root, NTR_CLASSTABLE)
				cDef := x.classTableLookup(cTable, idtext)
				if cDef != nil {
					return true // no need to wait on anything else, can perform lookup now
				}
			}
			return false
		},
		action: func(n Nod) {
			idtext := n.Data.(string)
			cTable := NodGetChild(x.Root, NTR_CLASSTABLE)
			cDef := x.classTableLookup(cTable, idtext)
			// rewrite as object initializer
			parentCall := NodGetParent(n, NTR_RECEIVERCALL_BASE)
			parentCall.NodeType = NT_OBJINIT
			NodSetChild(parentCall, NTR_RECEIVERCALL_BASE, cDef)
			n.NodeType = NT_IDENTIFIER_RESOLVED
		},
	}
}

func (x *XformerPocket) IRRNoscopesFuncLocalVar() *RewriteRule {
	// make progress towards resolving NT_IDENTIFIER_FUNC_NOSCOPE: lookup object initializer in class table
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_IDENTIFIER_FUNC_NOSCOPE {
				idtext := n.Data.(string)
				fDef := x.getContainingFuncDef(n)
				vTable := NodGetChild(fDef, NTR_VARTABLE)
				vDef := x.varTableLookup(vTable, idtext)
				if vDef != nil {
					return true // no need to wait on anything else, can perform lookup now
				}
			}
			return false
		},
		action: func(n Nod) {
			idtext := n.Data.(string)
			fDef := x.getContainingFuncDef(n)
			vTable := NodGetChild(fDef, NTR_VARTABLE)
			vDef := x.varTableLookup(vTable, idtext)
			// rewrite as call to variable
			parentCall := NodGetParent(n, NTR_RECEIVERCALL_BASE)
			varGetter := NodNew(NT_VAR_GETTER)
			NodSetChild(varGetter, NTR_VAR_NAME, n)
			NodSetChild(varGetter, NTR_VARDEF, vDef)
			NodSetChild(parentCall, NTR_RECEIVERCALL_BASE, varGetter)
			n.NodeType = NT_IDENTIFIER_RESOLVED
		},
	}
}

func (x *XformerPocket) IRRNoscopesFuncsOld() *RewriteRule {
	// make progress towards resolving NT_IDENTIFIER_FUNC_NOSCOPE: lookup in all tables
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_IDENTIFIER_FUNC_NOSCOPE {
				return true // no need to wait on anything else, can perform lookup now
			}
			return false
		},
		action: func(n Nod) {
			idtext := n.Data.(string)
			fTable := NodGetChild(x.Root, NTR_FUNCTABLE)
			fDef := x.funcTableLookup(fTable, idtext)
			isSys := isSystemFuncName(idtext)
			if fDef == nil {
				// check if object initializer
				cTable := NodGetChild(x.Root, NTR_CLASSTABLE)
				cDef := x.classTableLookup(cTable, idtext)
				if cDef != nil {
					// rewrite as object initializer
					parentCall := NodGetParent(n, NTR_RECEIVERCALL_BASE)
					parentCall.NodeType = NT_OBJINIT
					NodSetChild(parentCall, NTR_RECEIVERCALL_BASE, cDef)
					n.NodeType = NT_IDENTIFIER_RESOLVED
					return
				}
			}
			if fDef == nil && !isSys {
				panic("unknown function '" + idtext + "'")
			}
			n.NodeType = NT_IDENTIFIER_RESOLVED
			if !isSys {
				parentCall := NodGetParent(n, NTR_RECEIVERCALL_BASE)
				NodSetChild(parentCall, NTR_FUNCDEF, fDef)
			}
		},
	}
}

func (x *XformerPocket) IRRNoscopesLocals() *RewriteRule {
	// make progress towards resolving NT_IDENTIFIER_RVAL_NOSCOPEs: check for local variable
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_IDENTIFIER_RVAL_NOSCOPE {
				idtext := n.Data.(string)
				fDef := x.getContainingFuncDef(n)
				if fDef != nil {
					fTable := NodGetChild(fDef, NTR_VARTABLE)
					fVarDef := x.varTableLookup(fTable, idtext)
					if fVarDef != nil {
						return true
					}
				}
			}
			return false
		},
		action: func(n Nod) {
			idtext := n.Data.(string)
			fDef := x.getContainingFuncDef(n)
			fTable := NodGetChild(fDef, NTR_VARTABLE)
			fVarDef := x.varTableLookup(fTable, idtext)
			n.NodeType = NT_VAR_GETTER
			varName := NodNewData(NT_IDENTIFIER_RESOLVED, idtext)
			NodSetChild(n, NTR_VAR_NAME, varName)
			NodSetChild(n, NTR_VARDEF, fVarDef)
			n.Data = nil
		},
	}
}

func (x *XformerPocket) IRRSimpleVarWrites() *RewriteRule {
	// resolve simple variable writes: either existing local, new local, or class var
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_VARASSIGN {
				varName := NodGetChild(n, NTR_VAR_NAME)
				if varName.NodeType == NT_IDENTIFIER {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			varName := NodGetChild(n, NTR_VAR_NAME)
			idtext := varName.Data.(string)
			cCls := x.getContainingClassDef(n)
			if cCls != nil {
				cTable := NodGetChild(cCls, NTR_VARTABLE)
				clsVarDef := x.varTableLookup(cTable, idtext)
				if clsVarDef != nil {
					varName.NodeType = NT_IDENTIFIER_RESOLVED
					NodSetChild(n, NTR_VARDEF, clsVarDef)
					return
				}
			}
			tfunc := x.getContainingFuncDef(n)
			localVarTable := NodGetChild(tfunc, NTR_VARTABLE)
			localVarDef := x.varTableLookup(localVarTable, idtext)
			if localVarDef != nil {
				varName.NodeType = NT_IDENTIFIER_RESOLVED
				NodSetChild(n, NTR_VARDEF, localVarDef)
				return
			}

			// if we got here, it's a new local variable for sure
			x.linkNewVarDefWithName(n, localVarTable, idtext, VSCOPE_FUNCLOCAL)
		},
	}
}

func (x *XformerPocket) linkNewVarDefWithName(varRef Nod, varTable Nod, varName string, varScope int) {
	nvd := NodNew(NT_VARDEF)
	NodSetChild(nvd, NTR_VARDEF_SCOPE, NodNewData(NT_VARDEF_SCOPE, varScope))
	NodSetChild(nvd, NTR_VARDEF_NAME, NodNewData(NT_IDENTIFIER, varName))
	x.addVarToVartable(varTable, nvd)
	NodSetChild(varRef, NTR_VARDEF, nvd)
}

func (x *XformerPocket) getContainingClassDef(n Nod) Nod {
	return x.getContainingNodOrNil(n, func(n Nod) bool { return n.NodeType == NT_CLASSDEF })
}

func (x *XformerPocket) getLocalVarName(n Nod) string {
	return NodGetChild(n, NTR_VAR_NAME).Data.(string)
}

func (x *XformerPocket) varTableLookup(vt Nod, varName string) Nod {
	vDefs := NodGetChildList(vt)
	for _, vDef := range vDefs {
		vName := NodGetChild(vDef, NTR_VARDEF_NAME).Data.(string)
		if vName == varName {
			return vDef
		}
	}
	return nil
}

func (x *XformerPocket) funcTableLookup(ft Nod, funcName string) Nod {
	fDefs := NodGetChildList(ft)
	for _, fDef := range fDefs {
		fName := NodGetChild(fDef, NTR_FUNCDEF_NAME).Data.(string)
		if fName == funcName {
			return fDef
		}
	}
	return nil
}

func (x *XformerPocket) classTableLookup(ct Nod, className string) Nod {
	cDefs := NodGetChildList(ct)
	for _, cDef := range cDefs {
		cName := NodGetChild(cDef, NTR_CLASSDEF_NAME).Data.(string)
		if cName == className {
			return cDef
		}
	}
	return nil
}

func (x *XformerPocket) globalClassDefLookup(clsName string) Nod {
	clsTable := NodGetChild(x.Root, NTR_CLASSTABLE)
	return x.classTableLookup(clsTable, clsName)
}

func (x *XformerPocket) getContainingFuncDef(n Nod) Nod {
	return x.getContainingNodOrNil(n, func(n Nod) bool { return n.NodeType == NT_FUNCDEF })
}

func (x *XformerPocket) addVarToVartable(vt Nod, varDef Nod) {
	eList := NodGetChildList(vt)
	eList = append(eList, varDef)
	NodSetOutList(vt, eList)
}

func (x *XformerPocket) buildTableHierarchy() {
	// links vartables to their "parents"
	// in preparation for final variable resolving
	// for example, a local vartable's parent might be a class vartable

	varTables := x.SearchRoot(func(n Nod) bool { return n.NodeType == NT_VARTABLE })

	for _, varTable := range varTables {
		container := NodGetParent(varTable, NTR_VARTABLE)
		parentContainer := x.getContainingNodOrNil(container, func(n Nod) bool {
			return NodHasChild(n, NTR_VARTABLE) && n != container
		})
		if parentContainer != nil {
			parentVarTable := NodGetChild(parentContainer, NTR_VARTABLE)
			NodSetChild(varTable, NTR_TABLE_PARENT, parentVarTable)
		}
	}
}

func (x *XformerPocket) buildClassTable() {
	clsDefs := x.SearchRoot(func(n Nod) bool { return n.NodeType == NT_CLASSDEF })
	clsTable := NodNewChildList(NT_CLASSTABLE, clsDefs)
	NodSetChild(x.Root, NTR_CLASSTABLE, clsTable)
}

func (x *XformerPocket) buildClassVardefTable(clsDef Nod) {
	clsUnits := NodGetChildList(clsDef)
	varDefs := []Nod{}
	for _, clsUnit := range clsUnits {
		if clsUnit.NodeType == NT_CLASSFIELD {
			varDef := NodNew(NT_VARDEF)
			NodSetChild(varDef, NTR_VARDEF_NAME, NodGetChild(clsUnit, NTR_VARDEF_NAME))
			NodSetChild(varDef, NTR_VARDEF_SCOPE, NodNewData(NT_VARDEF_SCOPE, VSCOPE_CLASSFIELD))
			varDefs = append(varDefs, varDef)
		}
	}
	varTable := NodNewChildList(NT_VARTABLE, varDefs)
	NodSetChild(clsDef, NTR_VARTABLE, varTable)
}

func (x *XformerPocket) buildClassFuncdefTable(clsDef Nod) {
	clsUnits := NodGetChildList(clsDef)
	fDefs := []Nod{}
	for _, clsUnit := range clsUnits {
		if clsUnit.NodeType == NT_FUNCDEF {
			fDefs = append(fDefs, clsUnit)
		}
	}
	fTable := NodNewChildList(NT_FUNCTABLE, fDefs)
	NodSetChild(clsDef, NTR_FUNCTABLE, fTable)
}

func (x *XformerPocket) buildRootFuncTable() {
	topLevelUnits := NodGetChildList(x.Root)
	funcDefs := x.SearchNodList(topLevelUnits, func(n Nod) bool { return n.NodeType == NT_FUNCDEF })
	funcTable := NodNewChildList(NT_FUNCTABLE, funcDefs)
	NodSetChild(x.Root, NTR_FUNCTABLE, funcTable)
}

func (x *XformerPocket) buildLocalVarDefTables() {
	// find all local variable assignments and construct
	// the canonical union of variables

	// determine which (top-level) imperatives have which local vars
	localVarWrites := x.SearchRoot(func(n Nod) bool {
		return x.isLocalVarRef(n) && (n.NodeType == NT_PARAMETER || n.NodeType == NT_VARASSIGN)
	})
	impToVarRef := make(map[Nod][]Nod)
	for _, varRef := range localVarWrites {
		funcDef := x.findTopLevelFuncDef(varRef)
		if funcDef == nil {
			panic("failed to find top level imperative")
		}
		impToVarRef[funcDef] = append(impToVarRef[funcDef], varRef)
	}

	// next, generate the vartables for each imperative
	for imper, varRefs := range impToVarRef {
		varTable := x.generateVarTableFromLocalVarRefs(varRefs)
		NodSetChild(imper, NTR_VARTABLE, varTable)
	}

	x.linkVarsToVarTables()
	x.computeVariableScopes()

}

func (x *XformerPocket) linkVarsToVarTables() {

	// link up all unresolved var assignments and var getters to refer to this unified table
	varRefs := x.SearchRoot(func(n Nod) bool {
		return n.NodeType == NT_VAR_GETTER || n.NodeType == NT_VARASSIGN ||
			n.NodeType == NT_PARAMETER
	})
	for _, varRef := range varRefs {
		// get the var table associated with this var reference

		parentContainer := x.getContainingNodOrNil(varRef, func(n Nod) bool {
			return NodHasChild(n, NTR_VARTABLE)
		})

		if parentContainer == nil {
			panic("failed to find any variable table for var ref")
		}

		imParent := parentContainer
		imVartable := NodGetChild(imParent, NTR_VARTABLE)

		// get the var name as a string, which serves as the lookup key in the var table
		varName := x.getVarNameFromVarRef(varRef)

		// search the varTable for the name (naive linear search but should always be small list)
		matchedVarDef := x.searchVarTableForNameDeep(imVartable, varName)
		if matchedVarDef == nil {
			panic("unknown var: '" + varName + "'")
		}
		// finally, store a reference to the definition
		NodSetChild(varRef, NTR_VARDEF, matchedVarDef)
	}

}

func (x *XformerPocket) searchVarTableForNameDeep(varTable Nod, varName string) Nod {
	// search the varTable for the name (naive linear search but should always be small list)
	varDefs := NodGetChildList(varTable)
	for _, varDef := range varDefs {
		varDefVarName := NodGetChild(varDef, NTR_VARDEF_NAME).Data.(string)
		if varDefVarName == varName {
			return varDef
		}
	}
	// recurse to parent if nothing found locally
	if parentTable := NodGetChildOrNil(varTable, NTR_TABLE_PARENT); parentTable != nil {
		return x.searchVarTableForNameDeep(parentTable, varName)
	}
	return nil
}

func (x *XformerPocket) computeVariableScopes() {
	// varTables := x.SearchRoot(func(n Nod) bool { return n.NodeType == NT_VARTABLE })
	// for _, varTable := range varTables {
	// 	varDefs := NodGetChildList(varTable)
	// 	for _, varDef := range varDefs {
	// 		varScope := VSCOPE_FUNCLOCAL
	// 		incomingNods := varDef.In
	// 		for _, inEdge := range incomingNods {
	// 			inNod := inEdge.In
	// 			if inNod.NodeType == NT_PARAMETER {
	// 				varScope = VSCOPE_FUNCPARAM
	// 				break
	// 			}
	// 		}
	// 		// scope has been computed, now save it
	// 		NodSetChild(varDef, NTR_VARDEF_SCOPE, NodNewData(NT_VARDEF_SCOPE, varScope))
	// 	}
	// }
}

func (x *XformerPocket) getVarNameFromVarRef(varRef Nod) string {
	if varRef.NodeType == NT_PARAMETER {
		return NodGetChild(varRef, NTR_VARDEF_NAME).Data.(string)
	} else if varRef.NodeType == NT_VAR_GETTER || varRef.NodeType == NT_VARASSIGN {
		return NodGetChild(varRef, NTR_VAR_NAME).Data.(string)
	} else {
		panic("unhandled var ref type")
	}
}

func (x *XformerPocket) generateVarTableFromLocalVarRefs(varRefs []Nod) Nod {
	// incoming var refs are either parameters or assignments
	// an assignment defines a local variable iff no name match at the outer
	// class level

	varDefsByName := make(map[string]Nod)

	for _, varRef := range varRefs {
		varName := x.getVarNameFromVarRef(varRef)
		varDef := NodNew(NT_VARDEF)

		// annotate scope if we can
		var scope = -1
		if varRef.NodeType == NT_PARAMETER {
			scope = VSCOPE_FUNCPARAM
		} else if varRef.NodeType == NT_VARASSIGN {
			scope = VSCOPE_FUNCLOCAL
		}
		if scope != -1 {
			NodSetChild(varDef, NTR_VARDEF_SCOPE, NodNewData(NT_VARDEF_SCOPE, scope))
		}

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
	return x.getContainingNodOrNil(n, func(n Nod) bool {
		return n.NodeType == NT_FUNCDEF
	})
}
