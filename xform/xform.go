package xform

import (
	"fmt"
	"pocket-lang/parse"
)

type Xformer struct {
	root parse.Nod
}

func Xform(root parse.Nod) parse.Nod {
	fmt.Println("xforming node")

	xformer := &Xformer{}

	xformer.root = root
	xformer.Xform()
	return root
}

func (x *Xformer) Xform() {
	x.searchReplaceAll(
		func(n parse.Nod) bool { return n.NodeType == parse.NT_INLINEOPSTREAM },
		func(parse.Nod) parse.Nod {
			return &parse.Node{
				NodeType: parse.NT_LIT_INT,
				Data:     42,
			}
		},
	)

	x.searchReplaceAll(
		func(n parse.Nod) bool {
			if val, ok := n.Data.(int); ok {
				return val > 10
			}
			return false
		},
		func(parse.Nod) parse.Nod {
			return &parse.Node{
				NodeType: parse.NT_LIT_INT,
				Data:     1000,
			}
		},
	)
}

func (x *Xformer) searchReplaceAll(cond func(parse.Nod) bool, with func(parse.Nod) parse.Nod) {
	toReplace := x.searchFor(cond)
	fmt.Println("search results: ", parse.PrettyPrintNodes(toReplace))
	for _, ele := range toReplace {
		x.replace(ele, with(ele))
	}
}

func (x *Xformer) replace(what parse.Nod, with parse.Nod) {
	for _, ele := range what.In {
		ele.Out = with
	}
	with.In = what.In
}

type Searcher struct {
	alreadySeen map[parse.Nod]bool
	output      []parse.Nod
	condition   func(parse.Nod) bool // TODO: add more flexible condition mechanics
}

func SearcherNew() *Searcher {
	s := &Searcher{
		alreadySeen: make(map[parse.Nod]bool),
		output:      make([]parse.Nod, 0),
	}
	return s
}

func (x *Xformer) searchFor(condition func(parse.Nod) bool) []parse.Nod {
	s := SearcherNew()
	s.condition = condition
	s.search(x.root)
	return s.output
}

func (s *Searcher) search(node parse.Nod) {
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
