package xform

import (
	"fmt"
	. "pocket-lang/frontend/pocket/common"
	. "pocket-lang/parse"
)

type NSolver struct {
	xformer *XformerPocket
}

/*
valid NTS for nsolver:

*/

func (s *NSolver) solve() {
	s.initializeSymbolTables()

	// todo: loop this until convergence
	s.metaexecute()

	panic("NSolver.solve()")
}

func (s *NSolver) initializeSymbolTables() {
	// find all symbol table "hosts": those where it makes sense to speak of symbols in that scope
	symtableHosts := s.xformer.SearchRoot(func(n Nod) bool {
		return n.NodeType == NT_FUNCDEF ||
			n.NodeType == NT_CLASSDEF || n.NodeType == NT_TOPLEVEL
	})

	for _, host := range symtableHosts {
		s.initializeSymbolTable(host)
	}

	fmt.Println("after initializing symbol tables:", PrettyPrint(s.xformer.Root))
}

func (s *NSolver) initializeSymbolTable(host Nod) {
	table := NodNew(NNT_SYMTABLE)

	// determine which types of symbols (variables, functions, classes) are allowed
	var hasVariables bool = false
	var hasFunctions bool = false
	var hasClasses bool = false

	if host.NodeType == NT_TOPLEVEL {
		hasVariables = true
		hasFunctions = true
		hasClasses = true
	} else if host.NodeType == NT_FUNCDEF {
		hasVariables = true
	} else if host.NodeType == NT_CLASSDEF {
		hasClasses = true
		hasFunctions = true
		hasVariables = true
	} else {
		panic("unknown host type")
	}

	// insert the appropriate tables based on which types of symbols allowed
	if hasVariables {
		NodSetChild(table, NTR_VARTABLE, NodNewData(NT_VARTABLE, map[string]Nod{}))
	}
	if hasFunctions {
		NodSetChild(table, NTR_FUNCTABLE, NodNewData(NT_FUNCTABLE, map[string]Nod{}))
	}
	if hasClasses {
		NodSetChild(table, NTR_CLASSTABLE, NodNewData(NT_CLASSTABLE, map[string]Nod{}))
	}

	fmt.Println("inserting symtable", PrettyPrint(table))

	NodSetChild(host, NNTR_SYMTABLE, table)
}

func (s *NSolver) metaexecute() {
	// find all functions and meta-execute each
	funcdefs := s.xformer.SearchRoot(func(n Nod) bool {
		return n.NodeType == NT_FUNCDEF
	})
	// funcbodies := []Nod{}
	// for _, funcdef := range funcdefs {
	// 	funcbodies = append(funcbodies, NodGetChild(funcdef, NTR_FUNCDEF_CODE))
	// }

	// fmt.Println("func bodies:", PrettyPrintNodes(funcbodies))
	for _, funcdef := range funcdefs {
		metaExecutor := &MetaExecutor{s}
		metaExecutor.executeFunction(funcdef)
	}
}
