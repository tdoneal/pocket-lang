package parse

import (
	"pocket-lang/tokenize"
	"pocket-lang/types"
	"strconv"
)

type Parser struct {
	input  []types.Token
	pos    int
	output *Node
}

type ParseFunc func() Nod

type ParseError struct {
	msg      string
	location *types.SourceLocation
}

var _ error = ParseError{}

func (p ParseError) Error() string {
	return "At " + p.location.StringDebug() + ": " + p.msg
}

func Parse(tokens []types.Token) Nod {

	parser := &Parser{
		input: tokens,
		pos:   0,
	}

	return parser.parseImperative()
}

func (p *Parser) parseImperative() Nod {
	units := p.parseAtLeastOneGreedy(func() Nod {
		return p.parseImperativeUnit()
	})
	rv := NodNewChildList(NT_IMPERATIVE, units)
	return rv
}

func (p *Parser) parseManyGreedy(f ParseFunc) []Nod {
	rv := make([]Nod, 0)
	for !p.isEOF() {
		opos := p.pos
		n, err := p.tryparse(f)
		if err != nil {
			p.pos = opos
			break
		}
		rv = append(rv, n)
	}
	return rv
}

func (p *Parser) parseManyGreedyEnsureCount(f ParseFunc,
	countCheck func(int) bool) []Nod {
	rv := p.parseManyGreedy(f)
	if !countCheck(len(rv)) {
		p.raiseParseError("incorrect number of subelements")
	}
	return rv
}

func (p *Parser) parseAtLeastOneGreedy(f ParseFunc) []Nod {
	return p.parseManyGreedyEnsureCount(f, func(n int) bool { return n >= 1 })
}

func (p *Parser) parseImperativeUnit() Nod {
	return p.parseDisjunction([]ParseFunc{
		func() Nod { return p.parseImperativeBlock() },
		func() Nod { return p.parseStatement() },
	})
}

func (p *Parser) getCurrToken() *types.Token {
	return &p.input[p.pos]
}

func (p *Parser) parseImperativeBlock() Nod {
	p.parseToken(tokenize.TK_INCINDENT)
	imp := p.parseImperative()
	p.parseToken(tokenize.TK_DECINDENT)
	return imp
}

func (p *Parser) tryparse(parseFunc ParseFunc) (obj Nod, e error) {
	defer func() {
		if r := recover(); r != nil {
			e = r.(error)
		}
	}()
	e = nil
	obj = parseFunc()
	return
}

func (p *Parser) parseStatement() Nod {
	// expect variable assignment
	rv := p.parseStatementBody()
	p.parseEOL()
	return rv
}

func (p *Parser) parseStatementBody() Nod {
	return p.parseDisjunction([]ParseFunc{
		func() Nod { return p.parseVarInit() },
		func() Nod { return p.parseReceiverCall() },
	})
}

func (p *Parser) parseVarInit() Nod {
	name := p.parseVarName()
	p.parseColon()
	val := p.parseValue()
	rv := NodNew(NT_VARINIT)
	NodSetChild(rv, NTR_VARINIT_NAME, name)
	NodSetChild(rv, NTR_VARINIT_VALUE, val)
	return rv
}

func (p *Parser) parseVarName() Nod {
	return p.parseIdentifier()
}

func (p *Parser) parseValue() Nod {
	return p.parseDisjunction([]ParseFunc{
		func() Nod { return p.parseValueInlineOpStream() },
		func() Nod { return p.parseValueParenthetical() },
		func() Nod { return p.parseLiteralInt() },
		func() Nod { return p.parseReceiverCall() },
		func() Nod { return p.parseVarGetter() },
	})
}

func (p *Parser) parseValueParenthetical() Nod {
	p.parseToken(tokenize.TK_PARENL)
	innerVal := p.parseValue()
	p.parseToken(tokenize.TK_PARENR)
	return innerVal
}

