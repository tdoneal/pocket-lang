package pocket

import (
	"fmt"
	. "pocket-lang/parse"
	"pocket-lang/types"
	"strconv"
)

type ParserPocket struct {
	*Parser
}

func Parse(tokens []types.Token) Nod {

	parser := &ParserPocket{
		&Parser{
			Input: tokens,
			Pos:   0,
		},
	}

	return parser.parseTopLevel()
}

func (p *ParserPocket) parseFuncDef() Nod {
	funcName := p.ParseToken(TK_ALPHANUM).Data
	funcWord := p.ParseToken(TK_ALPHANUM).Data
	rv := NodNew(NT_FUNCDEF)
	if funcWord != "func" {
		p.RaiseParseError("missing func keyword")
	}

	// parse function type declarations if extant
	// for now, if they are extant, require an explicit in type and explicit out type
	funcInputType := p.ParseAtMostOne(func() Nod { return p.parseFuncDefTypeValue() })

	if funcInputType != nil {
		NodSetChild(rv, NTR_FUNCDEF_INTYPE, funcInputType)
		funcOutputType := p.ParseAtMostOne(func() Nod { return p.parseFuncDefTypeValue() })
		if funcOutputType != nil {
			NodSetChild(rv, NTR_FUNCDEF_OUTTYPE, funcOutputType)
		}
	}

	p.parseEOL()
	p.ParseToken(TK_INCINDENT)
	imp := p.parseImperative()
	p.ParseToken(TK_DECINDENT)

	funcNameNode := NodNewData(NT_IDENTIFIER, funcName)
	NodSetChild(rv, NTR_FUNCDEF_NAME, funcNameNode)
	NodSetChild(rv, NTR_FUNCDEF_CODE, imp)
	return rv
}

func (p *ParserPocket) parseImperative() Nod {
	units := p.ParseAtLeastOneGreedy(func() Nod {
		return p.parseImperativeUnit()
	})
	rv := NodNewChildList(NT_IMPERATIVE, units)
	return rv
}

func (p *ParserPocket) parseImperativeUnit() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseIf() },
		func() Nod { return p.parseWhile() },
		func() Nod { return p.parseFor() },
		func() Nod { return p.parseLoop() },
		func() Nod { return p.parseBreak() },
		func() Nod { return p.parseImperativeBlock() },
		func() Nod { return p.parseStatement() },
	})
}

func (p *ParserPocket) parseFor() Nod {
	p.ParseToken(TK_FOR)
	iterVar := p.parseIdentifier()
	p.ParseToken(TK_IN)
	iterOverValue := p.parseValue()
	p.parseEOL()
	body := p.parseImperativeBlock()
	rv := NodNew(NT_FOR)
	NodSetChild(rv, NTR_FOR_BODY, body)
	NodSetChild(rv, NTR_FOR_ITERVAR, iterVar)
	NodSetChild(rv, NTR_FOR_ITEROVER, iterOverValue)
	return rv
}

func (p *ParserPocket) parseBreak() Nod {
	p.ParseToken(TK_BREAK)
	p.parseEOL()
	return NodNew(NT_BREAK)
}

func (p *ParserPocket) parseWhile() Nod {
	p.ParseToken(TK_WHILE)
	cond := p.parseValue()
	p.parseEOL()
	body := p.parseImperativeBlock()

	rv := NodNew(NT_WHILE)
	NodSetChild(rv, NTR_WHILE_COND, cond)
	NodSetChild(rv, NTR_WHILE_BODY, body)
	return rv
}

func (p *ParserPocket) parseIf() Nod {
	p.ParseToken(TK_IF)
	cond := p.parseValue()
	p.parseEOL()
	body := p.parseImperativeBlock()

	elseNod := p.ParseAtMostOne(func() Nod { return p.parseElse() })

	rv := NodNew(NT_IF)
	NodSetChild(rv, NTR_IF_COND, cond)
	NodSetChild(rv, NTR_IF_BODY_TRUE, body)
	if elseNod != nil {
		NodSetChild(rv, NTR_IF_BODY_FALSE, elseNod)
	}
	return rv
}

