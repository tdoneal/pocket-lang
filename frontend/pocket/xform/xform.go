package xform

import (
	"fmt"
	. "pocket-lang/frontend/pocket/common"
	. "pocket-lang/parse"
	. "pocket-lang/xform"
	"reflect"
	"runtime"
	"strconv"

	"github.com/davecgh/go-spew/spew"
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

	x.NodCheckParentChildIntegrity()

	x.prepare()

	x.NodCheckParentChildIntegrity()
	x.desugar()

	x.NodCheckParentChildIntegrity()

	fmt.Println("after desugaring:", PrettyPrint(x.Root))

	x.solve()
	fmt.Println("after solving", PrettyPrint(x.Root))

	x.colorTypes()

	x.checkAllVarsResolved()
	x.checkAllCallsResolved()

	fmt.Println("after xform:", PrettyPrint(x.Root))
}

func (x *XformerPocket) solve() {
	// resolves identifiers and types to the best of our ability
	rules := x.getAllSolveRules()
	nodes := x.getSolvableNodes()
	x.initializeSolvableNodes(nodes)
	fmt.Println("after initializing solveable nodes", PrettyPrint(x.Root))

	x.applyGraphRewritesUntilStable(rules)
}

func (x *XformerPocket) initializeSolvableNodes(ns []Nod) {
	for _, n := range ns {
		x.initializeSolvableNode(n)
	}
	x.buildNamespaceHierarchy()
}

func (x *XformerPocket) buildNamespaceHierarchy() {
	// connects the name spaces to their "searchable" parents

	x.NodCheckParentChildIntegrity()

	allns := x.SearchRoot(func(n Nod) bool { return n.NodeType == NT_NAMESPACE })
	for _, ns := range allns {
		synContainer := NodGetParent(ns, NTR_NAMESPACE)

		parentContainer := x.getContainingNodOrNil(synContainer,
			func(ni Nod) bool { return NodHasChild(ni, NTR_NAMESPACE) && ni != synContainer })

		if parentContainer != nil {
			parentNs := NodGetChild(parentContainer, NTR_NAMESPACE)
			NodSetChild(ns, NTR_NAMESPACE_PARENT, parentNs)
		}
	}

}

func (x *XformerPocket) initializeSolvableNode(n Nod) {
	// TODO: start modularizing this, or move some of the logic to the solve rules
	nt := n.NodeType
	if isMypedValueType(nt) || nt == NT_VARDEF {
		x.ISNMyped(n)
	} else if nt == NT_FUNCDEF {
		x.ISNFuncDef(n)
	} else if nt == NT_CLASSDEF || nt == NT_CLASSDEFPARTIAL {
		x.ISNClassDef(n)
	} else if nt == NT_TOPLEVEL {
		x.ISNRoot(n)
	} else {
		// purposeful pass
	}
}

func (x *XformerPocket) ISNMyped(n Nod) {
	x.initializePosNegMypes(n)
}

func (x *XformerPocket) ISNRoot(n Nod) {
	x.buildClassTable()
	x.buildRootFuncTable()
	x.ISNInitNamespace(x.Root, false, true, true)
}

func (x *XformerPocket) ISNInitNamespace(n Nod, hasVars bool, hasFuncs bool, hasClasses bool) {
	tlNamespace := NodNew(NT_NAMESPACE)
	if hasVars {
		NodSetChild(tlNamespace, NTR_VARTABLE, NodGetChild(n, NTR_VARTABLE))
	}
	if hasFuncs {
		NodSetChild(tlNamespace, NTR_FUNCTABLE, NodGetChild(n, NTR_FUNCTABLE))
	}
	if hasClasses {
		NodSetChild(tlNamespace, NTR_CLASSTABLE, NodGetChild(n, NTR_CLASSTABLE))
	}
	NodSetChild(n, NTR_NAMESPACE, tlNamespace)
}

func (x *XformerPocket) ISNClassDef(n Nod) {
	x.buildClassVardefTable(n)
	x.buildClassFuncdefTable(n)
	cVarDefs := NodGetChildList(NodGetChild(n, NTR_VARTABLE))
	for _, cVarDef := range cVarDefs {
		x.initializePosNegMypes(cVarDef)
	}
	x.ISNInitNamespace(n, true, true, false)

	// classdefs can be values too, so init their type stuff
	// but only named classes can be referred to
	if n.NodeType == NT_CLASSDEF {
		x.initializePosNegMypes(n)
	}
}

func (x *XformerPocket) ISNFuncDef(n Nod) {
	varTable := NodNew(NT_VARTABLE)
	NodSetChild(n, NTR_VARTABLE, varTable)
	rvPlaceholder := NodNew(NT_FUNCDEF_RV_PLACEHOLDER)
	NodSetChild(n, NTR_RETURNVAL_PLACEHOLDER, rvPlaceholder)
	x.initializePosNegMypes(rvPlaceholder)

	// initialize the self def into the var table if applicable
	if selfDef := NodGetChildOrNil(n, NTR_METHOD_SELFDEF); selfDef != nil {
		x.addVarToVartable(varTable, selfDef)
	}

	// funcdefs can be values too, so init their type stuff
	x.initializePosNegMypes(n)

	x.ISNInitNamespace(n, true, false, false)
}

