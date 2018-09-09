package goback

import (
	"bytes"
	"fmt"
	. "pocket-lang/frontend/pocket"
	. "pocket-lang/parse"
	"pocket-lang/xform"
	"strconv"
)

type Generator struct {
	input         Nod
	buf           *bytes.Buffer
	tmpVarCounter int
}

func Generate(code Nod) string {

	preparer := &Preparer{&xform.Xformer{}}
	preparer.Prepare(code)

	fmt.Println("Prepared code:\n", PrettyPrint(code))

	generator := &Generator{
		buf:   &bytes.Buffer{},
		input: code,
	}

	generator.genSourceFile(code)

	return generator.buf.String()
}

func (g *Generator) genSourceFile(input Nod) {
	g.WS("package main\n\n")

	if fmtImport := NodGetChildOrNil(input, PNTR_GOIMPORTS); fmtImport != nil {
		g.WS("import \"fmt\"\n\n")
	}

	units := NodGetChildList(input)

	for _, unit := range units {
		if unit.NodeType == NT_FUNCDEF {
			g.genFuncDef(unit)
		} else {
			panic("unknown source unit type")
		}
	}
}

func (g *Generator) genFuncInType(n Nod) {
	if n.NodeType == NT_PARAMETER {
		g.genParameter(n)
	} else if n.NodeType == NT_LIT_LIST {
		g.genParameterList(n)
	}
}

func (g *Generator) genFuncOutType(n Nod) {
	if typeInt, ok := n.Data.(int); ok {
		if typeInt == TY_VOID {
			// in go there is no void keyword, so we don't output anything here
			return
		}
	}
	g.genType(n)
}

func (g *Generator) genParameterList(n Nod) {
	g.WS("args []interface{}")
}

func (g *Generator) genParameter(n Nod) {
	g.WS(NodGetChild(n, NTR_VARDEF_NAME).Data.(string))
	g.WS(" ")
	g.genType(NodGetChild(n, NTR_TYPE))
}

func (g *Generator) genFuncDef(n Nod) {
	funcName := NodGetChild(n, NTR_FUNCDEF_NAME).Data.(string)
	g.WS("func ")
	g.WS(funcName)
	g.WS("(")

	needsArgUnpacking := false
	if inType := NodGetChildOrNil(n, NTR_FUNCDEF_INTYPE); inType != nil {
		g.genFuncInType(inType)
		if inType.NodeType == NT_LIT_LIST {
			needsArgUnpacking = true
		}
	}

	g.WS(")")

	if outType := NodGetChildOrNil(n, NTR_FUNCDEF_OUTTYPE); outType != nil {
		g.genFuncOutType(outType)
	}

	g.WS(" {\n")

	if needsArgUnpacking {
		g.genArgUnpacking(NodGetChild(n, NTR_FUNCDEF_INTYPE))
	}

	g.genImperative(NodGetChild(n, NTR_FUNCDEF_CODE))

	g.WS("}\n")

}

func (g *Generator) genArgUnpacking(inTypeDef Nod) {
	params := NodGetChildList(inTypeDef)
	for ndx, param := range params {
		typeStr := g.getGenType(NodGetChild(param, NTR_TYPE))

		g.WS("var ")
		g.WS(NodGetChild(param, NTR_VARDEF_NAME).Data.(string))
		g.WS(" ")
		g.WS(typeStr)
		g.WS(" = ")
		g.WS("args[")
		g.WS(strconv.Itoa(ndx))
		g.WS("].(")
		g.WS(typeStr)
		g.WS(")")
		g.WS("\n")
	}
}

func (g *Generator) getContainingFuncDefOrNil(n Nod) Nod {
	for _, inEdge := range n.In {
		parent := inEdge.In
		if parent.NodeType == NT_FUNCDEF {
			return parent
		}
	}
	return nil
}

func (g *Generator) genImperative(input Nod) {

	if containingFuncDef := g.getContainingFuncDefOrNil(input); containingFuncDef != nil {
		if varTable := NodGetChildOrNil(containingFuncDef, NTR_VARTABLE); varTable != nil {
			g.genLocalVarTable(varTable)
		}
	}

	statements := NodGetChildList(input)
	for _, stmt := range statements {
		g.genImperativeUnit(stmt)
	}

}

