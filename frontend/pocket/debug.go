package pocket

import (
	"bytes"
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
	ntl[NT_IMPERATIVE] = "IMPERATIVE"
	ntl[NT_RECEIVERCALL] = "CALL"
	ntl[NT_RECEIVERCALL_CMD] = "CALLCMD"

	ntl[NT_LIT_INT] = "INT"
	ntl[NT_LIT_STRING] = "STRING"
	ntl[NT_INLINEOPSTREAM] = "OPSTREAM"
	ntl[NTR_RECEIVERCALL_NAME] = "NAME"
	ntl[NTR_RECEIVERCALL_VALUE] = "ARG"
	ntl[NT_ADDOP] = "ADD"
	ntl[NT_GTOP] = "GT"
	ntl[NT_LTOP] = "LT"
	ntl[NT_GTEQOP] = "GTEQ"
	ntl[NT_LTEQOP] = "LTEQ"
	ntl[NT_DOTOP] = "DOT"
	ntl[NTR_BINOP_LEFT] = "LEFT"
	ntl[NTR_BINOP_RIGHT] = "RIGHT"

	ntl[NT_VARINIT] = "VARINIT"
	ntl[NT_VARTABLE] = "VARTABLE"
	ntl[NT_VARDEF] = "VARDEF"
	ntl[NTR_VARDEF_NAME] = "VARNAME"
	ntl[NT_TOPLEVEL] = "TOPLEVEL"
	ntl[NT_FUNCDEF] = "FUNCDEF"
	ntl[NTR_FUNCDEF_NAME] = "NAME"
	ntl[NTR_FUNCDEF_CODE] = "BODY"
	ntl[NT_LIT_LIST] = "LIST"
	ntl[NTR_TYPE] = "TYPE"
	ntl[NT_TYPEARGED] = "TYPEARGED"
	ntl[NTR_TYPEARGED_BASE] = "BASE"
	ntl[NTR_TYPEARGED_ARG] = "ARG"
	ntl[NTR_TYPE_DECL] = "TYPEDECL"
	ntl[NT_TYPE] = "TYPE"
	ntl[NT_TYPEBASE] = "TYPEBASE"
	ntl[NT_MYPE] = "MYPE"
	ntl[NTR_MYPE_POS] = "MYPE_POS"
	ntl[NTR_MYPE_NEG] = "MYPE_NEG"
	ntl[NTR_MYPE_VALID] = "MYPE_VALID"
	ntl[NT_VARASSIGN] = "VARASSIGN"
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
	ntl[NT_VAR_GETTER] = "VARGET"
	ntl[NT_DOTOP_QUALIFIER] = "QUAL"
	ntl[NTR_VAR_GETTER_NAME] = "NAME"
	ntl[NT_PARAMETER] = "PARAM"
	ntl[NTR_FUNCDEF] = "FUNCDEF"
	ntl[NTR_FUNCDEF_INTYPE] = "IN"
	ntl[NTR_FUNCDEF_OUTTYPE] = "OUT"
	ntl[NTR_VARDEF_SCOPE] = "SCOPE"
	ntl[NT_VARDEF_SCOPE] = "VARSCOPE"
	ntl[NT_FUNCTABLE] = "FUNCTABLE"
	ntl[NTR_FUNCTABLE] = "FUNCTABLE"

	// mypearged
	ntl[MATYPE_ALL] = "MA_ALL"
	ntl[MATYPE_EMPTY] = "MA_EMPTY"
	ntl[MATYPE_SINGLE_BASE] = "MA_SINGLE_BASE"
	ntl[MATYPE_UNION] = "UNION"
	ntl[MATYPE_SINGLE_ARGED] = "MA_SINGLE_ARGED"
	ntl[MATYPER_ARG] = "ARG"
	ntl[MATYPER_BASE] = "BASE"

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

	d.initialized = true
}

func DebugPrinterNew() *DebugPrinter {
	DEBUG.ensureInitialized()
	return &DebugPrinter{}
}