func (p *ParserPocket) parseElse() Nod {
	p.ParseToken(TK_ELSE)
	p.parseEOL()
	body := p.parseImperativeBlock()
	return body
}

func (p *ParserPocket) parseLoop() Nod {
	p.ParseToken(TK_LOOP)
	arg := p.ParseAtMostOne(func() Nod { return p.parseValue() })
	p.parseEOL()
	body := p.parseImperativeBlock()
	rv := NodNewChild(NT_LOOP, NTR_LOOP_BODY, body)
	if arg != nil {
		NodSetChild(rv, NTR_LOOP_ARG, arg)
	}
	return rv
}

func (p *ParserPocket) parseTopLevel() Nod {
	units := p.ParseManyGreedy(func() Nod {
		return p.parseTopLevelUnit()
	})

	fmt.Println("top level units:", PrettyPrintNodes(units))

	if !p.IsEOF() {
		p.RaiseParseError("failed to consume all input")
	}

	return NodNewChildList(NT_TOPLEVEL, units)
}

func (p *ParserPocket) parseTopLevelUnit() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseFuncDef() },
		func() Nod { return p.parseClassDef() },
	})
}

func (p *ParserPocket) parseClassDef() Nod {
	name := p.parseIdentifier()
	p.ParseToken(TK_CLASS)
	p.parseEOL()
	rv := p.parseClassDefBlock()
	NodSetChild(rv, NTR_CLASSDEF_NAME, name)
	return rv
}

func (p *ParserPocket) parseClassDefBlock() Nod {
	p.ParseToken(TK_INCINDENT)
	units := p.ParseAtLeastOneGreedy(func() Nod {
		return p.parseClassDefUnit()
	})
	p.ParseToken(TK_DECINDENT)
	return NodNewChildList(NT_CLASSDEF, units)
}

func (p *ParserPocket) parseClassDefUnit() Nod {
	rv := p.parseClassDefField()
	p.parseEOL()
	return rv
}

func (p *ParserPocket) parseClassDefField() Nod {
	name := p.parseIdentifier()
	optType := p.ParseAtMostOne(func() Nod { return p.parseType() })

	rv := NodNew(NT_CLASSFIELD)
	NodSetChild(rv, NTR_VARDEF_NAME, name)
	if optType != nil {
		NodSetChild(rv, NTR_TYPE_DECL, optType)
	}
	return rv
}

func (p *ParserPocket) parseStatement() Nod {
	rv := p.parseStatementBody()
	p.parseEOL()
	return rv
}

func (p *ParserPocket) parseStatementBody() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseReturnStatement() },
		func() Nod { return p.parsePass() },
		func() Nod { return p.parseVarAssign() },
		func() Nod { return p.parseCommand() },
	})
}

func (p *ParserPocket) parsePass() Nod {
	p.ParseToken(TK_PASS)
	return NodNew(NT_PASS)
}

func (p *ParserPocket) parseReturnStatement() Nod {
	p.ParseToken(TK_RETURN)
	rv := NodNew(NT_RETURN)
	val := p.ParseAtMostOne(func() Nod { return p.parseValue() })
	if val != nil {
		NodSetChild(rv, NTR_RETURN_VALUE, val)
	}
	return rv
}

func (p *ParserPocket) parseVarAssign() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseVarAssignLocal() },
		func() Nod { return p.parseVarAssignComplex() },
	})
}

func (p *ParserPocket) parseVarAssignComplex() Nod {
	lval := p.parseLValue()
	p.parseColon()
	rval := p.parseValue()
	rv := NodNew(NT_VARASSIGN)
	NodSetChild(rv, NTR_VAR_NAME, lval)
	NodSetChild(rv, NTR_VARASSIGN_VALUE, rval)
	return rv
}

func (p *ParserPocket) parseVarAssignLocal() Nod {
	name := p.parseVarName()
	varType := p.ParseAtMostOne(func() Nod { return p.parseType() })
	p.parseColon()
	val := p.parseValue()
	rv := NodNew(NT_VARASSIGN)
	NodSetChild(rv, NTR_VAR_NAME, name)
	NodSetChild(rv, NTR_VARASSIGN_VALUE, val)
	if varType != nil {
		NodSetChild(rv, NTR_TYPE_DECL, varType)
	}
	return rv
}

