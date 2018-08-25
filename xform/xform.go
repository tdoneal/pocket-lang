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
	// x.buildFuncDefTables()
}

func (x *Xformer) buildVarDefTables() {
	// find all variable initializers
	// TODO: add support for multiple IMPERATIVEs, each with their own vartable

	// first, add an explicit link from each var init to it's corresponding
	// top-level imperative
	varInits := x.searchRoot(func(n Nod) bool { return n.NodeType == NT_VARINIT })

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

func (x *Xformer) findTopLevelImperative(n Nod) Nod {
	found := x.searchFor(n, func(n Nod) bool {
		return n.NodeType == NT_IMPERATIVE && x.oneParentIs(n,
			func(n Nod) bool {
				return n.NodeType == NT_FUNCDEF
			})
	}, func(n Nod) []Nod {
		return x.allInNodes(n)
	})

	return found[0]
}

func (x *Xformer) oneParentIs(n Nod, cond func(Nod) bool) bool {
	for _, ele := range n.In {
		parent := ele.In
		if cond(parent) {
			return true
		}
	}
	return false
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
	toReplace := x.searchRoot(cond)
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
	alreadySeen       map[Nod]bool
	output            []Nod
	condition         func(Nod) bool
	nextNodEnumerator func(Nod) []Nod
}

func SearcherNew() *Searcher {
	s := &Searcher{
		alreadySeen: make(map[Nod]bool),
		output:      make([]Nod, 0),
	}
	return s
}

func (x *Xformer) searchRoot(condition func(Nod) bool) []Nod {
	return x.searchFor(x.root,
		condition,
		func(n Nod) []Nod {
			return x.allOutNodes(n)
		})
}

func (x *Xformer) allOutNodes(n Nod) []Nod {
	rv := make([]Nod, 0)
	for _, ele := range n.Out {
		rv = append(rv, ele.Out)
	}
	return rv
}

func (x *Xformer) allInNodes(n Nod) []Nod {
	rv := make([]Nod, 0)
	for _, ele := range n.In {
		rv = append(rv, ele.In)
	}
	return rv
}

func (x *Xformer) searchFor(start Nod, condition func(Nod) bool, nextEnumerator func(Nod) []Nod) []Nod {
	s := SearcherNew()
	s.condition = condition
	s.nextNodEnumerator = nextEnumerator
	s.search(start)
	return s.output
}

func (x *Xformer) searchForNodeType(nodeType int) []Nod {
	return x.searchRoot(x.getNodeTypeCondition(nodeType))
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
	nextNodes := s.nextNodEnumerator(node)
	for _, nextNode := range nextNodes {
		s.search(nextNode)
	}
}