func (p *Parser) parseValueInlineOpStream() Nod {
	state := true // true: parsing non-stream ("atomic" value), false: parsing inline operator
	totalOps := 0
	elements := make([]Nod, 0)
	for {
		if state {
			aval, err := p.tryparse(func() Nod { return p.parseValueAtomic() })
			if err != nil {
				p.raiseParseError("expected atomic value")
				return nil
			}
			state = false
			elements = append(elements, aval)
		} else {
			oldpos := p.pos
			aval, err := p.tryparse(func() Nod { return p.parseInlineOp() })
			if err != nil {
				if totalOps == 0 {
					p.raiseParseError("expected inline op")
					return nil
				}
				// if here, we've simply reached the end of the inline op stream normally
				p.pos = oldpos
				break
			}
			state = true
			elements = append(elements, aval)
			totalOps++
		}
	}
	return NodNewChildList(NT_INLINEOPSTREAM, elements)
}

func (p *Parser) parseValueAtomic() Nod {
	return p.parseDisjunction([]ParseFunc{
		func() Nod { return p.parseValueParenthetical() },
		func() Nod { return p.parseLiteralInt() },
		func() Nod { return p.parseReceiverCall() },
		func() Nod { return p.parseVarGetter() },
	})
}

func (p *Parser) parseInlineOp() Nod {
	p.parseToken(tokenize.TK_ADDOP)
	return NodNew(NT_ADDOP)
}

func (p *Parser) parseVarGetter() Nod {
	return NodNewChild(NT_VAR_GETTER, NTR_VAR_GETTER_NAME, p.parseIdentifier())
}

func (p *Parser) parseTokenAlphanumeric() *types.Token {
	return p.parseToken(tokenize.TK_ALPHANUM)
}

func (p *Parser) parseIdentifier() Nod {
	return NodNewData(NT_IDENTIFIER, p.parseTokenAlphanumeric().Data)
}

func (p *Parser) parseDisjunction(funcs []ParseFunc) Nod {
	for i := 0; i < len(funcs); i++ {
		cfunc := funcs[i]
		oldpos := p.pos
		val, err := p.tryparse(cfunc)
		if err == nil {
			return val
		}
		p.pos = oldpos // backtrack
	}
	p.raiseParseError("parse error")
	return nil
}

func (p *Parser) parseReceiverCall() Nod {
	name := p.parseReceiverName()
	val := p.parseValue()
	// rv := &ReceiverCall{
	// 	ReceiverName:    &name,
	// 	ReceivedMessage: val,
	// }
	rv := NodNew(NT_RECEIVERCALL)
	NodSetChild(rv, NTR_RECEIVERCALL_NAME, name)
	NodSetChild(rv, NTR_RECEIVERCALL_VALUE, val)
	return rv
}

func (p *Parser) parseReceiverName() Nod {
	return p.parseIdentifier()
}

func (p *Parser) parseLiteralInt() Nod {
	tok := p.parseToken(tokenize.TK_LITERALINT)
	ival, err := strconv.Atoi(tok.Data)
	if err != nil {
		p.raiseParseError("in int literal")
	}
	return NodNewData(NT_LIT_INT, ival)
}

func (p *Parser) raiseParseError(msg string) {
	tok := p.currToken()
	pe := &ParseError{
		msg:      msg,
		location: tok.SourceLocation,
	}
	panic(pe)
}

func (p *Parser) parseColon() {
	p.parseToken(tokenize.TK_COLON)
}

func (p *Parser) parseEOL() {
	p.parseToken(tokenize.TK_EOL)
}

func (p *Parser) parseToken(tokenType int) *types.Token {
	cval := p.currToken()
	if cval.Type == tokenType {
		p.pos++
		return cval
	} else {
		p.raiseParseError("unexpected token")
		return nil
	}
}

func (p *Parser) currToken() *types.Token {
	return &p.input[p.pos]
}

func (p *Parser) isEOF() bool {
	return p.pos >= len(p.input)
}
