package main

import (
	"fmt"
	. "pocket-lang/backend/goback"
	"testing"
)

func TestDuck(t *testing.T) {
	fmt.Println("duck tested lolll")
	fmt.Println(P__duck_add(int64(1), int64(1)))
	fmt.Println(P__duck_add(1.34, 1.3))
	fmt.Println(P__duck_add(int64(1), 1.2))
	fmt.Println(P__duck_add(1.2, int64(1)))
	fmt.Println(P__duck_add("hi", " there"))
	fmt.Println(P__duck_add(int64(2), "hi"))
	fmt.Println(P__duck_add("hi", int64(2)))
	fmt.Println(P__duck_add(2.4, "hi"))
	fmt.Println(P__duck_add("hi", 2.4))
}
