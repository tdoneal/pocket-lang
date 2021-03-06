package pocket

import (
	"fmt"
	. "pocket-lang/frontend/pocket/common"
	. "pocket-lang/parse"
	"pocket-lang/types"
	"strconv"

	"github.com/davecgh/go-spew/spew"
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
		func() Nod { return p.parseFuncDefTL() },
		func() Nod { return p.parseClassDef() },
	})
}

func (p *ParserPocket) parseFuncDefTL() Nod {
	funcName := p.ParseToken(TK_ALPHANUM).Data
	// TODO: modifiers parsed here
	fDef := p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseFuncDefOnelineWithEOL() },
		func() Nod { return p.parseFuncDefClassic() },
	})
	NodSetChild(fDef, NTR_FUNCDEF_NAME, NodNewData(NT_IDENTIFIER_RESOLVED, funcName))
	return fDef
}

func (p *ParserPocket) parseFuncDef() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseFuncDefOneline() },
		func() Nod { return p.parseFuncDefClassic() },
	})
}

func (p *ParserPocket) parseFuncDefOnelineWithEOL() Nod {
	rv := p.parseFuncDefOneline()
	p.ParseAtMostOne(func() Nod { p.parseEOL(); return nil })
	return rv
}

func (p *ParserPocket) parseLiteralFunc() Nod {
	return p.parseFuncDef()
}

func (p *ParserPocket) parseFuncDefOneline() Nod {
	fDef := NodNew(NT_FUNCDEF)
	p.parseFuncHeaderInto(fDef)
	p.ParseToken(TK_COLON)
	fval := p.parseValue()
	NodSetChild(fDef, NTR_FUNCDEF_CODE, fval)
	return fDef
}

func (p *ParserPocket) parseFuncHeaderInto(fDef Nod) {
	funcWord := p.ParseToken(TK_ALPHANUM).Data
	if funcWord != "func" {
		p.RaiseParseError("missing func keyword")
	}
	// parse function type declarations if extant
	// for now, if they are extant, require an explicit in type and explicit out type
	funcInputType := p.ParseAtMostOne(func() Nod { return p.parseFuncDefTypeValue() })

	if funcInputType != nil {

		NodSetChild(fDef, NTR_FUNCDEF_INTYPE, funcInputType)
		funcOutputType := p.ParseAtMostOne(func() Nod { return p.parseFuncDefTypeValue() })

		if funcOutputType != nil {
			NodSetChild(fDef, NTR_FUNCDEF_OUTTYPE, funcOutputType)
		}
	}
}

func (p *ParserPocket) parseFuncDefClassic() Nod {
	rv := NodNew(NT_FUNCDEF)
	p.parseFuncHeaderInto(rv)

	p.parseEOL()
	p.ParseToken(TK_INCINDENT)
	imp := p.parseImperative()
	p.ParseToken(TK_DECINDENT)

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
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseForIn() },
		func() Nod { return p.parseForClassic() },
	})
}

func (p *ParserPocket) parseForClassic() Nod {
	initializer := p.parseStatementBody()
	p.parseComma()
	condition := p.parseValue()
	p.parseComma()
	progressor := p.parseStatementBody()
	p.parseEOL()
	body := p.parseImperativeBlock()
	rv := NodNew(NT_FOR_CLASSIC)
	NodSetChild(rv, NTR_FOR_BODY, body)
	NodSetChild(rv, NTR_FOR_INITIALIZER, initializer)
	NodSetChild(rv, NTR_WHILE_COND, condition)
	NodSetChild(rv, NTR_FOR_PROGRESSOR, progressor)
	return rv
}

