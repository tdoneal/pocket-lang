package main

import (
	"fmt"
	"strings"
)

// Parses a .syntax file

type ParserSyn struct {
}

func (parser *ParserSyn) parsesyn(file string) {
	fmt.Println("Parsing syntax file", file)

	// currclass := "root"

	lines := strings.Split(file, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		line = strings.Trim(line, " \t\n\r")
		tokens := strings.Split(line, " ")
		for j := 0; j < len(tokens); j++ {
			tokens[j] = strings.Trim(tokens[j], " \t\n\r")
		}
		parser.handleTokens(tokens)
	}
}

func (parser *ParserSyn) handleTokens(tokens []string) {
	fmt.Println("tokens", tokens)
}
