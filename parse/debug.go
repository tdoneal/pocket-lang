package parse

import (
	"bytes"
	"strconv"
)

type Debug struct {
	initialized    bool
	nodeTypeLookup map[int]string
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
	ntl[NT_LIT_INT] = "LIT_INT"
	ntl[NT_INLINEOPSTREAM] = "OPSTREAM"
	ntl[NTR_RECEIVERCALL_NAME] = "NAME"
	ntl[NTR_RECEIVERCALL_VALUE] = "ARG"
	ntl[NT_ADDOP] = "ADD"
	ntl[NT_VARINIT] = "VARINIT"
	d.initialized = true
}

func (n *Node) String() string {
	return n.PrettyPrint()
}

func PrettyPrint(n Nod) string {
	return ((*Node)(n)).PrettyPrint()
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

func (n *Node) PrettyPrint() string {
	dp := &DebugPrinter{}
	return dp.PrettyPrint(n)
}

func (d *DebugPrinter) PrettyPrint(node *Node) string {
	DEBUG.ensureInitialized()
	d.buf = bytes.Buffer{}
	d.alreadySeen = make(map[*Node]bool)
	d.indent = 0
	d.internalPrettyPrint(node)
	return d.buf.String()
}

func (d *DebugPrinter) internalPrettyPrint(node *Node) {
	if _, ok := d.alreadySeen[node]; ok {
		d.buf.WriteString("<SEEN>")
		return
	}
	d.printNodeType(node.NodeType)
	cnt := 0
	if len(node.Out) > 0 {
		d.incIndent(1)
		d.printEOL()
		for _, edge := range node.Out {
			d.printNodeType(edge.EdgeType)
			d.buf.WriteString("->")
			d.internalPrettyPrint(edge.Out)
			if cnt < (len(node.Out) - 1) {
				d.printEOL()
			}
			cnt++
		}
		d.incIndent(-1)
	} else {
		if val, ok := node.Data.(int); ok {
			d.buf.WriteString(": ")
			d.buf.WriteString(strconv.Itoa(val))
		} else if val, ok := node.Data.(string); ok {
			d.buf.WriteString(": \"")
			d.buf.WriteString(val)
			d.buf.WriteString("\"")
		}
	}

	d.alreadySeen[node] = true
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

func (d *DebugPrinter) printNodeType(nodeType int) {
	if val, ok := DEBUG.nodeTypeLookup[nodeType]; ok {
		d.buf.WriteString(val)
	} else if nodeType >= NTR_LIST_0 {
		ndx := nodeType - NTR_LIST_0
		d.buf.WriteString("[")
		d.buf.WriteString(strconv.Itoa(ndx))
		d.buf.WriteString("]")
	} else {
		d.buf.WriteString(strconv.Itoa(nodeType))
	}
}
