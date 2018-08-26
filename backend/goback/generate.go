package goback

import (
	"bytes"
	. "pocket-lang/frontend/pocket"
	. "pocket-lang/parse"
	"strconv"
)

type Generator struct {
	input Nod
	buf   *bytes.Buffer
}

func Generate(code Nod) string {

	preparer := &Preparer{}
	preparer.Prepare(code)

	generator := &Generator{
		buf:   &bytes.Buffer{},
		input: code,
	}

	generator.genSourceFile(code)

	return generator.buf.String()
}

func (g *Generator) genSourceFile(input Nod) {
	g.buf.WriteString("package main\n\n")

	units := NodGetChildList(input)

	for _, unit := range units {
		if unit.NodeType == NT_FUNCDEF {
			g.genFuncDef(unit)
		} else {
			panic("unknown source unit type")
		}
	}
}

func (g *Generator) genFuncDef(n Nod) {
	funcName := NodGetChild(n, NTR_FUNCDEF_NAME).Data.(string)
	g.buf.WriteString("func ")
	g.buf.WriteString(funcName)
	g.buf.WriteString("() ")

	if outType := NodGetChildOrNil(n, NTR_FUNCDEF_OUTTYPE); outType != nil {
		if outType.Data.(string) == "int" {
			g.buf.WriteString("int")
		}
	}

	g.buf.WriteString(" {\n")

	g.genImperative(NodGetChild(n, NTR_FUNCDEF_CODE))

	g.buf.WriteString("}\n")

}

func (g *Generator) genImperative(input Nod) {

	statements := NodGetChildList(input)
	for _, stmt := range statements {
		g.genStatement(stmt)
	}

}

func (g *Generator) genStatement(input Nod) {
	if input.NodeType == NT_VARINIT {
		g.genVarInit(input)
	} else if input.NodeType == NT_RECEIVERCALL {
		g.genReceiverCall(input)
	} else if input.NodeType == NT_RETURN {
		g.genReturn(input)
	}
	g.buf.WriteString("\n")
}

func (g *Generator) genReturn(input Nod) {
	g.buf.WriteString("return")
	if NodHasChild(input, NTR_RETURN_VALUE) {
		g.buf.WriteString(" (")
		g.genValue(NodGetChild(input, NTR_RETURN_VALUE))
		g.buf.WriteString(")")
	}
}

func (g *Generator) genVarInit(n Nod) {
	varName := NodGetChild(n, NTR_VARINIT_NAME).Data.(string)

	if ntype := NodGetChildOrNil(n, NTR_TYPE); ntype != nil {
		g.buf.WriteString("var ")
		g.buf.WriteString(varName)
		g.buf.WriteString(" ")
		g.buf.WriteString(ntype.Data.(string))
		g.buf.WriteString(" = ")
	} else {
		g.buf.WriteString(varName)
		g.buf.WriteString(" := ")
	}
	g.buf.WriteString("(")
	g.genValue(NodGetChild(n, NTR_VARINIT_VALUE))
	g.buf.WriteString(")")
}

func (g *Generator) genReceiverCall(n Nod) {
	rcvName := NodGetChild(n, NTR_RECEIVERCALL_NAME).Data.(string)

	if rcvName == "$li" {
		g.genListIndexor(n)
	} else {
		g.buf.WriteString(rcvName)
		g.buf.WriteString("(")
		if NodHasChild(n, NTR_RECEIVERCALL_VALUE) {
			g.genValue(NodGetChild(n, NTR_RECEIVERCALL_VALUE))
		}
		g.buf.WriteString(")")
	}
}

func (g *Generator) genListIndexor(n Nod) {
	args := NodGetChildList(NodGetChild(n, NTR_RECEIVERCALL_VALUE))
	g.genValue(args[0])
	g.buf.WriteString("[")
	g.genValue(args[1])
	g.buf.WriteString("]")
}

func (g *Generator) genValue(n Nod) {
	nt := n.NodeType
	if nt == NT_LIT_INT {
		g.buf.WriteString(strconv.Itoa(n.Data.(int)))
	} else if nt == NT_INLINEOPSTREAM {
		g.genOpStream(n)
	} else if nt == NT_VAR_GETTER {
		g.genVarGetter(n)
	} else if nt == NT_LIT_LIST {
		g.genLiteralList(n)
	} else if nt == NT_RECEIVERCALL {
		g.genReceiverCall(n)
	} else {
		g.buf.WriteString("value")
	}
}

func (g *Generator) genLiteralList(n Nod) {
	g.buf.WriteString("[]interface{}{")
	elements := NodGetChildList(n)
	for _, ele := range elements {
		g.genValue(ele)
		g.buf.WriteString(", ")
	}
	g.buf.WriteString("}")
}

func (g *Generator) genVarGetter(n Nod) {
	varName := NodGetChild(n, NTR_VAR_GETTER_NAME).Data.(string)
	g.buf.WriteString(varName)
}

func (g *Generator) genOpStream(n Nod) {
	g.buf.WriteString("(")
	children := NodGetChildList(n)
	for _, child := range children {
		if child.NodeType == NT_ADDOP {
			g.buf.WriteString("+")
		} else {
			g.genValue(child)
		}
	}
	g.buf.WriteString(")")
}
