package types

import (
	"strconv"
)

type SourceLocation struct {
	Line   int
	Column int
	Char   int
}

func (sl *SourceLocation) StringDebug() string {
	if sl == nil {
		return "(no location)"
	}
	humanLine := sl.Line + 1
	humanCol := sl.Column + 1
	return "line " + strconv.Itoa(humanLine) + " col " + strconv.Itoa(humanCol)
}

type Token struct {
	Type int
	Data string
	*SourceLocation
}

func (token *Token) String() string {
	return "\"" + token.Data + "\"(" + strconv.Itoa(token.Type) + ")"
}
