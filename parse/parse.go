package parse

import (
	"pocket-lang/tokenize"
	"pocket-lang/types"
	"strconv"
)

type Node struct {
	parent   *Node
	children []Node
}

type Parser struct {
	input  []types.Token
	pos    int
	output *Node
}

type ParseFunc func() interface{}

type ParseError struct {
	msg      string
	location *types.SourceLocation
}

var _ error = ParseError{}

func (p ParseError) Error() string {
	return "At " + p.location.StringDebug() + ": " + p.msg
}

func Parse(tokens []types.Token) *Imperative {

	parser := &Parser{
		input: tokens,
		pos:   0,
	}

	return parser.parseImperative()
}

func (p *Parser) parseImperative() *Imperative {
	// expect list of Statements
	stmts := make([]Statement, 0)
	for !p.isEOF() {
		stmt := p.parseStatement()
		stmts = append(stmts, stmt)
	}
	rv := &Imperative{
		Statements: stmts,
	}
	return rv
}

func (p *Parser) tryparse(parseFunc ParseFunc) (obj interface{}, e error) {
	defer func() {
		if r := recover(); r != nil {
			e = r.(error)
		}
	}()
	e = nil
	obj = parseFunc()
	return
}

func (p *Parser) parseStatement() Statement {
	// expect variable assignment
	rv := p.parseStatementBody()
	p.parseEOL()
	return rv
}

func (p *Parser) parseStatementBody() Statement {
	return p.parseDisjunction([]ParseFunc{
		func() interface{} { return p.parseVarInit() },
		func() interface{} { return p.parseReceiverCall() },
	})
}

func (p *Parser) parseVarInit() *VarInit {
	name := p.parseVarName()
	p.parseColon()
	val := p.parseValue()
	return &VarInit{
		VarName:  name,
		VarValue: val,
	}
}

func (p *Parser) parseVarName() string {
	tok := p.parseToken(tokenize.TK_ALPHANUM)
	return tok.Data
}

func (p *Parser) parseValue() Value {
	return p.parseDisjunction([]ParseFunc{
		func() interface{} { return p.parseLiteralInt() },
		func() interface{} { return p.parseReceiverCall() },
	})
}

func (p *Parser) parseDisjunction(funcs []ParseFunc) interface{} {
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

func (p *Parser) parseReceiverCall() *ReceiverCall {
	name := p.parseReceiverName()
	val := p.parseValue()
	rv := &ReceiverCall{
		ReceiverName:    &name,
		ReceivedMessage: val,
	}
	return rv
}

func (p *Parser) parseReceiverName() string {
	ctok := p.parseToken(tokenize.TK_ALPHANUM)
	return ctok.Data
}

func (p *Parser) parseLiteralInt() *LiteralInt {
	tok := p.parseToken(tokenize.TK_LITERALINT)
	ival, err := strconv.Atoi(tok.Data)
	if err != nil {
		p.raiseParseError("in int literal")
	}
	rv := &LiteralInt{
		Value: ival,
	}
	return rv
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
