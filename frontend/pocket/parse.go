package pocket

import (
	"fmt"
	. "pocket-lang/parse"
	"pocket-lang/tokenize"
	"pocket-lang/types"
	"strconv"

	"github.com/davecgh/go-spew/spew"
)

type ParserPocket struct {
	*Parser
}

func Parse(tokens []types.Token) Nod {

	parser := &ParserPocket{
		&Parser{
			Input: tokens,
			Pos:   0,
		},
	}

	return parser.parseTopLevel()
}

func (p *ParserPocket) parseFuncDef() Nod {
	fmt.Println("parsing func def at ", spew.Sdump(p.CurrToken()))
	funcName := p.ParseToken(tokenize.TK_ALPHANUM).Data
	fmt.Println("parsing potential func kw")
	funcWord := p.ParseToken(tokenize.TK_ALPHANUM).Data
	fmt.Println("gotit")
	rv := NodNew(NT_FUNCDEF)
	if funcWord != "func" {
		fmt.Println("missing func keyword")
		p.RaiseParseError("missing func keyword")
	}
	fmt.Println("got func keyw")

	// parse function type declarations if extant
	// for now, if they are extant, require an explicit in type and explicit out type
	funcInputType := p.ParseAtMostOne(func() Nod { return p.parseFuncDefTypeValue() })

	if funcInputType != nil {
		funcOutputType := p.parseFuncDefTypeValue()
		NodSetChild(rv, NTR_FUNCDEF_INTYPE, funcInputType)
		NodSetChild(rv, NTR_FUNCDEF_OUTTYPE, funcOutputType)
	}

	p.parseEOL()
	p.ParseToken(tokenize.TK_INCINDENT)
	imp := p.parseImperative()
	p.ParseToken(tokenize.TK_DECINDENT)

	funcNameNode := NodNewData(NT_IDENTIFIER, funcName)
	NodSetChild(rv, NTR_FUNCDEF_NAME, funcNameNode)
	NodSetChild(rv, NTR_FUNCDEF_CODE, imp)
	return rv
}

func (p *ParserPocket) parseImperative() Nod {
	units := p.ParseAtLeastOneGreedy(func() Nod {
		return p.parseImperativeUnit()
	})
	rv := NodNewChildList(NT_IMPERATIVE, units)
	return rv
}

func (p *ParserPocket) parseImperativeUnit() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseImperativeBlock() },
		func() Nod { return p.parseStatement() },
	})
}

func (p *ParserPocket) parseTopLevel() Nod {
	units := p.ParseManyGreedy(func() Nod {
		return p.parseFuncDef()
	})

	fmt.Println("top level units:", PrettyPrintNodes(units))

	if !p.IsEOF() {
		p.RaiseParseError("failed to consume all input")
	}

	return NodNewChildList(NT_TOPLEVEL, units)
}

func (p *ParserPocket) parseStatement() Nod {
	rv := p.parseStatementBody()
	p.parseEOL()
	return rv
}

func (p *ParserPocket) parseStatementBody() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseReturnStatement() },
		func() Nod { return p.parseVarInit() },
		func() Nod { return p.parseReceiverCallStatement() },
	})
}

func (p *ParserPocket) parseReturnStatement() Nod {
	p.ParseToken(tokenize.TK_RETURN)
	rv := NodNew(NT_RETURN)
	val := p.ParseAtMostOne(func() Nod { return p.parseValue() })
	if val != nil {
		NodSetChild(rv, NTR_RETURN_VALUE, val)
	}
	return rv
}

func (p *ParserPocket) parseVarInit() Nod {
	name := p.parseVarName()
	p.parseColon()
	val := p.parseValue()
	rv := NodNew(NT_VARINIT)
	NodSetChild(rv, NTR_VARINIT_NAME, name)
	NodSetChild(rv, NTR_VARINIT_VALUE, val)
	return rv
}

func (p *ParserPocket) parseVarName() Nod {
	return p.parseIdentifier()
}

func (p *ParserPocket) parseValue() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseValueInlineOpStream() },
		func() Nod { return p.parseValueParenthetical() },
		func() Nod { return p.parseLiteral() },
		func() Nod { return p.parseReceiverCall() },
		func() Nod { return p.parseVarGetter() },
	})
}

func (p *ParserPocket) parseValueParenthetical() Nod {
	p.ParseToken(tokenize.TK_PARENL)
	innerVal := p.parseValue()
	p.ParseToken(tokenize.TK_PARENR)
	return innerVal
}