func (p *ParserPocket) parseForIn() Nod {
	iterVar := p.parseIdentifier()
	p.ParseToken(TK_IN)
	iterOverValue := p.parseValue()
	p.parseEOL()
	body := p.parseImperativeBlock()
	rv := NodNew(NT_FOR_IN)
	NodSetChild(rv, NTR_FOR_BODY, body)
	NodSetChild(rv, NTR_FOR_IN_ITERVAR, iterVar)
	NodSetChild(rv, NTR_FOR_IN_ITEROVER, iterOverValue)
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

func (p *ParserPocket) parseClassDef() Nod {
	name := p.parseIdentifier()
	p.ParseToken(TK_CLASS)
	p.parseEOL()
	rv := p.parseClassDefBlock()
	NodSetChild(rv, NTR_CLASSDEF_NAME, name)
	rv.NodeType = NT_CLASSDEF
	return rv
}

func (p *ParserPocket) parseClassDefBlock() Nod {
	p.ParseToken(TK_INCINDENT)
	units := p.parseClassDefBlockInternals()
	p.ParseToken(TK_DECINDENT)
	return units
}

func (p *ParserPocket) parseClassDefBlockInternals() Nod {
	units := p.ParseAtLeastOneGreedy(func() Nod {
		return p.parseClassDefUnit()
	})
	return NodNewChildList(NT_CLASSDEFPARTIAL, units)
}

func (p *ParserPocket) parseClassDefUnit() Nod {
	rv := p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseFuncDefTL() },
		func() Nod { return p.parseClassDefField() },
		func() Nod { return p.parsePragma(func() Nod { return p.parseClassDefBlockInternals() }) },
	})
	return rv
}

func (p *ParserPocket) parsePragma(innerUnitParser func() Nod) Nod {
	p.ParseToken(TK_PRAGMA)
	modifiers := p.ParseManyGreedy(func() Nod {
		return p.parseModifier()
	})
	rv := NodNewChildList(NT_PRAGMACLAUSE, modifiers)
	p.parseEOL()
	p.ParseToken(TK_INCINDENT)
	innerUnit := innerUnitParser()
	p.ParseToken(TK_DECINDENT)

	NodSetChild(rv, NTR_PRAGMA_BODY, innerUnit)

	return rv
}

func (p *ParserPocket) parseModifier() Nod {
	atok := p.parseTokenAlphanumeric()
	adata := atok.Data
	if adata == "static" {
		return NodNew(NT_MODF_STATIC)
	} else if adata == "config" {
		return NodNew(NT_MODF_CONFIG)
	} else if adata == "private" {
		return NodNew(NT_MODF_PRIVATE)
	} else {
		p.RaiseParseError("invalid modifier")
		return nil
	}
}

func (p *ParserPocket) parseClassDefField() Nod {
	name := p.parseIdentifier()
	optType := p.ParseAtMostOne(func() Nod { return p.parseType() })
	optDefaultValue := p.ParseAtMostOne(func() Nod { return p.parseClassDefFieldDefaultValue() })
	p.parseEOL()
	rv := NodNew(NT_CLASSFIELD)
	NodSetChild(rv, NTR_VARDEF_NAME, name)
	if optType != nil {
		NodSetChild(rv, NTR_TYPE_DECL, optType)
	}
	if optDefaultValue != nil {
		NodSetChild(rv, NTR_VARASSIGN_VALUE, optDefaultValue)
	}

	return rv
}

func (p *ParserPocket) parseClassDefFieldDefaultValue() Nod {
	p.parseColon()
	val := p.parseValue()
	return val
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
		func() Nod { return p.parseVarIncrementor() },
		func() Nod { return p.parseCommand() },
	})
}

func (p *ParserPocket) parseVarIncrementor() Nod {
	lval := p.parseLValue()
	incop := p.parseIncrementorOp()
	rv := NodNew(NT_INCREMENTOR)
	NodSetChild(rv, NTR_INCREMENTOR_LVALUE, lval)
	NodSetChild(rv, NTR_INCREMENTOR_OP, incop)
	return rv
}

func (p *ParserPocket) parseIncrementorOp() Nod {
	ptok := p.ParseTokenOnCondition(func(t *types.Token) bool {
		return t.Type == TK_PLUSPLUS || t.Type == TK_MINUSMINUS
	})
	rv := NodNew(NT_INCREMENTOR_OP)
	if ptok.Type == TK_PLUSPLUS {
		rv.Data = true
	} else {
		rv.Data = false
	}
	return rv
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
		func() Nod { return p.parseVarAssignLocalTypeDecl() },
		func() Nod { return p.parseVarAssignGeneric() },
	})
}

