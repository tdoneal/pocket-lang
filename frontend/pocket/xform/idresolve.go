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
		x.IRRLocalOrClassVar(),
	}
	return rv
}

func (x *XformerPocket) IRRLocalOrClassVar() *RewriteRule {
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_VARASSIGN || n.NodeType == NT_VAR_GETTER {
				if !NodHasChild(n, NTR_VARDEF) {
					cCls := x.getContainingClassDef(n)
					if cCls != nil {
						return true
					}
				}
			}
			return false
		},
		action: func(n Nod) {
			cCls := x.getContainingClassDef(n)
			cTable := NodGetChild(cCls, NTR_VARTABLE)
			varName := x.getLocalVarName(n)

			clsVarDef := x.varTableLookup(cTable, varName)
			tfunc := x.getContainingFuncDef(n)
			localVarTable := NodGetChild(tfunc, NTR_VARTABLE)
			localVarDef := x.varTableLookup(localVarTable, varName)
			if localVarDef != nil {
				NodSetChild(n, NTR_VARDEF, localVarDef)
				return
			}
			if clsVarDef != nil {
				NodSetChild(n, NTR_VARDEF, clsVarDef)
				return
			}
			// if we got here, it's a new local variable for sure
			nvd := NodNew(NT_VARDEF)
			NodSetChild(nvd, NTR_VARDEF_NAME, NodNewData(NT_IDENTIFIER, varName))
			x.addVarToVartable(localVarTable, nvd)
			NodSetChild(n, NTR_VARDEF, nvd)
		},
	}
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
	for _, clsDef := range clsDefs {
		x.buildClassVardefTable(clsDef)
	}
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

func (x *XformerPocket) buildFuncDefTables() {
	topLevelUnits := NodGetChildList(x.Root)
	funcDefs := x.SearchNodList(topLevelUnits, func(n Nod) bool { return n.NodeType == NT_FUNCDEF })
	clsDefs := NodGetChildList(NodGetChild(x.Root, NTR_CLASSTABLE))
	funcTable := NodNewChildList(NT_FUNCTABLE, funcDefs)
	NodSetChild(x.Root, NTR_FUNCTABLE, funcTable)

	// link functional calls to their associated def (if found)
	calls := x.SearchRoot(func(n Nod) bool { return isReceiverCallType(n.NodeType) })
	for _, call := range calls {
		base := NodGetChild(call, NTR_RECEIVERCALL_BASE)
		if callName, ok := base.Data.(string); ok {
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

	// link calls to associated object initializer if aproppriate
	for _, call := range calls {
		base := NodGetChild(call, NTR_RECEIVERCALL_BASE)
		if callName, ok := base.Data.(string); ok {
			// lookup function in class table (indicating an object initializer)
			var matchedClsDef Nod
			for _, clsDef := range clsDefs {
				clsName := NodGetChild(clsDef, NTR_CLASSDEF_NAME).Data.(string)
				if callName == clsName {
					matchedClsDef = clsDef
					break
				}
			}
			if matchedClsDef != nil {
				// rewrite the call in-place -> object initializer
				NodSetChild(call, NTR_RECEIVERCALL_BASE, matchedClsDef)
				call.NodeType = NT_OBJINIT
			}
		}
	}
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