func (p *ParserPocket) parseValueInlineOpStream() Nod {
	state := true // true: parsing non-stream ("atomic" value), false: parsing inline operator
	totalOps := 0
	elements := make([]Nod, 0)
	for {
		if state {
			aval, err := p.Tryparse(func() Nod { return p.parseValueAtomic() })
			if err != nil {
				p.RaiseParseError("expected atomic value")
				return nil
			}
			state = false
			elements = append(elements, aval)
		} else {
			oldpos := p.Pos
			aval, err := p.Tryparse(func() Nod { return p.parseInlineOp() })
			if err != nil {
				if totalOps == 0 {
					p.RaiseParseError("expected inline op")
					return nil
				}
				// if here, we've simply reached the end of the inline op stream normally
				p.Pos = oldpos
				break
			}
			state = true
			elements = append(elements, aval)
			totalOps++
		}
	}
	return NodNewChildList(NT_INLINEOPSTREAM, elements)
}

func (p *ParserPocket) parseValueAtomic() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseValueParenthetical() },
		func() Nod { return p.parseLiteral() },
		func() Nod { return p.parseReceiverCall() },
		func() Nod { return p.parseVarGetter() },
	})
}

func (p *ParserPocket) parseLiteral() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseLiteralKeyword() },
		func() Nod { return p.parseLiteralList() },
		func() Nod { return p.parseLiteralInt() },
	})
}

func (p *ParserPocket) parseLiteralList() Nod {
	p.ParseToken(tokenize.TK_BRACKL)
	values := make([]Nod, 0)
	for {
		value := p.ParseAtMostOne(func() Nod { return p.parseValue() })
		if value == nil {
			break
		}
		values = append(values, value)
		p.ParseAtMostOne(func() Nod { p.ParseToken(tokenize.TK_COMMA); return nil })
	}
	p.ParseToken(tokenize.TK_BRACKR)

	rv := NodNewChildList(NT_LIT_LIST, values)
	return rv
}

func (p *ParserPocket) parseLiteralKeyword() Nod {
	fmt.Println("parseLiteralKeyword @ ", spew.Sdump(p.CurrToken()))
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseKeywordLitPrimitive(tokenize.TK_VOID, "void") },
		func() Nod { return p.parseKeywordLitPrimitive(tokenize.TK_INT, "int") },
	})
}

func (p *ParserPocket) parseKeywordLitPrimitive(tokenType int, data string) Nod {
	p.ParseToken(tokenType)
	return NodNewData(NT_LIT_PRIMITIVE, data)
}

func (p *ParserPocket) parseFuncDefTypeValue() Nod {
	return p.ParseDisjunction([]ParseFunc{
		// func() Nod { return p.parseValueParenthetical() },
		// func() Nod { return p.parseVarGetter() },
		func() Nod { return p.parseLiteralKeyword() },
	})
}

func (p *ParserPocket) parseInlineOp() Nod {
	p.ParseToken(tokenize.TK_ADDOP)
	return NodNew(NT_ADDOP)
}

func (p *ParserPocket) parseVarGetter() Nod {
	return NodNewChild(NT_VAR_GETTER, NTR_VAR_GETTER_NAME, p.parseIdentifier())
}

func (p *ParserPocket) parseTokenAlphanumeric() *types.Token {
	return p.ParseToken(tokenize.TK_ALPHANUM)
}

func (p *ParserPocket) parseIdentifier() Nod {
	return NodNewData(NT_IDENTIFIER, p.parseTokenAlphanumeric().Data)
}

func (p *ParserPocket) parseReceiverCallStatement() Nod {
	name := p.parseReceiverName()
	val := p.ParseAtMostOne(func() Nod { return p.parseValue() })
	rv := NodNew(NT_RECEIVERCALL)
	NodSetChild(rv, NTR_RECEIVERCALL_NAME, name)
	if val != nil {
		NodSetChild(rv, NTR_RECEIVERCALL_VALUE, val)
	}
	return rv
}

func (p *ParserPocket) parseReceiverCall() Nod {
	name := p.parseReceiverName()
	val := p.parseValue()
	rv := NodNew(NT_RECEIVERCALL)
	NodSetChild(rv, NTR_RECEIVERCALL_NAME, name)
	NodSetChild(rv, NTR_RECEIVERCALL_VALUE, val)
	return rv
}

func (p *ParserPocket) parseReceiverName() Nod {
	return p.parseIdentifier()
}

func (p *ParserPocket) parseLiteralInt() Nod {
	tok := p.ParseToken(tokenize.TK_LITERALINT)
	ival, err := strconv.Atoi(tok.Data)
	if err != nil {
		p.RaiseParseError("in int literal")
	}
	return NodNewData(NT_LIT_INT, ival)
}

func (p *ParserPocket) parseColon() {
	p.ParseToken(tokenize.TK_COLON)
}

func (p *ParserPocket) parseEOL() {
	p.ParseToken(tokenize.TK_EOL)
}

func (p *ParserPocket) parseImperativeBlock() Nod {
	p.ParseToken(tokenize.TK_INCINDENT)
	imp := p.parseImperative()
	p.ParseToken(tokenize.TK_DECINDENT)
	return imp
}
