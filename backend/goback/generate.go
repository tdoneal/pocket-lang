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
	input Nod
	buf   *bytes.Buffer
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

	if fmtImport := NodGetChildOrNil(input, PNR_GOIMPORTS); fmtImport != nil {
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

func (g *Generator) genParameterList(n Nod) {
	g.WS("args []interface{}")
}

func (g *Generator) genParameter(n Nod) {
	g.WS(NodGetChild(n, NTR_VARDEF_NAME).Data.(string))
	g.WS(" ")
	if paramType := NodGetChildOrNil(n, NTR_TYPE); paramType != nil {
		g.WS(paramType.Data.(string))
	} else {
		g.WS("interface{}")
	}
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
		if outType.Data.(string) == "int" {
			g.WS("int")
		}
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
		var typeName string
		if pType := NodGetChildOrNil(param, NTR_TYPE); pType != nil {
			typeName = pType.Data.(string)
		} else {
			typeName = "interface{}"
		}

		g.WS("var ")
		g.WS(NodGetChild(param, NTR_VARDEF_NAME).Data.(string))
		g.WS(" ")
		g.WS(typeName)
		g.WS(" = ")
		g.WS("args[")
		g.WS(strconv.Itoa(ndx))
		g.WS("].(")
		g.WS(typeName)
		g.WS(")")
		g.WS("\n")
	}
}

func (g *Generator) genImperative(input Nod) {

	if varTable := NodGetChildOrNil(input, NTR_VARTABLE); varTable != nil {
		g.genVarTable(varTable)
	}

	statements := NodGetChildList(input)
	for _, stmt := range statements {
		g.genImperativeUnit(stmt)
	}

}

func (g *Generator) genVarTable(n Nod) {
	varDefs := NodGetChildList(n)
	for _, varDef := range varDefs {
		g.genVarDef(varDef)
	}
}

func (g *Generator) genVarDef(n Nod) {
	varName := NodGetChild(n, NTR_VARDEF_NAME).Data.(string)
	g.WS("var ")
	g.WS(varName)
	g.WS("\n")
}

func (g *Generator) genImperativeUnit(n Nod) {
	if n.NodeType == NT_VARASSIGN {
		g.genVarAssign(n)
	} else if n.NodeType == NT_RECEIVERCALL {
		g.genReceiverCall(n)
	} else if n.NodeType == NT_RETURN {
		g.genReturn(n)
	} else if n.NodeType == NT_LOOP {
		g.genLoop(n)
	} else if n.NodeType == NT_IF {
		g.genIf(n)
	} else if n.NodeType == NT_BREAK {
		g.genBreak(n)
	} else {
		g.WS("command")
	}
	g.WS("\n")
}

func (g *Generator) genBreak(n Nod) {
	g.WS("break")
}

func (g *Generator) genLoop(input Nod) {
	g.WS("for {\n")
	g.genImperative(NodGetChild(input, NTR_LOOP_BODY))
	g.WS("}\n")
}

func (g *Generator) genIf(input Nod) {
	g.WS("if ")
	g.genValue(NodGetChild(input, NTR_IF_COND))
	g.WS("{\n")
	g.genImperative(NodGetChild(input, NTR_IF_BODY))
	g.WS("}\n")
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

	// if ntype := NodGetChildOrNil(n, NTR_TYPE); ntype != nil {
	// 	g.WS("var ")
	// 	g.WS(varName)
	// 	g.WS(" ")
	// 	g.WS(ntype.Data.(string))
	// 	g.WS(" = ")
	// } else {
	// 	g.WS(varName)
	// 	g.WS(" := ")
	// }
	g.WS(varName)
	g.WS(" = ")
	g.WS("(")
	g.genValue(NodGetChild(n, NTR_VARASSIGN_VALUE))
	g.WS(")")
}

func (g *Generator) genReceiverCall(n Nod) {
	rcvName := NodGetChild(n, NTR_RECEIVERCALL_NAME).Data.(string)

	if rcvName == "print" {
		rcvName = "fmt.Println"
	}

	if rcvName == "$li" || rcvName == "$mi" {
		g.genListIndexor(n)
	} else {
		g.WS(rcvName)
		g.WS("(")
		if NodHasChild(n, NTR_RECEIVERCALL_VALUE) {
			g.genValue(NodGetChild(n, NTR_RECEIVERCALL_VALUE))
		}
		g.WS(")")
	}
}

func (g *Generator) genListIndexor(n Nod) {
	args := NodGetChildList(NodGetChild(n, NTR_RECEIVERCALL_VALUE))
	g.genValue(args[0])
	g.WS("[")
	g.genValue(args[1])
	g.WS("]")
}

func (g *Generator) genValue(n Nod) {
	nt := n.NodeType
	if nt == NT_LIT_INT {
		g.WS(strconv.Itoa(n.Data.(int)))
	} else if nt == NT_INLINEOPSTREAM {
		g.genOpStream(n)
	} else if nt == NT_VAR_GETTER {
		g.genVarGetter(n)
	} else if nt == NT_LIT_LIST {
		g.genLiteralList(n)
	} else if nt == NT_LIT_MAP {
		g.genLiteralMap(n)
	} else if nt == NT_RECEIVERCALL {
		g.genReceiverCall(n)
	} else {
		g.WS("value")
	}
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
	g.WS("[]interface{}{")
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

func (g *Generator) genOpStream(n Nod) {
	g.WS("(")
	children := NodGetChildList(n)
	for _, child := range children {
		g.genOpStreamChild(child)
	}
	g.WS(")")
}

func (g *Generator) genOpStreamChild(n Nod) {
	if n.NodeType == NT_ADDOP {
		g.WS("+")
	} else if n.NodeType == NT_GTOP {
		g.WS(">")
	} else if n.NodeType == NT_LTOP {
		g.WS("<")
	} else {
		g.genValue(n)
	}
}

func (g *Generator) WS(s string) {
	g.buf.WriteString(s)
}