func (p *ParserPocket) parseVarName() Nod {
	return p.parseIdentifier()
}

func (p *ParserPocket) parseValue() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseValueInlineOpStream() },
		func() Nod { return p.parseValueParenthetical() },
		func() Nod { return p.parseLiteral() },
		func() Nod { return p.parseReceiverCall() },
		func() Nod { return p.parseVarGetter() },
	})
}

func (p *ParserPocket) parseValueParenthetical() Nod {
	p.ParseToken(TK_PARENL)
	innerVal := p.parseValue()
	p.ParseToken(TK_PARENR)
	return innerVal
}

func (p *ParserPocket) parseValueInlineOpStream() Nod {
	elements := p.ParseUnrolledSequenceGreedy([]ParseFunc{
		func() Nod { return p.parseValueAtomic() },
		func() Nod { return p.parseInlineOp() },
	})

	if len(elements) < 3 {
		p.RaiseParseError("invalid op stream")
	}
	if len(elements)%2 != 1 {
		p.RaiseParseError("invalid op stream")
	}

	return NodNewChildList(NT_INLINEOPSTREAM, elements)
}

func (p *ParserPocket) parseLValue() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseLValueDotStream() },
		func() Nod { return p.parseValueParenthetical() },
		func() Nod { return p.parseLiteral() },
		func() Nod { return p.parseReceiverCall() },
		func() Nod { return p.parseVarGetter() },
	})
}

func (p *ParserPocket) parseLValueDotStream() Nod {
	baseVal := p.parseValueAtomic()
	dotPattern := []ParseFunc{
		func() Nod { p.ParseToken(TK_DOT); return NodNew(NT_DOTOP) },
		func() Nod { return p.parseIdentifier() },
	}
	dotSeq := p.ParseUnrolledSequenceGreedy(dotPattern)
	streamNods := []Nod{baseVal}
	streamNods = append(streamNods, dotSeq...)
	rv := NodNew(NT_INLINEOPSTREAM)
	NodSetOutList(rv, streamNods)
	return rv
}

func (p *ParserPocket) parseValueAtomic() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseValueParenthetical() },
		func() Nod { return p.parseLiteral() },
		func() Nod { return p.parseReceiverCall() },
		func() Nod { return p.parseVarGetter() },
	})
}

func (p *ParserPocket) parseLiteral() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseLiteralKeyword() },
		func() Nod { return p.parseLiteralString() },
		func() Nod { return p.parseLiteralList() },
		func() Nod { return p.parseLiteralSet() },
		func() Nod { return p.parseLiteralMap() },
		func() Nod { return p.parseLiteralInt() },
		func() Nod { return p.parseLiteralFloat() },
	})
}

func (p *ParserPocket) parseLiteralInt() Nod {
	tok := p.ParseToken(TK_LITERALINT)
	ival, err := strconv.Atoi(tok.Data)
	if err != nil {
		p.RaiseParseError("in int literal")
	}
	return NodNewData(NT_LIT_INT, ival)
}

func (p *ParserPocket) parseLiteralFloat() Nod {
	tok := p.ParseToken(TK_LITERALFLOAT)
	ival, err := strconv.ParseFloat(tok.Data, 64)
	if err != nil {
		p.RaiseParseError("in float literal")
	}
	return NodNewData(NT_LIT_FLOAT, ival)

}

func (p *ParserPocket) parseLiteralString() Nod {
	tkn := p.ParseToken(TK_LITERALSTRING)
	return NodNewData(NT_LIT_STRING, tkn.Data)
}

func (p *ParserPocket) parseLiteralSet() Nod {
	p.ParseToken(TK_CURLYL)
	elements := p.parseManyOptDelimited(func() Nod { return p.parseValue() },
		func() Nod { return p.parseComma() })

	p.ParseToken(TK_CURLYR)
	return NodNewChildList(NT_LIT_SET, elements)
}

