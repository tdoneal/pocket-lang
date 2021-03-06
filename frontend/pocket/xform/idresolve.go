package xform

import (
	"fmt"
	. "pocket-lang/frontend/pocket/common"
	. "pocket-lang/parse"
)

func (x *XformerPocket) getIdentifierRewriteRules() []*RewriteRule {
	rv := []*RewriteRule{
		x.IRRBaseIdentifiers(),
		x.IRRBaseIdentifiersVarGet(),
		x.IRRBaseCallIdentifiers(),
		x.IRRTypeDeclIdentifiers(),
		x.IRRParametersToLocals(),

		// these operate on identifiers that have a bit of known information
		// about them
		x.IRRNoscopesLocals(),
		x.IRRNoscopesFuncGlobal(),
		x.IRRNoscopesFuncOwnClass(),
		x.IRRNoscopesFuncObjInit(),
		x.IRRNoscopesType(),
		x.IRRNoscopesFuncLocalVar(),

		// these operate on generic identifiers, about which little is known
		x.IRRNoscopesGeneric(),

		x.IRRKeywordArgsFuncDef(),
		x.IRRKeywordArgsObjInit(),
		// TODO: somehow re-use the var lookup framework to resolve certain Noscope Funcs
		x.IRRSimpleVarWrites(),
		x.IRRPlainObjInit(),
		x.IRRArgedObjInit(),
		x.IRRReturnToPlaceholder(),
		x.IRRResolveMethodCalls(),
	}
	return rv
}

func (x *XformerPocket) IRRKeywordArgsObjInit() *RewriteRule {
	// make progress on keyword arguments that refer to object initializer fields
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_IDENTIFIER_KWARG {
				varName := n.Data.(string)
				parentKwArg := NodGetParent(n, NTR_VAR_NAME)
				parentKwargs := NodGetParentByOrNil(parentKwArg, func(n Nod) bool { return n.NodeType == NT_KWARGS })
				parentCall := NodGetParent(parentKwargs, NTR_RECEIVERCALL_ARG)
				base := NodGetChild(parentCall, NTR_RECEIVERCALL_BASE)
				if parentCall.NodeType == NT_OBJINIT && base.NodeType == NT_CLASSDEF {
					cDef := base
					cvTable := NodGetChild(cDef, NTR_VARTABLE)
					vDef := x.varTableLookup(cvTable, varName)
					if vDef != nil {
						NodSetChild(parentKwArg, NTR_VARDEF, vDef)
						n.NodeType = NT_IDENTIFIER_RESOLVED
						return true
					}
				}
			}
			return false
		},
	}
}

func (x *XformerPocket) IRRKeywordArgsFuncDef() *RewriteRule {
	// make progress on keyword arguments that refer to function parameters
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_IDENTIFIER_KWARG {
				varName := n.Data.(string)
				parentKwArg := NodGetParent(n, NTR_VAR_NAME)
				parentKwargs := NodGetParentByOrNil(parentKwArg, func(n Nod) bool { return n.NodeType == NT_KWARGS })
				parentCall := NodGetParent(parentKwargs, NTR_RECEIVERCALL_ARG)
				if NodHasChild(parentCall, NTR_FUNCDEF) {
					fDef := NodGetChild(parentCall, NTR_FUNCDEF)
					fvTable := NodGetChild(fDef, NTR_VARTABLE)
					vDef := x.varTableLookup(fvTable, varName)
					if vDef != nil {
						return true
					}
				}
			}
			return false
		},
		action: func(n Nod) {
			varName := n.Data.(string)
			parentKwArg := NodGetParent(n, NTR_VAR_NAME)
			parentKwargs := NodGetParentByOrNil(parentKwArg,
				func(n Nod) bool { return n.NodeType == NT_KWARGS })
			parentCall := NodGetParent(parentKwargs, NTR_RECEIVERCALL_ARG)
			fDef := NodGetChild(parentCall, NTR_FUNCDEF)
			fvTable := NodGetChild(fDef, NTR_VARTABLE)
			vDef := x.varTableLookup(fvTable, varName)
			NodSetChild(parentKwArg, NTR_VARDEF, vDef)
			n.NodeType = NT_IDENTIFIER_RESOLVED
		},
	}
}

