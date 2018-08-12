package main

import "fmt"
import "pocket-lang/types"

// Parser interprets a syntax tree (Pattern) and parses an input token string based on that syntax tree
type Parser struct {
	main          Pattern
	patternLookup map[string]*Pattern
	tokenLookup   map[string]int
	tokens        []types.Token
}

type ParseResult struct {
	tokens   []types.Token
	valid    bool
	consumed int
}

// Main pattern matching function.  Returns rewritten tokens, whether valid and if valid, how many tokens consumed.
func (parser *Parser) parse(tokens []types.Token) ParseResult {
	fmt.Println("Checking validity of input string", tokens)
	parser.tokens = tokens
	return parser.parseNode(parser.main, 0)
}

func (parser *Parser) parseNode(pattern Pattern, offset int) ParseResult {
	fmt.Println("checking isvalidnode.  pattern", pattern.String(),
		"currToken", parser.tokens[offset].String())

	if pattern.Type == PATTERN_OPERATOR {
		if pattern.Operator == "^" {
			tokenId := pattern.Args[0].TokenId
			ctok := parser.tokens[offset]
			fmt.Println("pattern low-level token id", tokenId, "had", ctok.Type)
			if ctok.Type == tokenId {
				println("low-level token matched")
				return ParseResult{nil, true, 1}
			} else {
				println("low-level token mismatched")
				return ParseResult{nil, false, 0}
			}
		} else if pattern.Operator == "[]" {
			return parser.parseSequence(pattern.Args, offset)
		} else if pattern.Operator == "{}" {
			return parser.parseDisjunction(pattern.Args, offset)
		} else if pattern.Operator == "*" {
			return parser.parseManyGreedy(pattern.Args[0], offset)
		} else {
			panic("unknown operator \"" + pattern.Operator + "\"")
		}
	} else if pattern.Type == PATTERN_PATTERNREF {
		patternref := pattern.Data
		followedref := parser.patternLookup[patternref]
		println("following pattern reference '" + patternref + "'")
		return parser.parseNode(*followedref, offset)
	}

	return ParseResult{nil, false, 0}
}

func (parser *Parser) parseManyGreedy(repPattern Pattern, offset int) ParseResult {
	cons := 0
	for {
		argparse := parser.parseNode(repPattern, offset)
		if argparse.valid {
			offset += argparse.consumed
			cons += argparse.consumed
			if offset >= len(parser.tokens) {
				println("ManyGreedy: Reached EOF")
				break
			}
		} else {
			break
		}
	}
	return ParseResult{nil, true, cons}
}

func (parser *Parser) parseDisjunction(patterns []Pattern, offset int) ParseResult {
	for i := 0; i < len(patterns); i++ {
		arg := parser.parseNode(patterns[i], offset)
		if arg.valid {
			return ParseResult{nil, true, arg.consumed}
		}
	}
	return ParseResult{nil, false, 0}
}

func (parser *Parser) parseSequence(patterns []Pattern, offset int) ParseResult {
	cons := 0
	for i := 0; i < len(patterns); i++ {
		arg := parser.parseNode(patterns[i], offset)
		if !arg.valid {
			return ParseResult{nil, false, 0}
		}
		offset += arg.consumed
		cons += arg.consumed
	}
	return ParseResult{nil, true, cons}
}
