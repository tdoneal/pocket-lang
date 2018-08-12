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
		statements: stmts,
	}
	return rv
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
		varName:  name,
		varValue: val,
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
		panic("int literal syntax err")
	}
	rv := &LiteralInt{
		value: ival,
	}
	return rv
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
		panic("invalid token" + fmt.Sprint(cval))
	}
}

func (p *Parser) currToken() *types.Token {
	return &p.input[p.pos]
}

func (p *Parser) isEOF() bool {
	return p.pos >= len(p.input)
}
