package ast

import "github.com/sirbuffalo/datacraft/token"

type ScopeID uint32

type Node interface {
	Position() token.Position
}

type Program struct {
	Version   int
	Namespace string
	ScopeID   ScopeID
	Imports   []*Import
	Globals   []*Assignment
	Functions []*Function
}

type Import struct {
	Pos       token.Position
	Namespace string
	Names     []string
}

func (i *Import) Position() token.Position { return i.Pos }

type TypeRef struct {
	Pos      token.Position
	Name     string
	Element  *TypeRef
	Nullable bool
	Readonly bool
}

func (t *TypeRef) String() string {
	if t == nil {
		return ""
	}
	name := t.Name
	if t.Element != nil {
		name += "[" + t.Element.String() + "]"
	}
	if t.Nullable {
		name += "?"
	}
	if t.Readonly {
		name = "readonly " + name
	}
	return name
}

type Function struct {
	Pos            token.Position
	ScopeID        ScopeID
	Exposed        bool
	Name           string
	Parameters     []string
	ParameterTypes map[string]string
	Types          map[string]*TypeRef
	ReturnType     *TypeRef
	Body           []Statement
}

func (f *Function) Position() token.Position { return f.Pos }

type Statement interface {
	Node
	statement()
}

type Assignment struct {
	Pos          token.Position
	Name         string
	Index        Expression
	Indices      []Expression
	Operator     token.Kind
	Value        Expression
	DeclaredType *TypeRef
	Constant     bool
}

func (*Assignment) statement()                 {}
func (s *Assignment) Position() token.Position { return s.Pos }

type Return struct {
	Pos   token.Position
	Value Expression
}

func (*Return) statement()                 {}
func (s *Return) Position() token.Position { return s.Pos }

// Global declares that assignments to Names use the program's global scope.
type Global struct {
	Pos   token.Position
	Names []string
	Types map[string]string
}

func (*Global) statement()                 {}
func (s *Global) Position() token.Position { return s.Pos }

type ExpressionStatement struct {
	Pos        token.Position
	Expression Expression
}

func (*ExpressionStatement) statement()                 {}
func (s *ExpressionStatement) Position() token.Position { return s.Pos }

type Command struct {
	Pos  token.Position
	Text string
}

func (*Command) statement()                 {}
func (s *Command) Position() token.Position { return s.Pos }

type Break struct{ Pos token.Position }

func (*Break) statement()                 {}
func (s *Break) Position() token.Position { return s.Pos }

type Continue struct{ Pos token.Position }

func (*Continue) statement()                 {}
func (s *Continue) Position() token.Position { return s.Pos }

type If struct {
	Pos         token.Position
	BodyScopeID ScopeID
	ElseScopeID ScopeID
	Condition   Expression
	Body        []Statement
	Elifs       []ElseIf
	Else        []Statement
}

func (*If) statement()                 {}
func (s *If) Position() token.Position { return s.Pos }

type ElseIf struct {
	Pos       token.Position
	ScopeID   ScopeID
	Condition Expression
	Body      []Statement
}

// For describes source-level iteration. The data-pack emitter lowers this to
// a generated function that conditionally calls itself for the next iteration.
type For struct {
	Pos          token.Position
	ScopeID      ScopeID
	Variable     string
	VariableType *TypeRef
	Iterable     Expression
	Body         []Statement
}

func (*For) statement()                 {}
func (s *For) Position() token.Position { return s.Pos }

type While struct {
	Pos       token.Position
	ScopeID   ScopeID
	Condition Expression
	Body      []Statement
}

func (*While) statement()                 {}
func (s *While) Position() token.Position { return s.Pos }

type Expression interface {
	Node
	expression()
}

type Identifier struct {
	Pos  token.Position
	Name string
}

func (*Identifier) expression()                {}
func (e *Identifier) Position() token.Position { return e.Pos }

type Integer struct {
	Pos   token.Position
	Value int64
}

func (*Integer) expression()                {}
func (e *Integer) Position() token.Position { return e.Pos }

type String struct {
	Pos   token.Position
	Value string
}

type EntitySelector struct {
	Pos   token.Position
	Value string
}

func (*EntitySelector) expression()                {}
func (e *EntitySelector) Position() token.Position { return e.Pos }

func (*String) expression()                {}
func (e *String) Position() token.Position { return e.Pos }

type Boolean struct {
	Pos   token.Position
	Value bool
}

type NoneLiteral struct{ Pos token.Position }

func (*NoneLiteral) expression()                {}
func (e *NoneLiteral) Position() token.Position { return e.Pos }

func (*Boolean) expression()                {}
func (e *Boolean) Position() token.Position { return e.Pos }

type List struct {
	Pos      token.Position
	Elements []Expression
}

type Set struct {
	Pos      token.Position
	Elements []Expression
}

type NBT struct {
	Pos    token.Position
	Fields []NBTField
}

type NBTField struct {
	Pos   token.Position
	Key   string
	Value Expression
}

func (*NBT) expression()                {}
func (e *NBT) Position() token.Position { return e.Pos }

func (*Set) expression()                {}
func (e *Set) Position() token.Position { return e.Pos }

func (*List) expression()                {}
func (e *List) Position() token.Position { return e.Pos }

type Index struct {
	Pos    token.Position
	Target Expression
	Index  Expression
}

func (*Index) expression()                {}
func (e *Index) Position() token.Position { return e.Pos }

type Unary struct {
	Pos      token.Position
	Operator token.Kind
	Right    Expression
}

func (*Unary) expression()                {}
func (e *Unary) Position() token.Position { return e.Pos }

type Binary struct {
	Pos      token.Position
	Left     Expression
	Operator token.Kind
	Right    Expression
}

func (*Binary) expression()                {}
func (e *Binary) Position() token.Position { return e.Pos }

type Call struct {
	Pos       token.Position
	Callee    Expression
	Arguments []Expression
}

func (*Call) expression()                {}
func (e *Call) Position() token.Position { return e.Pos }

type Attribute struct {
	Pos    token.Position
	Target Expression
	Name   string
}

func (*Attribute) expression()                {}
func (e *Attribute) Position() token.Position { return e.Pos }
