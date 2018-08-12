package main

import (
	"fmt"
	"io/ioutil"
	"pocket-lang/parse"
	"pocket-lang/tokenize"
	"testing"
)

func TestTokenize(t *testing.T) {
	fmt.Println("running")
	dat, err := ioutil.ReadFile("./srcexample/hello.pk")
	if err != nil {
		panic(err)
	}

	tokens := tokenize.Tokenize(string(dat))
	fmt.Println("final tokens", tokens)

	parsed := parse.Parse(tokens)
	fmt.Println("final parsed", parsed)
}
