package pocket

import (
	"fmt"
	. "pocket-lang/parse"
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
	funcName := p.ParseToken(TK_ALPHANUM).Data
	funcWord := p.ParseToken(TK_ALPHANUM).Data
	rv := NodNew(NT_FUNCDEF)
	if funcWord != "func" {
		p.RaiseParseError("missing func keyword")
	}

	// parse function type declarations if extant
	// for now, if they are extant, require an explicit in type and explicit out type
	funcInputType := p.ParseAtMostOne(func() Nod { return p.parseFuncDefTypeValue() })

	if funcInputType != nil {
		funcOutputType := p.parseFuncDefTypeValue()
		NodSetChild(rv, NTR_FUNCDEF_INTYPE, funcInputType)
		NodSetChild(rv, NTR_FUNCDEF_OUTTYPE, funcOutputType)
	}

	p.parseEOL()
	p.ParseToken(TK_INCINDENT)
	imp := p.parseImperative()
	p.ParseToken(TK_DECINDENT)

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
		func() Nod { return p.parseIf() },
		func() Nod { return p.parseLoop() },
		func() Nod { return p.parseBreak() },
		func() Nod { return p.parseImperativeBlock() },
		func() Nod { return p.parseStatement() },
	})
}

func (p *ParserPocket) parseBreak() Nod {
	p.ParseToken(TK_BREAK)
	p.parseEOL()
	return NodNew(NT_BREAK)
}

func (p *ParserPocket) parseIf() Nod {
	p.ParseToken(TK_IF)
	cond := p.parseValue()
	p.parseEOL()
	body := p.parseImperativeBlock()
	rv := NodNew(NT_IF)
	NodSetChild(rv, NTR_IF_COND, cond)
	NodSetChild(rv, NTR_IF_BODY, body)
	return rv
}

func (p *ParserPocket) parseLoop() Nod {
	p.ParseToken(TK_LOOP)
	p.parseEOL()
	body := p.parseImperativeBlock()
	return NodNewChild(NT_LOOP, NTR_LOOP_BODY, body)
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
		func() Nod { return p.parseVarAssign() },
		func() Nod { return p.parseReceiverCallStatement() },
	})
}

func (p *ParserPocket) parseReturnStatement() Nod {
	p.ParseToken(TK_RETURN)
	rv := NodNew(NT_RETURN)
	val := p.ParseAtMostOne(func() Nod { return p.parseValue() })
	if val != nil {
		NodSetChild(rv, NTR_RETURN_VALUE, val)
	}
	return rv
}

func (p *ParserPocket) parseVarAssign() Nod {
	name := p.parseVarName()
	varType := p.ParseAtMostOne(func() Nod { return p.parseType() })
	p.parseColon()
	val := p.parseValue()
	rv := NodNew(NT_VARASSIGN)
	NodSetChild(rv, NTR_VAR_NAME, name)
	NodSetChild(rv, NTR_VARASSIGN_VALUE, val)
	if varType != nil {
		NodSetChild(rv, NTR_TYPE_DECL, varType)
	}
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
	p.ParseToken(TK_PARENL)
	innerVal := p.parseValue()
	p.ParseToken(TK_PARENR)
	return innerVal
}

func (p *ParserPocket) parseValueInlineOpStream() Nod {
	elements := p.ParseUnrolledSequenceGreedy([]ParseFunc{
		func() Nod { return p.parseValueAtomic() },
		func() Nod { return p.parseInlineOp() },
	})

	fmt.Println("value inline op stream: got", len(elements), "elements:")
	fmt.Println(PrettyPrintNodes(elements))

	if len(elements) < 3 {
		p.RaiseParseError("invalid op stream")
	}
	if len(elements)%2 != 1 {
		p.RaiseParseError("invalid op stream")
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
		func() Nod { return p.parseLiteralString() },
		func() Nod { return p.parseLiteralList() },
		func() Nod { return p.parseLiteralMap() },
		func() Nod { return p.parseLiteralInt() },
	})
}

func (p *ParserPocket) parseLiteralString() Nod {
	tkn := p.ParseToken(TK_LITERALSTRING)
	return NodNewData(NT_LIT_STRING, tkn.Data)
}

func (p *ParserPocket) parseLiteralMap() Nod {
	p.ParseToken(TK_CURLYL)

	kvpairs := p.parseManyOptDelimited(func() Nod { return p.parseMapKeyValuePair() },
		func() Nod { return p.parseComma() })

	p.ParseToken(TK_CURLYR)

	return NodNewChildList(NT_LIT_MAP, kvpairs)
}

