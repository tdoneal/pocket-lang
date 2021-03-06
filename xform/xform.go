package xform

import (
	. "pocket-lang/parse"
)

type Xformer struct {
	Root Nod
}

func (x *Xformer) OneParentIs(n Nod, cond func(Nod) bool) bool {
	for _, ele := range n.In {
		parent := ele.In
		if cond(parent) {
			return true
		}
	}
	return false
}

// deprecated.  new version: SearchReplaceAll2
func (x *Xformer) SearchReplaceAll(cond func(Nod) bool, with func(Nod) Nod) {
	toReplace := x.SearchRoot(cond)
	for _, ele := range toReplace {
		x.Replace(ele, with(ele))
	}
}

func (x *Xformer) SearchReplaceAll2(cond func(Nod) bool, with func(Nod) Nod) {
	toReplace := x.SearchRoot(cond)
	for _, ele := range toReplace {
		x.Replace2(ele, with)
	}
}

func (x *Xformer) Replace(what Nod, with Nod) {
	// replace one node with another
	// the new nod must not contain a reference (even indirectly to the old node)
	// for this to work as expected (unless the new nod is a direct ancestor)

	// redirect all incoming nodes of old to new
	for _, ele := range what.In {
		if ele.In != with { // properly handle the case where we replace with an ancestor of original node
			ele.Out = with
		}
	}

	// rebuild In table of new nod
	// with.In = what.In
	// with.In = Union(with.In, what.In)
	toAddToWithIn := []*Edge{}
	for _, edge := range what.In {
		if edge.In == with {
			continue
		}
		// add what.In.edge if not already in with.In.edges
		alreadyIn := false
		for _, withEdge := range with.In {
			if edge == withEdge {
				alreadyIn = true
				break
			}
		}
		if !alreadyIn {
			toAddToWithIn = append(toAddToWithIn, edge)
		}
	}
	with.In = append(with.In, toAddToWithIn...)
}

func (x *Xformer) Replace2(what Nod, with func(old Nod) Nod) {
	// handles self-referential replacements much better

	// inner workings:
	// replaces the given nod with a dummy Nod, then applies the
	// function to generate the new Nod, then finally
	// replaces the dummy with the new
	// key: any new pointers created by the generator function aren't overwritten
	// during the initial dummy replacement

	dummyNod := NodNew(NTR_LIST_0)
	for _, incomingEdge := range what.In {
		incomingEdge.Out = dummyNod
	}
	dummyNod.In = what.In
	what.In = nil

	newNod := with(what)

	newNod.In = dummyNod.In
	for _, incomingEdge := range newNod.In {
		incomingEdge.Out = newNod
	}
}

type Searcher struct {
	alreadySeen        map[Nod]bool
	output             []Nod
	condition          func(Nod) bool
	nextNodEnumerator  func(Nod) []Nod
	earlyStopCondition func([]Nod) bool
	terminated         bool
}

func SearcherNew() *Searcher {
	s := &Searcher{
		alreadySeen: make(map[Nod]bool),
		output:      make([]Nod, 0),
	}
	return s
}

func (x *Xformer) SearchRoot(condition func(Nod) bool) []Nod {
	return x.SearchFrom(x.Root,
		condition,
		func(n Nod) []Nod {
			return x.AllOutNodes(n)
		}, func(ns []Nod) bool { return false })
}

func (x *Xformer) SearchNodList(nods []Nod, condition func(Nod) bool) []Nod {
	rv := []Nod{}
	for _, ele := range nods {
		if condition(ele) {
			rv = append(rv, ele)
		}
	}
	return rv
}

func (x *Xformer) AllOutNodes(n Nod) []Nod {
	rv := make([]Nod, 0)
	for _, ele := range n.Out {
		rv = append(rv, ele.Out)
	}
	return rv
}

func (x *Xformer) AllInNodes(n Nod) []Nod {
	rv := make([]Nod, 0)
	for _, ele := range n.In {
		rv = append(rv, ele.In)
	}
	return rv
}

func (x *Xformer) SearchFrom(start Nod, condition func(Nod) bool, nextEnumerator func(Nod) []Nod, earlyStop func([]Nod) bool) []Nod {
	s := SearcherNew()
	s.condition = condition
	s.nextNodEnumerator = nextEnumerator
	s.earlyStopCondition = earlyStop
	s.search(start)
	return s.output
}

func (x *Xformer) SearchForNodeType(nodeType int) []Nod {
	return x.SearchRoot(x.GetNodeTypeCondition(nodeType))
}

func (x *Xformer) GetNodeTypeCondition(nodeType int) func(Nod) bool {
	return func(n Nod) bool { return n.NodeType == nodeType }
}

func (s *Searcher) search(node Nod) {
	if s.terminated {
		return
	}
	if _, ok := s.alreadySeen[node]; ok {
		return
	}
	if s.condition(node) {
		s.output = append(s.output, node)
		if s.earlyStopCondition(s.output) {
			s.terminated = true
			return
		}
	}
	s.alreadySeen[node] = true
	nextNodes := s.nextNodEnumerator(node)
	for _, nextNode := range nextNodes {
		s.search(nextNode)
	}
}
