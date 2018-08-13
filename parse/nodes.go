package parse

type Statement interface {
}

type Imperative struct {
	Statements []Statement
}

type VarInit struct {
	VarName  string
	VarValue Value
}

var _ Statement = VarInit{}

type Value interface {
}

type LiteralInt struct {
	Value int
}

var _ Value = LiteralInt{}
