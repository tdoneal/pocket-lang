package goback

import (
	"bytes"
	"fmt"
	. "pocket-lang/frontend/pocket/common"
	. "pocket-lang/parse"
	"pocket-lang/xform"
	"strconv"
)

type Generator struct {
	input         Nod
	buf           *bytes.Buffer
	tmpVarCounter int
}

func Generate(code Nod) string {

	preparer := &Preparer{&xform.Xformer{}}
	preparer.Prepare(code)

	fmt.Println("Prepared code:\n", PrettyPrint(code))

	generator := &Generator{
		buf:   &bytes.Buffer{},
		input: code,
	}

	generator.genSourceFile(code)

	return generator.buf.String()
}

func (g *Generator) genSourceFile(input Nod) {
	g.WS("package main\n\n")

	if fmtImport := NodGetChildOrNil(input, PNTR_GOIMPORTS); fmtImport != nil {
		g.WS("import \"fmt\"\n\n")
	}

	units := NodGetChildList(input)

	for _, unit := range units {
		if unit.NodeType == NT_FUNCDEF {
			g.genFuncDef(unit)
		} else if unit.NodeType == NT_CLASSDEF {
			g.genClassDef(unit)
		} else {
			panic("unknown source unit type")
		}
		g.WS("\n")
	}
}

func (g *Generator) genClassDef(n Nod) {
	g.WS("type ")
	g.WS(NodGetChild(n, NTR_CLASSDEF_NAME).Data.(string))
	g.WS(" struct {\n")
	clsUnits := NodGetChildList(n)

	for _, unit := range clsUnits {
		if unit.NodeType == NT_CLASSFIELD {
			g.genClassField(unit)
		}
		g.WS("\n")
	}
	g.WS("}\n")

	// generate all the methods
	for _, unit := range clsUnits {
		if unit.NodeType == NT_FUNCDEF {
			g.genFuncDefInner(unit, n)
		}
		g.WS("\n")
	}
}

func (g *Generator) genClassField(n Nod) {
	pkFieldName := NodGetChild(n, NTR_VARDEF_NAME).Data.(string)
	g.WS(g.convertToGoFieldName(pkFieldName))
	g.WS(" ")
	g.genType(NodGetChild(NodGetChild(n, NTR_VARDEF), NTR_TYPE))
}

func (g *Generator) genFuncInType(n Nod) {
	if n.NodeType == NT_PARAMETER {
		g.genParameter(n)
	} else if n.NodeType == NT_LIT_LIST {
		g.genParameterList(n)
	}
}

func (g *Generator) genRvPlaceholderFuncOutType(n Nod) {
	rvType := NodGetChild(n, NTR_TYPE)
	g.genFuncOutType(rvType)
}

func (g *Generator) genFuncOutType(n Nod) {
	if n.NodeType == DYPE_EMPTY {
		// in go there is no void keyword, so we don't output anything here
		return
	}
	g.genType(n)
}

func (g *Generator) genParameterList(n Nod) {
	g.WS("args []interface{}")
}

func (g *Generator) genParameter(n Nod) {
	g.WS(NodGetChild(n, NTR_VARDEF_NAME).Data.(string))
	g.WS(" ")
	g.genType(NodGetChild(n, NTR_TYPE))
}

func (g *Generator) genFuncDefInner(n Nod, rcvrDef Nod) {
	funcNameNod := NodGetChildOrNil(n, NTR_FUNCDEF_NAME)
	g.WS("func ")

	if rcvrDef != nil {
		g.WS("(self *")
		g.WS(NodGetChild(rcvrDef, NTR_CLASSDEF_NAME).Data.(string))
		g.WS(") ")
	}

	if funcNameNod != nil {
		g.WS(funcNameNod.Data.(string))
	}
	g.WS("(")

	needsArgUnpacking := false
	if inType := NodGetChildOrNil(n, NTR_FUNCDEF_INTYPE); inType != nil {
		g.genFuncInType(inType)
		if inType.NodeType == NT_LIT_LIST {
			needsArgUnpacking = true
		}
	}

	g.WS(")")

	g.genRvPlaceholderFuncOutType(NodGetChild(n, NTR_RETURNVAL_PLACEHOLDER))

	g.WS(" {\n")

	if needsArgUnpacking {
		g.genArgUnpacking(NodGetChild(n, NTR_FUNCDEF_INTYPE))
	}

	g.genImperative(NodGetChild(n, NTR_FUNCDEF_CODE))

	g.WS("}")
}

