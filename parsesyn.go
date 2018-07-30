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

type Parser struct {
	main        Pattern
	tokenLookup map[string]int
}

const (
	STOK_TOKENREF = 0
	STOK_OPERATOR = 1
)

type Pattern struct {
	Type     int
	Data     string
	TokenId  int
	Operator string
	Args     []Pattern
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
		parser.handleTokens(tokens)

	}

	return *parser.output
}

func (parser *ParserSyn) handleTokens(tokens []string) {
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
			tokname := tokens[0]
			fmt.Println("token definition", tokname)
			tokvalue, _ := parser.parseValue(tokens, 1, PS_VAL_CTX_GENERIC)
			fmt.Println("final token value", tokvalue)
			if tokname == "main" && parser.class == "root" {
				parser.output.main = tokvalue
			}
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
func (parser *ParserSyn) parseValue(tokens []string, offset int32, ctx int32) (Pattern, int) {
	ftok := tokens[offset]
	fmt.Println("Parsing at position", offset)
	if ftok == "^" {
		fmt.Println("operator ^ encountered")
		arg, argcons := parser.parseValue(tokens, offset+1, PS_VAL_CTX_LLTOK)
		rv := Pattern{
			Type:     STOK_OPERATOR,
			Operator: ftok,
			Args:     []Pattern{arg},
		}
		return rv, 1 + argcons
	} else {
		fmt.Println("token reference", ftok, "encountered")
		rv := Pattern{
			Type: STOK_TOKENREF,
			Data: ftok,
		}
		if ctx == PS_VAL_CTX_LLTOK {
			if itokid, ok := parser.tokenLookup[ftok]; ok {
				rv.TokenId = itokid
			} else {
				panic("invalid token" + ftok)
			}
		}
		return rv, 1
	}
}

func (parser *Parser) isValid(tokens []Token) bool {
	fmt.Println("Checking validity of input string", tokens)
	return parser.isValidNode(parser.main, tokens, 0)
}

func (parser *Parser) isValidNode(pattern Pattern, tokens []Token, offset int32) bool {
	if pattern.Operator == "^" {
		tokenId := pattern.Args[0].TokenId
		fmt.Println("token id", tokenId)
		ctok := tokens[offset]
		if ctok.Type == tokenId {
			return true
		}
	}
	return false
}
