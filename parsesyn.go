package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Parses a .syntax file

const (
	PS_CTX_INIT       = 0
	PS_CTX_TOK_LOOKUP = 1
	PS_CTX_CLASS      = 2
)

const (
	PS_VAL_CTX_GENERIC = 0
	PS_VAL_CTX_LLTOK   = 1
)

type ParserSyn struct {
	ctx         int
	class       string
	tokenLookup map[string]int
	output      *Parser
}

const (
	PATTERN_TOKENREF   = 0
	PATTERN_OPERATOR   = 1
	PATTERN_PATTERNREF = 2
)

type Pattern struct {
	Type     int
	Data     string
	TokenId  int
	Operator string
	Args     []Pattern
}

func (pattern *Pattern) String() string {
	rv := pattern.Operator + "(" + pattern.Data
	if pattern.Operator == "^" {
		rv += ":" + strconv.Itoa(pattern.Args[0].TokenId)
	}
	rv += ")"
	return rv
}

func (parser *ParserSyn) parsesyn(file string) Parser {
	fmt.Println("Parsing syntax file", file)

	// currclass := "root"

	parser.output = &Parser{}

	lines := strings.Split(file, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		line = strings.Trim(line, " \t\n\r")
		if len(line) == 0 {
			continue
		}
		tokens := strings.Split(line, " ")
		for j := 0; j < len(tokens); j++ {
			tokens[j] = strings.Trim(tokens[j], " \t\n\r")
		}
		parser.handleLineTokens(tokens)

	}

	return *parser.output
}

func (parser *ParserSyn) handleLineTokens(tokens []string) {
	ftok := tokens[0]

	if ftok == "class" {
		clsname := tokens[1]
		parser.class = clsname
		parser.ctx = PS_CTX_CLASS
		fmt.Println("Curr class set to", clsname)
	} else if ftok == "token" {
		if tokens[1] == "lookup" {
			parser.ctx = PS_CTX_TOK_LOOKUP
		}
	} else {
		if parser.ctx == PS_CTX_CLASS {
			pattname := tokens[0]
			fmt.Println("pattern definition", pattname)
			pattvalue, _ := parser.parseValue(tokens, 1, PS_VAL_CTX_GENERIC)
			fmt.Println("final pattern value", pattvalue)
			if pattname == "main" && parser.class == "root" {
				parser.output.main = pattvalue
			}
			if parser.output.patternLookup == nil {
				parser.output.patternLookup = make(map[string]Pattern)
			}
			parser.output.patternLookup[pattname] = pattvalue
		} else if parser.ctx == PS_CTX_TOK_LOOKUP {
			parser.handleTokenLookup(tokens)
		}
	}
}

func (parser *ParserSyn) handleTokenLookup(tokens []string) {
	alias := tokens[0]
	def := tokens[1]
	fmt.Println("token lookup line", "alias", alias, "def", def)

	if parser.tokenLookup == nil {
		parser.tokenLookup = make(map[string]int)
	}

	defInt, err := strconv.Atoi(def)
	check(err)

	parser.tokenLookup[alias] = defInt
}

// starts parsing from a given location
// returns the parsed value and how many tokens were consumed
func (parser *ParserSyn) parseValue(tokens []string, offset int, ctx int) (Pattern, int) {
	ftok := tokens[offset]
	fmt.Println("Parsing at position", offset)

	if ftok == "^" {
		fmt.Println("operator ^ encountered")
		arg, argcons := parser.parseValue(tokens, offset+1, PS_VAL_CTX_LLTOK)
		rv := Pattern{
			Type:     PATTERN_OPERATOR,
			Operator: ftok,
			Args:     []Pattern{arg},
		}
		return rv, 1 + argcons
	} else if ftok == "[" {
		return parser.parseSequence(tokens, offset, ctx)
	} else if ftok == "{" {
		return parser.parseDisjunction(tokens, offset, ctx)
	} else {
		rv := Pattern{}
		if ctx == PS_VAL_CTX_GENERIC {
			println("pattern reference", ftok, "encountered")
			rv.Data = ftok
			rv.Type = PATTERN_PATTERNREF
		} else if ctx == PS_VAL_CTX_LLTOK {
			fmt.Println("token reference", ftok, "encountered")
			if itokid, ok := parser.tokenLookup[ftok]; ok {
				rv.TokenId = itokid
			} else {
				panic("invalid low-level token reference " + ftok)
			}
		}
		return rv, 1
	}
}

// Parses the [ ... ] construction
func (parser *ParserSyn) parseSequence(tokens []string, offset int, ctx int) (Pattern, int) {
	patterns := make([]Pattern, 0)
	cons := 0

	// skip initial "["
	offset += 1
	cons += 1

	for offset < len(tokens) {
		ctok := tokens[offset]
		if ctok == "]" {
			rv := Pattern{
				Type:     PATTERN_OPERATOR,
				Operator: "[]",
				Args:     patterns,
			}
			fmt.Println("finished parsing sequence:", len(patterns), "elements")
			return rv, cons
		} else {
			patt, pcons := parser.parseValue(tokens, offset, PS_VAL_CTX_GENERIC)
			patterns = append(patterns, patt)
			offset += pcons
			cons += pcons
		}
	}
	panic("invalid syntax in sequential operator")
}

// Parses the { ... } (disjunction) construction
func (parser *ParserSyn) parseDisjunction(tokens []string, offset int, ctx int) (Pattern, int) {
	patterns := make([]Pattern, 0)
	cons := 0

	// skip initial "{"
	offset += 1
	cons += 1

	for offset < len(tokens) {
		ctok := tokens[offset]
		if ctok == "}" {
			rv := Pattern{
				Type:     PATTERN_OPERATOR,
				Operator: "{}",
				Args:     patterns,
			}
			fmt.Println("finished parsing disjunction:", len(patterns), "elements")
			return rv, cons
		} else {
			patt, pcons := parser.parseValue(tokens, offset, PS_VAL_CTX_GENERIC)
			patterns = append(patterns, patt)
			offset += pcons
			cons += pcons
		}
	}
	panic("invalid syntax in disjunction operator")
}
