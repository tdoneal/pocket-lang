package main

import (
	"fmt"
	"io/ioutil"
)

func main() {
	fmt.Printf("Hello, world.\n")

	dat, err := ioutil.ReadFile("./srcexample/hello.pk")
	check(err)
	fmt.Print(string(dat))

	tokens0 := tokenize0(string(dat))
	fmt.Println("tokens0", tokens0)
	tokens1 := tokenize1(tokens0)
	fmt.Println("tokens1", tokens1)
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
