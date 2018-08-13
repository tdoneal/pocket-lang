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
			fmt.Println("safely caught", r.(error))
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
	tok := p.parseTokenAlphanumeric()
	return tok.Data
}

func (p *Parser) parseValue() Value {
	return p.parseDisjunction([]ParseFunc{
		func() interface{} { return p.parseValueInlineOpStream() },
		func() interface{} { return p.parseValueParenthetical() },
		func() interface{} { return p.parseLiteralInt() },
		func() interface{} { return p.parseReceiverCall() },
		func() interface{} { return p.parseVarGetter() },
	})
}

func (p *Parser) parseValueParenthetical() Value {
	p.parseToken(tokenize.TK_PARENL)
	innerVal := p.parseValue()
	p.parseToken(tokenize.TK_PARENR)
	return innerVal
}

func (p *Parser) parseValueInlineOpStream() *ValueInlineOpStream {
	state := true // true: parsing non-stream ("atomic" value), false: parsing inline operator
	totalOps := 0
	elements := make([]ValueInlineOpStreamElement, 0)
	for {
		if state {
			aval, err := p.tryparse(func() interface{} { return p.parseValueAtomic() })
			fmt.Println(spew.Sdump(aval), spew.Sdump(err))
			if err != nil {
				p.raiseParseError("expected atomic value")
				return nil
			}
			state = false
			elements = append(elements, aval)
		} else {
			oldpos := p.pos
			aval, err := p.tryparse(func() interface{} { return p.parseInlineOp() })
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
	return &ValueInlineOpStream{
		Elements: elements,
	}
}

func (p *Parser) parseValueAtomic() Value {
	return p.parseDisjunction([]ParseFunc{
		func() interface{} { return p.parseValueParenthetical() },
		func() interface{} { return p.parseLiteralInt() },
		func() interface{} { return p.parseReceiverCall() },
		func() interface{} { return p.parseVarGetter() },
	})
}

func (p *Parser) parseInlineOp() InlineOp {
	fmt.Println("parsing inline op, currtoken=", spew.Sdump(p.currToken()))
	p.parseToken(tokenize.TK_ADDOP)
	fmt.Println("it worked...")
	return &AddOp{}
}

func (p *Parser) parseVarGetter() *VarGetter {
	ctok := p.parseTokenAlphanumeric()
	return &VarGetter{
		VarName: ctok.Data,
	}
}

func (p *Parser) parseTokenAlphanumeric() *types.Token {
	return p.parseToken(tokenize.TK_ALPHANUM)
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
	ctok := p.parseTokenAlphanumeric()
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
