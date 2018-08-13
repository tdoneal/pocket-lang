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
	fmt.Println("final tokens", spew.Sdump(tokens))

	parsed := parse.Parse(tokens)
	fmt.Println("final parsed", spew.Sdump(parsed))

	genned := goback.Generate(parsed)
	fmt.Println("final generated", genned)
}

type MyError struct {
	msg string
}

func (e MyError) Error() string {
	return e.msg
}
func TestGoSem(t *testing.T) {
	// mf := func() { fmt.Println("ran func") }
	mfp := func() { panic(&MyError{"huge fail"}) }
	obj, err := tryparse(mfp)
	fmt.Println("obj", obj, "err", err)
}

func tryparse(parseFunc func()) (obj interface{}, e error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("safely caught", r.(error))
			e = r.(error)
		}
	}()
	e = nil
	parseFunc()
	return
}
