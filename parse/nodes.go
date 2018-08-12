package parse

type Statement interface {
}

type Imperative struct {
	statements []Statement
}

type VarInit struct {
	varName  string
	varValue Value
}

var _ Statement = VarInit{}

type Value interface {
}

type LiteralInt struct {
	value int
}

var _ Value = LiteralInt{}
