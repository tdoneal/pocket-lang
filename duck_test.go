package main

import (
	"fmt"
	. "pocket-lang/backend/goback"
	"testing"
)

func TestDuck(t *testing.T) {
	tstMethodCalls()
}

type MyClass struct {
	Px int
}

func (*MyClass) Meth(a interface{}) interface{} {
	return a.(int) + 5
}

func tstMethodCalls() {
	obj := &MyClass{1}
	result := P__duck_method_call(obj, "Meth", 3)
	fmt.Println("result", result.(int))
}

func tstFieldWrites() {
	obj := &MyClass{1}
	P__duck_field_write(obj, "Px", 2)
	fmt.Println("result", obj.Px)
}

func tstFieldReads() {
	obj := &MyClass{6}
	result := P__duck_field_read(obj, "Px")
	fmt.Println("result: ", result.(int))
}

func tstBinOps() {
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
