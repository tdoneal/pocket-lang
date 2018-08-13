package parse

import (
	"fmt"
	"pocket-lang/tokenize"
	"pocket-lang/types"
	"strconv"

	"github.com/davecgh/go-spew/spew"
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

type ParseError struct {
	msg      string
	location *types.SourceLocation
}

var _ error = ParseError{}

func (p ParseError) Error() string {
	return "At " + p.location.StringDebug() + ": " + p.msg
}

func Parse(tokens []types.Token) *Imperative {
	fmt.Println("parsin", tokens)

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

func (p *Parser) tryparse(parseFunc func() interface{}) (obj interface{}, e error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
			e = r.(error)
		}
	}()
	e = nil
	obj = parseFunc()
	return
}

func (p *Parser) parseStatement() Statement {
	// expect variable assignment
	rv := p.parseVarInit()
	p.parseEOL()
	fmt.Println("emitted statement", spew.Sdump(rv))
	return rv
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
	// currently only int literals supported
	il := p.parseLiteralInt()
	return il
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
