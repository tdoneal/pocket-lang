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

	inFile := "./srcexample/hello.pk"
	fmt.Println("compiling input file", inFile)
	dat, err := ioutil.ReadFile(inFile)
	if err != nil {
		panic(err)
	}
	fmt.Println("input file:")
	fmt.Println(string(dat))

	tokens := tokenize.Tokenize(string(dat))
	fmt.Println("final tokens:\n", spew.Sdump(tokens))

	parsed := parse.Parse(tokens)
	fmt.Println("final parsed:\n", parse.PrettyPrint(parsed))

	xformed := xform.Xform(parsed)
	fmt.Println("final xformed:\n", parse.PrettyPrint(xformed))

	genned := goback.Generate(parsed)
	fmt.Println("final generated:\n", genned)

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
