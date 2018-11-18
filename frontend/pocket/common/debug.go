package common

import (
	"bytes"
	"fmt"
	. "pocket-lang/parse"
	"strconv"
)

type Debug struct {
	initialized    bool
	nodeTypeLookup map[int]string // NT_IMPERATIVE -> "IMPERATIVE", etc
	typeLookup     map[int]string // TY_INT -> "int", etc
}

type DebugPrinter struct {
	buf         bytes.Buffer
	alreadySeen map[*Node]bool
	indent      int
}

var DEBUG *Debug = &Debug{} // singleton

func (d *Debug) ensureInitialized() {
	if !d.initialized {
		d.initialize()
	}
}

func (d *Debug) initialize() {
	d.nodeTypeLookup = make(map[int]string)
	ntl := d.nodeTypeLookup
	ntl[NT_IDENTIFIER] = "IDENTIFIER"
	ntl[NT_IDENTIFIER_RVAL] = "IDENTRVAL"
	ntl[NT_IDENTIFIER_LVAL] = "IDENTLVAL"
	ntl[NT_IDENTIFIER_NOSCOPE] = "IDENTNOSC"
	ntl[NT_IDENTIFIER_RESOLVED] = "IDENTRESOLVED"
	ntl[NT_IDENTIFIER_FUNC_NOSCOPE] = "IDENTFUNCNOSC"
	ntl[NT_IDENTIFIER_TYPE_NOSCOPE] = "IDENTTYPE"
	ntl[NT_IDENTIFIER_KWARG] = "IDENTKWARG"

	ntl[NT_IMPERATIVE] = "IMPERATIVE"
	ntl[NT_RECEIVERCALL] = "CALL"
	ntl[NT_RECEIVERCALL_CMD] = "CALLCMD"
	ntl[NT_RECEIVERCALL_METHOD] = "CALLMETHOD"
	ntl[NT_KWARGS] = "KEYWORDARGS"
	ntl[NT_KWARG] = "KEYWORDARG"

	ntl[NT_LIT_INT] = "INT"
	ntl[NT_LIT_STRING] = "STRING"
	ntl[NT_INLINEOPSTREAM] = "OPSTREAM"
	ntl[NT_VALUE_MOLECULE] = "OPMOLECULE"

	ntl[NTR_RECEIVERCALL_BASE] = "BASE"
	ntl[NTR_RECEIVERCALL_ARG] = "ARG"
	ntl[NT_EMPTYARGLIST] = "ARGEMPTY"
	ntl[NT_ADDOP] = "ADD"
	ntl[NT_SUBOP] = "SUB"
	ntl[NT_MULOP] = "MUL"
	ntl[NT_DIVOP] = "DIV"
	ntl[NT_MODOP] = "MOD"
	ntl[NT_GTOP] = "GT"
	ntl[NT_LTOP] = "LT"
	ntl[NT_GTEQOP] = "GTEQ"
	ntl[NT_LTEQOP] = "LTEQ"
	ntl[NT_DOTOP] = "DOT"
	ntl[NTR_BINOP_LEFT] = "LEFT"
	ntl[NTR_BINOP_RIGHT] = "RIGHT"

	ntl[NT_REFERENCEOP] = "REF"

	ntl[NT_INCREMENTOR] = "INCREMENTOR"
	ntl[NT_INCREMENTOR_OP] = "INCREMENTOROP"
	ntl[NTR_INCREMENTOR_LVALUE] = "LVALUE"

	ntl[NT_VARINIT] = "VARINIT"
	ntl[NT_VARTABLE] = "VARTABLE"
	ntl[NTR_VARTABLE] = "VARTABLE"
	ntl[NT_VARDEF] = "VARDEF"
	ntl[NTR_VARDEF] = "VARDEF"
	ntl[NTR_VARDEF_NAME] = "VARDEFNAME"
	ntl[NT_TOPLEVEL] = "TOPLEVEL"
	ntl[NT_FUNCDEF] = "FUNCDEF"
	ntl[NTR_FUNCDEF_NAME] = "NAME"
	ntl[NTR_FUNCDEF_CODE] = "BODY"
	ntl[NT_FUNCDEF_RV_PLACEHOLDER] = "RVPLACEHLD"
	ntl[NT_PASS] = "PASS"
	ntl[NT_LIT_LIST] = "LITLIST"
	ntl[NT_LIT_SET] = "LITSET"
	ntl[NT_LIT_MAP] = "LITMAP"
	ntl[NTR_TYPE] = "TYPE"
	ntl[NT_TYPEARGED] = "TYPEARGED"
	ntl[NTR_TYPEARGED_BASE] = "BASE"
	ntl[NTR_TYPEARGED_ARG] = "ARG"
	ntl[NTR_TYPE_DECL] = "TYPEDECL"
	ntl[NT_TYPE] = "TYPE"
	ntl[NT_TYPEBASE] = "TYPEBASE"
	ntl[NT_DYPE] = "DYPE"
	ntl[NTR_MYPE_POS] = "MYPE_POS"
	ntl[NTR_MYPE_NEG] = "MYPE_NEG"
	ntl[NTR_MYPE_VALID] = "MYPE_VALID"
	ntl[NTR_TYPECOND_DEFS] = "TYCONDDEFS"
	ntl[NT_TYPECOND_DEFS] = "TYCONDDEFS"
	ntl[NT_VARASSIGN] = "VARASSIGN"
	ntl[NT_VARASSIGN_ARITH] = "VARASSIGNARITH"
	ntl[NTR_VARASSIGN_ARITHOP] = "ARITHOP"
	ntl[NTR_VAR_NAME] = "VARNAME"
	ntl[NTR_VARASSIGN_VALUE] = "ASSIGNVAL"
	ntl[NTR_KVPAIR_KEY] = "KEY"
	ntl[NTR_KVPAIR_VAL] = "VALUE"
	ntl[NT_LOOP] = "LOOP"
	ntl[NTR_LOOP_BODY] = "BODY"
	ntl[NT_FOR] = "FOR"
	ntl[NTR_FOR_BODY] = "BODY"
	ntl[NTR_FOR_ITERVAR] = "ITERVAR"
	ntl[NTR_FOR_ITEROVER] = "ITEROVER"
	ntl[NT_WHILE] = "WHILE"
	ntl[NTR_WHILE_BODY] = "BODY"
	ntl[NTR_WHILE_COND] = "COND"
	ntl[NT_IF] = "IF"
	ntl[NTR_IF_COND] = "COND"
	ntl[NTR_IF_BODY_TRUE] = "IFTRUE"
	ntl[NTR_IF_BODY_FALSE] = "ELSE"
	ntl[NT_BREAK] = "BREAK"
	ntl[NT_RETURN] = "RETURN"
	ntl[NTR_RETURN_VALUE] = "RETURNVAL"
	ntl[NT_VAR_GETTER] = "VARGET"
	ntl[NT_DOTOP_QUALIFIER] = "QUAL"
	ntl[NT_PARAMETER] = "PARAM"
	ntl[NTR_FUNCDEF] = "FUNCDEF"
	ntl[NTR_FUNCDEF_INTYPE] = "IN"
	ntl[NTR_FUNCDEF_OUTTYPE] = "OUT"
	ntl[NTR_VARDEF_SCOPE] = "SCOPE"
	ntl[NT_VARDEF_SCOPE] = "VARSCOPE"
	ntl[NT_FUNCTABLE] = "FUNCTABLE"
	ntl[NTR_FUNCTABLE] = "FUNCTABLE"
	ntl[NT_CLASSDEF] = "CLASSDEF"
	ntl[NT_CLASSDEFPARTIAL] = "CDEFPARTIAL"
	ntl[NTR_CLASSDEF_STATICZONE] = "STATICZONE"
	ntl[NTR_CLASSDEF_NAME] = "NAME"
	ntl[NT_CLASSFIELD] = "CLASSFIELD"
	ntl[NT_OBJFIELD_ACCESSOR] = "OBJFIELDACCESS"
	ntl[NTR_OBJFIELD_ACCESSOR_NAME] = "FIELDNAME"
	ntl[NT_CLASSTABLE] = "CLASSTABLE"
	ntl[NTR_CLASSTABLE] = "CLASSTABLE"
	ntl[NT_OBJINIT] = "OBJINIT"
	ntl[NT_NAMESPACE] = "NAMESPACE"
	ntl[NTR_NAMESPACE] = "NAMESPACE"
	ntl[NTR_NAMESPACE_PARENT] = "NSPARENT"
	ntl[NT_TYPECALL] = "TYPECALL"
	ntl[NT_PRAGMACLAUSE] = "PRAGMACLAUSE"
	ntl[NTR_PRAGMA_BODY] = "BODY"
	ntl[NT_MODF_STATIC] = "MODFSTATIC"
	ntl[NTR_PRAGMAPAINT] = "PRAGMAPAINT"
	ntl[NT_PRAGMAPAINT] = "PRAGMAPAINT"

	// dype
	ntl[DYPE_ALL] = "DYPE_ALL"
	ntl[DYPE_EMPTY] = "DYPE_EMPTY"
	ntl[DYPE_UNION] = "UNION"
	ntl[DYPE_XSECT] = "XSECT"

	d.typeLookup = map[int]string{}
	tl := d.typeLookup
	tl[TY_BOOL] = "bool"
	tl[TY_FLOAT] = "float"
	tl[TY_INT] = "int"
	tl[TY_STRING] = "string"
	tl[TY_DUCK] = "duck"
	tl[TY_LIST] = "list"
	tl[TY_MAP] = "map"
	tl[TY_SET] = "set"
	tl[TY_VOID] = "void"
	tl[TY_FUNC] = "func"

	d.initialized = true
}

