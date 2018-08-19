package main

import (
	"fmt"
	"io/ioutil"
	"pocket-lang/generate/goback"
	"pocket-lang/parse"
	"pocket-lang/tokenize"
	"pocket-lang/xform"
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
	fmt.Println("final tokens", spew.Sdump(tokens))

	parsed := parse.Parse(tokens)
	fmt.Println("final parsed", spew.Sdump(parsed))

	xformed := xform.Xform(parsed)
	fmt.Println("final xformed:", spew.Sdump(xformed))

	genned := goback.Generate(parsed)
	fmt.Println("final generated", genned)
}

type MyError struct {
	msg string
}

func (e MyError) Error() string {
	return e.msg
}
