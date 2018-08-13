package tokenize

// Tokenizes an input string

import (
	"bytes"
	"fmt"
	"pocket-lang/types"
	"strconv"
)

const (
	TKS_INIT      = 0
	TK_EOL        = 1
	TK_INCINDENT  = 2
	TK_DECINDENT  = 3
	TK_LITERALINT = 10
	TK_ALPHANUM   = 20
	TK_COLON      = 30
	TK_ADDOP      = 40
	TK_COMMENT    = 50
	TK_PARENL     = 60
	TK_PARENR     = 61
)

type LiteralInt struct {
	*types.Token
	value *int
}

type Tokenizer struct {
	input   string
	pos     int
	state   int
	srcLoc  *types.SourceLocation
	tokbuf  *bytes.Buffer
	outtoks []types.Token
}

func Tokenize(input string) []types.Token {
	fmt.Println("input", input)

	tkzr := &Tokenizer{
		input: input,
		pos:   0,
		state: 0,
		srcLoc: &types.SourceLocation{
			Char:   0,
			Line:   0,
			Column: 0,
		},
		tokbuf: &bytes.Buffer{},
	}

	for tkzr.pos < len(input) {
		var crune rune = rune(input[tkzr.pos])
		tkzr.process(crune)
	}

	return tkzr.outtoks
}

func (tkzr *Tokenizer) process(input rune) {
	fmt.Println("state " + strconv.Itoa(tkzr.state) + ". proc rune '" + string(input) + "'")
	if tkzr.state == TKS_INIT {
		tkzr.processInit(input)
	} else if tkzr.state == TK_ALPHANUM {
		tkzr.processAlphanum(input)
	} else {
		panic("unknown state that i'm in")
	}
}

func (tkzr *Tokenizer) processInit(input rune) {
	if isAlphic(input) {
		tkzr.state = TK_ALPHANUM
	} else if input == ':' {
		tkzr.emitTokenRune(TK_COLON, input)
		tkzr.incr()
	} else if isDigit(input) {
		tkzr.processLiteralInt()
	} else if input == ' ' {
		tkzr.processSpace()
	} else if input == '\t' {
		tkzr.processTab()
	} else if input == '\n' {
		tkzr.processEOL()
	} else if input == '+' {
		tkzr.emitTokenRune(TK_ADDOP, input)
		tkzr.incr()
	} else if input == '(' {
		tkzr.emitTokenRune(TK_PARENL, input)
		tkzr.incr()
	} else if input == ')' {
		tkzr.emitTokenRune(TK_PARENR, input)
		tkzr.incr()
	} else {
		tkzr.incr()
	}
}

func (tkzr *Tokenizer) processLiteralInt() {
	tkzr.state = TK_LITERALINT
	for {
		chr := tkzr.getCurrRune()
		if isDigit(chr) {
			tkzr.tokbuf.WriteRune(chr)
			tkzr.incr()
		} else {
			break
		}
	}
	tkzr.endBufedToken()
}

func (tkzr *Tokenizer) getCurrRune() rune {
	return rune(tkzr.input[tkzr.pos])
}

func (tkzr *Tokenizer) processSpace() {
	// skip for now
	tkzr.incr()
}

func (tkzr *Tokenizer) processTab() {
	panic("unsupported")
}

func (tkzr *Tokenizer) processEOL() {
	fmt.Println("process EOL")

	lastTok := &tkzr.outtoks[len(tkzr.outtoks)-1]
	if lastTok.Type == TK_EOL {
		// skip token emission
		fmt.Println("skipping EOL token emission")
	} else {
		tkzr.emitTokenRune(TK_EOL, '\n')
	}
	tkzr.incrLine()
}

func (tkzr *Tokenizer) processAlphanum(input rune) {
	fmt.Println("proc char '" + string(input) + "'")
	if isAlphic(input) {
		tkzr.tokbuf.WriteRune(input)
		tkzr.incr()
	} else {
		tkzr.endBufedToken()
	}
}
func (tkzr *Tokenizer) endBufedToken() {
	tkzr.emitAndEnd(&types.Token{
		Data:           tkzr.tokbuf.String(),
		Type:           tkzr.state,
		SourceLocation: tkzr.createCurrSourceLocation(),
	})
	tkzr.tokbuf.Reset()
}

func (tkzr *Tokenizer) emitTokenRune(tokenType int, input rune) {
	tkzr.emitAndEnd(&types.Token{
		Data:           string(input),
		Type:           tokenType,
		SourceLocation: tkzr.createCurrSourceLocation(),
	})
}

func (tkzr *Tokenizer) createCurrSourceLocation() *types.SourceLocation {
	rv := &types.SourceLocation{
		Char:   tkzr.srcLoc.Char,
		Line:   tkzr.srcLoc.Line,
		Column: tkzr.srcLoc.Column,
	}
	return rv
}

func (tkzr *Tokenizer) incr() {
	tkzr.pos++
	tkzr.srcLoc.Char++
	tkzr.srcLoc.Column++
}

func (tkzr *Tokenizer) incrLine() {
	tkzr.pos++
	tkzr.srcLoc.Char++
	tkzr.srcLoc.Column = 0
	tkzr.srcLoc.Line++
}

func (tkzr *Tokenizer) emitAndEnd(token *types.Token) {
	tkzr.outtoks = append(tkzr.outtoks, *token)
	tkzr.state = TKS_INIT
}

func isAlphic(input rune) bool {
	return (input >= 'a' && input <= 'z') ||
		(input >= 'A' && input <= 'Z')
}

func isDigit(input rune) bool {
	return (input >= '0' && input <= '9')
}
