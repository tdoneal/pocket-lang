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

type ReceiverCall struct {
	ReceiverName    *string
	ReceivedMessage Value
}

type VarGetter struct {
	VarName string
}

type ValueInlineOpStreamElement interface{}

type ValueInlineOpStream struct {
	Elements []ValueInlineOpStreamElement
}

type InlineOp interface{}

type AddOp struct {
	Args []Value
}
