package types

import (
	"strconv"
)

type SourceLocation struct {
	Line   int
	Column int
	Char   int
}

type Token struct {
	Type int
	Data string
	*SourceLocation
}

func (token *Token) String() string {
	return "\"" + token.Data + "\"(" + strconv.Itoa(token.Type) + ")"
}
