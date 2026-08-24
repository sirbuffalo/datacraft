package token

import "fmt"

type Kind string

const (
	Illegal Kind = "ILLEGAL"
	EOF     Kind = "EOF"
	Newline Kind = "NEWLINE"
	Indent  Kind = "INDENT"
	Dedent  Kind = "DEDENT"

	Identifier Kind = "IDENTIFIER"
	Integer    Kind = "INTEGER"
	String     Kind = "STRING"
	Selector   Kind = "SELECTOR"
	Command    Kind = "COMMAND"

	Version   Kind = "VERSION"
	Def       Kind = "DEF"
	Expose    Kind = "EXPOSE"
	Export    Kind = "EXPORT" // Reserved to provide a migration diagnostic.
	From      Kind = "FROM"
	Import    Kind = "IMPORT"
	Namespace Kind = "NAMESPACE"
	Global    Kind = "GLOBAL"
	Return    Kind = "RETURN"
	If        Kind = "IF"
	Elif      Kind = "ELIF"
	Else      Kind = "ELSE"
	For       Kind = "FOR"
	While     Kind = "WHILE"
	Break     Kind = "BREAK"
	Continue  Kind = "CONTINUE"
	In        Kind = "IN"
	Is        Kind = "IS"
	True      Kind = "TRUE"
	False     Kind = "FALSE"
	And       Kind = "AND"
	Or        Kind = "OR"
	Not       Kind = "NOT"
	Const     Kind = "CONST"
	Readonly  Kind = "READONLY"
	None      Kind = "NONE"

	Assign       Kind = "="
	PlusAssign   Kind = "+="
	MinusAssign  Kind = "-="
	StarAssign   Kind = "*="
	SlashAssign  Kind = "/="
	Plus         Kind = "+"
	Minus        Kind = "-"
	Star         Kind = "*"
	Slash        Kind = "/"
	Percent      Kind = "%"
	Equal        Kind = "=="
	NotEqual     Kind = "!="
	Less         Kind = "<"
	LessEqual    Kind = "<="
	Greater      Kind = ">"
	GreaterEqual Kind = ">="
	LeftParen    Kind = "("
	RightParen   Kind = ")"
	LeftBracket  Kind = "["
	RightBracket Kind = "]"
	Comma        Kind = ","
	Colon        Kind = ":"
	Dot          Kind = "."
	Ampersand    Kind = "&"
	Arrow        Kind = "->"
	Question     Kind = "?"
	LeftBrace    Kind = "{"
	RightBrace   Kind = "}"
	Pipe         Kind = "|"
)

type Position struct {
	Line   int
	Column int
}

type Token struct {
	Kind     Kind
	Lexeme   string
	Position Position
}

func (t Token) String() string {
	return fmt.Sprintf("%s(%q) at %d:%d", t.Kind, t.Lexeme, t.Position.Line, t.Position.Column)
}

var Keywords = map[string]Kind{
	"version": Version, "def": Def, "expose": Expose, "export": Export, "from": From, "import": Import, "namespace": Namespace, "global": Global, "return": Return, "if": If, "elif": Elif, "else": Else,
	"for": For, "while": While, "break": Break, "continue": Continue, "in": In, "is": Is,
	"True": True, "False": False, "and": And, "or": Or, "not": Not,
	"mod":   Percent,
	"const": Const, "readonly": Readonly, "None": None,
}