func (x *XformerPocket) IRRTypeDeclIdentifiers() *RewriteRule {
	// make progress on identifiers directly within type declarations
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_VARASSIGN || n.NodeType == NT_CLASSFIELD {
				if typeDecl := NodGetChildOrNil(n, NTR_TYPE_DECL); typeDecl != nil {
					if typeDecl.NodeType == NT_IDENTIFIER {
						return true
					}
				}
			}
			return false
		},
		action: func(n Nod) {
			typeDecl := NodGetChild(n, NTR_TYPE_DECL)
			typeDecl.NodeType = NT_IDENTIFIER_TYPE_NOSCOPE
		},
	}
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

func (x *XformerPocket) IRRNoscopesType() *RewriteRule {
	// make progress on single-word references to known types
	return &RewriteRule{
		condition: func(n Nod) bool {

			if n.NodeType == NT_IDENTIFIER_TYPE_NOSCOPE {

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
			n.NodeType = NT_IDENTIFIER_RESOLVED
			x.Replace(n, cDef)
		},
	}
}

func (x *XformerPocket) IRRPlainObjInit() *RewriteRule {
	// look up single-word references to classes, treat as either object initializers
	// or direct class references, depending on the context
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_IDENTIFIER_NOSCOPE {
				idtext := n.Data.(string)
				cDef := x.globalClassDefLookup(idtext)
				if cDef != nil {
					// if in ref op, treat as that
					if prnt := NodGetParentByOrNil(n,
						func(n0 Nod) bool { return n0.NodeType == NT_REFERENCEOP }); prnt != nil {
						x.Replace(prnt, cDef) // replace the whole ref op with the classdef
						n.NodeType = NT_IDENTIFIER_RESOLVED
					} else { // otherwise treat as object initializer
						n.NodeType = NT_OBJINIT
						NodSetChild(n, NTR_RECEIVERCALL_BASE, cDef)
						NodSetChild(n, NTR_RECEIVERCALL_ARG, NodNew(NT_EMPTYARGLIST))
					}
					return true
				}
			}
			return false
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
			n.NodeType = NT_IDENTIFIER_NOSCOPE
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

func (x *XformerPocket) IRRBaseIdentifiersVarGet() *RewriteRule {
	// simple identifiers inside a var get can be rewritten as IDENTIFIER_RVAL_NOSCOPE
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_VAR_GETTER {
				lvalue := NodGetChild(n, NTR_VAR_NAME)
				if lvalue.NodeType == NT_IDENTIFIER {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			lvalue := NodGetChild(n, NTR_VAR_NAME)
			lvalue.NodeType = NT_IDENTIFIER_NOSCOPE
		},
	}
}

func (x *XformerPocket) resolveIdentifierRValNoscopeAsVar(ident Nod, varDef Nod) {
	idtext := ident.Data.(string)
	if varGetter := NodGetParentOrNil(ident, NTR_VAR_NAME); varGetter != nil {
		// rewrite the existing vargetter
		ident.NodeType = NT_IDENTIFIER_RESOLVED
		NodSetChild(varGetter, NTR_VARDEF, varDef)
	} else {
		// create a vargetter out of this identifier
		ident.NodeType = NT_VAR_GETTER
		varName := NodNewData(NT_IDENTIFIER_RESOLVED, idtext)
		NodSetChild(ident, NTR_VAR_NAME, varName)
		NodSetChild(ident, NTR_VARDEF, varDef)
		ident.Data = nil
	}

}

func (x *XformerPocket) IRRNoscopesGeneric() *RewriteRule {
	// make progress on generic rvalue references: look up as either class, func, or var
	return &RewriteRule{
		condaction: func(n Nod) bool {
			if n.NodeType == NT_IDENTIFIER_NOSCOPE {
				idtext := n.Data.(string)
				fmt.Println("looking up unresolved generic identifier:", idtext)
				iDef := x.containingNamespaceLookup(n, idtext)
				if iDef != nil {
					if iDef.NodeType == NT_FUNCDEF {
						n.NodeType = NT_IDENTIFIER_RESOLVED
						NodSetChild(n, NTR_FUNCDEF, iDef)
					} else if iDef.NodeType == NT_VARDEF {
						n.NodeType = NT_VAR_GETTER
						NodSetChild(n, NTR_VARDEF, iDef)
						n.Data = nil
						NodSetChild(n, NTR_VAR_NAME, NodNewData(NT_IDENTIFIER_RESOLVED, idtext))
					} else if iDef.NodeType == NT_CLASSDEF {
						x.Replace(n, iDef)
					} else {
						panic("unsupported resolved identifier type")
					}
					return true
				}
			}
			return false
		},
	}
}

func (x *XformerPocket) IRRNoscopesFuncOwnClass() *RewriteRule {
	// make progress towards resolving NT_IDENTIFIER_FUNC_NOSCOPE: lookup in own class func table
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_IDENTIFIER_FUNC_NOSCOPE {
				idtext := n.Data.(string)
				cCls := x.getContainingClassDef(n)
				if cCls != nil {
					fTable := NodGetChild(cCls, NTR_FUNCTABLE)
					fDef := x.funcTableLookup(fTable, idtext)
					if fDef != nil {
						return true
					}
				}
			}
			return false
		},
		action: func(n Nod) {
			// area() -> self.area()

			idtext := n.Data.(string)
			cCls := x.getContainingClassDef(n)
			fTable := NodGetChild(cCls, NTR_FUNCTABLE)
			fDef := x.funcTableLookup(fTable, idtext)

			parentCall := NodGetParent(n, NTR_RECEIVERCALL_BASE)
			NodSetChild(parentCall, NTR_FUNCDEF, fDef)
			parentCall.NodeType = NT_RECEIVERCALL_METHOD

			NodSetChild(parentCall, NTR_RECEIVERCALL_METHOD_NAME, n)
			n.NodeType = NT_IDENTIFIER_RESOLVED

			newBase := NodNewData(NT_IDENTIFIER_NOSCOPE, "self")
			NodSetChild(parentCall, NTR_RECEIVERCALL_BASE, newBase)

			x.initializeSolvableNode(newBase)

		},
	}
}
func (x *XformerPocket) containingNamespaceLookup(start Nod, idtext string) Nod {
	contns := x.getContainingNodOrNil(start, func(n Nod) bool { return NodHasChild(n, NTR_NAMESPACE) })
	if contns == nil {
		return nil
	}
	immedNamespace := NodGetChild(contns, NTR_NAMESPACE)
	return x.namespaceLookupRecursive(immedNamespace, idtext)
}

