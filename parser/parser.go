package parser

import (
	"fmt"
	"strconv"

	"mccomp/ast"
	"mccomp/lexer"
	"mccomp/token"
)

type Error struct {
	Position token.Position
	Message  string
}

func (e Error) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Position.Line, e.Position.Column, e.Message)
}

// Parse accepts source text rather than a file, keeping the parser portable to
// WebAssembly. Reading files belongs in a native CLI adapter.
func Parse(source string) (*ast.Program, error) {
	tokens, err := lexer.Lex(source)
	if err != nil {
		return nil, err
	}
	p := parser{tokens: tokens}
	return p.parseProgram()
}

type parser struct {
	tokens []token.Token
	index  int
}

func (p *parser) parseProgram() (*ast.Program, error) {
	program := &ast.Program{}
	p.skipNewlines()
	if p.match(token.Namespace) {
		name, err := p.consume(token.Identifier, "expected namespace ID")
		if err != nil {
			return nil, err
		}
		program.Namespace = name.Lexeme
		if err = p.endStatement(); err != nil {
			return nil, err
		}
		p.skipNewlines()
	}
	for !p.check(token.EOF) {
		function, err := p.parseFunction()
		if err != nil {
			return nil, err
		}
		program.Functions = append(program.Functions, function)
		p.skipNewlines()
	}
	return program, nil
}

func (p *parser) parseFunction() (*ast.Function, error) {
	exported := p.match(token.Export)
	start, err := p.consume(token.Def, "expected 'def' at the top level")
	if err != nil {
		return nil, err
	}
	name, err := p.consume(token.Identifier, "expected function name")
	if err != nil {
		return nil, err
	}
	if _, err = p.consume(token.LeftParen, "expected '(' after function name"); err != nil {
		return nil, err
	}
	parameters := []string{}
	parameterTypes := map[string]string{}
	if !p.check(token.RightParen) {
		for {
			parameter, consumeErr := p.consume(token.Identifier, "expected parameter name")
			if consumeErr != nil {
				return nil, consumeErr
			}
			parameters = append(parameters, parameter.Lexeme)
			if p.match(token.Colon) {
				typeName, typeErr := p.consume(token.Identifier, "expected parameter type after ':'")
				if typeErr != nil {
					return nil, typeErr
				}
				parameterTypes[parameter.Lexeme] = typeName.Lexeme
			}
			if !p.match(token.Comma) {
				break
			}
		}
	}
	if _, err = p.consume(token.RightParen, "expected ')' after parameters"); err != nil {
		return nil, err
	}
	if _, err = p.consume(token.Colon, "expected ':' after function signature"); err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.Function{Pos: start.Position, Exported: exported, Name: name.Lexeme, Parameters: parameters, ParameterTypes: parameterTypes, Body: body}, nil
}

func (p *parser) parseBlock() ([]ast.Statement, error) {
	if _, err := p.consume(token.Newline, "expected a newline before the block"); err != nil {
		return nil, err
	}
	for p.match(token.Newline) {
	}
	if _, err := p.consume(token.Indent, "expected an indented block"); err != nil {
		return nil, err
	}

	var statements []ast.Statement
	p.skipNewlines()
	for !p.check(token.Dedent) && !p.check(token.EOF) {
		statement, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		statements = append(statements, statement)
		p.skipNewlines()
	}
	if _, err := p.consume(token.Dedent, "expected the end of the indented block"); err != nil {
		return nil, err
	}
	if len(statements) == 0 {
		return nil, Error{p.previous().Position, "blocks cannot be empty"}
	}
	return statements, nil
}