func (p *ParserPocket) parseLiteralMap() Nod {
	p.ParseToken(TK_CURLYL)
	kvpairs := p.parseManyOptDelimited(func() Nod { return p.parseMapKeyValuePair() },
		func() Nod { return p.parseComma() })

	p.ParseToken(TK_CURLYR)
	return NodNewChildList(NT_LIT_MAP, kvpairs)
}

func (p *ParserPocket) parseMapKeyValuePair() Nod {
	key := p.parseValue()
	p.parseColon()
	val := p.parseValue()
	rv := NodNew(NT_LIT_MAP_KVPAIR)
	NodSetChild(rv, NTR_KVPAIR_KEY, key)
	NodSetChild(rv, NTR_KVPAIR_VAL, val)
	return rv
}

func (p *ParserPocket) parseLiteralList() Nod {
	p.ParseToken(TK_BRACKL)
	elements := p.parseManyOptDelimited(
		func() Nod { return p.parseValue() },
		func() Nod { return p.parseComma() },
	)
	p.ParseToken(TK_BRACKR)
	return NodNewChildList(NT_LIT_LIST, elements)
}

func (p *ParserPocket) parseLiteralKeyword() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseLiteralBool() },
		func() Nod { return p.parseKeywordPrimitive(TK_VOID, NT_TYPEBASE, TY_VOID) },
		func() Nod { return p.parseKeywordPrimitive(TK_INT, NT_TYPEBASE, TY_INT) },
		func() Nod { return p.parseKeywordPrimitive(TK_BOOL, NT_TYPEBASE, TY_BOOL) },
		func() Nod { return p.parseKeywordPrimitive(TK_FLOAT, NT_TYPEBASE, TY_FLOAT) },
		func() Nod { return p.parseKeywordPrimitive(TK_STRING, NT_TYPEBASE, TY_STRING) },
		func() Nod { return p.parseKeywordPrimitive(TK_LIST, NT_TYPEBASE, TY_LIST) },
		func() Nod { return p.parseKeywordPrimitive(TK_SET, NT_TYPEBASE, TY_SET) },
		func() Nod { return p.parseKeywordPrimitive(TK_MAP, NT_TYPEBASE, TY_MAP) },
	})
}

func (p *ParserPocket) parseLiteralBool() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseTrue() },
		func() Nod { return p.parseFalse() },
	})
}

func (p *ParserPocket) parseTrue() Nod {
	p.ParseToken(TK_TRUE)
	return NodNewData(NT_LIT_BOOL, true)
}

func (p *ParserPocket) parseFalse() Nod {
	p.ParseToken(TK_FALSE)
	return NodNewData(NT_LIT_BOOL, false)
}

func (p *ParserPocket) parseKeywordPrimitive(tokenType int, nodeType int, data interface{}) Nod {
	p.ParseToken(tokenType)
	return NodNewData(nodeType, data)
}

func (p *ParserPocket) parseFuncDefTypeValue() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseFDTEmptyParenBracketLike() },
		func() Nod { return p.parseParameterParenthetical() },
		func() Nod { return p.parseParameterList() },
		func() Nod { return p.parseLiteralKeyword() },
	})
}

func (p *ParserPocket) parseFDTEmptyParenBracketLike() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseFDTEmptyParen(TK_PARENL, TK_PARENR) },
		func() Nod { return p.parseFDTEmptyParen(TK_CURLYL, TK_CURLYR) },
		func() Nod { return p.parseFDTEmptyParen(TK_BRACKL, TK_BRACKR) },
	})
}

func (p *ParserPocket) parseFDTEmptyParen(tok0 int, tok1 int) Nod {
	p.ParseToken(tok0)
	p.ParseToken(tok1)
	// interpret same as if the keyword "void" was there
	return NodNewData(NT_TYPEBASE, TY_VOID)
}

func (p *ParserPocket) parseType() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseTypeArged() },
		func() Nod { return p.parseTypeBase() },
	})
}

func (p *ParserPocket) parseTypeBase() Nod {
	return p.parseLiteralKeyword()
}

