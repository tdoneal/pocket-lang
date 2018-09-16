package pocket

import (
	"bytes"
	"fmt"
	"pocket-lang/tokenize"
	"pocket-lang/types"
)

type TokenizerPocket struct {
	isPreline   bool // whether we are in the "first whitespace" on a line
	indentLevel int
	*tokenize.Tokenizer
}

const (
	TKS_INIT         = 0
	TK_EOL           = 1
	TK_INCINDENT     = 2
	TK_DECINDENT     = 3
	TK_LITERALINT    = 10
	TK_LITERALFLOAT  = 11
	TK_LITERALSTRING = 15
	TK_ALPHANUM      = 20
	TK_COLON         = 30
	TK_DOT           = 32
	TK_DOTPIPE       = 33
	TK_EQOP          = 35
	TK_ADDOP         = 40
	TK_SUBOP         = 41
	TK_MULTOP        = 42
	TK_DIVOP         = 43
	TK_MOD           = 44
	TK_LT            = 45
	TK_LTEQ          = 46
	TK_GT            = 47
	TK_GTEQ          = 48
	TK_EQ            = 49
	TK_OR            = 50
	TK_AND           = 51
	TK_COMMENT       = 55
	TK_PARENL        = 60
	TK_PARENR        = 61
	TK_BRACKL        = 62
	TK_BRACKR        = 63
	TK_CURLYL        = 64
	TK_CURLYR        = 65
	TK_COMMA         = 66
	TK_IF            = 75
	TK_ELSE          = 76
	TK_LOOP          = 80
	TK_FOR           = 81
	TK_IN            = 82
	TK_WHILE         = 83
	TK_BREAK         = 85
	TK_PASS          = 90
	TK_RETURN        = 100
	TK_VOID          = 110
	TK_BOOL          = 120
	TK_INT           = 121
	TK_FLOAT         = 122
	TK_STRING        = 123
	TK_LIST          = 124
	TK_SET           = 125
	TK_MAP           = 126
	TK_FALSE         = 130
	TK_TRUE          = 131
	TK_CLASS         = 150
)

func Tokenize(input string) []types.Token {

	tkzr := &TokenizerPocket{
		Tokenizer: &tokenize.Tokenizer{
			Input: input,
			Pos:   0,
			State: 0,
			SrcLoc: &types.SourceLocation{
				Char:   0,
				Line:   0,
				Column: 0,
			},
			Tokbuf: &bytes.Buffer{},
		},
		isPreline:   true,
		indentLevel: 0,
	}

	for !tkzr.IsEOF() {
		tkzr.process()
	}

	tkzr.addFinalEOLIfMissing()
	tkzr.cleanUpDanglingIndents()

	return tkzr.Outtoks
}

func (tkzr *TokenizerPocket) process() {
	// fmt.Println("state " + strconv.Itoa(tkzr.state) + ". proc rune '" + string(input) + "'")
	if tkzr.isPreline {
		tkzr.processPreline()
		tkzr.isPreline = false
		return
	}
	if tkzr.State == TKS_INIT {
		tkzr.processInit()
	} else if tkzr.State == TK_COMMENT {
		tkzr.processComment()
	} else {
		panic("unknown state that i'm in")
	}
}

func (tkzr *TokenizerPocket) addFinalEOLIfMissing() {
	if len(tkzr.Outtoks) > 0 && tkzr.Outtoks[len(tkzr.Outtoks)-1].Type != TK_EOL {
		tkzr.EmitTokenNoData(TK_EOL)
	}
}

func (tkzr *TokenizerPocket) processPreline() {
	// now indents must be 4 spaces
	nspaces := 0
	lineEmpty := true
	lineComment := false
	for !tkzr.IsEOF() {
		chr := tkzr.CurrRune()
		if isSpace(chr) || chr == '\r' {
			nspaces++
			tkzr.Incr()
		} else if isEOL(chr) {
			break
		} else if isCommentStart(chr) {
			lineComment = true
			break
		} else {
			lineEmpty = false
			break
		}
	}
	if lineComment {
		tkzr.State = TK_COMMENT
		return
	}
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
		tkzr.EmitTokenNoData(TK_INCINDENT)
	} else {
		nDecIndents := (expectedSpaces - nspaces) / 4
		tkzr.deindent(nDecIndents)
	}
}

