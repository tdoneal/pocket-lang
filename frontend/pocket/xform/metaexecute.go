package xform

import (
	"fmt"
	. "pocket-lang/frontend/pocket/common"
	. "pocket-lang/parse"
)

type MetaExecutor struct {
	solver *NSolver
}

func (e *MetaExecutor) executeFunction(function Nod) {
	imperativeBody := NodGetChild(function, NTR_FUNCDEF_CODE)
	fmt.Println("meta-executing", PrettyPrint(imperativeBody))

	executableNodes := e.solver.xformer.SearchRoot(func(n Nod) bool {
		return e.isSolvable(n.NodeType)
	})

	for _, execNode := range executableNodes {
		e.executeNode(execNode)
	}

	fmt.Println("after executing function", PrettyPrint(function))

	panic("done executing function")
}

func (e *MetaExecutor) isSolvable(nt int) bool {
	return nt == NT_LIT_INT
}

func (e *MetaExecutor) executeNode(n Nod) bool { // returns whether change made
	fmt.Println("Executing node", PrettyPrint(n))

	nt := n.NodeType

	outKnow := []Nod{}

	if nt == NT_LIT_INT {
		runtype := NodNewData(NT_TYPEBASE, TY_INT)
		outKnow = append(outKnow, NodNewData(KNOW_RUNTYPE, runtype))
		outKnow = append(outKnow, NodNewData(KNOW_RUNVALUE, ConstructRunValue(runtype, n.Data)))
	} else {
		panic("can't execute nt")
	}

	if !NodHasChild(n, NNTR_KNOWLEDGE) {
		knowledgeNod := NodNewChildList(NNT_KNOWLEDGE_DISJUNCTION, outKnow)
		fmt.Println("output knowledge:", PrettyPrint(knowledgeNod))

		NodSetChild(n, NNTR_KNOWLEDGE, knowledgeNod)
		return true
	}

	panic("unsupported")
}

func ConstructRunValue(runType Nod, entropy interface{}) Nod {
	rv := NodNew(NNT_RUNVALUE)
	NodSetChild(rv, NNTR_RUNVALUE_TYPE, runType)
	rv.Data = entropy
	return rv
}
