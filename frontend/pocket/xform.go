package pocket

import (
	"fmt"
	. "pocket-lang/parse"
	. "pocket-lang/xform"
)

type XformerPocket struct {
	*Xformer
}

func Xform(root Nod) Nod {
	fmt.Println("xforming node")

	xformer := &XformerPocket{
		&Xformer{},
	}

	xformer.Root = root
	xformer.Xform()
	return root
}

func (x *XformerPocket) Xform() {
	x.buildVarDefTables()
	// x.buildFuncDefTables()

	x.applyRewrite(&RewriteRule{
		condition: func(n Nod) bool {
			return n.NodeType == NT_INLINEOPSTREAM
		},
		action: func(n Nod) {
			x.Replace(n, x.parseInlineOpStream(n))
		},
	})

	x.annotateTypes()
}

type RewriteRule struct {
	condition func(n Nod) bool
	action    func(n Nod)
}

func (x *XformerPocket) annotateTypes() {

	TARs := []*RewriteRule{
		&RewriteRule{
			condition: func(n Nod) bool {
				return n.NodeType == NT_LIT_INT && !NodHasChild(n, NTR_TYPE)
			},
			action: func(n Nod) {
				NodSetChild(n, NTR_TYPE, NodNewData(NT_TYPE, "int"))
			},
		},
		// propagate var assign values to the var's assignment
		&RewriteRule{
			condition: func(n Nod) bool {
				if n.NodeType == NT_VARASSIGN && !NodHasChild(n, NTR_TYPE) {
					if viv := NodGetChildOrNil(n, NTR_VARASSIGN_VALUE); viv != nil {
						return NodHasChild(viv, NTR_TYPE)
					}
				}
				return false
			},
			action: func(n Nod) {
				NodSetChild(n, NTR_TYPE,
					NodGetChild(NodGetChild(n, NTR_VARASSIGN_VALUE), NTR_TYPE))
			},
		},
	}

	x.applyRewritesUntilStable(TARs)

}

func (x *XformerPocket) applyRewritesUntilStable(rules []*RewriteRule) {
	for {
		maxApplied := 0
		for _, rule := range rules {
			nApplied := x.applyRewrite(rule)
			if nApplied > maxApplied {
				maxApplied = nApplied
			}
		}
		if maxApplied == 0 {
			break
		}
	}
}

func (x *XformerPocket) applyRewrite(rule *RewriteRule) int {
	found := x.SearchRoot(rule.condition)
	for _, ele := range found {
		rule.action(ele)
	}
	fmt.Println("applied rewrite rule to", len(found), "elements")
	return len(found)
}

func (x *XformerPocket) buildVarDefTables() {
	// find all variable assignments and construct
	// the canonical union of variables

	// determine which (top-level) imperatives have which vars
	varInits := x.SearchRoot(func(n Nod) bool { return n.NodeType == NT_VARASSIGN })
	fmt.Println("all variable initializers:", len(varInits))
	impVarInits := make(map[Nod][]Nod)
	for _, ele := range varInits {
		tlImper := x.findTopLevelImperative(ele)
		if tlImper == nil {
			panic("failed to find top level imperative")
		}
		impVarInits[tlImper] = append(impVarInits[tlImper], ele)
	}

	// next, generate the vartables for each imperative
	for imper, varInits := range impVarInits {
		varTable := x.generateVarTableFromVarInits(varInits)
		NodSetChild(imper, NTR_VARTABLE, varTable)
	}

}

func (x *XformerPocket) generateVarTableFromVarInits(varInits []Nod) Nod {
	varDefsByName := make(map[string]Nod)
	for _, varInit := range varInits {
		varName := NodGetChild(varInit, NTR_VAR_NAME).Data.(string)
		varDef := NodNew(NT_VARDEF)
		NodSetChild(varDef, NTR_VARDEF_NAME, NodNewData(NT_IDENTIFIER, varName))
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