func (tkzr *TokenizerPocket) deindent(nDecIndents int) {
	for i := 0; i < nDecIndents; i++ {
		tkzr.EmitTokenNoData(TK_DECINDENT)
	}
	tkzr.indentLevel -= nDecIndents
	fmt.Println("Deindented (", nDecIndents, ") block(s)")
}

func (tkzr *TokenizerPocket) cleanUpDanglingIndents() {
	tkzr.deindent(tkzr.indentLevel)
}

func (tkzr *TokenizerPocket) processComment() {
	if isEOL(tkzr.CurrRune()) {
		tkzr.emitEOLIfNotRedundant()
		tkzr.State = TKS_INIT
		tkzr.isPreline = true
		tkzr.Incr()
		return
	}
	tkzr.Incr()
}

func (tkzr *TokenizerPocket) processInit() {
	input := tkzr.CurrRune()
	if isAlphic(input) {
		tkzr.processAlphanum()
	} else if input == ':' {
		tkzr.EmitTokenRuneAndIncr(TK_COLON)
	} else if input == '=' {
		tkzr.EmitTokenRuneAndIncr(TK_EQOP)
	} else if isDigit(input) {
		tkzr.processLiteralNumeric()
	} else if isStringDelim(input) {
		tkzr.processLiteralString()
	} else if input == ' ' {
		tkzr.processSpace()
	} else if input == '\t' {
		tkzr.processTab()
	} else if isEOL(input) {
		tkzr.processEOL()
	} else if input == '+' {
		tkzr.EmitTokenRuneAndIncr(TK_ADDOP)
	} else if input == '-' {
		tkzr.EmitTokenRuneAndIncr(TK_SUBOP)
	} else if input == '*' {
		tkzr.EmitTokenRuneAndIncr(TK_MULTOP)
	} else if input == '/' {
		tkzr.EmitTokenRuneAndIncr(TK_DIVOP)
	} else if input == '(' {
		tkzr.EmitTokenRuneAndIncr(TK_PARENL)
	} else if input == ')' {
		tkzr.EmitTokenRuneAndIncr(TK_PARENR)
	} else if input == '[' {
		tkzr.EmitTokenRuneAndIncr(TK_BRACKL)
	} else if input == ']' {
		tkzr.EmitTokenRuneAndIncr(TK_BRACKR)
	} else if input == '{' {
		tkzr.EmitTokenRuneAndIncr(TK_CURLYL)
	} else if input == '}' {
		tkzr.EmitTokenRuneAndIncr(TK_CURLYR)
	} else if input == '>' {
		tkzr.processGT()
	} else if input == '<' {
		tkzr.processLT()
	} else if input == ',' {
		tkzr.EmitTokenRuneAndIncr(TK_COMMA)
	} else if input == '|' {
		tkzr.EmitTokenRuneAndIncr(TK_OR)
	} else if input == '&' {
		tkzr.EmitTokenRuneAndIncr(TK_AND)
	} else if input == '%' {
		tkzr.EmitTokenRuneAndIncr(TK_MOD)
	} else if input == '.' {
		tkzr.processDot()
	} else if input == '#' {
		tkzr.processPound()
	} else {
		tkzr.Incr()
	}
}

func (tkzr *TokenizerPocket) processDot() {
	tkzr.process1Or2CharOp('.', '>', TK_DOT, TK_DOTPIPE)
}

func (tkzr *TokenizerPocket) process1Or2CharOp(firstRune rune, secondRune rune, tok1 int, tok2 int) {
	// skip first rune
	tkzr.Incr()
	if tkzr.IsEOF() {
		panic("unexpected EOF")
	}
	nxtRune := tkzr.CurrRune()
	if nxtRune == secondRune {
		tkzr.EmitToken(tok2, string(firstRune)+string(secondRune))
		tkzr.Incr()
	} else {
		tkzr.EmitToken(tok1, string(firstRune))
	}
}

func (tkzr *TokenizerPocket) processLT() {
	tkzr.processGTOrLT('<', TK_LT, TK_LTEQ)
}

func (tkzr *TokenizerPocket) processGT() {
	tkzr.processGTOrLT('>', TK_GT, TK_GTEQ)
}