func (g *Generator) genFuncDef(n Nod) {
	g.genFuncDefInner(n, nil)
}

func (g *Generator) genArgUnpacking(inTypeDef Nod) {
	params := NodGetChildList(inTypeDef)
	for ndx, param := range params {
		typeNod := NodGetChild(param, NTR_TYPE)

		g.WS("var ")
		g.WS(NodGetChild(param, NTR_VARDEF_NAME).Data.(string))
		g.WS(" ")
		g.genType(typeNod)
		g.WS(" = ")
		g.WS("args[")
		g.WS(strconv.Itoa(ndx))
		g.WS("].(")
		g.genType(typeNod)
		g.WS(")")
		g.WS("\n")
	}
}

func (g *Generator) getContainingFuncDefOrNil(n Nod) Nod {
	for _, inEdge := range n.In {
		parent := inEdge.In
		if parent.NodeType == NT_FUNCDEF {
			return parent
		}
	}
	return nil
}

func (g *Generator) genImperative(input Nod) {

	if containingFuncDef := g.getContainingFuncDefOrNil(input); containingFuncDef != nil {
		if varTable := NodGetChildOrNil(containingFuncDef, NTR_VARTABLE); varTable != nil {
			g.genLocalVarTable(varTable)
		}
	}

	statements := NodGetChildList(input)
	for _, stmt := range statements {
		g.genImperativeUnit(stmt)
	}

}

func (g *Generator) genLocalVarTable(n Nod) {
	varDefs := NodGetChildList(n)
	for _, varDef := range varDefs {
		vds := NodGetChild(varDef, NTR_VARDEF_SCOPE).Data.(int)
		if vds == VSCOPE_FUNCLOCAL {
			g.genVarDef(varDef)
		}
	}
}

func (g *Generator) genVarDef(n Nod) {
	varName := NodGetChild(n, NTR_VARDEF_NAME).Data.(string)
	g.WS("var ")
	g.WS(varName)
	if nType := NodGetChildOrNil(n, NTR_TYPE); nType != nil {
		g.WS(" ")
		g.genType(NodGetChild(n, NTR_TYPE))
	}
	g.WS("\n")
}

func (g *Generator) getGenTypeBase(n Nod) string {
	if n.NodeType == DYPE_ALL {
		return "interface{}"
	}
	lut := map[int]string{
		TY_BOOL:   "bool",
		TY_INT:    "int64",
		TY_FLOAT:  "float64",
		TY_NUMBER: "number",
		TY_STRING: "string",
		TY_DUCK:   "interface{}",
		TY_LIST:   "[]interface{}",
		TY_SET:    "map[interface{}]bool",
		TY_MAP:    "map[interface{}]interface{}",
	}
	if val, ok := lut[n.Data.(int)]; ok {
		return val
	} else {
		return "<basetype>"
	}
}

func (g *Generator) getGenResult(printRoutine func(subGenerator *Generator)) string {
	subg := &Generator{
		buf: &bytes.Buffer{},
	}
	printRoutine(subg)
	return subg.buf.String()
}

func (g *Generator) genType(n Nod) {
	if n.NodeType == DYPE_UNION {
		if NodHasChild(n, PNTR_TYPE_INDEXABLE) {
			g.WS("[]interface{}")
		} else {
			g.WS("interface{}")
		}
	} else if n.NodeType == NT_TYPEBASE || n.NodeType == DYPE_ALL {
		g.WS(g.getGenTypeBase(n))
	} else if n.NodeType == NT_TYPEARGED {
		panic("typearged is obsolete; use TYPECALL instead")
	} else if n.NodeType == NT_CLASSDEF {
		g.WS("*")
		g.WS(NodGetChild(n, NTR_CLASSDEF_NAME).Data.(string))
	} else if n.NodeType == NT_FUNCDEF {
		g.genTypeFuncDef(n)
	} else {
		g.WS("<type>")
	}

}

