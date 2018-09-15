package xform

import (
	. "pocket-lang/frontend/pocket/common"
	. "pocket-lang/parse"
	"strconv"
)

func (x *XformerPocket) desugar() {
	x.rewriteDotPipesAsFunctionCalls()
	x.rewriteForInLoops()
}

func (x *XformerPocket) getTempVarName() string {
	rv := "__pkx" + strconv.Itoa(x.tempVarCounter) + "__"
	x.tempVarCounter++
	return rv
}

func (x *XformerPocket) rewriteDotPipesAsFunctionCalls() {
	x.SearchReplaceAll(
		func(n Nod) bool {
			return n.NodeType == NT_DOTPIPEOP
		},
		func(n Nod) Nod {
			dpFun := NodGetChild(n, NTR_BINOP_RIGHT)
			if dpFun.NodeType == NT_TYPEBASE {
				rv := NodNew(NT_OBJINIT)
				NodSetChild(rv, NTR_RECEIVERCALL_BASE, NodGetChild(n, NTR_BINOP_RIGHT))
				NodSetChild(rv, NTR_RECEIVERCALL_ARG, NodGetChild(n, NTR_BINOP_LEFT))
				return rv
			}
			panic("unsupported right arg of dotpipe")
		},
	)
}

func (x *XformerPocket) rewriteForInLoops() {
	forLoops := x.SearchRoot(func(n Nod) bool { return n.NodeType == NT_FOR })
	for _, forLoop := range forLoops {
		x.Replace(forLoop, x.rewriteForInLoop(forLoop))
	}
}

func (x *XformerPocket) rewriteForInLoop(forLoop Nod) Nod {
	// rewrites the for <var> in <list> : body syntax
	// to a lower level form involving a while loop and an index variable
	rvSeq := []Nod{}
	loopOver := NodGetChild(forLoop, NTR_FOR_ITEROVER)
	declaredElementVarName := NodGetChild(forLoop, NTR_FOR_ITERVAR).Data.(string)
	ndxVarName := x.getTempVarName()
	iterOverVarName := x.getTempVarName()
	// generate the effective code: __ndx_var__ : 0
	ndxVarInitializer := NodNew(NT_VARASSIGN)
	ndxVarIdentifier := NodNewData(NT_IDENTIFIER, ndxVarName)
	ndxVarInitValue := NodNewData(NT_LIT_INT, 0)
	NodSetChild(ndxVarInitializer, NTR_VAR_NAME, ndxVarIdentifier)
	NodSetChild(ndxVarInitializer, NTR_VARASSIGN_VALUE, ndxVarInitValue)

	// __iterover_var__: <iterover>
	iterOverVarInitializer := NodNew(NT_VARASSIGN)
	iterOverVarIdentifier := NodNewData(NT_IDENTIFIER, iterOverVarName)
	iterOverVarInitValue := loopOver
	NodSetChild(iterOverVarInitializer, NTR_VAR_NAME, iterOverVarIdentifier)
	NodSetChild(iterOverVarInitializer, NTR_VARASSIGN_VALUE, iterOverVarInitValue)

	loopBodySeq := []Nod{}
	// while __ndx__ < seq.len
	termCond := NodNew(NT_LTOP)
	termCondNdxVarGetter := NodNewChild(NT_VAR_GETTER, NTR_VAR_NAME,
		NodNewData(NT_IDENTIFIER, ndxVarName))
	termCondLenGetter := NodNew(NT_DOTOP)
	NodSetChild(termCondLenGetter, NTR_BINOP_LEFT,
		NodNewChild(NT_VAR_GETTER, NTR_VAR_NAME,
			NodNewData(NT_IDENTIFIER, iterOverVarName)))
	NodSetChild(termCondLenGetter, NTR_BINOP_RIGHT, NodNewData(NT_IDENTIFIER, "len"))
	NodSetChild(termCond, NTR_BINOP_LEFT, termCondNdxVarGetter)
	NodSetChild(termCond, NTR_BINOP_RIGHT, termCondLenGetter)

	// <itervar>: __iterover_var__[__ndx__]
	iterVarAssigner := NodNew(NT_VARASSIGN)
	NodSetChild(iterVarAssigner, NTR_VAR_NAME, NodNewData(NT_IDENTIFIER, declaredElementVarName))
	iterVarAssignerValue := NodNew(NT_RECEIVERCALL)
	// generate the list indexor as a receiver call
	NodSetChild(iterVarAssignerValue, NTR_RECEIVERCALL_BASE, NodNewData(NT_IDENTIFIER, iterOverVarName))
	NodSetChild(iterVarAssignerValue, NTR_RECEIVERCALL_ARG,
		NodNewChild(NT_VAR_GETTER, NTR_VAR_NAME,
			NodNewData(NT_IDENTIFIER, ndxVarName)))
	NodSetChild(iterVarAssigner, NTR_VARASSIGN_VALUE, iterVarAssignerValue)
	loopBodySeq = append(loopBodySeq, iterVarAssigner)

	// actual user body
	loopBodySeq = append(loopBodySeq, NodGetChild(forLoop, NTR_FOR_BODY))

	// __ndx__++
	ndxVarIncrementor := NodNew(NT_VARASSIGN)
	NodSetChild(ndxVarIncrementor, NTR_VAR_NAME, NodNewData(NT_IDENTIFIER, ndxVarName))
	ndxVarIncrementorValue := NodNew(NT_ADDOP)
	NodSetChild(ndxVarIncrementorValue, NTR_BINOP_LEFT, NodNewChild(
		NT_VAR_GETTER, NTR_VAR_NAME, NodNewData(NT_IDENTIFIER, ndxVarName)))
	NodSetChild(ndxVarIncrementorValue, NTR_BINOP_RIGHT, NodNewData(
		NT_LIT_INT, 1))
	NodSetChild(ndxVarIncrementor, NTR_VARASSIGN_VALUE, ndxVarIncrementorValue)
	loopBodySeq = append(loopBodySeq, ndxVarIncrementor)

	// put it all together and return
	whileLoop := NodNew(NT_WHILE)
	NodSetChild(whileLoop, NTR_WHILE_COND, termCond)
	NodSetChild(whileLoop, NTR_WHILE_BODY, NodNewChildList(NT_IMPERATIVE, loopBodySeq))

	rvSeq = append(rvSeq, ndxVarInitializer)
	rvSeq = append(rvSeq, iterOverVarInitializer)
	rvSeq = append(rvSeq, whileLoop)
	rv := NodNewChildList(NT_IMPERATIVE, rvSeq)
	return rv

}
