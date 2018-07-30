package main

import (
	"fmt"
)

const (
	Token1Iden        = 0
	Token1Punc        = 1
	Token1StartIndent = 2
	Token1EndIndent   = 3
	Token1EOL         = 4

	T1ST_INIT = 0
)

type Token1 struct {
	Data string
	Type int
	*SourceLocation
}

func tokenize1(input []Token0) []Token1 {
	fmt.Println("Tokenize 1", input)

	// indentlevel := 0
	// state := T1ST_INIT

	for i := 0; i < len(input); i++ {
		intok := input[i]
		fmt.Println("data", intok.Data)
		if intok.Type == 0 { // alphanumeric literal

		}
	}

	return nil
}