func (p *parser) parseStatement() (ast.Statement, error) {
	switch {
	case p.match(token.Return):
		return p.parseReturn(p.previous())
	case p.match(token.Global):
		return p.parseGlobal(p.previous())
	case p.match(token.If):
		return p.parseIf(p.previous())
	case p.match(token.For):
		return p.parseFor(p.previous())
	case p.match(token.While):
		return p.parseWhile(p.previous())
	case p.match(token.Break):
		start := p.previous()
		if err := p.endStatement(); err != nil {
			return nil, err
		}
		return &ast.Break{Pos: start.Position}, nil
	case p.match(token.Continue):
		start := p.previous()
		if err := p.endStatement(); err != nil {
			return nil, err
		}
		return &ast.Continue{Pos: start.Position}, nil
	case p.match(token.Command):
		command := p.previous()
		if err := p.endStatement(); err != nil {
			return nil, err
		}
		return &ast.Command{Pos: command.Position, Text: command.Lexeme}, nil
	case p.check(token.Identifier) && p.checkNext(token.Assign, token.PlusAssign, token.MinusAssign, token.StarAssign, token.SlashAssign):
		return p.parseAssignment()
	case p.check(token.Identifier) && p.checkNext(token.LeftBracket):
		return p.parseIndexedAssignment()
	default:
		pos := p.current().Position
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if err = p.endStatement(); err != nil {
			return nil, err
		}
		return &ast.ExpressionStatement{Pos: pos, Expression: expr}, nil
	}
}

func (p *parser) parseGlobal(start token.Token) (ast.Statement, error) {
	names := []string{}
	types := map[string]string{}
	for {
		name, err := p.consume(token.Identifier, "expected variable name after 'global'")
		if err != nil {
			return nil, err
		}
		names = append(names, name.Lexeme)
		if p.match(token.Colon) {
			typeName, typeErr := p.consume(token.Identifier, "expected global type after ':'")
			if typeErr != nil {
				return nil, typeErr
			}
			types[name.Lexeme] = typeName.Lexeme
		}
		if !p.match(token.Comma) {
			break
		}
	}
	if err := p.endStatement(); err != nil {
		return nil, err
	}
	return &ast.Global{Pos: start.Position, Names: names, Types: types}, nil
}

func (p *parser) parseFor(start token.Token) (ast.Statement, error) {
	variable, err := p.consume(token.Identifier, "expected loop variable after 'for'")
	if err != nil {
		return nil, err
	}
	if _, err = p.consume(token.In, "expected 'in' after loop variable"); err != nil {
		return nil, err
	}
	iterable, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err = p.consume(token.Colon, "expected ':' after for loop"); err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.For{Pos: start.Position, Variable: variable.Lexeme, Iterable: iterable, Body: body}, nil
}

func (p *parser) parseWhile(start token.Token) (ast.Statement, error) {
	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err = p.consume(token.Colon, "expected ':' after while condition"); err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.While{Pos: start.Position, Condition: condition, Body: body}, nil
}

func (p *parser) parseAssignment() (ast.Statement, error) {
	name := p.advance()
	operator := p.advance()
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if err = p.endStatement(); err != nil {
		return nil, err
	}
	return &ast.Assignment{Pos: name.Position, Name: name.Lexeme, Operator: operator.Kind, Value: value}, nil
}

func (p *parser) parseIndexedAssignment() (ast.Statement, error) {
	target, err := p.parseCall()
	if err != nil {
		return nil, err
	}
	indexed, ok := target.(*ast.Index)
	if !ok {
		return nil, Error{target.Position(), "assignment target must be a list item"}
	}
	root, indices := indexedRootAndIndices(indexed)
	if root == nil {
		return nil, Error{indexed.Pos, "indexed assignment requires a list variable"}
	}
	operator := p.current()
	if !p.match(token.Assign, token.PlusAssign, token.MinusAssign, token.StarAssign, token.SlashAssign) {
		return nil, Error{operator.Position, "expected an assignment operator after list item"}
	}
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if err = p.endStatement(); err != nil {
		return nil, err
	}
	return &ast.Assignment{Pos: root.Pos, Name: root.Name, Index: indices[0], Indices: indices, Operator: operator.Kind, Value: value}, nil
}

