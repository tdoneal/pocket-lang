package pocket

import (
	"fmt"
	. "pocket-lang/parse"
	. "pocket-lang/xform"
)

const (
	TY_OBJECT = 1
	TY_NUMBER = 2
	TY_INT    = 3
	TY_FLOAT  = 4
	TY_STRING = 5
	TY_SET    = 6
	TY_MAP    = 7
	TY_LIST   = 8
	TY_DUCK   = 30
)

type Mype interface{}

type MypeExplicit struct {
	types map[int]bool
}

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

func (x *XformerPocket) getInitMype() Nod {
	md := &MypeExplicit{
		types: map[int]bool{
			TY_OBJECT: true,
			TY_INT:    true,
		},
	}
	return NodNewData(NT_MYPE, md)
}

func isLiteralNodeType(nt int) bool {
	return nt == NT_LIT_INT || nt == NT_LIT_STRING
}

func getLiteralTypeAnnDataFromNT(nt int) int {
	lut := map[int]int{
		NT_LIT_INT:    TY_INT,
		NT_LIT_STRING: TY_STRING,
	}
	return lut[nt]
}

func isBinaryOpType(nt int) bool {
	return nt == NT_ADDOP || nt == NT_GTOP || nt == NT_LTOP
}

func isVarReferenceNT(nt int) bool {
	return nt == NT_VAR_GETTER || nt == NT_VARASSIGN || nt == NT_PARAMETER
}

func (x *XformerPocket) solveTypes() {
	// // assign a concrete type to every node

	// first: initialize all mypes in the vartable
	varDefs := x.SearchRoot(func(n Nod) bool { return n.NodeType == NT_VARDEF })
	for _, value := range varDefs {
		NodSetChild(value, NTR_MYPE, x.getInitMype())
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
			NodSetChild(value, NTR_MYPE, NodGetChild(varDef, NTR_MYPE))
		} else {
			NodSetChild(value, NTR_MYPE, x.getInitMype())
		}
	}

	fmt.Println("after initial mype assignments:", PrettyPrintMypes(values))

	// apply repeated solve rules until convergence (for system 1 semantics)
	x.applyRewritesUntilStable(values, []*RewriteRule{
		marLiterals(),
		marVarAssign(),
		marAddOp(),
	})

	// for now: enforce that all type must be fully resolved by now
	// TODO: support uncertainty here followed by an explicit search over the possible types
	// followed by static type verification (the verification is a different set of rules)
	allResolved := true
	allToCheck := append(values, varDefs...)
	for _, ele := range allToCheck {
		if NodGetChild(ele, NTR_MYPE).NodeType != NT_TYPE {
			fmt.Println("mypes after applying heuristics:", PrettyPrintMype(ele))
			allResolved = false
			break
		}
	}

	if !allResolved {
		// TODO: don't panic here
		fmt.Println("Unable to resolve all types, ambiguity existed in at least one mype")
	}
	// explicitly convert mypes to types (and remove the mypes in the process)
	x.applyRewriteOnGraph(&RewriteRule{
		condition: func(n Nod) bool {
			return NodHasChild(n, NTR_MYPE)
		},
		action: func(n Nod) {
			mype := NodGetChild(n, NTR_MYPE)
			var assignType Nod
			if _, ok := mype.Data.(int); ok {
				assignType = mype
			}
			if _, ok := mype.Data.(*MypeExplicit); ok {
				assignType = NodNewData(NT_TYPE, TY_DUCK)
			}
			if assignType == nil {
				panic("unhandled mype type")
			}
			NodSetChild(n, NTR_TYPE, assignType)
			NodRemoveChild(n, NTR_MYPE)
		},
	})

	fmt.Println("Final type assignments:", PrettyPrintMypes(values))

}

func marVarAssign() *RewriteRule {
	// propagate var assign values from rhs -> lhs
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_VARASSIGN {
				mypeLHS := NodGetChild(n, NTR_MYPE)
				mypeRHS := NodGetChild(NodGetChild(n, NTR_VARASSIGN_VALUE), NTR_MYPE)
				lhsIsConcrete := mypeLHS.NodeType == NT_TYPE
				rhsIsConcrete := mypeRHS.NodeType == NT_TYPE
				if rhsIsConcrete && !lhsIsConcrete {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			writeTypeAndData(NodGetChild(n, NTR_MYPE), NodGetChild(NodGetChild(n, NTR_VARASSIGN_VALUE), NTR_MYPE))
			fmt.Println("Applied MAR: VarAssign")
		},
	}
}

func marAddOp() *RewriteRule {
	// for now, say that all add ops force all involved mypes (both args and result) to int
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_ADDOP {
				resultMype := NodGetChild(n, NTR_MYPE)
				argMypes := []Nod{
					NodGetChild(NodGetChild(n, NTR_BINOP_LEFT), NTR_MYPE),
					NodGetChild(NodGetChild(n, NTR_BINOP_RIGHT), NTR_MYPE),
				}
				if resultMype.NodeType != NT_TYPE || argMypes[0].NodeType != NT_TYPE ||
					argMypes[1].NodeType != NT_TYPE {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			intType := NodNewData(NT_TYPE, TY_INT)
			writeTypeAndData(NodGetChild(n, NTR_MYPE), intType)
			writeTypeAndData(NodGetChild(NodGetChild(n, NTR_BINOP_LEFT), NTR_MYPE), intType)
			writeTypeAndData(NodGetChild(NodGetChild(n, NTR_BINOP_RIGHT), NTR_MYPE), intType)
			fmt.Println("Applied add op int forcer to", PrettyPrintMype(n))
		},
	}
}

func writeTypeAndData(dst Nod, src Nod) {
	dst.NodeType = src.NodeType
	dst.Data = src.Data
}

func marLiterals() *RewriteRule {
	return &RewriteRule{
		condition: func(n Nod) bool {
			if isLiteralNodeType(n.NodeType) {
				if NodGetChild(n, NTR_MYPE).NodeType != NT_TYPE {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			ty := getLiteralTypeAnnDataFromNT(n.NodeType)
			NodSetChild(n, NTR_MYPE, NodNewData(NT_TYPE, ty))
			fmt.Println("Applied marLitIntsBase to", PrettyPrintMype(n))
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