func (p *ParserPocket) parseMapKeyValuePair() Nod {
	key := p.parseValue()
	p.parseColon()
	val := p.parseValue()
	rv := NodNew(NT_LIT_MAP_KVPAIR)
	NodSetChild(rv, NTR_KVPAIR_KEY, key)
	NodSetChild(rv, NTR_KVPAIR_VAL, val)
	return rv
}

func (p *ParserPocket) parseLiteralList() Nod {
	p.ParseToken(TK_BRACKL)
	elements := p.parseManyOptDelimited(
		func() Nod { return p.parseValue() },
		func() Nod { return p.parseComma() },
	)
	p.ParseToken(TK_BRACKR)
	return NodNewChildList(NT_LIT_LIST, elements)
}

func (p *ParserPocket) parseLiteralKeyword() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseKeywordLitPrimitive(TK_VOID, "void") },
		func() Nod { return p.parseKeywordLitPrimitive(TK_INT, "int") },
	})
}

func (p *ParserPocket) parseKeywordLitPrimitive(tokenType int, data string) Nod {
	p.ParseToken(tokenType)
	return NodNewData(NT_LIT_PRIMITIVE, data)
}

func (p *ParserPocket) parseFuncDefTypeValue() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseParameterParenthetical() },
		func() Nod { return p.parseParameterList() },
		func() Nod { return p.parseLiteralKeyword() },
	})
}

func (p *ParserPocket) parseType() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseLiteralKeyword() },
	})
}

func (p *ParserPocket) parseParameterList() Nod {
	p.ParseToken(TK_BRACKL)
	parameters := p.parseManyOptDelimited(
		func() Nod { return p.parseParameterSingle() },
		func() Nod { return p.parseComma() },
	)
	p.ParseToken(TK_BRACKR)
	return NodNewChildList(NT_LIT_LIST, parameters)
}

func (p *ParserPocket) parseComma() Nod {
	p.ParseToken(TK_COMMA)
	return nil
}

func (p *ParserPocket) parseManyOptDelimited(elementParser func() Nod,
	delimiter func() Nod) []Nod {
	values := make([]Nod, 0)
	for {
		value := p.ParseAtMostOne(elementParser)
		if value == nil {
			break
		}
		values = append(values, value)
		p.ParseAtMostOne(delimiter)
	}
	return values
}

func (p *ParserPocket) parseParameterSingle() Nod {
	varname := p.parseVarName()
	vartype := p.ParseAtMostOne(func() Nod { return p.parseType() })
	rv := NodNew(NT_PARAMETER)
	NodSetChild(rv, NTR_VARDEF_NAME, varname)
	if vartype != nil {
		NodSetChild(rv, NTR_TYPE_DECL, vartype)
	}
	return rv
}

func (p *ParserPocket) parseParameterParenthetical() Nod {
	p.ParseToken(TK_PARENL)
	rv := p.parseParameterSingle()
	p.ParseToken(TK_PARENR)
	return rv
}

func (p *ParserPocket) parseInlineOp() Nod {
	ctok := p.CurrToken().Type
	fmt.Println("checking if inline op: ", spew.Sdump(p.CurrToken()))
	nt := p.inlineOpTokenToNT(ctok)
	if nt != -1 {
		p.ParseToken(ctok)
		rv := NodNew(nt)
		return rv
	}
	fmt.Println("wasn't an inline op")
	p.RaiseParseError("invalid inline op")
	return nil

}

func (p *ParserPocket) inlineOpTokenToNT(tokenType int) int {
	if tokenType == TK_ADDOP {
		return NT_ADDOP
	} else if tokenType == TK_LT {
		return NT_LTOP
	} else if tokenType == TK_GT {
		return NT_GTOP
	} else {
		return -1
	}
}

func (p *ParserPocket) parseVarGetter() Nod {
	return NodNewChild(NT_VAR_GETTER, NTR_VAR_GETTER_NAME, p.parseIdentifier())
}

func (p *ParserPocket) parseTokenAlphanumeric() *types.Token {
	return p.ParseToken(TK_ALPHANUM)
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
	tok := p.ParseToken(TK_LITERALINT)
	ival, err := strconv.Atoi(tok.Data)
	if err != nil {
		p.RaiseParseError("in int literal")
	}
	return NodNewData(NT_LIT_INT, ival)
}

func (p *ParserPocket) parseColon() {
	p.ParseToken(TK_COLON)
}

func (p *ParserPocket) parseEOL() {
	p.ParseToken(TK_EOL)
}

func (p *ParserPocket) parseImperativeBlock() Nod {
	p.ParseToken(TK_INCINDENT)
	imp := p.parseImperative()
	p.ParseToken(TK_DECINDENT)
	return imp
}
