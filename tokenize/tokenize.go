package tokenize

// Tokenizes an input string

import (
	"bytes"
	"fmt"
	"pocket-lang/types"
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
	TK_SUBOP      = 41
	TK_MULTOP     = 42
	TK_DIVOP      = 43
	TK_LT         = 45
	TK_LTEQ       = 46
	TK_GT         = 47
	TK_GTEQ       = 48
	TK_EQ         = 49
	TK_COMMENT    = 50
	TK_PARENL     = 60
	TK_PARENR     = 61
	TK_BRACKL     = 62
	TK_BRACKR     = 63
	TK_CURLYL     = 64
	TK_CURLYR     = 65
	TK_COMMA      = 66
	TK_IF         = 75
	TK_ELSE       = 76
	TK_LOOP       = 80
	TK_FOR        = 81
	TK_WHILE      = 82
	TK_BREAK      = 85
	TK_RETURN     = 100
	TK_VOID       = 110
	TK_INT        = 120
)

type LiteralInt struct {
	*types.Token
	value *int
}

type Tokenizer struct {
	input       string
	pos         int
	state       int
	isPreline   bool // whether we are in the "first whitespace" on a line
	indentLevel int
	srcLoc      *types.SourceLocation
	tokbuf      *bytes.Buffer
	outtoks     []types.Token
}

func Tokenize(input string) []types.Token {

	tkzr := &Tokenizer{
		input:       input,
		pos:         0,
		state:       0,
		isPreline:   true,
		indentLevel: 0,
		srcLoc: &types.SourceLocation{
			Char:   0,
			Line:   0,
			Column: 0,
		},
		tokbuf: &bytes.Buffer{},
	}

	for !tkzr.isEOF() {
		tkzr.process()
	}

	tkzr.addFinalEOLIfMissing()
	tkzr.cleanUpDanglingIndents()

	return tkzr.outtoks
}

func (tkzr *Tokenizer) process() {
	// fmt.Println("state " + strconv.Itoa(tkzr.state) + ". proc rune '" + string(input) + "'")
	if tkzr.isPreline {
		tkzr.processPreline()
		tkzr.isPreline = false
		return
	}
	if tkzr.state == TKS_INIT {
		tkzr.processInit()
	} else if tkzr.state == TK_COMMENT {
		tkzr.processComment()
	} else {
		panic("unknown state that i'm in")
	}
}

func (tkzr *Tokenizer) addFinalEOLIfMissing() {
	if len(tkzr.outtoks) > 0 && tkzr.outtoks[len(tkzr.outtoks)-1].Type != TK_EOL {
		tkzr.emitTokenNoData(TK_EOL)
	}
}

func (tkzr *Tokenizer) processPreline() {
	// now indents must be 4 spaces
	nspaces := 0
	lineEmpty := true
	for !tkzr.isEOF() {
		chr := tkzr.getCurrRune()
		if isSpace(chr) || chr == '\r' {
			nspaces++
			tkzr.incr()
		} else if isEOL(chr) {
			break
		} else {
			fmt.Println("offending character:", chr)
			lineEmpty = false
			break
		}
	}
	fmt.Println("preline line", tkzr.createCurrSourceLocation().Line, "lineEmpty", lineEmpty)
	if lineEmpty {
		return
	}
	expectedSpaces := tkzr.indentLevel * 4
	if nspaces%4 != 0 {
		panic("Invalid indent")
	}
	if nspaces-expectedSpaces > 4 {
		panic("Invalid indent")
	}
	if nspaces-expectedSpaces == 4 {
		fmt.Println("Indented block")
		tkzr.indentLevel++
		tkzr.emitTokenNoData(TK_INCINDENT)
	} else {
		nDecIndents := (expectedSpaces - nspaces) / 4
		tkzr.deindent(nDecIndents)
	}
}

func (tkzr *Tokenizer) deindent(nDecIndents int) {
	for i := 0; i < nDecIndents; i++ {
		tkzr.emitTokenNoData(TK_DECINDENT)
	}
	tkzr.indentLevel -= nDecIndents
	fmt.Println("Deindented (", nDecIndents, ") block(s)")
}

func (tkzr *Tokenizer) cleanUpDanglingIndents() {
	tkzr.deindent(tkzr.indentLevel)
}

func (tkzr *Tokenizer) processComment() {
	if isEOL(tkzr.getCurrRune()) {
		tkzr.state = TKS_INIT
		return
	}
	tkzr.incr()
}

