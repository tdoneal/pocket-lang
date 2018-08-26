package tokenize

// Tokenizes an input string

import (
	"bytes"
	"pocket-lang/types"
)

type Tokenizer struct {
	Input   string
	Pos     int
	State   int
	SrcLoc  *types.SourceLocation
	Tokbuf  *bytes.Buffer
	Outtoks []types.Token
}

func (tkzr *Tokenizer) EmitTokenRuneAndIncr(tokenType int) {
	tkzr.EmitTokenRune(tokenType, tkzr.CurrRune())
	tkzr.Incr()
}

func (tkzr *Tokenizer) CurrRune() rune {
	return rune(tkzr.Input[tkzr.Pos])
}

func (tkzr *Tokenizer) EndBufedToken() {
	tkzr.EmitTokenObject(&types.Token{
		Data:           tkzr.Tokbuf.String(),
		Type:           tkzr.State,
		SourceLocation: tkzr.CreateCurrSourceLocation(),
	})
	tkzr.Tokbuf.Reset()
}

func (tkzr *Tokenizer) EmitTokenRune(tokenType int, input rune) {
	tkzr.EmitTokenObject(&types.Token{
		Data:           string(input),
		Type:           tokenType,
		SourceLocation: tkzr.CreateCurrSourceLocation(),
	})
}

func (tkzr *Tokenizer) CreateCurrSourceLocation() *types.SourceLocation {
	rv := &types.SourceLocation{
		Char:   tkzr.SrcLoc.Char,
		Line:   tkzr.SrcLoc.Line,
		Column: tkzr.SrcLoc.Column,
	}
	return rv
}

func (tkzr *Tokenizer) Incr() {
	tkzr.Pos++
	tkzr.SrcLoc.Char++
	tkzr.SrcLoc.Column++
}

func (tkzr *Tokenizer) IncrLine() {
	tkzr.Pos++
	tkzr.SrcLoc.Char++
	tkzr.SrcLoc.Column = 0
	tkzr.SrcLoc.Line++
}

func (tkzr *Tokenizer) IsEOF() bool {
	return tkzr.Pos >= len(tkzr.Input)
}

func (tkzr *Tokenizer) EmitTokenNoData(tokenType int) {
	tkzr.EmitToken(tokenType, "")
}

func (tkzr *Tokenizer) EmitToken(tokenType int, data string) {
	tkzr.EmitTokenObject(&types.Token{
		SourceLocation: tkzr.CreateCurrSourceLocation(),
		Type:           tokenType,
		Data:           data,
	})
}

func (tkzr *Tokenizer) EmitTokenObject(token *types.Token) {
	tkzr.Outtoks = append(tkzr.Outtoks, *token)
}
