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
}
