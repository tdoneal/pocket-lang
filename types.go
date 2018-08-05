package main

import (
	"strconv"
)

type SourceLocation struct {
	line   int
	column int
	char   int
}

type Token struct {
	Type int
	Data string
	*SourceLocation
}

func (token *Token) String() string {
	return "\"" + token.Data + "\"(" + strconv.Itoa(token.Type) + ")"
}
