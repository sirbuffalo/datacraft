package parser_test

import (
	"testing"

	"github.com/sirbuffalo/datacraft/ast"
	"github.com/sirbuffalo/datacraft/parser"
	"github.com/sirbuffalo/datacraft/token"
)

func TestParseFunctionAndArithmetic(t *testing.T) {
	source := "namespace demo\n\ndef add(a, b):\n    total = a + b * 2\n    return total\n"
	program, err := parser.ParseLegacy(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(program.Functions) != 1 {
		t.Fatalf("got %d functions, want 1", len(program.Functions))
	}
	if program.Namespace != "demo" {
		t.Fatalf("namespace = %q, want demo", program.Namespace)
	}
	function := program.Functions[0]
	if function.Name != "add" || len(function.Parameters) != 2 {
		t.Fatalf("unexpected function: %#v", function)
	}
	assignment, ok := function.Body[0].(*ast.Assignment)
	if !ok {
		t.Fatalf("first statement is %T, want *ast.Assignment", function.Body[0])
	}
	if assignment.Operator != token.Assign {
		t.Fatalf("operator = %s, want %s", assignment.Operator, token.Assign)
	}
	addition, ok := assignment.Value.(*ast.Binary)
	if !ok || addition.Operator != token.Plus {
		t.Fatalf("assignment value = %#v, want addition", assignment.Value)
	}
	if multiplication, ok := addition.Right.(*ast.Binary); !ok || multiplication.Operator != token.Star {
		t.Fatalf("addition right = %#v, want multiplication", addition.Right)
	}
}

func TestParseIfAndFor(t *testing.T) {
	source := "def reward():\n    players = [1, 2, 3]\n    for player in players:\n        if ready(player):\n            give(player, 1)\n        else:\n            /say Player is not ready\n"
	program, err := parser.ParseLegacy(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	assignment, ok := program.Functions[0].Body[0].(*ast.Assignment)
	if !ok {
		t.Fatalf("statement is %T, want *ast.Assignment", program.Functions[0].Body[0])
	}
	if list, ok := assignment.Value.(*ast.List); !ok || len(list.Elements) != 3 {
		t.Fatalf("assignment value = %#v, want three-element list", assignment.Value)
	}
	loop, ok := program.Functions[0].Body[1].(*ast.For)
	if !ok {
		t.Fatalf("statement is %T, want *ast.For", program.Functions[0].Body[1])
	}
	if loop.Variable != "player" {
		t.Fatalf("loop variable = %q, want player", loop.Variable)
	}
	conditional, ok := loop.Body[0].(*ast.If)
	if !ok || len(conditional.Else) != 1 {
		t.Fatalf("loop body = %#v, want if with else", loop.Body)
	}
}

func TestRejectTabs(t *testing.T) {
	_, err := parser.ParseLegacy("def bad():\n\treturn 1\n")
	if err == nil {
		t.Fatal("Parse() error = nil, want indentation error")
	}
}

func TestParseGlobalDeclaration(t *testing.T) {
	program, err := parser.ParseLegacy("def update():\n    global counter, timer\n    counter += 1\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	declaration, ok := program.Functions[0].Body[0].(*ast.Global)
	if !ok {
		t.Fatalf("statement is %T, want *ast.Global", program.Functions[0].Body[0])
	}
	if len(declaration.Names) != 2 || declaration.Names[0] != "counter" || declaration.Names[1] != "timer" {
		t.Fatalf("global names = %#v, want counter and timer", declaration.Names)
	}
}

func TestParseExposedFunction(t *testing.T) {
	program, err := parser.ParseLegacy("expose def reward(player):\n    return 5\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(program.Functions) != 1 || !program.Functions[0].Exposed {
		t.Fatalf("function = %#v, want exposed function", program.Functions)
	}
}

func TestExportReportsExposeMigration(t *testing.T) {
	_, err := parser.ParseLegacy("export def reward():\n    return 5\n")
	if err == nil || err.Error() != "2:1: 'export' was renamed to 'expose'" {
		t.Fatalf("ParseLegacy() error = %v", err)
	}
}

func TestParseImports(t *testing.T) {
	program, err := parser.Parse("namespace app\n\nfrom math import add, clamp\n\ndef run() -> None:\n    say(add(2, 3))\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(program.Imports) != 1 || program.Imports[0].Namespace != "math" {
		t.Fatalf("imports = %#v", program.Imports)
	}
	if got := program.Imports[0].Names; len(got) != 2 || got[0] != "add" || got[1] != "clamp" {
		t.Fatalf("imported names = %#v", got)
	}
}

func TestParseListIndex(t *testing.T) {
	program, err := parser.ParseLegacy("def read():\n    items = [1, 2, 3]\n    value = items[1]\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	assignment := program.Functions[0].Body[1].(*ast.Assignment)
	index, ok := assignment.Value.(*ast.Index)
	if !ok {
		t.Fatalf("value = %T, want *ast.Index", assignment.Value)
	}
	if index.Target.(*ast.Identifier).Name != "items" || index.Index.(*ast.Integer).Value != 1 {
		t.Fatalf("index = %#v", index)
	}
}

func TestParseTypeTest(t *testing.T) {
	program, err := parser.ParseLegacy("namespace demo\n\ndef test():\n    result = value is bool\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	assignment := program.Functions[0].Body[0].(*ast.Assignment)
	binary, ok := assignment.Value.(*ast.Binary)
	if !ok || binary.Operator != token.Is || binary.Right.(*ast.Identifier).Name != "bool" {
		t.Fatalf("type test = %#v", assignment.Value)
	}
}

func TestParseModKeyword(t *testing.T) {
	program, err := parser.ParseLegacy("namespace demo\n\ndef test():\n    result = 14 mod 5\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	assignment := program.Functions[0].Body[0].(*ast.Assignment)
	binary, ok := assignment.Value.(*ast.Binary)
	if !ok || binary.Operator != token.Percent {
		t.Fatalf("mod expression = %#v", assignment.Value)
	}
}

func TestParseEntitySelectors(t *testing.T) {
	program, err := parser.ParseLegacy("namespace demo\n\ndef test():\n    target = @s\n    entities = [@s, @e[type=minecraft:pig,limit=1]]\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	first := program.Functions[0].Body[0].(*ast.Assignment).Value.(*ast.EntitySelector)
	if first.Value != "@s" {
		t.Fatalf("selector = %q", first.Value)
	}
	list := program.Functions[0].Body[1].(*ast.Assignment).Value.(*ast.List)
	if got := list.Elements[1].(*ast.EntitySelector).Value; got != "@e[type=minecraft:pig,limit=1]" {
		t.Fatalf("list selector = %q", got)
	}
}

func TestParseVersion2TypesConstReadonlySetAndNone(t *testing.T) {
	source := "version 2\nnamespace strict\n\nconst LIMIT: int = 5\n\ndef inspect(values: readonly list[int], target: entity?) -> None:\n    unique: set[str] = {\"a\", \"b\"}\n    missing: entity? = None\n    for value: str in unique:\n        say(value)\n"
	program, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if program.Version != 2 || len(program.Globals) != 1 || !program.Globals[0].Constant {
		t.Fatalf("program header/globals = %#v", program)
	}
	function := program.Functions[0]
	if function.ReturnType.String() != "None" || function.Types["values"].String() != "readonly list[int]" || function.Types["target"].String() != "entity?" {
		t.Fatalf("function types = %#v", function.Types)
	}
	declaration := function.Body[0].(*ast.Assignment)
	if declaration.DeclaredType.String() != "set[str]" {
		t.Fatalf("declaration type = %s", declaration.DeclaredType.String())
	}
	if _, ok := declaration.Value.(*ast.Set); !ok {
		t.Fatalf("set literal = %T", declaration.Value)
	}
}

func TestHeaderlessSourceDefaultsToVersion2(t *testing.T) {
	program, err := parser.Parse("def identity(value: int) -> int:\n    return value\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if program.Version != 2 {
		t.Fatalf("version = %d, want 2", program.Version)
	}
}

func TestHeaderlessSourceUsesStrictVersion2Grammar(t *testing.T) {
	_, err := parser.Parse("def legacy(value):\n    return value\n")
	if err == nil {
		t.Fatal("Parse() error = nil, want missing type error")
	}
}

func TestExplicitVersion1UsesLegacyGrammar(t *testing.T) {
	program, err := parser.Parse("version 1\ndef legacy(value):\n    return value\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if program.Version != 1 {
		t.Fatalf("version = %d, want 1", program.Version)
	}
}

func TestParseCollectionOperators(t *testing.T) {
	program, err := parser.Parse("def combine(a: set[int], b: set[int]) -> set[int]:\n    return a | b & a\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	union := program.Functions[0].Body[0].(*ast.Return).Value.(*ast.Binary)
	if union.Operator != token.Ampersand {
		t.Fatalf("outer operator = %s, want &", union.Operator)
	}
	if left := union.Left.(*ast.Binary); left.Operator != token.Pipe {
		t.Fatalf("left operator = %s, want |", left.Operator)
	}
}

func TestParseEntitySetLiteral(t *testing.T) {
	program, err := parser.Parse("def nearby() -> set[entity]:\n    entities: set[entity] = {@e[type=minecraft:pig]}\n    return entities\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	literal := program.Functions[0].Body[0].(*ast.Assignment).Value.(*ast.Set)
	if got := literal.Elements[0].(*ast.EntitySelector).Value; got != "@e[type=minecraft:pig]" {
		t.Fatalf("selector = %q", got)
	}
}

func TestParseDirectEntitySelectorForSet(t *testing.T) {
	program, err := parser.Parse("def nearby() -> set[entity]:\n    entities: set[entity] = @e[type=minecraft:pig]\n    return entities\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	selector := program.Functions[0].Body[0].(*ast.Assignment).Value.(*ast.EntitySelector)
	if selector.Value != "@e[type=minecraft:pig]" {
		t.Fatalf("selector = %q", selector.Value)
	}
}

func TestParseListItemAssignment(t *testing.T) {
	program, err := parser.ParseLegacy("def update(index):\n    items = [1, 2]\n    items[0] = 5\n    items[index] = 7\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	constant := program.Functions[0].Body[1].(*ast.Assignment)
	if constant.Name != "items" || constant.Index.(*ast.Integer).Value != 0 {
		t.Fatalf("constant assignment = %#v", constant)
	}
	dynamic := program.Functions[0].Body[2].(*ast.Assignment)
	if dynamic.Name != "items" || dynamic.Index.(*ast.Identifier).Name != "index" {
		t.Fatalf("dynamic assignment = %#v", dynamic)
	}
}

func TestParseForAndWhile(t *testing.T) {
	program, err := parser.ParseLegacy("def loops(items):\n    for item in items:\n        say(item)\n    while True:\n        return 1\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if _, ok := program.Functions[0].Body[0].(*ast.For); !ok {
		t.Fatalf("first statement = %T", program.Functions[0].Body[0])
	}
	if _, ok := program.Functions[0].Body[1].(*ast.While); !ok {
		t.Fatalf("second statement = %T", program.Functions[0].Body[1])
	}
}

func TestParseFunctionAndGlobalTypes(t *testing.T) {
	program, err := parser.ParseLegacy("def typed(value: int, text: str, items: list):\n    global title: str, count: int\n    return value\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	function := program.Functions[0]
	if function.ParameterTypes["value"] != "int" || function.ParameterTypes["text"] != "str" || function.ParameterTypes["items"] != "list" {
		t.Fatalf("parameter types = %#v", function.ParameterTypes)
	}
	global := function.Body[0].(*ast.Global)
	if global.Types["title"] != "str" || global.Types["count"] != "int" {
		t.Fatalf("global types = %#v", global.Types)
	}
}