func (g *Generator) genTypeFuncDef(n Nod) {
	g.WS("func(")
	param := NodGetChild(n, NTR_FUNCDEF_INTYPE)
	if param.NodeType == NT_PARAMETER {
		g.genType(NodGetChild(param, NTR_TYPE))
	} else if param.NodeType == NT_LIT_LIST {
		panic("multi arg not supported")
	} else {
		panic("unknown parameter structure")
	}
	g.WS(")")
	// TODO: probably remove the path that relies on NTR_FUNCDEF_OUTTYPE
	if explicitOutType := NodGetChildOrNil(n, NTR_FUNCDEF_OUTTYPE); explicitOutType != nil {
		g.genFuncOutType(explicitOutType)
	} else {
		g.genFuncOutType(NodGetChild(NodGetChild(n, NTR_RETURNVAL_PLACEHOLDER), NTR_TYPE))
	}

}

func isReceiverCallType(nt int) bool {
	return nt == NT_RECEIVERCALL || nt == NT_RECEIVERCALL_CMD || nt == NT_RECEIVERCALL_METHOD
}

func (g *Generator) genImperativeUnit(n Nod) {
	if n.NodeType == NT_VARASSIGN {
		g.genVarAssign(n)
	} else if isReceiverCallType(n.NodeType) {
		g.genReceiverCall(n)
	} else if n.NodeType == NT_RETURN {
		g.genReturn(n)
	} else if n.NodeType == NT_LOOP {
		g.genLoop(n)
	} else if n.NodeType == NT_WHILE {
		g.genWhile(n)
	} else if n.NodeType == NT_IF {
		g.genIf(n)
	} else if n.NodeType == NT_BREAK {
		g.genBreak(n)
	} else if n.NodeType == NT_IMPERATIVE {
		g.genImperative(n)
	} else if n.NodeType == NT_PASS {
		g.genPass(n)
	} else if n.NodeType == PNT_DUCK_FIELD_WRITE {
		g.genDuckFieldWrite(n)
	} else if n.NodeType == PNT_DUCK_METHOD_CALL {
		g.genDuckMethodCall(n)
	} else {
		g.WS("command")
	}
	g.WS("\n")
}

func (g *Generator) genDuckMethodCall(n Nod) {
	base := NodGetChild(n, NTR_RECEIVERCALL_BASE)
	name := NodGetChild(n, NTR_RECEIVERCALL_METHOD_NAME).Data.(string)
	arg := NodGetChildOrNil(n, NTR_RECEIVERCALL_ARG)

	g.WS("P__duck_method_call(")
	g.genValue(base)
	g.WS(", ")
	g.genLiteralStringRaw(name)
	g.WS(", ")
	g.genDuckMethodCallArg(arg)
	g.WS(")")
}

func (g *Generator) genDuckMethodCallArg(n Nod) {
	if n.NodeType == NT_EMPTYARGLIST {
		g.WS("nil")
	} else {
		g.genValue(n)
	}
}

func (g *Generator) genDuckFieldWrite(n Nod) {
	obj := NodGetChild(n, PNTR_DUCK_FIELD_WRITE_OBJ)
	name := NodGetChild(n, PNTR_DUCK_FIELD_WRITE_NAME)
	val := NodGetChild(n, PNTR_DUCK_FIELD_WRITE_VAL)
	g.WS("P__duck_field_write(")
	g.genValue(obj)
	g.WS(", ")
	fieldName := name.Data.(string)
	g.genLiteralStringRaw(g.convertToGoFieldName(fieldName))
	g.WS(", ")
	g.genValue(val)
	g.WS(")")
}

func (g *Generator) genPass(n Nod) {
	g.WS("")
}

func (g *Generator) genWhile(n Nod) {
	g.WS("for ")
	g.genValue(NodGetChild(n, NTR_WHILE_COND))
	g.WS("{\n")
	g.genImperative(NodGetChild(n, NTR_WHILE_BODY))
	g.WS("}")
	g.WS("\n")
}

func (g *Generator) genBreak(n Nod) {
	g.WS("break")
}

