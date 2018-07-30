package main

type SourceLocation struct {
	line   int
	column int
	char   int
}

type Token struct {
	Type int
	Data string
	*SourceLocation
}
