package main

import (
	"pocket-lang/pktest"
	"testing"
)

func TestRunAll(t *testing.T) {
	cases := pktest.ListDirFiles("./testcases")
	for _, cse := range cases {
		pktest.RunCase(cse)
	}
}