func (g *Generator) genLocalVarTable(n Nod) {
	varDefs := NodGetChildList(n)
	for _, varDef := range varDefs {
		vds := NodGetChild(varDef, NTR_VARDEF_SCOPE).Data.(int)
		if vds == VSCOPE_FUNCLOCAL {
			g.genVarDef(varDef)
		}
	}
}

func (g *Generator) genVarDef(n Nod) {
	varName := NodGetChild(n, NTR_VARDEF_NAME).Data.(string)
	g.WS("var ")
	g.WS(varName)
	if nType := NodGetChildOrNil(n, NTR_TYPE); nType != nil {
		g.WS(" ")
		g.genType(NodGetChild(n, NTR_TYPE))
	}
	g.WS("\n")
}

func (g *Generator) genType(n Nod) {
	g.WS(g.getGenType(n))
}

func (g *Generator) getGenTypeBase(n Nod) string {
	lut := map[int]string{
		TY_BOOL:   "bool",
		TY_INT:    "int64",
		TY_FLOAT:  "float64",
		TY_NUMBER: "number",
		TY_STRING: "string",
		TY_DUCK:   "interface{}",
		TY_LIST:   "[]interface{}",
		TY_SET:    "map[interface{}]bool",
		TY_MAP:    "map[interface{}]interface{}",
	}
	if val, ok := lut[n.Data.(int)]; ok {
		return val
	} else {
		return "<basetype>"
	}
}

func (g *Generator) getGenType(n Nod) string {
	if n.NodeType == NT_TYPEBASE {
		return g.getGenTypeBase(n)
	} else if n.NodeType == NT_TYPEARGED {
		return g.getGenTypeArged(n)
	} else {
		return "<type>"
	}
}

func (g *Generator) getGenTypeArged(n Nod) string {
	bt := NodGetChild(n, NTR_TYPEARGED_BASE).Data.(int)
	if bt == TY_LIST {
		argStr := g.getGenType(NodGetChild(n, NTR_TYPEARGED_ARG))
		return "[]" + argStr
	} else {
		return "<typearged>"
	}
}

func isReceiverCallType(nt int) bool {
	return nt == NT_RECEIVERCALL || nt == NT_RECEIVERCALL_CMD
}

func (g *Generator) genImperativeUnit(n Nod) {
	if n.NodeType == NT_VARASSIGN {
		g.genVarAssign(n)
	} else if isReceiverCallType(n.NodeType) {
		g.genReceiverCall(n)
	} else if n.NodeType == NT_RETURN {
		g.genReturn(n)
	} else if n.NodeType == NT_LOOP {
		g.genLoop(n)
	} else if n.NodeType == NT_WHILE {
		g.genWhile(n)
	} else if n.NodeType == NT_IF {
		g.genIf(n)
	} else if n.NodeType == NT_BREAK {
		g.genBreak(n)
	} else if n.NodeType == NT_IMPERATIVE {
		g.genImperative(n)
	} else {
		g.WS("command")
	}
	g.WS("\n")
}

func (g *Generator) genWhile(n Nod) {
	g.WS("for ")
	g.genValue(NodGetChild(n, NTR_WHILE_COND))
	g.WS("{\n")
	g.genImperative(NodGetChild(n, NTR_WHILE_BODY))
	g.WS("}")
	g.WS("\n")
}

func (g *Generator) genBreak(n Nod) {
	g.WS("break")
}

func (g *Generator) getTempVarName() string {
	rv := "_pk_" + strconv.Itoa(g.tmpVarCounter)
	g.tmpVarCounter++
	return rv
}

func (g *Generator) genLoop(input Nod) {
	g.WS("for ")
	if loopArg := NodGetChildOrNil(input, NTR_LOOP_ARG); loopArg != nil {
		tmpVarName := g.getTempVarName()
		g.WS(" ")
		g.WS(tmpVarName)
		g.WS(" := 0; ")
		g.WS(tmpVarName)
		g.WS(" < ")
		g.genValue(loopArg)
		g.WS("; ")
		g.WS(tmpVarName)
		g.WS("++ ")
	}
	g.WS("{\n")
	g.genImperative(NodGetChild(input, NTR_LOOP_BODY))
	g.WS("}\n")
}