func isSpecialAssignToken(ty int) bool {
	return ty == TK_ADDASSIGN || ty == TK_SUBASSIGN ||
		ty == TK_MULTASSIGN || ty == TK_DIVASSIGN || ty == TK_MODASSIGN ||
		ty == TK_ORASSIGN || ty == TK_ANDASSIGN
}

func (p *ParserPocket) specialVarAssignOp(tktype int) int {
	// maps a special assignment token type (e.g. +:, -:, to the corresponding
	// node type , e.g. +, -)
	if tktype == TK_ADDASSIGN {
		return NT_ADDOP
	} else if tktype == TK_SUBASSIGN {
		return NT_SUBOP
	} else if tktype == TK_MULTASSIGN {
		return NT_MULOP
	} else if tktype == TK_DIVASSIGN {
		return NT_DIVOP
	} else if tktype == TK_MODASSIGN {
		return NT_MODOP
	} else if tktype == TK_ORASSIGN {
		return NT_OROP
	} else if tktype == TK_ANDASSIGN {
		return NT_ANDOP
	}
	panic("unknown special var assign type")
}

func (p *ParserPocket) parseVarAssignGeneric() Nod {
	lval := p.parseLValue()
	assgnTok := p.ParseTokenOnCondition(func(t *types.Token) bool {
		return t.Type == TK_COLON || isSpecialAssignToken(t.Type)
	})
	rval := p.parseValue()

	rv := NodNew(NT_VARASSIGN)
	NodSetChild(rv, NTR_VAR_NAME, lval)
	NodSetChild(rv, NTR_VARASSIGN_VALUE, rval)

	if isSpecialAssignToken(assgnTok.Type) {
		rv.NodeType = NT_VARASSIGN_ARITH
		NodSetChild(rv, NTR_VARASSIGN_ARITHOP, NodNew(p.specialVarAssignOp(assgnTok.Type)))
	}

	return rv
}

func (p *ParserPocket) parseVarAssignLocalTypeDecl() Nod {
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
		func() Nod { return p.parseValueMolecular() },
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
		func() Nod { return p.parseValueMolecular() },
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

func (p *ParserPocket) parseValueMolecular() Nod {
	prefixOps := p.ParseManyGreedy(func() Nod {
		return p.parsePrefixOp()
	})
	atom := p.parseValueAtomic()
	suffixOps := p.ParseManyGreedy(func() Nod {
		return p.parseSuffixOp()
	})

	// quick optimization: if no prefix or suffix found (common), just return the atom
	if len(prefixOps) == 0 && len(suffixOps) == 0 {
		return atom
	}
	elems := []Nod{}
	elems = append(elems, prefixOps...)
	elems = append(elems, atom)
	elems = append(elems, suffixOps...)
	rv := NodNewChildList(NT_VALUE_MOLECULE, elems)
	fmt.Println("returning molecule:", PrettyPrint(rv))
	return rv
}

func (p *ParserPocket) parsePrefixOp() Nod {
	optok := p.ParseTokenOnCondition(func(t *types.Token) bool {
		return t.Type == TK_REF
	})
	return NodNew(p.prefixOpTokenToNT(optok.Type))
}

func (p *ParserPocket) parseSuffixOp() Nod {
	p.RaiseParseError("no suffix ops")
	return nil
}

func (p *ParserPocket) parseLValue() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseLValueDotStream() },
		func() Nod { return p.parseValueParenthetical() },
		func() Nod { return p.parseLiteral() },
		func() Nod { return p.parseReceiverCall() },
		func() Nod { return p.parseIdentifier() },
	})
}

