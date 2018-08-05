package main

import "fmt"

// Parser interprets a syntax tree (Pattern) and parses an input token string based on that syntax tree
type Parser struct {
	main          Pattern
	patternLookup map[string]*Pattern
	tokenLookup   map[string]int
	tokens        []Token
}

func (parser *Parser) isValid(tokens []Token) (bool, int) {
	fmt.Println("Checking validity of input string", tokens)
	parser.tokens = tokens
	return parser.isValidNode(parser.main, 0)
}

// Main pattern matching function.  Returns whether valid and if valid, how many tokens consumed.
func (parser *Parser) isValidNode(pattern Pattern, offset int) (bool, int) {
	fmt.Println("checking isvalidnode.  pattern", pattern.String(),
		"currToken", parser.tokens[offset].String())

	if pattern.Type == PATTERN_OPERATOR {
		if pattern.Operator == "^" {
			tokenId := pattern.Args[0].TokenId
			ctok := parser.tokens[offset]
			fmt.Println("pattern low-level token id", tokenId, "had", ctok.Type)
			if ctok.Type == tokenId {
				println("low-level token matched")
				return true, 1
			} else {
				println("low-level token mismatched")
				return false, 0
			}
		} else if pattern.Operator == "[]" {
			return parser.isValidSequence(pattern.Args, offset)
		} else if pattern.Operator == "{}" {
			return parser.isValidDisjunction(pattern.Args, offset)
		} else if pattern.Operator == "*" {
			return parser.isValidManyGreedy(pattern.Args[0], offset)
		} else {
			panic("unknown operator \"" + pattern.Operator + "\"")
		}
	} else if pattern.Type == PATTERN_PATTERNREF {
		patternref := pattern.Data
		followedref := parser.patternLookup[patternref]
		println("following pattern reference '" + patternref + "'")
		return parser.isValidNode(*followedref, offset)
	}

	return false, 0
}

func (parser *Parser) isValidManyGreedy(repPattern Pattern, offset int) (bool, int) {
	cons := 0
	for {
		argvalid, argcons := parser.isValidNode(repPattern, offset)
		if argvalid {
			offset += argcons
			cons += argcons
			if offset >= len(parser.tokens) {
				println("ManyGreedy: Reached EOF")
				break
			}
		} else {
			break
		}
	}
	return true, cons
}

func (parser *Parser) isValidDisjunction(patterns []Pattern, offset int) (bool, int) {
	for i := 0; i < len(patterns); i++ {
		argValid, argcons := parser.isValidNode(patterns[i], offset)
		if argValid {
			return true, argcons
		}
	}
	return false, 0
}

func (parser *Parser) isValidSequence(patterns []Pattern, offset int) (bool, int) {
	cons := 0
	for i := 0; i < len(patterns); i++ {
		argValid, argcons := parser.isValidNode(patterns[i], offset)
		if !argValid {
			return false, 0
		}
		offset += argcons
		cons += argcons
	}
	return true, cons
}