func (tkzr *Tokenizer) processInit() {
	input := tkzr.getCurrRune()
	if isAlphic(input) {
		tkzr.processAlphanum()
	} else if input == ':' {
		tkzr.emitTokenRune(TK_COLON, input)
		tkzr.incr()
	} else if isDigit(input) {
		tkzr.processLiteralInt()
	} else if input == ' ' {
		tkzr.processSpace()
	} else if input == '\t' {
		tkzr.processTab()
	} else if isEOL(input) {
		tkzr.processEOL()
	} else if input == '+' {
		tkzr.emitTokenRuneAndIncr(TK_ADDOP)
	} else if input == '(' {
		tkzr.emitTokenRuneAndIncr(TK_PARENL)
	} else if input == ')' {
		tkzr.emitTokenRuneAndIncr(TK_PARENR)
	} else if input == '[' {
		tkzr.emitTokenRuneAndIncr(TK_BRACKL)
	} else if input == ']' {
		tkzr.emitTokenRuneAndIncr(TK_BRACKR)
	} else if input == '{' {
		tkzr.emitTokenRuneAndIncr(TK_CURLYL)
	} else if input == '}' {
		tkzr.emitTokenRuneAndIncr(TK_CURLYR)
	} else if input == '>' {
		tkzr.emitTokenRuneAndIncr(TK_GT)
	} else if input == '<' {
		tkzr.emitTokenRuneAndIncr(TK_LT)
	} else if input == ',' {
		tkzr.emitTokenRuneAndIncr(TK_COMMA)
	} else if input == '#' {
		tkzr.processPound()
	} else {
		tkzr.incr()
	}
}

func (tkzr *Tokenizer) emitTokenRuneAndIncr(tokenType int) {
	tkzr.emitTokenRune(tokenType, tkzr.getCurrRune())
	tkzr.incr()
}

func (tkzr *Tokenizer) processPound() {
	tkzr.state = TK_COMMENT
	tkzr.incr()
}

func (tkzr *Tokenizer) processLiteralInt() {
	tkzr.state = TK_LITERALINT
	for !tkzr.isEOF() {
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

	if len(tkzr.outtoks) == 0 || tkzr.outtoks[len(tkzr.outtoks)-1].Type == TK_EOL {
		// skip token emission
	} else {
		tkzr.emitTokenRune(TK_EOL, '\n')
	}
	tkzr.isPreline = true
	tkzr.incrLine()
}

func (tkzr *Tokenizer) processAlphanum() {
	tkzr.state = TK_ALPHANUM
	for !tkzr.isEOF() {
		chr := tkzr.getCurrRune()
		if isAlphic(chr) {
			tkzr.tokbuf.WriteRune(chr)
			tkzr.incr()
		} else {
			break
		}
	}

	keywordType := tkzr.checkKeyword(tkzr.tokbuf.String())

	if keywordType == -1 {
		tkzr.emitToken(TK_ALPHANUM, tkzr.tokbuf.String())
	} else {
		tkzr.emitToken(keywordType, tkzr.tokbuf.String())
	}
	tkzr.tokbuf.Reset()
	tkzr.state = TKS_INIT
}

func (tkzr *Tokenizer) checkKeyword(word string) int {
	// returns TK_TYPE if keyword, -1 otherwise
	if word == "return" {
		return TK_RETURN
	} else if word == "void" {
		return TK_VOID
	} else if word == "int" {
		return TK_INT
	} else if word == "loop" {
		return TK_LOOP
	} else if word == "for" {
		return TK_FOR
	} else if word == "while" {
		return TK_WHILE
	} else if word == "if" {
		return TK_IF
	} else if word == "else" {
		return TK_ELSE
	} else if word == "break" {
		return TK_BREAK
	}
	return -1
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

func (tkzr *Tokenizer) isEOF() bool {
	return tkzr.pos >= len(tkzr.input)
}

func (tkzr *Tokenizer) emitAndEnd(token *types.Token) {
	tkzr.outtoks = append(tkzr.outtoks, *token)
	tkzr.state = TKS_INIT
}

func (tkzr *Tokenizer) emitTokenNoData(tokenType int) {
	tkzr.emitToken(tokenType, "")
}

func (tkzr *Tokenizer) emitToken(tokenType int, data string) {
	tkzr.outtoks = append(tkzr.outtoks, types.Token{
		SourceLocation: tkzr.createCurrSourceLocation(),
		Type:           tokenType,
		Data:           data,
	})
}

func isAlphic(input rune) bool {
	return (input >= 'a' && input <= 'z') ||
		(input >= 'A' && input <= 'Z') || input == '$' || input == '_'
}

func isDigit(input rune) bool {
	return (input >= '0' && input <= '9')
}

func isEOL(input rune) bool {
	return input == '\n'
}

func isSpace(input rune) bool {
	return input == ' '
}
