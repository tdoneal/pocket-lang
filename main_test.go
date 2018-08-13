package main

import (
	"fmt"
	"io/ioutil"
	"pocket-lang/generate/goback"
	"pocket-lang/parse"
	"pocket-lang/tokenize"
	"testing"

	"github.com/davecgh/go-spew/spew"
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
	fmt.Println("final parsed", spew.Sdump(parsed))

	genned := goback.Generate(parsed)
	fmt.Println("final generated", genned)
}