func indexedRootAndIndices(indexed *ast.Index) (*ast.Identifier, []ast.Expression) {
	indices := []ast.Expression{indexed.Index}
	target := indexed.Target
	for {
		if parent, ok := target.(*ast.Index); ok {
			indices = append([]ast.Expression{parent.Index}, indices...)
			target = parent.Target
			continue
		}
		root, _ := target.(*ast.Identifier)
		return root, indices
	}
}

func (p *parser) parseReturn(start token.Token) (ast.Statement, error) {
	var value ast.Expression
	var err error
	if !p.check(token.Newline) {
		value, err = p.parseExpression()
		if err != nil {
			return nil, err
		}
	}
	if err = p.endStatement(); err != nil {
		return nil, err
	}
	return &ast.Return{Pos: start.Position, Value: value}, nil
}

func (p *parser) parseIf(start token.Token) (ast.Statement, error) {
	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err = p.consume(token.Colon, "expected ':' after condition"); err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	result := &ast.If{Pos: start.Position, Condition: condition, Body: body}

	for p.match(token.Elif) {
		elifToken := p.previous()
		elifCondition, parseErr := p.parseExpression()
		if parseErr != nil {
			return nil, parseErr
		}
		if _, parseErr = p.consume(token.Colon, "expected ':' after elif condition"); parseErr != nil {
			return nil, parseErr
		}
		elifBody, parseErr := p.parseBlock()
		if parseErr != nil {
			return nil, parseErr
		}
		result.Elifs = append(result.Elifs, ast.ElseIf{Pos: elifToken.Position, Condition: elifCondition, Body: elifBody})
	}
	if p.match(token.Else) {
		if _, err = p.consume(token.Colon, "expected ':' after else"); err != nil {
			return nil, err
		}
		result.Else, err = p.parseBlock()
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (p *parser) parseExpression() (ast.Expression, error) { return p.parseOr() }

func (p *parser) parseOr() (ast.Expression, error) {
	return p.parseBinary(p.parseAnd, token.Or)
}

func (p *parser) parseAnd() (ast.Expression, error) {
	return p.parseBinary(p.parseEquality, token.And)
}

func (p *parser) parseEquality() (ast.Expression, error) {
	return p.parseBinary(p.parseComparison, token.Equal, token.NotEqual, token.Is)
}

func (p *parser) parseComparison() (ast.Expression, error) {
	return p.parseBinary(p.parseConcat, token.Less, token.LessEqual, token.Greater, token.GreaterEqual)
}

func (p *parser) parseConcat() (ast.Expression, error) {
	return p.parseBinary(p.parseTerm, token.Ampersand)
}

func (p *parser) parseTerm() (ast.Expression, error) {
	return p.parseBinary(p.parseFactor, token.Plus, token.Minus)
}

func (p *parser) parseFactor() (ast.Expression, error) {
	return p.parseBinary(p.parseUnary, token.Star, token.Slash, token.Percent)
}

func (p *parser) parseBinary(next func() (ast.Expression, error), kinds ...token.Kind) (ast.Expression, error) {
	left, err := next()
	if err != nil {
		return nil, err
	}
	for p.match(kinds...) {
		operator := p.previous()
		right, parseErr := next()
		if parseErr != nil {
			return nil, parseErr
		}
		left = &ast.Binary{Pos: operator.Position, Left: left, Operator: operator.Kind, Right: right}
	}
	return left, nil
}

func (p *parser) parseUnary() (ast.Expression, error) {
	if p.match(token.Not, token.Minus, token.Plus) {
		operator := p.previous()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.Unary{Pos: operator.Position, Operator: operator.Kind, Right: right}, nil
	}
	return p.parseCall()
}

func (p *parser) parseCall() (ast.Expression, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		if p.match(token.Dot) {
			dot := p.previous()
			name, consumeErr := p.consume(token.Identifier, "expected attribute name after '.'")
			if consumeErr != nil {
				return nil, consumeErr
			}
			expr = &ast.Attribute{Pos: dot.Position, Target: expr, Name: name.Lexeme}
			continue
		}
		if p.match(token.LeftParen) {
			open := p.previous()
			arguments := []ast.Expression{}
			if !p.check(token.RightParen) {
				for {
					argument, parseErr := p.parseExpression()
					if parseErr != nil {
						return nil, parseErr
					}
					arguments = append(arguments, argument)
					if !p.match(token.Comma) {
						break
					}
				}
			}
			if _, err = p.consume(token.RightParen, "expected ')' after arguments"); err != nil {
				return nil, err
			}
			expr = &ast.Call{Pos: open.Position, Callee: expr, Arguments: arguments}
			continue
		}
		if p.match(token.LeftBracket) {
			open := p.previous()
			index, parseErr := p.parseExpression()
			if parseErr != nil {
				return nil, parseErr
			}
			if _, err = p.consume(token.RightBracket, "expected ']' after list index"); err != nil {
				return nil, err
			}
			expr = &ast.Index{Pos: open.Position, Target: expr, Index: index}
			continue
		}
		break
	}
	return expr, nil
}

func (p *parser) parsePrimary() (ast.Expression, error) {
	if p.match(token.Integer) {
		value, err := strconv.ParseInt(p.previous().Lexeme, 10, 64)
		if err != nil {
			return nil, Error{p.previous().Position, "integer is out of range"}
		}
		return &ast.Integer{Pos: p.previous().Position, Value: value}, nil
	}
	if p.match(token.String) {
		return &ast.String{Pos: p.previous().Position, Value: p.previous().Lexeme}, nil
	}
	if p.match(token.Selector) {
		selector := p.previous()
		return &ast.EntitySelector{Pos: selector.Position, Value: selector.Lexeme}, nil
	}
	if p.match(token.True, token.False) {
		return &ast.Boolean{Pos: p.previous().Position, Value: p.previous().Kind == token.True}, nil
	}
	if p.match(token.Identifier) {
		return &ast.Identifier{Pos: p.previous().Position, Name: p.previous().Lexeme}, nil
	}
	if p.match(token.LeftParen) {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if _, err = p.consume(token.RightParen, "expected ')' after expression"); err != nil {
			return nil, err
		}
		return expr, nil
	}
	if p.match(token.LeftBracket) {
		open := p.previous()
		elements := []ast.Expression{}
		if !p.check(token.RightBracket) {
			for {
				element, err := p.parseExpression()
				if err != nil {
					return nil, err
				}
				elements = append(elements, element)
				if !p.match(token.Comma) {
					break
				}
			}
		}
		if _, err := p.consume(token.RightBracket, "expected ']' after list elements"); err != nil {
			return nil, err
		}
		return &ast.List{Pos: open.Position, Elements: elements}, nil
	}
	return nil, Error{p.current().Position, fmt.Sprintf("expected expression, found %s", p.current().Kind)}
}

func (p *parser) endStatement() error {
	if _, err := p.consume(token.Newline, "expected a newline after statement"); err != nil {
		return err
	}
	return nil
}

func (p *parser) skipNewlines() {
	for p.match(token.Newline) {
	}
}

func (p *parser) match(kinds ...token.Kind) bool {
	for _, kind := range kinds {
		if p.check(kind) {
			p.advance()
			return true
		}
	}
	return false
}

func (p *parser) consume(kind token.Kind, message string) (token.Token, error) {
	if p.check(kind) {
		return p.advance(), nil
	}
	return token.Token{}, Error{p.current().Position, message}
}

func (p *parser) check(kind token.Kind) bool { return p.current().Kind == kind }

func (p *parser) checkNext(kinds ...token.Kind) bool {
	if p.index+1 >= len(p.tokens) {
		return false
	}
	for _, kind := range kinds {
		if p.tokens[p.index+1].Kind == kind {
			return true
		}
	}
	return false
}

func (p *parser) advance() token.Token {
	current := p.current()
	if p.index < len(p.tokens)-1 {
		p.index++
	}
	return current
}

func (p *parser) current() token.Token { return p.tokens[p.index] }

func (p *parser) previous() token.Token {
	if p.index == 0 {
		return p.tokens[0]
	}
	return p.tokens[p.index-1]
}
