package main

import (
	"fmt"
	"io/ioutil"
	"pocket-lang/backend/goback"
	"pocket-lang/parse"
	"pocket-lang/tokenize"
	"pocket-lang/xform"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestMain(t *testing.T) {
	fmt.Println("running")
	dat, err := ioutil.ReadFile("./srcexample/hello.pk")
	if err != nil {
		panic(err)
	}

	tokens := tokenize.Tokenize(string(dat))
	fmt.Println("final tokens", spew.Sdump(tokens))

	parsed := parse.Parse(tokens)
	fmt.Println("final parsed", parse.PrettyPrint(parsed))

	xformed := xform.Xform(parsed)
	fmt.Println("final xformed:", parse.PrettyPrint(xformed))

	genned := goback.Generate(parsed)
	fmt.Println("final generated", genned)

	err = ioutil.WriteFile("./outcode/out.go", []byte(genned), 0644)
	if err != nil {
		panic(err)
	}

	goback.RunFile("./outcode/out.go")
}

type MyError struct {
	msg string
}

func (e MyError) Error() string {
	return e.msg
}

func TestGenCode(t *testing.T) {

}
