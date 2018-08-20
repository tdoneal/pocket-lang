package xform

import (
	"fmt"
	. "pocket-lang/parse"
)

type Xformer struct {
	root Nod
}

func Xform(root Nod) Nod {
	fmt.Println("xforming node")

	xformer := &Xformer{}

	xformer.root = root
	xformer.Xform()
	return root
}

func (x *Xformer) Xform() {
	x.buildVarDefTables()
}

func (x *Xformer) buildVarDefTables() {
	// find all variable initializers
	// TODO: add support for multiple IMPERATIVEs, each with their own vartable
	varInits := x.searchFor(func(n Nod) bool { return n.NodeType == NT_VARINIT })

	fmt.Println("pretty nodes:", len(varInits))
	fmt.Println(PrettyPrintNodes(varInits))

	var varDefs = make([]Nod, 0)

	for _, ele := range varInits {
		varNameNode := ele.Out[NTR_VARINIT_NAME].Out
		varName := varNameNode.Data.(string)
		fmt.Println("varName", varName)
		varDef := NodNewChild(NT_VARDEF, NTR_VARDEF_NAME, varNameNode)
		varDefs = append(varDefs, varDef)
	}

	imperatives := x.searchForNodeType(NT_IMPERATIVE)

	fmt.Println("imperatives", PrettyPrintNodes(imperatives))

	if len(imperatives) != 1 {
		panic("incorrect # of imperatives")
	}

	varTable := NodNewChildList(NT_VARTABLE, varDefs)
	imperative := (*Node)(imperatives[0])

	NodSetChild(imperative, NTR_VARTABLE, varTable)

	fmt.Println("final imperative", PrettyPrint(imperative))

}

func (x *Xformer) XformTest() {
	x.searchReplaceAll(
		func(n Nod) bool { return n.NodeType == NT_INLINEOPSTREAM },
		func(Nod) Nod {
			return &Node{
				NodeType: NT_LIT_INT,
				Data:     42,
			}
		},
	)

	x.searchReplaceAll(
		func(n Nod) bool {
			if val, ok := n.Data.(int); ok {
				return val > 10
			}
			return false
		},
		func(Nod) Nod {
			return &Node{
				NodeType: NT_LIT_INT,
				Data:     1000,
			}
		},
	)
}

func (x *Xformer) searchReplaceAll(cond func(Nod) bool, with func(Nod) Nod) {
	toReplace := x.searchFor(cond)
	fmt.Println("search results: ", PrettyPrintNodes(toReplace))
	for _, ele := range toReplace {
		x.replace(ele, with(ele))
	}
}

func (x *Xformer) replace(what Nod, with Nod) {
	for _, ele := range what.In {
		ele.Out = with
	}
	with.In = what.In
}

type Searcher struct {
	alreadySeen map[Nod]bool
	output      []Nod
	condition   func(Nod) bool // TODO: add more flexible condition mechanics
}

func SearcherNew() *Searcher {
	s := &Searcher{
		alreadySeen: make(map[Nod]bool),
		output:      make([]Nod, 0),
	}
	return s
}

func (x *Xformer) searchFor(condition func(Nod) bool) []Nod {
	s := SearcherNew()
	s.condition = condition
	s.search(x.root)
	return s.output
}

func (x *Xformer) searchForNodeType(nodeType int) []Nod {
	return x.searchFor(x.getNodeTypeCondition(nodeType))
}

func (x *Xformer) getNodeTypeCondition(nodeType int) func(Nod) bool {
	return func(n Nod) bool { return n.NodeType == nodeType }
}

func (s *Searcher) search(node Nod) {
	if _, ok := s.alreadySeen[node]; ok {
		return
	}
	if s.condition(node) {
		s.output = append(s.output, node)
	}
	s.alreadySeen[node] = true
	for _, edge := range node.Out {
		s.search(edge.Out)
	}
}