func (g *Generator) getTempVarName() string {
	rv := "_pk_" + strconv.Itoa(g.tmpVarCounter)
	g.tmpVarCounter++
	return rv
}

func (g *Generator) genLoop(input Nod) {
	g.WS("for ")
	if loopArg := NodGetChildOrNil(input, NTR_LOOP_ARG); loopArg != nil {
		tmpVarName := g.getTempVarName()
		g.WS(" ")
		g.WS(tmpVarName)
		g.WS(" := 0; ")
		g.WS(tmpVarName)
		g.WS(" < ")
		g.genValue(loopArg)
		g.WS("; ")
		g.WS(tmpVarName)
		g.WS("++ ")
	}
	g.WS("{\n")
	g.genImperative(NodGetChild(input, NTR_LOOP_BODY))
	g.WS("}\n")
}

func (g *Generator) genIf(input Nod) {
	g.WS("if ")
	g.genValue(NodGetChild(input, NTR_IF_COND))
	g.WS("{\n")
	g.genImperative(NodGetChild(input, NTR_IF_BODY_TRUE))
	g.WS("}")
	if elseBody := NodGetChildOrNil(input, NTR_IF_BODY_FALSE); elseBody != nil {
		g.WS(" else {\n")
		g.genImperative(elseBody)
		g.WS("}")
	}
	g.WS("\n")
}

func (g *Generator) genReturn(input Nod) {
	g.WS("return")
	if NodHasChild(input, NTR_RETURN_VALUE) {
		g.WS(" (")
		g.genValue(NodGetChild(input, NTR_RETURN_VALUE))
		g.WS(")")
	}
}

func (g *Generator) genVarAssign(n Nod) {

	lvalue := NodGetChild(n, NTR_VAR_NAME)
	varDef := NodGetChildOrNil(n, NTR_VARDEF)
	g.genLValue(lvalue, varDef)

	g.WS(" = ")
	g.WS("(")
	g.genValue(NodGetChild(n, NTR_VARASSIGN_VALUE))
	g.WS(")")
}

func (g *Generator) genLValue(n Nod, varDef Nod) {
	// varDef is nil if unknown or not applicable

	// prepend "self." to simple class variables
	if varDef != nil && NodHasChild(varDef, NTR_VARDEF_SCOPE) {
		if NodGetChild(varDef, NTR_VARDEF_SCOPE).Data.(int) == VSCOPE_CLASSFIELD &&
			n.NodeType == NT_IDENTIFIER {
			g.WS("self.")
		}
	}
	if n.NodeType == NT_DOTOP {
		// TODO: remove this path, it's rather lazy
		g.genValue(NodGetChild(n, NTR_BINOP_LEFT))
		g.WS(".")
		fieldName := NodGetChild(n, NTR_BINOP_RIGHT).Data.(string)
		g.WS(g.convertToGoFieldName(fieldName))
	} else if n.NodeType == NT_OBJFIELD_ACCESSOR {
		g.genValue(NodGetChild(n, NTR_RECEIVERCALL_BASE))
		g.WS(".")
		fieldName := NodGetChild(n, NTR_OBJFIELD_ACCESSOR_NAME).Data.(string)
		g.WS(g.convertToGoFieldName(fieldName))
	} else if n.NodeType == NT_IDENTIFIER || n.NodeType == NT_IDENTIFIER_RESOLVED ||
		n.NodeType == NT_IDENTIFIER_FUNC_NOSCOPE {
		g.WS(n.Data.(string))
	} else {
		g.WS("lvalue")
	}
}

func (g *Generator) genReceiverCall(n Nod) {

	if n.NodeType == NT_RECEIVERCALL_METHOD {
		g.genReceiverCallMethod(n)
		return
	}

	base := NodGetChild(n, NTR_RECEIVERCALL_BASE)

	if rcvName, ok := base.Data.(string); ok {
		if rcvName == "$li" {
			g.genListIndexor(n)
			return
		}
	}

	g.genReceiverCallBase(base)

	arg := NodGetChildOrNil(n, NTR_RECEIVERCALL_ARG)

	g.genArg(arg)
}