func (tkzr *TokenizerPocket) processGTOrLT(firstRune rune, tokNoEq int, tokEq int) {
	tkzr.process1Or2CharOp(firstRune, '=', tokNoEq, tokEq)
}

func (tkzr *TokenizerPocket) processPound() {
	tkzr.State = TK_COMMENT
	tkzr.Incr()
}

func isCommentStart(chr rune) bool {
	return chr == '#'
}

func (tkzr *TokenizerPocket) processLiteralNumeric() {
	tkzr.State = TK_LITERALINT
	decPointFound := false
	for !tkzr.IsEOF() {
		chr := tkzr.CurrRune()
		if isDigit(chr) {
			tkzr.Tokbuf.WriteRune(chr)
			tkzr.Incr()
		} else if isDecimalPoint(chr) {
			if decPointFound {
				panic("too many decimal points")
			}
			decPointFound = true
			tkzr.Tokbuf.WriteRune(chr)
			tkzr.Incr()
		} else {
			break
		}
	}
	var tokType int
	if decPointFound {
		tokType = TK_LITERALFLOAT
	} else {
		tokType = TK_LITERALINT
	}
	tkzr.EndBufedToken(tokType)
	tkzr.State = TKS_INIT
}

func (tkzr *TokenizerPocket) processSpace() {
	// skip for now
	tkzr.Incr()
}

func (tkzr *TokenizerPocket) processTab() {
	panic("unsupported")
}

func (tkzr *TokenizerPocket) processEOL() {

	tkzr.emitEOLIfNotRedundant()
	tkzr.isPreline = true
	tkzr.IncrLine()
}

func (tkzr *TokenizerPocket) emitEOLIfNotRedundant() {
	if len(tkzr.Outtoks) == 0 || tkzr.Outtoks[len(tkzr.Outtoks)-1].Type == TK_EOL {
		// skip token emission
	} else {
		tkzr.EmitTokenRune(TK_EOL, '\n')
	}
}

func (tkzr *TokenizerPocket) processAlphanum() {
	tkzr.State = TK_ALPHANUM
	for !tkzr.IsEOF() {
		chr := tkzr.CurrRune()
		if isAlphic(chr) {
			tkzr.Tokbuf.WriteRune(chr)
			tkzr.Incr()
		} else {
			break
		}
	}

	keywordType := tkzr.checkKeyword(tkzr.Tokbuf.String())

	if keywordType == -1 {
		tkzr.EmitToken(TK_ALPHANUM, tkzr.Tokbuf.String())
	} else {
		tkzr.EmitToken(keywordType, tkzr.Tokbuf.String())
	}
	tkzr.Tokbuf.Reset()
	tkzr.State = TKS_INIT
}

func (tkzr *TokenizerPocket) processLiteralString() {
	// skip initial quote
	tkzr.Incr()
	terminated := false
	for !tkzr.IsEOF() {
		chr := tkzr.CurrRune()
		if isStringDelim(chr) {
			terminated = true
			tkzr.Incr()
			break
		} else {
			tkzr.Tokbuf.WriteRune(chr)
			tkzr.Incr()
		}
	}
	if !terminated {
		panic("unterminated string literal")
	}
	tkzr.EmitToken(TK_LITERALSTRING, tkzr.Tokbuf.String())
	tkzr.Tokbuf.Reset()
	tkzr.State = TKS_INIT
}

func (tkzr *TokenizerPocket) checkKeyword(word string) int {
	// returns TK_TYPE if keyword, -1 otherwise
	if word == "return" {
		return TK_RETURN
	} else if word == "void" {
		return TK_VOID
	} else if word == "bool" {
		return TK_BOOL
	} else if word == "int" {
		return TK_INT
	} else if word == "float" {
		return TK_FLOAT
	} else if word == "string" {
		return TK_STRING
	} else if word == "list" {
		return TK_LIST
	} else if word == "set" {
		return TK_SET
	} else if word == "map" {
		return TK_MAP
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
	} else if word == "true" {
		return TK_TRUE
	} else if word == "false" {
		return TK_FALSE
	} else if word == "in" {
		return TK_IN
	} else if word == "pass" {
		return TK_PASS
	} else if word == "class" {
		return TK_CLASS
	}
	return -1
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

func isStringDelim(input rune) bool {
	return input == '\''
}

func isDecimalPoint(input rune) bool {
	return input == '.'
}
