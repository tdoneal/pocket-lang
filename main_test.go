package main

import (
	"fmt"
	"pocket-lang/pktest"
	"testing"
)

func TestMain(t *testing.T) {

	inFile := "./srcexample/hello.pk"
	fmt.Println("compiling input file", inFile)
	pktest.CompileAndRunFile(inFile)
}

type MyError struct {
	msg string
}

func (e MyError) Error() string {
	return e.msg
}

func TestGenCode(t *testing.T) {

}