func (g *Generator) genReceiverCallBase(n Nod) {
	if n.NodeType == NT_VAR_GETTER {
		g.genValue(n)
	} else {
		g.genLValue(n, nil)
	}
}

func (g *Generator) genReceiverCallMethod(n Nod) {
	base := NodGetChild(n, NTR_RECEIVERCALL_BASE)
	name := NodGetChild(n, NTR_RECEIVERCALL_METHOD_NAME).Data.(string)

	g.genValue(base)

	g.WS(".")
	g.WS(name)

	arg := NodGetChildOrNil(n, NTR_RECEIVERCALL_ARG)

	g.genArg(arg)

}

func (g *Generator) genArg(arg Nod) {
	if arg == nil || arg.NodeType == NT_EMPTYARGLIST {
		g.WS("()")
	} else {
		g.WS("(")
		g.genValue(arg)
		g.WS(")")
	}
}

func (g *Generator) genListIndexor(n Nod) {
	args := NodGetChildList(NodGetChild(n, NTR_RECEIVERCALL_ARG))
	g.genValue(args[0])
	g.WS("[")
	g.genValue(args[1])
	g.WS("]")
}

func (g *Generator) genValue(n Nod) {
	nt := n.NodeType
	if nt == NT_LIT_INT {
		g.genLiteralInt(n)
	} else if nt == NT_LIT_FLOAT {
		g.genLiteralFloat(n)
	} else if nt == NT_LIT_STRING {
		g.genLiteralString(n)
	} else if nt == NT_LIT_BOOL {
		g.genLiteralBool(n)
	} else if nt == NT_VAR_GETTER {
		g.genVarGetter(n)
	} else if nt == NT_LIT_LIST {
		g.genLiteralList(n)
	} else if nt == NT_LIT_SET {
		g.genLiteralSet(n)
	} else if nt == NT_LIT_MAP {
		g.genLiteralMap(n)
	} else if nt == NT_RECEIVERCALL {
		g.genReceiverCall(n)
	} else if nt == NT_RECEIVERCALL_METHOD {
		g.genReceiverCallMethod(n)
	} else if nt == NT_COLLECTION_INDEXOR {
		g.genCollectionIndexor(n)
	} else if nt == NT_OBJINIT {
		g.genObjInit(n)
	} else if nt == NT_DOTOP {
		g.genValueDotOp(n)
	} else if nt == NT_OBJFIELD_ACCESSOR {
		g.genObjFieldAccessor(n)
	} else if g.isBinaryInlineDuckOpType(n.NodeType) {
		g.genDuckOp(n)
	} else if isBinaryInlineOpType(n.NodeType) {
		g.genBinaryInlineOp(n)
	} else if n.NodeType == PNT_DUCK_FIELD_READ {
		g.genDuckFieldRead(n)
	} else if n.NodeType == PNT_DUCK_METHOD_CALL {
		g.genDuckMethodCall(n)
	} else if n.NodeType == NT_REFERENCEOP {
		g.genReferenceOp(n)
	} else if n.NodeType == NT_FUNCDEF {
		g.genValueFuncDef(n)
	} else {
		g.WS("value")
	}
}

func (g *Generator) genValueFuncDef(n Nod) {
	g.genFuncDef(n)
}

func (g *Generator) genReferenceOp(n Nod) {
	g.genLValue(NodGetChild(n, NTR_RECEIVERCALL_ARG), nil)
}

func (g *Generator) genObjFieldAccessor(n Nod) {
	obj := NodGetChild(n, NTR_RECEIVERCALL_BASE)
	fieldName := NodGetChild(n, NTR_OBJFIELD_ACCESSOR_NAME).Data.(string)
	g.genValue(obj)
	g.WS(".")
	g.WS(g.convertToGoFieldName(fieldName))
}

func (g *Generator) genDuckFieldRead(n Nod) {
	g.WS("P__duck_field_read(")
	g.genValue(NodGetChild(n, NTR_RECEIVERCALL_BASE))
	g.WS(", ")
	pkFieldName := NodGetChild(n, NTR_OBJFIELD_ACCESSOR_NAME).Data.(string)
	g.genLiteralStringRaw(g.convertToGoFieldName(pkFieldName))
	g.WS(")")
}

