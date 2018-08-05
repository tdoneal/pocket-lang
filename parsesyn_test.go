package main

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func TestThing(t *testing.T) {
	fmt.Println("Hello world")
	dat, err := ioutil.ReadFile("./langs/basic.syntax")
	if err != nil {
		panic(err)
	}
	parser := &ParserSyn{}
	parser.parsesyn(string(dat))

	fakeTokens := []Token{
		Token{
			Data: "test",
			Type: TOK0_ALPHANUMERIC,
		},
		Token{
			Data: "me",
			Type: TOK0_ALPHANUMERIC,
		},
	}

	fmt.Println("final token lookup table", parser.tokenLookup)
	fmt.Println("final generated parser", *parser.output)

	finalIsValid, finalCons := parser.output.isValid(fakeTokens)

	fmt.Println("final isValid?", finalIsValid, "finalConsumed", finalCons)
}
