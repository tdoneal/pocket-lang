package goback

import (
	"bytes"
	"pocket-lang/parse"
)

type Generator struct {
	input *parse.Imperative
	buf   *bytes.Buffer
}

func Generate(code *parse.Imperative) string {

	generator := &Generator{
		buf:   &bytes.Buffer{},
		input: code,
	}

	generator.genImperative(code)

	return generator.buf.String()
}

func (g *Generator) genImperative(input *parse.Imperative) {
	g.buf.WriteString("func main() {\n")

	for i := 0; i < len(input.Statements); i++ {
		stmt := input.Statements[i]
		g.genStatement(stmt)
	}
	g.buf.WriteString("}\n")
}

func (g *Generator) genStatement(input parse.Statement) {
	g.buf.WriteString("statement\n")
	// if input assignable *parse.VarInit {
	// 	g.buf.WriteString("var init\n")
	// }else {
	// 	g.buf.WriteString("unknown stmt\n")
	// }
}
