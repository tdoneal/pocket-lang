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
	x.solveTypes()
	x.buildVarDefTables()
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

func (x *XformerPocket) solveTypes() {
	// assign a concrete type to every node
	// first gather all value nodes
	// TODO: multi-function support
	values := x.SearchRoot(func(n Nod) bool {
		nt := n.NodeType
		return nt == NT_LIT_INT || nt == NT_ADDOP || nt == NT_VAR_GETTER || nt == NT_VARASSIGN
	})

	// assign an initial mype to all
	for _, value := range values {
		mype := &MypeExplicit{
			types: map[int]bool{
				TY_OBJECT: true,
				TY_INT:    true,
			},
		}
		NodSetChild(value, NTR_MYPE, NodNewData(NT_MYPE, mype))
	}

	// apply repeated solve rules until convergence (for system 1 semantics)
	x.applyRewritesUntilStable(values, []*RewriteRule{
		marLitIntsBase(),
		marVarAssign(),
		marAddOp(),
	})

	// for now: enforce that all type must be fully resolved by now
	// TODO: support uncertainty here followed by an explicit search over the possible types
	// followed by static type verification (the verification is a different set of rules)
	allResolved := true
	for _, ele := range values {
		if NodGetChild(ele, NTR_MYPE).NodeType != NT_TYPE {
			fmt.Println("mypes after applying heuristics:", PrettyPrintMypes(values))
			allResolved = false
			break
		}
	}

	if !allResolved {
		// TODO: don't panic here
		panic("Unable to resolve types, ambiguity existed in at least one mype")
	}
	// explicitly convert mypes to types (and remove the mypes in the process)
	x.applyRewriteOnGraph(&RewriteRule{
		condition: func(n Nod) bool {
			return NodHasChild(n, NTR_MYPE)
		},
		action: func(n Nod) {
			mype := NodGetChild(n, NTR_MYPE)
			NodSetChild(n, NTR_TYPE, mype)
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
			NodSetChild(n, NTR_MYPE,
				NodGetChild(NodGetChild(n, NTR_VARASSIGN_VALUE), NTR_MYPE))
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
			NodSetChild(n, NTR_MYPE, intType)
			NodSetChild(NodGetChild(n, NTR_BINOP_LEFT), NTR_MYPE, intType)
			NodSetChild(NodGetChild(n, NTR_BINOP_RIGHT), NTR_MYPE, intType)
			fmt.Println("Applied add op int forcer to", PrettyPrintMype(n))
		},
	}
}

func marLitIntsBase() *RewriteRule {
	return &RewriteRule{
		condition: func(n Nod) bool {
			if n.NodeType == NT_LIT_INT {
				if NodGetChild(n, NTR_MYPE).NodeType != NT_TYPE {
					return true
				}
			}
			return false
		},
		action: func(n Nod) {
			NodSetChild(n, NTR_MYPE, NodNewData(NT_TYPE, TY_INT))
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

	// determine which (top-level) imperatives have which vars
	varAssigns := x.SearchRoot(func(n Nod) bool { return n.NodeType == NT_VARASSIGN })
	impVarAssigns := make(map[Nod][]Nod)
	for _, ele := range varAssigns {
		tlImper := x.findTopLevelImperative(ele)
		if tlImper == nil {
			panic("failed to find top level imperative")
		}
		impVarAssigns[tlImper] = append(impVarAssigns[tlImper], ele)
	}

	// next, generate the vartables for each imperative
	for imper, varAssigns := range impVarAssigns {
		varTable := x.generateVarTableFromVarAssigns(varAssigns)
		NodSetChild(imper, NTR_VARTABLE, varTable)
	}
}

func (x *XformerPocket) generateVarTableFromVarAssigns(varAssigns []Nod) Nod {
	varDefsByName := make(map[string]Nod)
	for _, varAssign := range varAssigns {
		varName := NodGetChild(varAssign, NTR_VAR_NAME).Data.(string)
		varDef := NodNew(NT_VARDEF)
		NodSetChild(varDef, NTR_VARDEF_NAME, NodNewData(NT_IDENTIFIER, varName))
		if NodHasChild(varAssign, NTR_TYPE) {
			NodSetChild(varDef, NTR_TYPE, NodGetChild(varAssign, NTR_TYPE))
		} else {
			panic("var assign missing type")
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
		return n.NodeType == NT_IMPERATIVE && x.OneParentIs(n,
			func(n Nod) bool {
				return n.NodeType == NT_FUNCDEF
			})
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
