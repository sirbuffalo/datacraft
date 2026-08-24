package lexer

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sirbuffalo/datacraft/token"
)

type Error struct {
	Position token.Position
	Message  string
}

func (e Error) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Position.Line, e.Position.Column, e.Message)
}

// Lex converts source text into tokens. It has no filesystem or OS dependencies,
// so the same implementation can be used by native and WebAssembly front ends.
func Lex(source string) ([]token.Token, error) {
	l := lexer{
		source:  source,
		line:    1,
		column:  1,
		atStart: true,
		indents: []int{0},
	}
	return l.run()
}

type lexer struct {
	source  string
	offset  int
	line    int
	column  int
	atStart bool
	indents []int
	tokens  []token.Token
}

func (l *lexer) run() ([]token.Token, error) {
	for !l.done() {
		if l.atStart {
			if err := l.lexIndentation(); err != nil {
				return nil, err
			}
			if l.done() {
				break
			}
		}

		r, _ := l.peek()
		switch {
		case r == ' ' || r == '\t' || r == '\r':
			l.advance()
		case r == '\n':
			pos := l.position()
			l.advance()
			l.emit(token.Newline, "", pos)
			l.atStart = true
		case r == '#':
			l.skipComment()
		case r == '/' && l.onlyWhitespaceSinceLineStart():
			l.lexCommand()
		case r == '@':
			l.lexSelector()
		case unicode.IsLetter(r) || r == '_':
			l.lexIdentifier()
		case unicode.IsDigit(r):
			l.lexInteger()
		case r == '"' || r == '\'':
			if err := l.lexString(); err != nil {
				return nil, err
			}
		default:
			if err := l.lexOperator(); err != nil {
				return nil, err
			}
		}
	}

	if len(l.tokens) > 0 && l.tokens[len(l.tokens)-1].Kind != token.Newline {
		l.emit(token.Newline, "", l.position())
	}
	for len(l.indents) > 1 {
		l.indents = l.indents[:len(l.indents)-1]
		l.emit(token.Dedent, "", l.position())
	}
	l.emit(token.EOF, "", l.position())
	return l.tokens, nil
}

func (l *lexer) lexSelector() {
	pos, start := l.position(), l.offset
	depth := 0
	quote := rune(0)
	for !l.done() {
		r, _ := l.peek()
		if quote != 0 {
			l.advance()
			if r == quote {
				quote = 0
			}
			continue
		}
		if r == '"' || r == '\'' {
			quote = r
			l.advance()
			continue
		}
		if r == '[' {
			depth++
			l.advance()
			continue
		}
		if r == ']' {
			if depth == 0 {
				break
			}
			depth--
			l.advance()
			continue
		}
		if depth == 0 && (unicode.IsSpace(r) || strings.ContainsRune(",):}", r)) {
			break
		}
		l.advance()
	}
	l.emit(token.Selector, l.source[start:l.offset], pos)
}

func (l *lexer) lexIndentation() error {
	start := l.position()
	spaces := 0
	for !l.done() {
		r, _ := l.peek()
		if r == ' ' {
			spaces++
			l.advance()
			continue
		}
		if r == '\t' {
			return Error{l.position(), "tabs are not allowed; indent with spaces"}
		}
		break
	}

	if l.done() {
		return nil
	}
	r, _ := l.peek()
	if r == '\n' || r == '\r' || r == '#' {
		l.atStart = false
		return nil
	}
	if spaces%4 != 0 {
		return Error{start, "indentation must use multiples of four spaces"}
	}

	current := l.indents[len(l.indents)-1]
	switch {
	case spaces > current:
		l.indents = append(l.indents, spaces)
		l.emit(token.Indent, "", start)
	case spaces < current:
		for len(l.indents) > 1 && spaces < l.indents[len(l.indents)-1] {
			l.indents = l.indents[:len(l.indents)-1]
			l.emit(token.Dedent, "", start)
		}
		if spaces != l.indents[len(l.indents)-1] {
			return Error{start, "indentation does not match an outer block"}
		}
	}
	l.atStart = false
	return nil
}

