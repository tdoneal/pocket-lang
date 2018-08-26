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
	x.annotateTypes()
}

func (x *XformerPocket) annotateTypes() {
	x.applyTAR(func(n Nod) bool { return n.NodeType == NT_LIT_INT },
		func(n Nod) {
			NodSetChild(n, NTR_TYPE, NodNewData(NT_TYPE, "int"))
		})

	// TODO: only update annotated type if new type carries additional information
	x.applyTAR(
		func(n Nod) bool {
			if n.NodeType == NT_VARINIT {
				if viv := NodGetChildOrNil(n, NTR_VARINIT_VALUE); viv != nil {
					return NodHasChild(viv, NTR_TYPE)
				}
			}
			return false
		},
		func(n Nod) {
			NodSetChild(n, NTR_TYPE,
				NodGetChild(NodGetChild(n, NTR_VARINIT_VALUE), NTR_TYPE))
		},
	)
}

func (x *XformerPocket) applyTAR(condition func(Nod) bool, action func(Nod)) {
	found := x.SearchRoot(condition)
	for _, ele := range found {
		action(ele)
	}
	fmt.Println("applied TAR to", len(found), "elements")
}

func (x *XformerPocket) buildVarDefTables() {
	// find all variable initializers
	// TODO: add support for multiple IMPERATIVEs, each with their own vartable

	// first, add an explicit link from each var init to it's corresponding
	// top-level imperative
	varInits := x.SearchRoot(func(n Nod) bool { return n.NodeType == NT_VARINIT })

	fmt.Println("all variable initializers:", len(varInits))
	fmt.Println(PrettyPrintNodes(varInits))

	for _, ele := range varInits {
		tlImper := x.findTopLevelImperative(ele)
		if tlImper == nil {
			panic("failed to find top level imperative")
		}
		NodSetChild(ele, NTR_TOPLEVEL_IMPERATIVE, tlImper)

	}

	fmt.Println("after linking tl imperatives: var inits:\n", PrettyPrintNodes(varInits))

	// next, generate the vartables for each tl imperative
	impVartables := make(map[Nod][]Nod)
	for _, ele := range varInits {
		varNameNode := ele.Out[NTR_VARINIT_NAME].Out
		varName := varNameNode.Data.(string)
		fmt.Println("varName", varName)
		varDef := NodNewChild(NT_VARDEF, NTR_VARDEF_NAME, varNameNode)
		tlImper := NodGetChild(ele, NTR_TOPLEVEL_IMPERATIVE)
		// lazily initialize the tables
		if _, ok := impVartables[tlImper]; !ok {
			impVartables[tlImper] = make([]Nod, 0)
		}
		impVartables[tlImper] = append(impVartables[tlImper], varDef)
	}

	// last, explicitly put the vartables in the graph
	for imp, varTable := range impVartables {
		varTableNod := NodNewChildList(NT_VARTABLE, varTable)
		NodSetChild(imp, NTR_VARTABLE, varTableNod)
	}
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