func PrettyPrintNodes(nodes []Nod) string {
	var buf bytes.Buffer = bytes.Buffer{}
	buf.WriteString("[\n")
	for _, ele := range nodes {
		buf.WriteString(PrettyPrint(ele))
		buf.WriteString("\n")
	}
	buf.WriteString("]\n")
	return buf.String()
}

func DebugPrinterNew() *DebugPrinter {
	DEBUG.ensureInitialized()
	d := &DebugPrinter{}
	d.buf = bytes.Buffer{}
	d.alreadySeen = make(map[*Node]bool)
	d.indent = 0
	return d
}

func PrettyPrint(n Nod) string {
	d := DebugPrinterNew()
	d.internalPrettyPrint(n, -1)
	return d.String()
}

func PrettyPrintDepth(node *Node, depth int) string {
	d := DebugPrinterNew()
	d.internalPrettyPrint(node, depth)
	return d.String()
}

func (d *DebugPrinter) String() string {
	return d.buf.String()
}

func (d *DebugPrinter) internalPrettyPrint(node *Node, depth int) {
	// -1 means unlimited depth
	seen := false
	if _, ok := d.alreadySeen[node]; ok {
		seen = true
	}
	d.alreadySeen[node] = true
	d.PrintNodeType(node.NodeType)

	d.PrintLocalDataIfExtant(node)

	if depth == 0 {
		return
	}

	// update depth counter
	if depth > 0 {
		depth--
	}

	// print children
	cnt := 0
	if len(node.Out) > 0 && !seen {
		d.incIndent(1)
		d.printEOL()
		for _, edge := range node.Out {
			d.PrintNodeType(edge.EdgeType)
			d.buf.WriteString("->")
			childDepth := d.getAppropriateChildDepth(
				node.NodeType, edge.EdgeType, edge.Out.NodeType, depth)
			d.internalPrettyPrint(edge.Out, childDepth)
			if cnt < (len(node.Out) - 1) {
				d.printEOL()
			}
			cnt++
		}
		d.incIndent(-1)
	}

	if seen {
		d.buf.WriteString(" (SEEN)")
	}

}