func (p *ParserPocket) parseLValueDotStream() Nod {
	baseVal := p.parseValueAtomic()
	dotPattern := []ParseFunc{
		func() Nod { p.ParseToken(TK_DOT); return NodNew(NT_DOTOP) },
		func() Nod { return p.parseIdentifier() },
	}
	dotSeq := p.ParseUnrolledSequenceGreedy(dotPattern)
	if len(dotSeq) < 2 {
		p.RaiseParseError("not a dot stream")
	}
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
		func() Nod { return p.parseValueIdentifier() },
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
		func() Nod { return p.parseLiteralFunc() },
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
		func() Nod { return p.parseType() },
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
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseLiteralKeyword() },
		// TODO: add support for scoped type identifiers (e.g Geometry.Point)
		func() Nod { return p.parseIdentifier() },
	})
}

func (p *ParserPocket) parseTypeArged() Nod {
	baseType := p.parseTypeBase()
	typeArg := p.parseTypeArg()
	rv := NodNew(NT_TYPECALL)
	NodSetChild(rv, NTR_RECEIVERCALL_BASE, baseType)
	NodSetChild(rv, NTR_RECEIVERCALL_ARG, typeArg)
	return rv
}

func (p *ParserPocket) parseTypeArg() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseTypeBase() },
		func() Nod { return p.parseConfigArgs() },
	})
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

func (p *ParserPocket) prefixOpTokenToNT(ty int) int {
	if ty == TK_REF {
		return NT_REFERENCEOP
	}
	panic("unknown prefix op type")
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

func (p *ParserPocket) parseValueIdentifier() Nod {
	return NodNewData(NT_IDENTIFIER_RVAL, p.parseTokenAlphanumeric().Data)
}

func (p *ParserPocket) parseTokenAlphanumeric() *types.Token {
	return p.ParseToken(TK_ALPHANUM)
}

func (p *ParserPocket) parseIdentifier() Nod {
	return NodNewData(NT_IDENTIFIER, p.parseTokenAlphanumeric().Data)
}

func (p *ParserPocket) parseCommand() Nod {
	fmt.Println("tryina parse command", spew.Sdump(p.CurrToken()))
	target := p.parseReceiverBase()
	fmt.Println("receiver base", PrettyPrint(target))
	rv := NodNew(NT_RECEIVERCALL_CMD)
	NodSetChild(rv, NTR_RECEIVERCALL_BASE, target)
	parenArg := p.ParseAtMostOne(func() Nod { return p.parseCommandParentheticalArg() })
	if parenArg != nil {
		NodSetChild(rv, NTR_RECEIVERCALL_ARG, parenArg)
	} else {
		args := p.ParseManyGreedy(func() Nod { return p.parseValue() })
		if len(args) > 1 {
			p.RaiseParseError("only zro and one arg cmds supported for now")
		}
		if len(args) > 0 {
			NodSetChild(rv, NTR_RECEIVERCALL_ARG, args[0])
		}
		if len(args) == 0 {
			NodSetChild(rv, NTR_RECEIVERCALL_ARG, NodNew(NT_EMPTYARGLIST))
		}
	}

	return rv
}

func (p *ParserPocket) parseReceiverBase() Nod {
	return p.ParseDisjunction([]ParseFunc{
		func() Nod { return p.parseLValueDotStream() },
		func() Nod { return p.parseValueParenthetical() },
		func() Nod { return p.parseLiteral() },
		func() Nod { return p.parseIdentifier() },
	})
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

	cfgArg := p.ParseAtMostOne(func() Nod { return p.parseConfigArgs() })

	p.parseOpenParenlikeToken()
	p.Pos--
	arg := p.parseReceiverCallParentheticalArg()
	rv := NodNew(NT_RECEIVERCALL)
	NodSetChild(rv, NTR_RECEIVERCALL_BASE, name)
	if arg != nil {
		NodSetChild(rv, NTR_RECEIVERCALL_ARG, arg)
	}
	if cfgArg != nil {
		NodSetChild(rv, NTR_RECEIVERCALL_CFG_ARG, cfgArg)
	}
	return rv
}

func (p *ParserPocket) parseConfigArgs() Nod {
	p.ParseToken(TK_TILDE)
	val := p.parseValue()
	p.ParseToken(TK_TILDE)
	return val
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