func (g *Generator) genIf(input Nod) {
	g.WS("if ")
	g.genValue(NodGetChild(input, NTR_IF_COND))
	g.WS("{\n")
	g.genImperative(NodGetChild(input, NTR_IF_BODY_TRUE))
	g.WS("}")
	if elseBody := NodGetChildOrNil(input, NTR_IF_BODY_FALSE); elseBody != nil {
		g.WS(" else {\n")
		g.genImperative(elseBody)
		g.WS("}")
	}
	g.WS("\n")
}

func (g *Generator) genReturn(input Nod) {
	g.WS("return")
	if NodHasChild(input, NTR_RETURN_VALUE) {
		g.WS(" (")
		g.genValue(NodGetChild(input, NTR_RETURN_VALUE))
		g.WS(")")
	}
}

func (g *Generator) genVarAssign(n Nod) {
	varName := NodGetChild(n, NTR_VAR_NAME).Data.(string)

	g.WS(varName)
	g.WS(" = ")
	g.WS("(")
	g.genValue(NodGetChild(n, NTR_VARASSIGN_VALUE))
	g.WS(")")
}

func (g *Generator) genReceiverCall(n Nod) {
	rcvName := NodGetChild(n, NTR_RECEIVERCALL_BASE).Data.(string)

	if rcvName == "print" {
		rcvName = "fmt.Println"
	}

	if rcvName == "$li" {
		g.genListIndexor(n)
	} else {
		g.WS(rcvName)
		g.WS("(")
		if NodHasChild(n, NTR_RECEIVERCALL_ARG) {
			g.genValue(NodGetChild(n, NTR_RECEIVERCALL_ARG))
		}
		g.WS(")")
	}
}

func (g *Generator) genListIndexor(n Nod) {
	args := NodGetChildList(NodGetChild(n, NTR_RECEIVERCALL_ARG))
	g.genValue(args[0])
	g.WS("[")
	g.genValue(args[1])
	g.WS("]")
}

func (g *Generator) genValue(n Nod) {
	nt := n.NodeType
	if nt == NT_LIT_INT {
		g.genLiteralInt(n)
	} else if nt == NT_LIT_FLOAT {
		g.genLiteralFloat(n)
	} else if nt == NT_LIT_STRING {
		g.genLiteralString(n)
	} else if nt == NT_LIT_BOOL {
		g.genLiteralBool(n)
	} else if nt == NT_VAR_GETTER {
		g.genVarGetter(n)
	} else if nt == NT_LIT_LIST {
		g.genLiteralList(n)
	} else if nt == NT_LIT_SET {
		g.genLiteralSet(n)
	} else if nt == NT_LIT_MAP {
		g.genLiteralMap(n)
	} else if nt == NT_RECEIVERCALL {
		g.genReceiverCall(n)
	} else if nt == NT_CALLOBJINIT {
		g.genCallObjInit(n)
	} else if nt == NT_DOTOP {
		g.genDotOp(n)
	} else if g.isBinaryInlineDuckOpType(n.NodeType) {
		g.genDuckOp(n)
	} else if g.isBinaryInlineOpType(n.NodeType) {
		g.genBinaryInlineOp(n)
	} else {
		g.WS("value")
	}
}

func (g *Generator) genCallObjInit(n Nod) {
	// for now only will work for primitive types (outputs as go casts)
	argNodType := NodGetChild(NodGetChild(n, NTR_RECEIVERCALL_ARG), NTR_TYPE)
	if argNodType.NodeType == NT_TYPEBASE {
		fmt.Println("curr type gen", PrettyPrint(argNodType))
		if argNodType.Data.(int) == TY_DUCK {
			// use go type assertions
			g.genValue(NodGetChild(n, NTR_RECEIVERCALL_ARG))
			g.WS(".")
			g.WS("(")
			g.genType(NodGetChild(n, NTR_RECEIVERCALL_BASE))
			g.WS(")")
		} else {
			// use go casts
			g.genType(NodGetChild(n, NTR_RECEIVERCALL_BASE))
			g.WS("(")
			g.genValue(NodGetChild(n, NTR_RECEIVERCALL_ARG))
			g.WS(")")
		}
	} else {
		g.WS("(object initializer)")
	}

}

func (g *Generator) isBinaryInlineDuckOpType(nt int) bool {
	return nt == PNT_DUCK_ADDOP
}