func (p *ParserPocket) parseTypeArged() Nod {
	baseType := p.parseTypeBase()
	// for now only support single type args
	typeArg := p.parseTypeBase()
	rv := NodNew(NT_TYPEARGED)
	NodSetChild(rv, NTR_TYPEARGED_BASE, baseType)
	NodSetChild(rv, NTR_TYPEARGED_ARG, typeArg)
	return rv
}

func (p *ParserPocket) parseParameterList() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod {
			p.ParseToken(TK_PARENL)
			rv := p.parseParameterListInner()
			p.ParseToken(TK_PARENR)
			return rv
		},
		func() Nod {
			p.ParseToken(TK_BRACKL)
			rv := p.parseParameterListInner()
			p.ParseToken(TK_BRACKR)
			return rv
		},
	})
}

func (p *ParserPocket) parseParameterListInner() Nod {
	parameters := p.parseManyOptDelimited(
		func() Nod { return p.parseParameterSingle() },
		func() Nod { return p.parseComma() },
	)
	return NodNewChildList(NT_LIT_LIST, parameters)
}

func (p *ParserPocket) parseComma() Nod {
	p.ParseToken(TK_COMMA)
	return nil
}

func (p *ParserPocket) parseManyOptDelimited(elementParser func() Nod,
	delimiter func() Nod) []Nod {
	values := make([]Nod, 0)
	for {
		value := p.ParseAtMostOne(elementParser)
		if value == nil {
			break
		}
		values = append(values, value)
		p.ParseAtMostOne(delimiter)
	}
	return values
}

func (p *ParserPocket) parseParameterSingle() Nod {
	varname := p.parseVarName()
	vartype := p.ParseAtMostOne(func() Nod { return p.parseType() })
	rv := NodNew(NT_PARAMETER)
	NodSetChild(rv, NTR_VARDEF_NAME, varname)
	if vartype != nil {
		NodSetChild(rv, NTR_TYPE_DECL, vartype)
	}
	return rv
}

func (p *ParserPocket) parseParameterParenthetical() Nod {
	p.ParseToken(TK_PARENL)
	rv := p.parseParameterSingle()
	p.ParseToken(TK_PARENR)
	return rv
}

func (p *ParserPocket) parseInlineOp() Nod {
	ctok := p.CurrToken().Type
	nt := p.inlineOpTokenToNT(ctok)
	if nt != -1 {
		p.ParseToken(ctok)
		rv := NodNew(nt)
		return rv
	}
	p.RaiseParseError("invalid inline op")
	return nil

}

func (p *ParserPocket) inlineOpTokenToNT(tokenType int) int {
	if tokenType == TK_ADDOP {
		return NT_ADDOP
	} else if tokenType == TK_MULTOP {
		return NT_MULOP
	} else if tokenType == TK_SUBOP {
		return NT_SUBOP
	} else if tokenType == TK_DIVOP {
		return NT_DIVOP
	} else if tokenType == TK_LT {
		return NT_LTOP
	} else if tokenType == TK_GT {
		return NT_GTOP
	} else if tokenType == TK_LTEQ {
		return NT_LTEQOP
	} else if tokenType == TK_GTEQ {
		return NT_GTEQOP
	} else if tokenType == TK_EQOP {
		return NT_EQOP
	} else if tokenType == TK_OR {
		return NT_OROP
	} else if tokenType == TK_AND {
		return NT_ANDOP
	} else if tokenType == TK_MOD {
		return NT_MODOP
	} else if tokenType == TK_DOT {
		return NT_DOTOP
	} else if tokenType == TK_DOTPIPE {
		return NT_DOTPIPEOP
	} else {
		return -1
	}
}

func (p *ParserPocket) parseVarGetter() Nod {
	return NodNewChild(NT_VAR_GETTER, NTR_VAR_NAME, p.parseIdentifier())
}

func (p *ParserPocket) parseTokenAlphanumeric() *types.Token {
	return p.ParseToken(TK_ALPHANUM)
}

func (p *ParserPocket) parseIdentifier() Nod {
	return NodNewData(NT_IDENTIFIER, p.parseTokenAlphanumeric().Data)
}