func (x *XformerPocket) getSolvableNodes() []Nod {
	return x.SearchRoot(func(n Nod) bool {
		return x.isSolvableNode(n)
	})
}

func (x *XformerPocket) isSolvableNode(n Nod) bool {
	return true
}

func (x *XformerPocket) getAllSolveRules() []*RewriteRule {
	typeRules := x.getAllSolveTypeRules()
	idRules := x.getIdentifierRewriteRules()

	rv := append(typeRules, idRules...)

	// for ndx, rule := range rv {
	// 	// fmt.Println("rule", ndx, rule, GetRewriteRuleDebugInfo(rule))
	// }

	return rv
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

func (x *XformerPocket) checkAllVarsResolved() {
	getters := x.SearchRoot(func(n Nod) bool {
		return n.NodeType == NT_VAR_GETTER
	})
	for _, getter := range getters {
		if !NodHasChild(getter, NTR_VARDEF) {
			varBase := NodGetChild(getter, NTR_VAR_NAME)
			// for now, only error if the variable is simple
			if varBase.NodeType == NT_IDENTIFIER ||
				varBase.NodeType == NT_IDENTIFIER_NOSCOPE {
				panic("unknown variable '" + varBase.Data.(string) + "'")
			}
		}
	}

	x.SearchRoot(func(n Nod) bool {
		if n.NodeType == NT_IDENTIFIER_KWARG {
			panic("unresolved keyword argument: '" + n.Data.(string) + "'")
		}
		return false
	})
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

type RewriteRule struct {
	// returns whether progress made
	condaction func(n Nod) bool

	// old-school condition-action paradigm (deprecated)
	// TODO: replace all instances of this, and remove
	condition func(n Nod) bool
	action    func(n Nod)
}

// generic go function
func GetRewriteRuleDebugInfo(rule *RewriteRule) string {
	if rule.condaction != nil {
		return GetFPointerDebugInfo(rule.condaction)
	}
	return GetFPointerDebugInfo(rule.condition)
}

func GetFPointerDebugInfo(f interface{}) string {
	fPointer := reflect.ValueOf(f).Pointer()
	funcObject := runtime.FuncForPC(fPointer)
	file, line := funcObject.FileLine(fPointer)
	return file + ": " + strconv.Itoa(line)
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

func isUnaryOpType(nt int) bool {
	return isPrefixOpType(nt)
}

func isPrefixOpType(nt int) bool {
	return nt == NT_REFERENCEOP
}

func isSuffixOpType(nt int) bool {
	return false
}

func isRValVarReferenceNT(nt int) bool {
	return nt == NT_VAR_GETTER || nt == NT_VARASSIGN || nt == NT_PARAMETER ||
		nt == NT_IDENTIFIER_RVAL || nt == NT_IDENTIFIER_NOSCOPE || nt == NT_OBJFIELD_ACCESSOR
}

func isCallType(nt int) bool {
	return nt == NT_RECEIVERCALL || nt == NT_RECEIVERCALL_CMD ||
		nt == NT_RECEIVERCALL_METHOD || nt == NT_OBJINIT
}

func (x *XformerPocket) applyGraphRewritesUntilStable(rules []*RewriteRule) {
	// repeatedly apply rewrite rules, allowing for new nodes to pop in or out of existence
	// by re-generating the node list after each pass
	nPasses := 0
	for {
		maxApplied := 0
		allNods := x.SearchRoot(func(n Nod) bool { return true })
		for _, rule := range rules {
			// fmt.Println("applying rule", GetRewriteRuleDebugInfo(rule))
			nApplied := x.applyRewriteRuleOnJust(allNods, rule)
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
	// fmt.Println("START RULE APPLY to nods", rule, GetRewriteRuleDebugInfo(rule))
	for _, ele := range nods {
		if rule.condaction != nil {
			applied := rule.condaction(ele)
			if applied {
				nApplied++
				integrity := x.NodComputeParentChildIntegrity()
				if !integrity {
					fmt.Println("integrity check after rule", spew.Sdump(rule.condaction))
					// for finding a stack trace of the rule, try to break it
					rule.condaction(nil)
					panic("integrity check failed")
				}
			}
		} else {
			// TODO: remove this old-school condition-action approach
			if rule.condition(ele) {
				rule.action(ele)
				nApplied++
			}
		}

	}
	return nApplied
}

func (x *XformerPocket) removeNodListAt(nods []Nod, removeAt int) []Nod {
	return append(nods[:removeAt], nods[removeAt+1:]...)
}