func (g *Generator) genDotOp(n Nod) {
	qualName := NodGetChild(n, NTR_BINOP_RIGHT).Data.(string)
	objNod := NodGetChild(n, NTR_BINOP_LEFT)
	if qualName == "len" {
		g.WS("int64(len(")
		g.genValue(objNod)
		g.WS("))")
	} else {
		g.WS("(")
		g.genValue(objNod)
		g.WS(")")
		g.WS(".")
		g.WS("qualname")
	}
}

func (g *Generator) genLiteralInt(n Nod) {
	g.WS("int64(")
	g.WS(strconv.Itoa(n.Data.(int)))
	g.WS(")")
}

func (g *Generator) genLiteralFloat(n Nod) {
	g.WS(strconv.FormatFloat(n.Data.(float64), 'g', -1, 64))
}

func (g *Generator) genLiteralBool(n Nod) {
	lv := n.Data.(bool)
	if lv {
		g.WS("true")
	} else {
		g.WS("false")
	}
}

func (g *Generator) genLiteralString(n Nod) {
	g.WS("\"")
	g.WS(n.Data.(string))
	g.WS("\"")
}

func (g *Generator) isBinaryInlineOpType(nType int) bool {
	return nType == NT_ADDOP || nType == NT_GTOP || nType == NT_LTOP ||
		nType == NT_GTEQOP || nType == NT_LTEQOP || nType == NT_EQOP ||
		nType == NT_SUBOP || nType == NT_MULOP || nType == NT_DIVOP ||
		nType == NT_OROP || nType == NT_ANDOP || nType == NT_MODOP

}

func (g *Generator) getBinaryInlineOpSymbol(nType int) string {
	lut := map[int]string{
		NT_ADDOP:  "+",
		NT_SUBOP:  "-",
		NT_MULOP:  "*",
		NT_DIVOP:  "/",
		NT_GTOP:   ">",
		NT_LTOP:   "<",
		NT_GTEQOP: ">=",
		NT_LTEQOP: "<=",
		NT_EQOP:   "==",
		NT_OROP:   "||",
		NT_ANDOP:  "&&",
		NT_MODOP:  "%",
	}
	return lut[nType]
}

func (g *Generator) genBinaryInlineOp(n Nod) {
	g.WS("(")
	g.genValue(NodGetChild(n, NTR_BINOP_LEFT))
	g.WS(g.getBinaryInlineOpSymbol(n.NodeType))
	g.genValue(NodGetChild(n, NTR_BINOP_RIGHT))
	g.WS(")")
}

func (g *Generator) genDuckOp(n Nod) {
	g.WS("Pduck_add(")
	g.genValue(NodGetChild(n, NTR_BINOP_LEFT))
	g.WS(",")
	g.genValue(NodGetChild(n, NTR_BINOP_RIGHT))
	g.WS(")")
}

func (g *Generator) genLiteralSet(n Nod) {
	g.WS("map[interface{}]bool{")
	elements := NodGetChildList(n)
	for _, element := range elements {
		g.genValue(element)
		g.WS(": true")
		g.WS(", ")
	}
	g.WS("}")
}

func (g *Generator) genLiteralMap(n Nod) {
	g.WS("map[interface{}]interface{}{")
	kvpairs := NodGetChildList(n)
	for _, kvpair := range kvpairs {
		g.genMapKVPair(kvpair)
		g.WS(", ")
	}
	g.WS("}")
}

func (g *Generator) genMapKVPair(n Nod) {
	g.genValue(NodGetChild(n, NTR_KVPAIR_KEY))
	g.WS(": ")
	g.genValue(NodGetChild(n, NTR_KVPAIR_VAL))
}

func (g *Generator) genLiteralList(n Nod) {
	g.genType(NodGetChild(n, NTR_TYPE))
	g.WS("{")
	elements := NodGetChildList(n)
	for _, ele := range elements {
		g.genValue(ele)
		g.WS(", ")
	}
	g.WS("}")
}

func (g *Generator) genVarGetter(n Nod) {
	varName := NodGetChild(n, NTR_VAR_GETTER_NAME).Data.(string)
	g.WS(varName)
}

func (g *Generator) WS(s string) {
	g.buf.WriteString(s)
}