func (g *Generator) convertToGoFieldName(pkFieldName string) string {
	return "P" + pkFieldName // gotta capitalize these names so Go treats them as public
}

func (g *Generator) genCollectionIndexor(n Nod) {
	base := NodGetChild(n, NTR_RECEIVERCALL_BASE)
	arg := NodGetChild(n, NTR_RECEIVERCALL_ARG)

	g.genValue(base)
	g.WS("[")
	g.genValue(arg)
	g.WS("]")
}

func (g *Generator) genObjInit(n Nod) {
	baseNod := NodGetChild(n, NTR_RECEIVERCALL_BASE)
	// argNod := NodGetChild(n, NTR_RECEIVERCALL_ARG)

	if baseNod.NodeType == NT_CLASSDEF {
		// struct initializer
		clsName := NodGetChild(baseNod, NTR_CLASSDEF_NAME).Data.(string)
		g.WS("&")
		g.WS(clsName)
		g.genObjInitArg(NodGetChild(n, NTR_RECEIVERCALL_ARG))
	} else {
		panic("couldn't handle obj init base:" + PrettyPrint(baseNod))
	}
}

func (g *Generator) genObjInitArg(n Nod) {
	g.WS("{")
	if n.NodeType == NT_EMPTYARGLIST {
		// pass
	} else if n.NodeType == NT_LIT_LIST {
		g.genArgListInternals(n)
	} else if n.NodeType == NT_KWARGS {
		g.genObjInitKwargs(n)
	} else {
		g.genValue(n)
	}
	g.WS("}")
}

func (g *Generator) genObjInitKwargs(n Nod) {
	kwargs := NodGetChildList(n)
	for _, kwarg := range kwargs {
		fieldName := NodGetChild(kwarg, NTR_VAR_NAME).Data.(string)
		g.WS(g.convertToGoFieldName(fieldName))
		g.WS(": ")
		g.genValue(NodGetChild(kwarg, NTR_VARASSIGN_VALUE))
		g.WS(", ")
	}
}

func (g *Generator) genArgListInternals(n Nod) {
	// assumes input is a lit_list for now
	args := NodGetChildList(n)
	for _, arg := range args {
		g.genValue(arg)
		g.WS(", ")
	}
}

func (g *Generator) isBinaryInlineDuckOpType(nt int) bool {
	return nt == PNT_DUCK_BINOP
}

func (g *Generator) genValueDotOp(n Nod) {
	qualName := NodGetChild(n, NTR_BINOP_RIGHT).Data.(string)
	objNod := NodGetChild(n, NTR_BINOP_LEFT)
	if qualName == "len" {
		g.WS("int64(len(")
		g.genValue(objNod)
		g.WS("))")
	} else {
		g.WS("(")
		g.genValue(objNod)
		g.WS(")")
		g.WS(".")
		g.WS(g.convertToGoFieldName(qualName))
	}
}

func (g *Generator) genLiteralInt(n Nod) {
	g.WS("int64(")
	g.WS(strconv.Itoa(n.Data.(int)))
	g.WS(")")
}

func (g *Generator) genLiteralFloat(n Nod) {
	g.WS(strconv.FormatFloat(n.Data.(float64), 'g', -1, 64))
}

func (g *Generator) genLiteralBool(n Nod) {
	lv := n.Data.(bool)
	if lv {
		g.WS("true")
	} else {
		g.WS("false")
	}
}

func (g *Generator) genLiteralString(n Nod) {
	g.genLiteralStringRaw(n.Data.(string))
}

func (g *Generator) genLiteralStringRaw(s string) {
	g.WS("\"")
	g.WS(s)
	g.WS("\"")
}

func isBinaryInlineOpType(nType int) bool {
	return nType == NT_ADDOP || nType == NT_GTOP || nType == NT_LTOP ||
		nType == NT_GTEQOP || nType == NT_LTEQOP || nType == NT_EQOP ||
		nType == NT_SUBOP || nType == NT_MULOP || nType == NT_DIVOP ||
		nType == NT_OROP || nType == NT_ANDOP || nType == NT_MODOP

}

