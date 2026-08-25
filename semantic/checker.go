// Package semantic validates version-2 programs before Minecraft code generation.
package semantic

import (
	"fmt"
	"strings"

	"github.com/sirbuffalo/datacraft/ast"
	"github.com/sirbuffalo/datacraft/token"
)

type Error struct {
	Position token.Position
	Message  string
}

func (e Error) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Position.Line, e.Position.Column, e.Message)
}

type Type struct {
	Name     string
	Element  *Type
	Nullable bool
	Readonly bool
}

func (t Type) String() string {
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

type binding struct {
	Type     Type
	Constant bool
	Global   bool
}
type checker struct {
	globals         map[string]binding
	functions       map[string]*ast.Function
	writableGlobals map[string]struct{}
}

func Check(program *ast.Program) error {
	return CheckWithImports(program, nil)
}

func CheckWithImports(program *ast.Program, imported map[string]*ast.Function) error {
	if program.Version != 2 {
		return nil
	}
	c := &checker{globals: map[string]binding{}, functions: map[string]*ast.Function{}}
	for name, function := range imported {
		c.functions[name] = function
	}
	for _, declaration := range program.Globals {
		if err := c.declare(c.globals, declaration, nil); err != nil {
			return err
		}
		value := c.globals[declaration.Name]
		value.Global = true
		c.globals[declaration.Name] = value
	}
	for _, function := range program.Functions {
		if _, exists := c.functions[function.Name]; exists {
			return Error{function.Pos, fmt.Sprintf("duplicate function %q", function.Name)}
		}
		c.functions[function.Name] = function
		if function.ReturnType == nil {
			return Error{function.Pos, "version 2 functions require a return type"}
		}
		if _, err := resolveType(function.ReturnType); err != nil {
			return err
		}
	}
	for _, function := range program.Functions {
		if err := c.checkFunction(function); err != nil {
			return err
		}
	}
	return nil
}

func resolveType(ref *ast.TypeRef) (Type, error) {
	if ref == nil {
		return Type{}, Error{Message: "missing type"}
	}
	valid := ref.Name == "int" || ref.Name == "bool" || ref.Name == "str" || ref.Name == "entity" || ref.Name == "nbt" || ref.Name == "None" || ref.Name == "list" || ref.Name == "set"
	if !valid {
		return Type{}, Error{ref.Pos, fmt.Sprintf("unknown type %q", ref.Name)}
	}
	result := Type{Name: ref.Name, Nullable: ref.Nullable, Readonly: ref.Readonly}
	collection := ref.Name == "list" || ref.Name == "set"
	if collection && ref.Element == nil {
		return Type{}, Error{ref.Pos, fmt.Sprintf("%s requires one element type", ref.Name)}
	}
	if !collection && ref.Element != nil {
		return Type{}, Error{ref.Pos, fmt.Sprintf("%s cannot have an element type", ref.Name)}
	}
	if ref.Readonly && !collection {
		return Type{}, Error{ref.Pos, "readonly is only valid for list and set types"}
	}
	if ref.Name == "None" && ref.Nullable {
		return Type{}, Error{ref.Pos, "None cannot be nullable"}
	}
	if ref.Element != nil {
		element, err := resolveType(ref.Element)
		if err != nil {
			return Type{}, err
		}
		result.Element = &element
	}
	return result, nil
}

func (c *checker) checkFunction(function *ast.Function) error {
	c.writableGlobals = map[string]struct{}{}
	if err := c.collectGlobalDeclarations(function.Body); err != nil {
		return err
	}
	scope := map[string]binding{}
	for name, global := range c.globals {
		scope[name] = global
	}
	for _, name := range function.Parameters {
		ref := function.Types[name]
		if ref == nil {
			return Error{function.Pos, fmt.Sprintf("parameter %q requires a type", name)}
		}
		typeValue, err := resolveType(ref)
		if err != nil {
			return err
		}
		scope[name] = binding{Type: typeValue}
	}
	returnType, err := resolveType(function.ReturnType)
	if err != nil {
		return err
	}
	return c.checkStatements(function.Body, scope, returnType)
}

func (c *checker) checkStatements(statements []ast.Statement, scope map[string]binding, returnType Type) error {
	for _, statement := range statements {
		switch statement := statement.(type) {
		case *ast.Assignment:
			if statement.DeclaredType != nil {
				if err := c.declare(scope, statement, &returnType); err != nil {
					return err
				}
				continue
			}
			current, found := scope[statement.Name]
			if !found {
				return Error{statement.Pos, fmt.Sprintf("variable %q must be declared with a type before assignment", statement.Name)}
			}
			if current.Constant {
				return Error{statement.Pos, fmt.Sprintf("cannot assign to constant %q", statement.Name)}
			}
			if current.Type.Name == "nbt" && statement.Index != nil {
				if statement.Operator != token.Assign {
					return Error{statement.Pos, "NBT fields only support '=' assignment"}
				}
				for _, index := range statement.Indices {
					if _, ok := index.(*ast.String); !ok {
						return Error{index.Position(), "NBT keys must be string literals"}
					}
				}
				if err := c.validateNBTValue(statement.Value, scope); err != nil {
					return err
				}
				continue
			}
			if current.Global {
				if _, writable := c.writableGlobals[statement.Name]; !writable {
					return Error{statement.Pos, fmt.Sprintf("assigning namespace global %q requires 'global %s'", statement.Name, statement.Name)}
				}
			}
			if current.Type.Readonly && statement.Index != nil {
				return Error{statement.Pos, fmt.Sprintf("cannot mutate readonly %s", current.Type.String())}
			}
			actual, err := c.expressionType(statement.Value, scope, &current.Type)
			if err != nil {
				return err
			}
			if !assignable(current.Type, actual) {
				return Error{statement.Value.Position(), fmt.Sprintf("cannot assign %s to %s", actual.String(), current.Type.String())}
			}
		case *ast.Return:
			actual := Type{Name: "None"}
			var err error
			if statement.Value != nil {
				actual, err = c.expressionType(statement.Value, scope, &returnType)
			}
			if err != nil {
				return err
			}
			if !assignable(returnType, actual) {
				return Error{statement.Pos, fmt.Sprintf("function returns %s, not %s", actual.String(), returnType.String())}
			}
		case *ast.ExpressionStatement:
			if call, ok := statement.Expression.(*ast.Call); ok {
				if err := c.checkMutation(call, scope); err != nil {
					return err
				}
			}
			if _, err := c.expressionType(statement.Expression, scope, nil); err != nil {
				return err
			}
		case *ast.If:
			if _, err := c.expressionType(statement.Condition, scope, nil); err != nil {
				return err
			}
			if err := c.checkStatements(statement.Body, clone(scope), returnType); err != nil {
				return err
			}
			for _, branch := range statement.Elifs {
				if err := c.checkStatements(branch.Body, clone(scope), returnType); err != nil {
					return err
				}
			}
			if err := c.checkStatements(statement.Else, clone(scope), returnType); err != nil {
				return err
			}
		case *ast.While:
			if _, err := c.expressionType(statement.Condition, scope, nil); err != nil {
				return err
			}
			if err := c.checkStatements(statement.Body, clone(scope), returnType); err != nil {
				return err
			}
		case *ast.For:
			if statement.VariableType == nil {
				return Error{statement.Pos, "version 2 loop variables require a type"}
			}
			loopType, err := resolveType(statement.VariableType)
			if err != nil {
				return err
			}
			iterable, err := c.expressionType(statement.Iterable, scope, nil)
			if err != nil {
				return err
			}
			if iterable.Name != "list" && iterable.Name != "set" {
				return Error{statement.Iterable.Position(), "for loop requires a list or set"}
			}
			if iterable.Element == nil || !assignable(loopType, *iterable.Element) {
				return Error{statement.Pos, fmt.Sprintf("loop variable %s does not accept %s elements", loopType.String(), iterable.String())}
			}
			inner := clone(scope)
			inner[statement.Variable] = binding{Type: loopType}
			if err := c.checkStatements(statement.Body, inner, returnType); err != nil {
				return err
			}
		case *ast.Global:
			// Declarations are collected before checking so they apply throughout
			// the function, matching Python's function-wide global semantics.
		}
	}
	return nil
}

func (c *checker) collectGlobalDeclarations(statements []ast.Statement) error {
	for _, statement := range statements {
		switch statement := statement.(type) {
		case *ast.Global:
			for _, name := range statement.Names {
				if _, exists := c.globals[name]; !exists {
					return Error{statement.Pos, fmt.Sprintf("namespace global %q is not declared", name)}
				}
				c.writableGlobals[name] = struct{}{}
			}
		case *ast.If:
			if err := c.collectGlobalDeclarations(statement.Body); err != nil {
				return err
			}
			for _, branch := range statement.Elifs {
				if err := c.collectGlobalDeclarations(branch.Body); err != nil {
					return err
				}
			}
			if err := c.collectGlobalDeclarations(statement.Else); err != nil {
				return err
			}
		case *ast.For:
			if err := c.collectGlobalDeclarations(statement.Body); err != nil {
				return err
			}
		case *ast.While:
			if err := c.collectGlobalDeclarations(statement.Body); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *checker) declare(scope map[string]binding, declaration *ast.Assignment, _ *Type) error {
	if _, exists := scope[declaration.Name]; exists {
		return Error{declaration.Pos, fmt.Sprintf("variable %q is already declared", declaration.Name)}
	}
	declared, err := resolveType(declaration.DeclaredType)
	if err != nil {
		return err
	}
	actual, err := c.expressionType(declaration.Value, scope, &declared)
	if err != nil {
		return err
	}
	if !assignable(declared, actual) {
		return Error{declaration.Value.Position(), fmt.Sprintf("cannot initialize %s with %s", declared.String(), actual.String())}
	}
	if selector, ok := declaration.Value.(*ast.EntitySelector); ok && declared.Name == "entity" && !singularSelector(selector.Value) {
		return Error{selector.Pos, "entity selector may select multiple entities; use @n or add limit=1"}
	}
	scope[declaration.Name] = binding{Type: declared, Constant: declaration.Constant}
	return nil
}

func (c *checker) expressionType(expression ast.Expression, scope map[string]binding, expected *Type) (Type, error) {
	switch expression := expression.(type) {
	case *ast.Integer:
		return Type{Name: "int"}, nil
	case *ast.Boolean:
		return Type{Name: "bool"}, nil
	case *ast.String:
		return Type{Name: "str"}, nil
	case *ast.EntitySelector:
		if expected != nil && expected.Name == "set" && expected.Element != nil && expected.Element.Name == "entity" {
			return *expected, nil
		}
		return Type{Name: "entity", Nullable: true}, nil
	case *ast.NoneLiteral:
		return Type{Name: "None"}, nil
	case *ast.Identifier:
		value, found := scope[expression.Name]
		if !found {
			return Type{}, Error{expression.Pos, fmt.Sprintf("undefined variable %q", expression.Name)}
		}
		return value.Type, nil
	case *ast.List:
		return c.collectionLiteral("list", expression.Elements, expression.Pos, scope, expected)
	case *ast.Set:
		if expected != nil && expected.Name == "nbt" && len(expression.Elements) == 0 {
			return Type{Name: "nbt"}, nil
		}
		return c.collectionLiteral("set", expression.Elements, expression.Pos, scope, expected)
	case *ast.NBT:
		for _, field := range expression.Fields {
			if err := c.validateNBTValue(field.Value, scope); err != nil {
				return Type{}, err
			}
		}
		return Type{Name: "nbt"}, nil
	case *ast.Unary:
		value, err := c.expressionType(expression.Right, scope, nil)
		if err != nil {
			return Type{}, err
		}
		if value.Name != "int" && value.Name != "bool" {
			return Type{}, Error{expression.Pos, "unary operator requires int or bool"}
		}
		return Type{Name: "int"}, nil
	case *ast.Binary:
		if expression.Operator == token.Is {
			if _, err := c.expressionType(expression.Left, scope, nil); err != nil {
				return Type{}, err
			}
			typeName, ok := expression.Right.(*ast.Identifier)
			if !ok || (typeName.Name != "bool" && typeName.Name != "int" && typeName.Name != "str" && typeName.Name != "list" && typeName.Name != "entity" && typeName.Name != "nbt") {
				return Type{}, Error{expression.Right.Position(), "is expects bool, int, str, list, entity, or nbt"}
			}
			return Type{Name: "bool"}, nil
		}
		var operandExpected *Type
		if expected != nil && (expression.Operator == token.Plus || expression.Operator == token.Pipe || expression.Operator == token.Ampersand) {
			operandExpected = expected
		}
		left, err := c.expressionType(expression.Left, scope, operandExpected)
		if err != nil {
			return Type{}, err
		}
		right, err := c.expressionType(expression.Right, scope, operandExpected)
		if err != nil {
			return Type{}, err
		}
		if expression.Operator == token.Plus && left.Name == "str" && right.Name == "str" {
			return Type{Name: "str"}, nil
		}
		if expression.Operator == token.Plus && left.Name == "list" && right.Name == "list" {
			if !sameElementType(left, right) {
				return Type{}, Error{expression.Pos, "+ requires lists with the same element type"}
			}
			return mutableCollection(left), nil
		}
		if expression.Operator == token.Pipe || expression.Operator == token.Ampersand {
			if left.Name != "set" || right.Name != "set" || !sameElementType(left, right) {
				return Type{}, Error{expression.Pos, fmt.Sprintf("%s requires sets with the same element type", expression.Operator)}
			}
			return mutableCollection(left), nil
		}
		if expression.Operator == token.In {
			if right.Name != "set" || right.Element == nil || !assignable(*right.Element, left) {
				return Type{}, Error{expression.Pos, "in requires a value compatible with the set element type"}
			}
			return Type{Name: "bool"}, nil
		}
		if expression.Operator == token.Equal || expression.Operator == token.NotEqual || expression.Operator == token.Less || expression.Operator == token.LessEqual || expression.Operator == token.Greater || expression.Operator == token.GreaterEqual || expression.Operator == token.And || expression.Operator == token.Or || expression.Operator == token.Is {
			return Type{Name: "bool"}, nil
		}
		if (left.Name != "int" && left.Name != "bool") || (right.Name != "int" && right.Name != "bool") {
			return Type{}, Error{expression.Pos, "arithmetic requires int or bool operands"}
		}
		return Type{Name: "int"}, nil
	case *ast.Index:
		if root, indices := indexedRoot(expression); root != nil {
			if binding, found := scope[root.Name]; found && (binding.Type.Name == "nbt" || binding.Type.Name == "entity") {
				for _, index := range indices {
					if _, ok := index.(*ast.String); !ok {
						return Type{}, Error{index.Position(), binding.Type.Name + " NBT keys must be string literals"}
					}
				}
				if expected == nil || (binding.Type.Name == "entity" && expected.Name != "int" && expected.Name != "bool" && expected.Name != "str" && expected.Name != "nbt") {
					return Type{}, Error{expression.Pos, "an " + binding.Type.Name + " NBT read requires an expected type"}
				}
				return *expected, nil
			}
		}
		target, err := c.expressionType(expression.Target, scope, nil)
		if err != nil {
			return Type{}, err
		}
		index, err := c.expressionType(expression.Index, scope, nil)
		if err != nil {
			return Type{}, err
		}
		if target.Name == "nbt" {
			if _, ok := expression.Index.(*ast.String); !ok {
				return Type{}, Error{expression.Index.Position(), "NBT keys must be string literals"}
			}
			if expected == nil || expected.Name == "None" {
				return Type{}, Error{expression.Pos, "an NBT field read requires an expected type"}
			}
			return *expected, nil
		}
		if target.Name == "entity" {
			if _, ok := expression.Index.(*ast.String); !ok {
				return Type{}, Error{expression.Index.Position(), "entity NBT keys must be string literals"}
			}
			if expected == nil || (expected.Name != "int" && expected.Name != "bool" && expected.Name != "str" && expected.Name != "nbt") {
				return Type{}, Error{expression.Pos, "an entity NBT read requires an int, bool, str, or nbt destination"}
			}
			return *expected, nil
		}
		if target.Name != "list" || target.Element == nil {
			return Type{}, Error{expression.Pos, "only lists and NBT compounds support indexing"}
		}
		if index.Name != "int" && index.Name != "bool" {
			return Type{}, Error{expression.Index.Position(), "list index must be int"}
		}
		return *target.Element, nil
	case *ast.Call:
		return c.callType(expression, scope)
	}
	return Type{}, Error{expression.Position(), fmt.Sprintf("cannot type expression %T", expression)}
}

func indexedRoot(expression *ast.Index) (*ast.Identifier, []ast.Expression) {
	indices := []ast.Expression{expression.Index}
	target := expression.Target
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

func (c *checker) validateNBTValue(expression ast.Expression, scope map[string]binding) error {
	switch value := expression.(type) {
	case *ast.Integer, *ast.Boolean, *ast.String:
		return nil
	case *ast.NBT:
		for _, field := range value.Fields {
			if err := c.validateNBTValue(field.Value, scope); err != nil {
				return err
			}
		}
		return nil
	case *ast.List:
		for _, element := range value.Elements {
			if err := c.validateNBTValue(element, scope); err != nil {
				return err
			}
		}
		return nil
	case *ast.Identifier:
		actual, err := c.expressionType(value, scope, nil)
		if err != nil {
			return err
		}
		if actual.Name == "int" || actual.Name == "bool" || actual.Name == "str" || actual.Name == "list" || actual.Name == "nbt" {
			return nil
		}
	case *ast.Index:
		_, err := c.expressionType(value.Target, scope, nil)
		if err == nil {
			if _, ok := value.Index.(*ast.String); ok {
				return nil
			}
		}
	}
	return Error{expression.Position(), "NBT values support int, bool, str, list, and nbt; None and entities are not allowed"}
}

func (c *checker) collectionLiteral(name string, elements []ast.Expression, pos token.Position, scope map[string]binding, expected *Type) (Type, error) {
	if expected == nil || expected.Name != name || expected.Element == nil {
		return Type{}, Error{pos, fmt.Sprintf("%s literal requires an expected %s[T] type", name, name)}
	}
	for _, element := range elements {
		if name == "set" && expected.Element.Name == "entity" {
			if _, selector := element.(*ast.EntitySelector); selector {
				continue
			}
		}
		actual, err := c.expressionType(element, scope, expected.Element)
		if err != nil {
			return Type{}, err
		}
		if !assignable(*expected.Element, actual) {
			return Type{}, Error{element.Position(), fmt.Sprintf("%s element is %s, expected %s", name, actual.String(), expected.Element.String())}
		}
	}
	return Type{Name: name, Element: expected.Element}, nil
}

func sameElementType(left, right Type) bool {
	return left.Element != nil && right.Element != nil && assignable(*left.Element, *right.Element) && assignable(*right.Element, *left.Element)
}

func mutableCollection(value Type) Type {
	value.Readonly = false
	return value
}

func (c *checker) callType(call *ast.Call, scope map[string]binding) (Type, error) {
	identifier, ok := call.Callee.(*ast.Identifier)
	if !ok {
		if attribute, attrOK := call.Callee.(*ast.Attribute); attrOK {
			target, err := c.expressionType(attribute.Target, scope, nil)
			if err != nil {
				return Type{}, err
			}
			if target.Name != "list" && target.Name != "set" {
				return Type{}, Error{attribute.Pos, "collection method requires list or set"}
			}
			if target.Name == "set" {
				switch attribute.Name {
				case "add", "discard", "remove":
					if len(call.Arguments) != 1 {
						return Type{}, Error{call.Pos, attribute.Name + " expects one element"}
					}
					if selector, selectorOK := call.Arguments[0].(*ast.EntitySelector); selectorOK && target.Element != nil && target.Element.Name == "entity" {
						if !singularSelector(selector.Value) {
							return Type{}, Error{selector.Pos, "set element selector may select multiple entities"}
						}
						return Type{Name: "None"}, nil
					}
					actual, typeErr := c.expressionType(call.Arguments[0], scope, target.Element)
					if typeErr != nil {
						return Type{}, typeErr
					}
					if target.Element == nil || !assignable(*target.Element, actual) {
						return Type{}, Error{call.Arguments[0].Position(), fmt.Sprintf("set element is %s, expected %s", actual.String(), target.Element.String())}
					}
					return Type{Name: "None"}, nil
				case "clear":
					if len(call.Arguments) != 0 {
						return Type{}, Error{call.Pos, "clear expects no arguments"}
					}
					return Type{Name: "None"}, nil
				default:
					return Type{}, Error{attribute.Pos, fmt.Sprintf("unknown set method %q", attribute.Name)}
				}
			}
			if attribute.Name == "remove" && target.Element != nil {
				return *target.Element, nil
			}
			return Type{Name: "None"}, nil
		}
		return Type{}, Error{call.Pos, "invalid call target"}
	}
	switch identifier.Name {
	case "say":
		return Type{Name: "None"}, nil
	case "len":
		return Type{Name: "int"}, nil
	case "bool", "is_bool":
		return Type{Name: "bool"}, nil
	case "str":
		return Type{Name: "str"}, nil
	case "list", "set":
		if len(call.Arguments) != 1 {
			return Type{}, Error{call.Pos, identifier.Name + " expects one collection"}
		}
		source, err := c.expressionType(call.Arguments[0], scope, nil)
		if err != nil {
			return Type{}, err
		}
		if source.Name != "list" && source.Name != "set" {
			return Type{}, Error{call.Pos, identifier.Name + " expects a list or set"}
		}
		return Type{Name: identifier.Name, Element: source.Element}, nil
	}
	function, found := c.functions[identifier.Name]
	if !found {
		return Type{}, Error{identifier.Pos, fmt.Sprintf("undefined function %q", identifier.Name)}
	}
	if len(call.Arguments) != len(function.Parameters) {
		return Type{}, Error{call.Pos, fmt.Sprintf("function %q expects %d arguments", function.Name, len(function.Parameters))}
	}
	for index, argument := range call.Arguments {
		expected, _ := resolveType(function.Types[function.Parameters[index]])
		actual, err := c.expressionType(argument, scope, &expected)
		if err != nil {
			return Type{}, err
		}
		if !assignable(expected, actual) {
			return Type{}, Error{argument.Position(), fmt.Sprintf("argument is %s, expected %s", actual.String(), expected.String())}
		}
	}
	return resolveType(function.ReturnType)
}

func (c *checker) checkMutation(call *ast.Call, scope map[string]binding) error {
	attribute, ok := call.Callee.(*ast.Attribute)
	if !ok {
		return nil
	}
	identifier, ok := attribute.Target.(*ast.Identifier)
	if !ok {
		return nil
	}
	value, found := scope[identifier.Name]
	if !found {
		return nil
	}
	mutation := attribute.Name == "append" || attribute.Name == "insert" || attribute.Name == "remove" || attribute.Name == "discard" || attribute.Name == "add" || attribute.Name == "clear"
	if mutation && value.Global {
		if _, writable := c.writableGlobals[identifier.Name]; !writable {
			return Error{call.Pos, fmt.Sprintf("mutating namespace global %q requires 'global %s'", identifier.Name, identifier.Name)}
		}
	}
	if mutation && (value.Constant || value.Type.Readonly) {
		return Error{call.Pos, fmt.Sprintf("cannot mutate %s collection %q", qualifier(value), identifier.Name)}
	}
	return nil
}

func assignable(destination, source Type) bool {
	if source.Name == "None" {
		return destination.Nullable
	}
	if destination.Name == "int" && source.Name == "bool" {
		return true
	}
	if destination.Name != source.Name {
		return false
	}
	if source.Nullable && !destination.Nullable {
		return false
	}
	if destination.Element != nil {
		if source.Element == nil || !assignable(*destination.Element, *source.Element) {
			return false
		}
	}
	if !destination.Readonly && source.Readonly {
		return false
	}
	return true
}

func singularSelector(selector string) bool {
	if selector == "@s" || strings.HasPrefix(selector, "@n") || strings.HasPrefix(selector, "@p") || strings.HasPrefix(selector, "@r") {
		return true
	}
	return strings.HasPrefix(selector, "@e[") && strings.Contains(selector, "limit=1")
}

func clone(source map[string]binding) map[string]binding {
	result := map[string]binding{}
	for key, value := range source {
		result[key] = value
	}
	return result
}
func qualifier(value binding) string {
	if value.Constant {
		return "constant"
	}
	return "readonly"
}
