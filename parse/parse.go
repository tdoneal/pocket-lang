package parse

import (
	"pocket-lang/types"
)

type Parser struct {
	Input  []types.Token
	Pos    int
	Output *Node
}

type ParseFunc func() Nod

type ParseError struct {
	Msg      string
	Location *types.SourceLocation
}

var _ error = ParseError{}

func (p ParseError) Error() string {
	return "At " + p.Location.StringDebug() + ": " + p.Msg
}

// returns nil if zero parsed, node if one parsed
func (p *Parser) ParseAtMostOne(f ParseFunc) Nod {
	opos := p.Pos
	nod, err := p.Tryparse(f)
	if err == nil {
		return nod
	} else {
		p.Pos = opos
		return nil
	}
}

func (p *Parser) ParseManyGreedy(f ParseFunc) []Nod {
	rv := make([]Nod, 0)
	for !p.IsEOF() {
		opos := p.Pos
		n, err := p.Tryparse(f)
		if err != nil {
			p.Pos = opos
			break
		}
		rv = append(rv, n)
	}
	return rv
}

func (p *Parser) ParseManyGreedyEnsureCount(f ParseFunc,
	countCheck func(int) bool) []Nod {
	rv := p.ParseManyGreedy(f)
	if !countCheck(len(rv)) {
		p.RaiseParseError("incorrect number of subelements")
	}
	return rv
}

func (p *Parser) ParseAtLeastOneGreedy(f ParseFunc) []Nod {
	return p.ParseManyGreedyEnsureCount(f, func(n int) bool { return n >= 1 })
}

func (p *Parser) GetCurrToken() *types.Token {
	return &p.Input[p.Pos]
}

func (p *Parser) Tryparse(parseFunc ParseFunc) (obj Nod, e error) {
	defer func() {
		if r := recover(); r != nil {
			e = r.(*ParseError)
		}
	}()
	e = nil
	obj = parseFunc()
	return
}

func (p *Parser) ParseDisjunction(funcs []ParseFunc) Nod {
	for i := 0; i < len(funcs); i++ {
		cfunc := funcs[i]
		oldpos := p.Pos
		val, err := p.Tryparse(cfunc)
		if err == nil {
			return val
		}
		p.Pos = oldpos // backtrack
	}
	p.RaiseParseError("parse error")
	return nil
}

func (p *Parser) RaiseParseError(msg string) {
	tok := p.CurrToken()
	pe := &ParseError{
		Msg:      msg,
		Location: tok.SourceLocation,
	}
	panic(pe)
}

func (p *Parser) ParseToken(tokenType int) *types.Token {
	cval := p.CurrToken()
	if cval.Type == tokenType {
		p.Pos++
		return cval
	} else {
		p.RaiseParseError("unexpected token")
		return nil
	}
}

func (p *Parser) CurrToken() *types.Token {
	return &p.Input[p.Pos]
}

func (p *Parser) IsEOF() bool {
	return p.Pos >= len(p.Input)
}
