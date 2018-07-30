package main

import (
	"fmt"
	"io/ioutil"
)

func main() {
	fmt.Printf("Hello, world.\n")

	dat, err := ioutil.ReadFile("./srcexample/hello.t")
	check(err)
	fmt.Print(string(dat))

	tokens := tokenize(string(dat))
	fmt.Println("tokens", tokens)
}

type ParserRules struct {
	Rules []ParserRule
}

type ParserRule struct {
	TokenName string
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
