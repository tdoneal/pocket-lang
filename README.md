# pocket-lang
Pocket Language

Pocket is a general purpose, typesafe, compiled programming language.

The purpose of Pocket is to do for Python what Crystal does for Ruby.  In addition, I hope to integrate AI/ML features into the language to support modern problem solving workflows.

Pocket also supports metaprogramming features and novel OO constructs.


## What's in this repo
This repository contains the Pocket compiler.  It comprises a handwritten lexer, parser, ASG transformer, type inference engine, and output generator.

The compiler's approach is to use Go as an IR (by compiling Pocket code to Go, then calling the Go compiler).

## Hello, world!
A basic example of a Pocket program.
```
main func
    print('Hello, world!')
```

## A more involved example
Please note, the below algorithm is not an optimized version of mergesort in any way.  It simply serves to show the syntactic flavor of Pocket.

```
mergeSort func (m list) list
    if m.len <= 1
        return m

    left : []
    right : []
    for i : 0, i < m.len, i ++
        x : m(i)
        if i < m.len / 2
            left : left + [x]
        else
            right : right + [x]

    left : mergeSort(left)
    right : mergeSort(right)

    return merge(left, right)

merge func (left list, right list) list
    result : []
    
    lptr : 0
    rptr : 0
    while lptr < left.len & rptr < right.len
        if left(lptr) < right(rptr)
            result : result + [left(lptr)]
            lptr ++
        else
            result : result + [right(rptr)]
            rptr ++

    while lptr < left.len
        result : result + [left(lptr)]
        lptr ++

    while rptr < right.len
        result : result + [right(rptr)]
        rptr ++

    return result
```

## Development status
Pocket is an independent experimental research project currently in pre-alpha development.  The lexer, parser, transformer, and generator are somewhat stable, but the type inference engine is in need of a rewrite.  Additionally, the language has yet to be fully defined!  Please contact me if you have any good ideas!

## Running tests
See the main_test.go and case_test.go if you dare.