func (x *XformerPocket) namespaceLookupRecursive(ns Nod, idtext string) Nod {
	immedResult := x.namespaceLookupImmediate(ns, idtext)
	if immedResult != nil {
		return immedResult
	}
	if parentNs := NodGetChildOrNil(ns, NTR_NAMESPACE_PARENT); parentNs != nil {
		return x.namespaceLookupRecursive(parentNs, idtext)
	}
	return nil
}

func (x *XformerPocket) namespaceLookupImmediate(ns Nod, idtext string) Nod {
	// todo: support class/var defs
	fmt.Println("ns lookup immediate: '", idtext, "' in", PrettyPrint(ns))
	if fTable := NodGetChildOrNil(ns, NTR_FUNCTABLE); fTable != nil {
		fResult := x.funcTableLookup(fTable, idtext)
		if fResult != nil {
			return fResult
		}
	}
	if varTable := NodGetChildOrNil(ns, NTR_VARTABLE); varTable != nil {
		vResult := x.varTableLookup(varTable, idtext)
		if vResult != nil {
			return vResult
		}
	}
	if clsTable := NodGetChildOrNil(ns, NTR_CLASSTABLE); clsTable != nil {
		clsResult := x.classTableLookup(clsTable, idtext)
		if clsResult != nil {
			return clsResult
		}
	}
	return nil
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
					return true
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
					return true
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
					return true
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
			x.initializeSolvableNode(varGetter)
			NodSetChild(varGetter, NTR_VAR_NAME, n)
			NodSetChild(varGetter, NTR_VARDEF, vDef)
			NodSetChild(parentCall, NTR_RECEIVERCALL_BASE, varGetter)
			n.NodeType = NT_IDENTIFIER_RESOLVED

		},
	}
}

func (x *XformerPocket) IRRNoscopesLocals() *RewriteRule {
	// make progress towards resolving NT_IDENTIFIER_RVAL_NOSCOPEs: check for local variable
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_IDENTIFIER_NOSCOPE {
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
			x.resolveIdentifierRValNoscopeAsVar(n, fVarDef)
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
			NodSetChild(clsUnit, NTR_VARDEF, varDef)
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
