package main

import (
	"bytes"
	"fmt"
)

type Token0 struct {
	Type int
	Data string
	*SourceLocation
}

var TokenGroups = []string{
	"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
	" \t\r\n",
}

func tokenize0(input string) []Token0 {
	fmt.Println("input", input)

	var tokenBuffer bytes.Buffer
	var group int = -1

	var tokenLookup = make(map[byte]int)

	var rv = make([]Token0, 0)

	// build tokenLookup
	for i := 0; i < len(TokenGroups); i++ {
		tg := TokenGroups[i]
		for j := 0; j < len(tg); j++ {
			tokenLookup[tg[j]] = i
		}
	}

	for i := 0; i < len(input); i++ {
		var cbyte byte = input[i]
		var oldGroup = group
		if val, ok := tokenLookup[cbyte]; ok {
			group = val
		} else {
			group = -1
		}
		if group != oldGroup || group == -1 {
			token := Token0{}
			token.Data = tokenBuffer.String()
			token.Type = oldGroup
			sl := &SourceLocation{}
			sl.char = i
			token.SourceLocation = sl
			rv = append(rv, token)
			tokenBuffer.Reset()
			tokenBuffer.WriteByte(cbyte)
		} else {
			tokenBuffer.WriteByte(cbyte)
		}
	}

	return rv
}