func PrettyPrint(n Nod) string {
	dp := DebugPrinterNew()
	dp.PrettyPrint(n)
	return dp.String()
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

func (d *DebugPrinter) PrettyPrint(node *Node) {
	d.buf = bytes.Buffer{}
	d.alreadySeen = make(map[*Node]bool)
	d.indent = 0
	d.internalPrettyPrint(node)
}

func (d *DebugPrinter) String() string {
	return d.buf.String()
}

func (d *DebugPrinter) internalPrettyPrint(node *Node) {
	seen := false
	if _, ok := d.alreadySeen[node]; ok {
		seen = true
	}
	d.alreadySeen[node] = true
	d.PrintNodeType(node.NodeType)

	d.PrintLocalDataIfExtant(node)

	// print children
	cnt := 0
	if len(node.Out) > 0 && !seen {
		d.incIndent(1)
		d.printEOL()
		for _, edge := range node.Out {
			d.PrintNodeType(edge.EdgeType)
			d.buf.WriteString("->")
			d.internalPrettyPrint(edge.Out)
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

func (d *DebugPrinter) PrintLocalDataIfExtant(node *Node) {
	if val, ok := node.Data.(int); ok {
		d.buf.WriteString(": ")
		if node.NodeType == NT_TYPE || node.NodeType == NT_TYPEBASE {
			d.PrintType(val)
		} else {
			d.buf.WriteString(strconv.Itoa(val))
		}
	} else if val, ok := node.Data.(string); ok {
		d.buf.WriteString(": \"")
		d.buf.WriteString(val)
		d.buf.WriteString("\"")
	} else if val, ok := node.Data.(*MypeExplicit); ok {
		d.llPrintMypeObject(val)
	} else if val, ok := node.Data.(*MypeArged); ok {
		subPrinter := DebugPrinterNew()
		subPrinter.PrettyPrint(val.Node)
		d.buf.WriteString(";; ")
		d.buf.WriteString(subPrinter.buf.String())
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

func (d *DebugPrinter) PrettyPrintMypes(nods []Nod) {
	for _, ele := range nods {
		d.PrettyPrintMype(ele)
	}
}

func (d *DebugPrinter) PrintType(t int) {
	if val, ok := DEBUG.typeLookup[t]; ok {
		d.buf.WriteString(val)
	} else {
		d.buf.WriteString(strconv.Itoa(t))
	}
}

func (d *DebugPrinter) llPrintMypeObject(m Mype) {
	if me, ok := m.(*MypeExplicit); ok {
		d.buf.WriteString("{")
		for key := range me.Types {
			d.PrintType(key)
			d.buf.WriteString(", ")
		}
		d.buf.WriteString("}")
	} else {
		panic("invalid mype object")
	}
}

func (d *DebugPrinter) PrettyPrintMype(nod Nod) {
	d.PrintNodeType(nod.NodeType)
	d.PrintLocalDataIfExtant(nod)
	if nod.NodeType == NT_VAR_GETTER {
		d.buf.WriteString(" ")
		d.buf.WriteString(NodGetChild(nod, NTR_VAR_GETTER_NAME).Data.(string))
	} else if nod.NodeType == NT_VARDEF {
		d.buf.WriteString(" ")
		d.buf.WriteString(NodGetChild(nod, NTR_VARDEF_NAME).Data.(string))
	}

	if NodHasChild(nod, NTR_MYPE_POS) {
		d.buf.WriteString(" +<")
		mypeNod := NodGetChild(nod, NTR_MYPE_POS)
		d.PrintNodeType(mypeNod.NodeType)
		d.PrintLocalDataIfExtant(mypeNod)
		d.buf.WriteString(">")
	}
	if NodHasChild(nod, NTR_MYPE_NEG) {
		d.buf.WriteString(" -~<")
		mypeNod := NodGetChild(nod, NTR_MYPE_NEG)
		d.PrintNodeType(mypeNod.NodeType)
		d.PrintLocalDataIfExtant(mypeNod)
		d.buf.WriteString(">")
	}
	if NodHasChild(nod, NTR_MYPE_VALID) {
		d.buf.WriteString(" =<")
		mypeNod := NodGetChild(nod, NTR_MYPE_VALID)
		d.PrintNodeType(mypeNod.NodeType)
		d.PrintLocalDataIfExtant(mypeNod)
		d.buf.WriteString(">")
	}
	if NodHasChild(nod, NTR_TYPE) {
		d.buf.WriteString(" :: ")
		typeNod := NodGetChild(nod, NTR_TYPE)
		d.PrintNodeType(typeNod.NodeType)
		d.PrintLocalDataIfExtant(typeNod)
	}
	d.buf.WriteString("\n")
}

func PrettyPrintOp(printOp func(*DebugPrinter)) string {
	d := DebugPrinterNew()
	printOp(d)
	return d.String()
}

func PrettyPrintMype(nod Nod) string {
	return PrettyPrintOp(func(d *DebugPrinter) { d.PrettyPrintMype(nod) })
}

func PrettyPrintMypes(nods []Nod) string {
	return PrettyPrintOp(func(d *DebugPrinter) { d.PrettyPrintMypes(nods) })
}
