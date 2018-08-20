package goback

import (
	"bytes"
	. "pocket-lang/parse"
	"strconv"
)

type Generator struct {
	input Nod
	buf   *bytes.Buffer
}

func Generate(code Nod) string {

	generator := &Generator{
		buf:   &bytes.Buffer{},
		input: code,
	}

	generator.genImperative(code)

	return generator.buf.String()
}

func (g *Generator) genImperative(input Nod) {
	g.buf.WriteString("func main() {\n")

	statements := NodGetChildList(input)
	for _, stmt := range statements {
		g.genStatement(stmt)
	}

	g.buf.WriteString("}\n")
}

func (g *Generator) genStatement(input Nod) {
	if input.NodeType == NT_VARINIT {
		g.genVarInit(input)
	} else if input.NodeType == NT_RECEIVERCALL {
		g.genReceiverCall(input)
	}
	g.buf.WriteString("\n")
}

func (g *Generator) genVarInit(n Nod) {
	varName := NodGetChild(n, NTR_VARINIT_NAME).Data.(string)
	g.buf.WriteString(varName)
	g.buf.WriteString(" := (")
	g.genValue(NodGetChild(n, NTR_VARINIT_VALUE))
	g.buf.WriteString(")")
}

func (g *Generator) genReceiverCall(n Nod) {
	rcvName := NodGetChild(n, NTR_RECEIVERCALL_NAME).Data.(string)
	g.buf.WriteString(rcvName)
	g.buf.WriteString("(")
	g.genValue(NodGetChild(n, NTR_RECEIVERCALL_VALUE))
	g.buf.WriteString(")")
}

func (g *Generator) genValue(n Nod) {
	nt := n.NodeType
	if nt == NT_LIT_INT {
		g.buf.WriteString(strconv.Itoa(n.Data.(int)))
	} else if nt == NT_INLINEOPSTREAM {
		g.genOpStream(n)
	} else {
		g.buf.WriteString("value")
	}
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