func (g *Generator) getBinaryInlineOpSymbol(nType int) string {
	lut := map[int]string{
		NT_ADDOP:  "+",
		NT_SUBOP:  "-",
		NT_MULOP:  "*",
		NT_DIVOP:  "/",
		NT_GTOP:   ">",
		NT_LTOP:   "<",
		NT_GTEQOP: ">=",
		NT_LTEQOP: "<=",
		NT_EQOP:   "==",
		NT_OROP:   "||",
		NT_ANDOP:  "&&",
		NT_MODOP:  "%",
	}
	return lut[nType]
}

func (g *Generator) genBinaryInlineOp(n Nod) {
	g.WS("(")
	g.genValue(NodGetChild(n, NTR_BINOP_LEFT))
	g.WS(g.getBinaryInlineOpSymbol(n.NodeType))
	g.genValue(NodGetChild(n, NTR_BINOP_RIGHT))
	g.WS(")")
}

func (g *Generator) getGenDuckOpName(nt int) string {
	lut := map[int]string{
		NT_ADDOP:  "add",
		NT_SUBOP:  "sub",
		NT_MULOP:  "mul",
		NT_DIVOP:  "div",
		NT_MODOP:  "mod",
		NT_GTOP:   "gt",
		NT_GTEQOP: "gteq",
		NT_LTOP:   "lt",
		NT_LTEQOP: "lteq",
		NT_EQOP:   "defeq",
	}
	if val, ok := lut[nt]; ok {
		return val
	} else {
		return "someop(nt " + strconv.Itoa(nt) + ")"
	}
}

func (g *Generator) getGenFullDuckOpName(nt int) string {
	return "P__duck_" + g.getGenDuckOpName(nt)
}

func (g *Generator) genDuckOp(n Nod) {
	g.WS(g.getGenFullDuckOpName(n.Data.(int)))
	g.WS("(")
	g.genValue(NodGetChild(n, NTR_BINOP_LEFT))
	g.WS(",")
	g.genValue(NodGetChild(n, NTR_BINOP_RIGHT))
	g.WS(")")
}

func (g *Generator) genLiteralSet(n Nod) {
	g.WS("map[interface{}]bool{")
	elements := NodGetChildList(n)
	for _, element := range elements {
		g.genValue(element)
		g.WS(": true")
		g.WS(", ")
	}
	g.WS("}")
}

func (g *Generator) genLiteralMap(n Nod) {
	g.WS("map[interface{}]interface{}{")
	kvpairs := NodGetChildList(n)
	for _, kvpair := range kvpairs {
		g.genMapKVPair(kvpair)
		g.WS(", ")
	}
	g.WS("}")
}

func (g *Generator) genMapKVPair(n Nod) {
	g.genValue(NodGetChild(n, NTR_KVPAIR_KEY))
	g.WS(": ")
	g.genValue(NodGetChild(n, NTR_KVPAIR_VAL))
}

func (g *Generator) isDuckType(n Nod) bool {
	return n.NodeType == DYPE_ALL || n.NodeType == DYPE_UNION
}

func (g *Generator) genLiteralList(n Nod) {
	if g.isDuckType(NodGetChild(n, NTR_TYPE)) {
		g.WS("[]interface{}")
	} else {
		g.genType(NodGetChild(n, NTR_TYPE))
	}
	g.WS("{")
	elements := NodGetChildList(n)
	for _, ele := range elements {
		g.genValue(ele)
		g.WS(", ")
	}
	g.WS("}")
}

func (g *Generator) genVarGetter(n Nod) {
	varDef := NodGetChild(n, NTR_VARDEF)
	isClassField := false
	if scope := NodGetChildOrNil(varDef, NTR_VARDEF_SCOPE); scope != nil {
		if scope.Data.(int) == VSCOPE_CLASSFIELD {
			isClassField = true
		}
	}

	varName := NodGetChild(n, NTR_VAR_NAME).Data.(string)

	if isClassField {
		varName = g.convertToGoFieldName(varName)
		g.WS("self.")
	}
	g.WS(varName)
}

func (g *Generator) WS(s string) {
	g.buf.WriteString(s)
}