func (d *DebugPrinter) getAppropriateChildDepth(
	parTy int, edgeTy int, childTy int, defaultDepth int) int {
	// for "general purpose viewing" compute an appropriate depth for displaying
	newDepth := defaultDepth

	if parTy == NT_NAMESPACE {
		newDepth = 0
	}

	if parTy == NT_VARTABLE || parTy == NT_FUNCTABLE || parTy == NT_CLASSTABLE {
		newDepth = 2
	}

	if (defaultDepth >= 0) && (newDepth > defaultDepth) {
		newDepth = defaultDepth
	}
	return newDepth
}

func (d *DebugPrinter) PrintVarScopeType(ty int) {
	if ty == VSCOPE_FUNCPARAM {
		d.buf.WriteString("funcparam")
	} else if ty == VSCOPE_FUNCLOCAL {
		d.buf.WriteString("local")
	} else if ty == VSCOPE_CLASSFIELD {
		d.buf.WriteString("classfield")
	} else {
		d.buf.WriteString(strconv.Itoa(ty))
	}
}

func (d *DebugPrinter) PrintLocalDataIfExtant(node *Node) {
	if val, ok := node.Data.(int); ok {
		d.buf.WriteString(": ")
		if node.NodeType == NT_TYPE || node.NodeType == NT_TYPEBASE {
			d.PrintType(val)
		} else if node.NodeType == NT_VARDEF_SCOPE {
			d.PrintVarScopeType(val)
		} else {
			d.buf.WriteString(strconv.Itoa(val))
		}
	} else if val, ok := node.Data.(string); ok {
		d.buf.WriteString(": \"")
		d.buf.WriteString(val)
		d.buf.WriteString("\"")
	} else if val, ok := node.Data.(Nod); ok && node.NodeType == NT_DYPE {
		d.buf.WriteString(": ")
		d.internalPrettyPrint(val, 3)
	} else if node.NodeType == NT_NAMESPACE {
		d.buf.WriteString(" @ ")
		d.buf.WriteString(fmt.Sprintf("%p", node))
	}
}

func (d *DebugPrinter) printEOL() {
	d.buf.WriteString("\n")
	for i := 0; i < d.indent; i++ {
		d.buf.WriteString("\t")
	}
}

func (d *DebugPrinter) incIndent(by int) {
	d.indent += by
}

func (d *DebugPrinter) PrintNodeType(nodeType int) {
	if val, ok := DEBUG.nodeTypeLookup[nodeType]; ok {
		d.buf.WriteString(val)
	} else if nodeType >= NTR_LIST_0 && nodeType < NTR_LIST_MAX {
		ndx := nodeType - NTR_LIST_0
		d.buf.WriteString("[")
		d.buf.WriteString(strconv.Itoa(ndx))
		d.buf.WriteString("]")
	} else {
		d.buf.WriteString(strconv.Itoa(nodeType))
	}
}

func (d *DebugPrinter) PrintType(t int) {
	if val, ok := DEBUG.typeLookup[t]; ok {
		d.buf.WriteString(val)
	} else {
		d.buf.WriteString(strconv.Itoa(t))
	}
}

func PrettyPrintOp(printOp func(*DebugPrinter)) string {
	d := DebugPrinterNew()
	printOp(d)
	return d.String()
}