func (l *lexer) lexIdentifier() {
	pos, start := l.position(), l.offset
	for !l.done() {
		r, _ := l.peek()
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			break
		}
		l.advance()
	}
	text := l.source[start:l.offset]
	kind := token.Identifier
	if keyword, ok := token.Keywords[text]; ok {
		kind = keyword
	}
	l.emit(kind, text, pos)
}

func (l *lexer) lexInteger() {
	pos, start := l.position(), l.offset
	for !l.done() {
		r, _ := l.peek()
		if !unicode.IsDigit(r) {
			break
		}
		l.advance()
	}
	l.emit(token.Integer, l.source[start:l.offset], pos)
}

func (l *lexer) lexString() error {
	pos := l.position()
	quote, _ := l.advance()
	var value strings.Builder
	for !l.done() {
		r, _ := l.advance()
		if r == quote {
			l.emit(token.String, value.String(), pos)
			return nil
		}
		if r == '\n' {
			return Error{pos, "unterminated string"}
		}
		if r == '\\' {
			if l.done() {
				break
			}
			escaped, _ := l.advance()
			switch escaped {
			case 'n':
				value.WriteRune('\n')
			case 't':
				value.WriteRune('\t')
			case '\\', '\'', '"':
				value.WriteRune(escaped)
			default:
				return Error{l.position(), "unsupported escape sequence"}
			}
			continue
		}
		value.WriteRune(r)
	}
	return Error{pos, "unterminated string"}
}

func (l *lexer) lexCommand() {
	pos, start := l.position(), l.offset
	for !l.done() {
		r, _ := l.peek()
		if r == '\n' {
			break
		}
		l.advance()
	}
	l.emit(token.Command, strings.TrimSpace(l.source[start+1:l.offset]), pos)
}

func (l *lexer) lexOperator() error {
	pos := l.position()
	r, _ := l.advance()
	one := string(r)
	two := one
	if !l.done() {
		next, _ := l.peek()
		two += string(next)
	}
	twoKinds := map[string]token.Kind{
		"=": token.Assign, "+=": token.PlusAssign, "-=": token.MinusAssign,
		"*=": token.StarAssign, "/=": token.SlashAssign, "==": token.Equal,
		"!=": token.NotEqual, "<=": token.LessEqual, ">=": token.GreaterEqual,
		"->": token.Arrow,
	}
	if kind, ok := twoKinds[two]; ok && len(two) == 2 {
		l.advance()
		l.emit(kind, two, pos)
		return nil
	}
	oneKinds := map[string]token.Kind{
		"=": token.Assign, "+": token.Plus, "-": token.Minus, "*": token.Star,
		"/": token.Slash, "%": token.Percent, "<": token.Less, ">": token.Greater,
		"(": token.LeftParen, ")": token.RightParen, ",": token.Comma, ":": token.Colon, ".": token.Dot,
		"[": token.LeftBracket, "]": token.RightBracket,
		"&": token.Ampersand,
		"?": token.Question, "{": token.LeftBrace, "}": token.RightBrace, "|": token.Pipe,
	}
	if kind, ok := oneKinds[one]; ok {
		l.emit(kind, one, pos)
		return nil
	}
	return Error{pos, fmt.Sprintf("unexpected character %q", r)}
}

func (l *lexer) skipComment() {
	for !l.done() {
		r, _ := l.peek()
		if r == '\n' {
			return
		}
		l.advance()
	}
}

func (l *lexer) onlyWhitespaceSinceLineStart() bool {
	i := l.offset - 1
	for i >= 0 && l.source[i] != '\n' {
		if l.source[i] != ' ' && l.source[i] != '\t' && l.source[i] != '\r' {
			return false
		}
		i--
	}
	return true
}

func (l *lexer) done() bool { return l.offset >= len(l.source) }

func (l *lexer) peek() (rune, int) {
	r, size := utf8.DecodeRuneInString(l.source[l.offset:])
	return r, size
}

func (l *lexer) advance() (rune, int) {
	r, size := l.peek()
	l.offset += size
	if r == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
	return r, size
}

func (l *lexer) position() token.Position {
	return token.Position{Line: l.line, Column: l.column}
}

func (l *lexer) emit(kind token.Kind, lexeme string, pos token.Position) {
	l.tokens = append(l.tokens, token.Token{Kind: kind, Lexeme: lexeme, Position: pos})
}