func (p *ParserPocket) parseCommand() Nod {
	name := p.parseReceiverName()
	rv := NodNew(NT_RECEIVERCALL_CMD)
	NodSetChild(rv, NTR_RECEIVERCALL_BASE, name)
	parenArg := p.ParseAtMostOne(func() Nod { return p.parseCommandParentheticalArg() })
	if parenArg != nil {
		if parenArg.NodeType != NT_EMPTYARGLIST {
			NodSetChild(rv, NTR_RECEIVERCALL_ARG, parenArg)
		}
	} else {
		args := p.ParseManyGreedy(func() Nod { return p.parseValue() })
		if len(args) > 1 {
			p.RaiseParseError("only zro and one arg cmds supported for now")
		}
		if len(args) > 0 {
			NodSetChild(rv, NTR_RECEIVERCALL_ARG, args[0])
		}
	}

	return rv
}

func (p *ParserPocket) parseCommandParentheticalArg() Nod {
	p.parseOpenParenlikeToken()
	p.Pos--
	return p.parseReceiverCallParentheticalArg()
}

func (p *ParserPocket) parseReceiverCall() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseReceiverCallParentheticalStyle() },
		func() Nod { return p.parseReceiverCallCommandStyle() },
	})
}

func (p *ParserPocket) parseReceiverCallCommandStyle() Nod {
	name := p.parseReceiverName()
	val := p.parseValue()
	rv := NodNew(NT_RECEIVERCALL)
	NodSetChild(rv, NTR_RECEIVERCALL_BASE, name)
	NodSetChild(rv, NTR_RECEIVERCALL_ARG, val)
	return rv
}

func (p *ParserPocket) parseReceiverCallParentheticalStyle() Nod {
	name := p.parseReceiverName()
	p.parseOpenParenlikeToken()
	p.Pos--
	arg := p.parseReceiverCallParentheticalArg()
	rv := NodNew(NT_RECEIVERCALL)
	NodSetChild(rv, NTR_RECEIVERCALL_BASE, name)
	if arg != nil {
		NodSetChild(rv, NTR_RECEIVERCALL_ARG, arg)
	}
	return rv
}

func (p *ParserPocket) parseReceiverCallParentheticalArg() Nod {
	// returns NT_EMPTYARGLIST if empty arg list, value if arg found, error if neither
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseRCPAEmpty() },
		func() Nod { return p.parseRCPAParenList() },
		func() Nod { return p.parseValueAtomic() },
	})
}

func (p *ParserPocket) parseRCPAParenList() Nod {
	p.ParseToken(TK_PARENL)
	elements := p.parseManyOptDelimited(func() Nod { return p.parseValue() },
		func() Nod { return p.parseComma() })
	if len(elements) < 2 {
		p.RaiseParseError("needed 2 more elements to be considered a list")
	}
	p.ParseToken(TK_PARENR)
	return NodNewChildList(NT_LIT_LIST, elements)
}

func (p *ParserPocket) parseRCPAEmpty() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseRCPAEmptyParen(TK_PARENL, TK_PARENR) },
		func() Nod { return p.parseRCPAEmptyParen(TK_CURLYL, TK_CURLYR) },
		func() Nod { return p.parseRCPAEmptyParen(TK_BRACKL, TK_BRACKR) },
	})
}

func (p *ParserPocket) parseRCPAEmptyParen(tok0 int, tok1 int) Nod {
	p.ParseToken(tok0)
	p.ParseToken(tok1)
	return NodNew(NT_EMPTYARGLIST)
}

func (p *ParserPocket) parseOpenParenlikeToken() *types.Token {
	tk := p.ParseTokenOnCondition(func(t *types.Token) bool {
		return t.Type == TK_PARENL || t.Type == TK_BRACKL || t.Type == TK_CURLYL
	})
	return tk
}

func (p *ParserPocket) parseReceiverName() Nod {
	return p.parseIdentifier()
}

func (p *ParserPocket) parseColon() {
	p.ParseToken(TK_COLON)
}

func (p *ParserPocket) parseEOL() {
	p.ParseToken(TK_EOL)
}

func (p *ParserPocket) parseImperativeBlock() Nod {
	p.ParseToken(TK_INCINDENT)
	imp := p.parseImperative()
	p.ParseToken(TK_DECINDENT)
	return imp
}
