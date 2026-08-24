package compiler_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sirbuffalo/datacraft/compiler"
	"github.com/sirbuffalo/datacraft/parser"
)

func TestCompileConstantsAssignmentsAndExpressions(t *testing.T) {
	source := "namespace demo\n\ndef calculate(a, b):\n    total = a + b * 5\n    total -= 2\n    return total\n"
	program, err := parser.ParseLegacy(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "demo")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	wantLoad := []string{
		"scoreboard objectives add demo dummy",
		"scoreboard players set #c2 demo 2",
		"scoreboard players set #c5 demo 5",
	}
	if !reflect.DeepEqual(output.Load, wantLoad) {
		t.Fatalf("load = %#v, want %#v", output.Load, wantLoad)
	}

	wantFunction := []string{
		"scoreboard players operation #t0 demo = #v1 demo",
		"scoreboard players operation #t0 demo *= #c5 demo",
		"scoreboard players operation #t1 demo = #v0 demo",
		"scoreboard players operation #t1 demo += #t0 demo",
		"scoreboard players operation #v2 demo = #t1 demo",
		"scoreboard players operation #v2 demo -= #c2 demo",
		"scoreboard players operation #r0 demo = #v2 demo",
		"return run scoreboard players get #r0 demo",
	}
	if !reflect.DeepEqual(output.Functions["_0"], wantFunction) {
		t.Fatalf("function = %#v, want %#v", output.Functions["_0"], wantFunction)
	}
	if output.FunctionNames["calculate"] != "_0" {
		t.Fatalf("function mapping = %#v", output.FunctionNames)
	}
	if len(output.Variables) != 3 || output.Variables[2].Name != "total" || output.Variables[2].ID != 2 {
		t.Fatalf("variable mapping = %#v", output.Variables)
	}
}

func TestCompileGlobalAssignment(t *testing.T) {
	program, err := parser.ParseLegacy("def update():\n    global counter\n    counter = 5\n    counter += 1\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "pack")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	want := []string{
		"scoreboard players operation #v0 pack = #c5 pack",
		"scoreboard players operation #v0 pack += #c1 pack",
	}
	if !reflect.DeepEqual(output.Functions["_0"], want) {
		t.Fatalf("function = %#v, want %#v", output.Functions["_0"], want)
	}
}

func TestCompileVersion2TypedNamespaceGlobals(t *testing.T) {
	source := `namespace globals

const LIMIT: int = 5
message: str = "ready"
values: list[int] = [1, 2]
unique: set[int] = {2, 3}
target: entity? = None

def update() -> int:
    global message, values, unique, target
    message = "running"
    values.append(LIMIT)
    unique.add(LIMIT)
    target = @n[type=minecraft:pig]
    return LIMIT
`
	program, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "globals")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	load := strings.Join(output.Load, "\n")
	for _, wanted := range []string{
		"scoreboard objectives add globals dummy",
		"scoreboard players set #c5 globals 5",
		"scoreboard players operation #v0 globals = #c5 globals",
		`data modify storage globals:data strings.v1 set value "ready"`,
		"data modify storage globals:data lists.v2 set value []",
		"data remove storage globals:data entities.v4",
	} {
		if !strings.Contains(load, wanted) {
			t.Fatalf("load is missing %q:\n%s", wanted, load)
		}
	}
	update := strings.Join(output.Functions["_0"], "\n")
	for _, wanted := range []string{
		`data modify storage globals:data strings.v1 set value "running"`,
		"data modify storage globals:data lists.v2 append value 0",
		"execute as @n[type=minecraft:pig] run function globals:",
		"return run scoreboard players get #r0 globals",
	} {
		if !strings.Contains(update, wanted) {
			t.Fatalf("update is missing %q:\n%s", wanted, update)
		}
	}
}

func TestCompileComparisonAndBooleanExpression(t *testing.T) {
	program, err := parser.ParseLegacy("def test(a, b):\n    result = a < b and not False\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "pack")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(output.Functions["_0"]) == 0 {
		t.Fatal("comparison generated no commands")
	}
}

func TestCompileFunctionMappingsPreserveExpose(t *testing.T) {
	program, err := parser.ParseLegacy("expose def public():\n    return 1\n\ndef private():\n    return 2\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "pack")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if !output.FunctionMappings[0].Exposed || output.FunctionMappings[1].Exposed {
		t.Fatalf("function mappings = %#v", output.FunctionMappings)
	}
	if output.FunctionMappings[0].GeneratedName != "_0" || output.FunctionMappings[1].GeneratedName != "_1" {
		t.Fatalf("function mappings = %#v", output.FunctionMappings)
	}
}

func TestCompileInternalFunctionCall(t *testing.T) {
	source := "def add(a, b):\n    return a + b\n\ndef main():\n    result = add(2, 3)\n"
	program, err := parser.ParseLegacy(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "pack")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	want := []string{
		"scoreboard players operation #t1 pack = #c2 pack",
		"scoreboard players operation #t2 pack = #c3 pack",
		"scoreboard players operation #v0 pack = #t1 pack",
		"scoreboard players operation #v1 pack = #t2 pack",
		"function pack:_0",
		"scoreboard players operation #v2 pack = #r0 pack",
	}
	if !reflect.DeepEqual(output.Functions["_1"], want) {
		t.Fatalf("main function = %#v, want %#v", output.Functions["_1"], want)
	}
}

func TestNamespaceOwnsScoreboard(t *testing.T) {
	program, err := parser.ParseLegacy("namespace combat\n\ndef load():\n    damage = 5\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "bundle")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if output.ScoreboardName != "combat" {
		t.Fatalf("scoreboard = %q, want combat", output.ScoreboardName)
	}
	if output.Load[0] != "scoreboard objectives add combat dummy" {
		t.Fatalf("load command = %q", output.Load[0])
	}
	if output.Functions["_0"][0] != "scoreboard players operation #v0 combat = #c5 combat" {
		t.Fatalf("function = %#v", output.Functions["_0"])
	}
}

func TestCompileIfElifElseAndBooleans(t *testing.T) {
	source := "namespace demo\n\ndef test(value):\n    enabled = True\n    disabled = False\n    if value:\n        result = 1\n    elif enabled:\n        result = 2\n    else:\n        result = 3\n"
	program, err := parser.ParseLegacy(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "demo")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	wantLoad := []string{
		"scoreboard objectives add demo dummy",
		"scoreboard players set #c0 demo 0",
		"scoreboard players set #c1 demo 1",
		"scoreboard players set #c2 demo 2",
		"scoreboard players set #c3 demo 3",
	}
	if !reflect.DeepEqual(output.Load, wantLoad) {
		t.Fatalf("load = %#v, want %#v", output.Load, wantLoad)
	}
	main := output.Functions["_0"]
	if !contains(main, "scoreboard players operation #v1 demo = #c1 demo") {
		t.Fatalf("True assignment did not use 1: %#v", main)
	}
	if !contains(main, "scoreboard players operation #v2 demo = #c0 demo") {
		t.Fatalf("False assignment did not use 0: %#v", main)
	}
	if !contains(main, "execute if score #t0 demo matches 0 unless score #v0 demo matches 0 run function demo:_1") {
		t.Fatalf("if did not use nonzero truthiness: %#v", main)
	}
	if !contains(main, "execute if score #t0 demo matches 0 run function demo:_3") {
		t.Fatalf("else helper was not generated: %#v", main)
	}
	if len(output.Functions["_1"]) == 0 || len(output.Functions["_2"]) == 0 || len(output.Functions["_3"]) == 0 {
		t.Fatalf("branch helpers = %#v", output.Functions)
	}
}

func TestInternalFunctionsReturnExactNumbers(t *testing.T) {
	program, err := parser.ParseLegacy("namespace logic\n\ndef one():\n    return True\n\ndef two():\n    return 2\n\ndef three():\n    return 3\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "logic")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	want := map[string][]string{
		"_0": {"scoreboard players operation #r0 logic = #c1 logic", "return run scoreboard players get #r0 logic"},
		"_1": {"scoreboard players operation #r1 logic = #c2 logic", "return run scoreboard players get #r1 logic"},
		"_2": {"scoreboard players operation #r2 logic = #c3 logic", "return run scoreboard players get #r2 logic"},
	}
	for name, commands := range want {
		if !reflect.DeepEqual(output.Functions[name], commands) {
			t.Fatalf("%s = %#v, want %#v", name, output.Functions[name], commands)
		}
	}
}

func TestReturnInsideIfElifElsePropagates(t *testing.T) {
	program, err := parser.ParseLegacy("namespace logic\n\ndef choose(value):\n    if value == 1:\n        return 1\n    elif value == 2:\n        return 2\n    else:\n        return 3\n    return 0\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "logic")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	main := output.Functions["_0"]
	if main[0] != "scoreboard players set #rf0 logic 0" {
		t.Fatalf("return flag was not reset: %#v", main)
	}
	propagate := "execute if score #rf0 logic matches 1 run return run scoreboard players get #r0 logic"
	count := 0
	for _, command := range main {
		if command == propagate {
			count++
		}
	}
	if count != 3 {
		t.Fatalf("return propagation count = %d, want 3: %#v", count, main)
	}
	for _, helper := range []string{"_1", "_2", "_3"} {
		if !contains(output.Functions[helper], "scoreboard players set #rf0 logic 1") {
			t.Fatalf("%s does not signal a return: %#v", helper, output.Functions[helper])
		}
	}
}

func TestNestedConditionalReturnPropagatesThroughEveryHelper(t *testing.T) {
	program, err := parser.ParseLegacy("namespace logic\n\ndef nested(a, b):\n    if a:\n        if b:\n            return 3\n    return 0\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "logic")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	propagate := "execute if score #rf0 logic matches 1 run return run scoreboard players get #r0 logic"
	if !contains(output.Functions["_0"], propagate) {
		t.Fatalf("outer function does not propagate: %#v", output.Functions["_0"])
	}
	if !contains(output.Functions["_1"], propagate) {
		t.Fatalf("outer helper does not propagate: %#v", output.Functions["_1"])
	}
}

func TestListsUseDataStorageAndItemsCanBeLoadedIntoScores(t *testing.T) {
	program, err := parser.ParseLegacy("namespace inventory\n\ndef read(index):\n    items = [1, index + 2, 3]\n    first = items[0]\n    selected = items[index]\n    return selected\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "pack")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if !contains(output.Load, "data modify storage inventory:data lists set value {}") {
		t.Fatalf("storage was not initialized: %#v", output.Load)
	}
	main := output.Functions["_0"]
	if !contains(main, "data modify storage inventory:data lists.v1 set value []") {
		t.Fatalf("list was not created in storage: %#v", main)
	}
	if !contains(main, "execute store result storage inventory:data lists.v1[-1] int 1 run scoreboard players get #c1 inventory") {
		t.Fatalf("literal item was not stored: %#v", main)
	}
	if !contains(main, "execute store result score #t1 inventory run data get storage inventory:data lists.v1[0] 1") {
		t.Fatalf("literal-index item was not loaded into a score: %#v", main)
	}
	if !contains(main, "execute store result storage inventory:data scratch.index0 int 1 run scoreboard players get #v0 inventory") {
		t.Fatalf("dynamic index was not prepared: %#v", main)
	}
	if !contains(main, "function pack:_1 with storage inventory:data scratch") {
		t.Fatalf("dynamic item loader was not called: %#v", main)
	}
	wantMacro := "$execute store result score #t2 inventory run data get storage inventory:data lists.v1[$(index0)] 1"
	if !contains(output.Functions["_1"], wantMacro) {
		t.Fatalf("dynamic item loader = %#v, want %q", output.Functions["_1"], wantMacro)
	}
}

func TestListItemsCanBeAssigned(t *testing.T) {
	program, err := parser.ParseLegacy("namespace inventory\n\ndef update(index, amount):\n    items = [1, 2, 3]\n    items[0] = 5\n    items[index] = amount + 1\n    return items[0]\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "pack")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	main := output.Functions["_0"]
	constantWrite := "execute store result storage inventory:data lists.v2[0] int 1 run scoreboard players get #c5 inventory"
	if !contains(main, constantWrite) {
		t.Fatalf("constant-index write missing: %#v", main)
	}
	if !contains(main, "execute store result storage inventory:data scratch.index0 int 1 run scoreboard players get #v0 inventory") {
		t.Fatalf("dynamic index was not stored: %#v", main)
	}
	if !contains(main, "function pack:_1 with storage inventory:data scratch") {
		t.Fatalf("dynamic writer was not called: %#v", main)
	}
	wantMacro := "$execute store result storage inventory:data lists.v2[$(index0)] int 1 run scoreboard players get #t0 inventory"
	if !contains(output.Functions["_1"], wantMacro) {
		t.Fatalf("dynamic writer = %#v, want %q", output.Functions["_1"], wantMacro)
	}
}

func TestListsCanBeFunctionInputsAndOutputs(t *testing.T) {
	source := "namespace inventory\n\ndef first(items):\n    return items[0]\n\ndef passthrough(items):\n    return items\n\ndef main():\n    source = [4, 5]\n    result = passthrough(source)\n    return first(result)\n"
	program, err := parser.ParseLegacy(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "pack")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if !output.FunctionMappings[0].Parameters[0].IsList || !output.FunctionMappings[1].Parameters[0].IsList {
		t.Fatalf("list parameters were not inferred: %#v", output.FunctionMappings)
	}
	if !output.FunctionMappings[1].ReturnsList {
		t.Fatalf("list return was not inferred: %#v", output.FunctionMappings[1])
	}
	passthrough := output.Functions["_1"]
	if !contains(passthrough, "data modify storage inventory:data returns.r1 set from storage inventory:data lists.v1") {
		t.Fatalf("list output was not stored: %#v", passthrough)
	}
	if !contains(passthrough, "return run scoreboard players get #r1 inventory") {
		t.Fatalf("list ID was not returned with /return: %#v", passthrough)
	}
	main := output.Functions["_2"]
	if !contains(main, "data modify storage inventory:data lists.v1 set from storage inventory:data lists.v2") {
		t.Fatalf("list input was not copied: %#v", main)
	}
	if !contains(main, "data modify storage inventory:data lists.v3 set from storage inventory:data returns.r1") {
		t.Fatalf("list output was not received: %#v", main)
	}
}

func TestSayNumbersBooleansExpressionsAndLists(t *testing.T) {
	source := "namespace demo\n\ndef show(value):\n    items = [1, value]\n    say(value + 2)\n    say(False)\n    say(items)\n    say([True, 3])\n"
	program, err := parser.ParseLegacy(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "pack")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	commands := output.Functions["_0"]
	if !contains(commands, "tellraw @a [{\"score\":{\"name\":\"#t0\",\"objective\":\"demo\"}}]") {
		t.Fatalf("expression was not displayed from its score: %#v", commands)
	}
	if !contains(commands, "tellraw @a [{\"score\":{\"name\":\"#c0\",\"objective\":\"demo\"}}]") {
		t.Fatalf("boolean was not displayed as 0/1: %#v", commands)
	}
	if !contains(commands, "tellraw @a [{\"nbt\":\"lists.v1\",\"storage\":\"demo:data\"}]") {
		t.Fatalf("list variable was not displayed from storage: %#v", commands)
	}
	if !contains(commands, "tellraw @a [{\"nbt\":\"scratch.say1\",\"storage\":\"demo:data\"}]") {
		t.Fatalf("list expression was not displayed from storage: %#v", commands)
	}
}

func TestSayJoinsStringsAndDynamicValues(t *testing.T) {
	program, err := parser.ParseLegacy("namespace demo\n\ndef show(value):\n    items = [1, 2]\n    say(\"test: \", value + 5, \" items: \", items)\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "pack")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	want := "tellraw @a [{\"text\":\"test: \"},{\"score\":{\"name\":\"#t0\",\"objective\":\"demo\"}},{\"text\":\" items: \"},{\"nbt\":\"lists.v1\",\"storage\":\"demo:data\"}]"
	if !contains(output.Functions["_0"], want) {
		t.Fatalf("joined say command missing: %#v", output.Functions["_0"])
	}
}

func TestSayRequiresOneArgumentAndHasNoValue(t *testing.T) {
	program, err := parser.ParseLegacy("def bad():\n    result = say(1)\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	_, err = compiler.Compile(program, "pack")
	if err == nil || !strings.Contains(err.Error(), "say does not return a value") {
		t.Fatalf("Compile() error = %v", err)
	}
}

func TestForListRangeAndWhileCompileRecursively(t *testing.T) {
	source := "namespace loops\n\ndef test():\n    total = 0\n    items = [1, 2, 3]\n    for item in items:\n        total += item\n    for number in range(1, 4):\n        total += number\n    counter = 0\n    while counter < 3:\n        counter += 1\n    return total + counter\n"
	program, err := parser.ParseLegacy(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "loops")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	main := output.Functions["_0"]
	if !containsPrefix(main, "execute store result score ") {
		t.Fatalf("list length was not loaded: %#v", main)
	}
	recursive := 0
	for name, commands := range output.Functions {
		for _, command := range commands {
			if strings.HasSuffix(command, "run function loops:"+name) {
				recursive++
			}
		}
	}
	if recursive < 3 {
		t.Fatalf("recursive loop helpers = %d, functions=%#v", recursive, output.Functions)
	}
}

func TestReturnInsideWhilePropagates(t *testing.T) {
	program, err := parser.ParseLegacy("namespace loops\n\ndef find():\n    value = 0\n    while value < 3:\n        value += 1\n        if value == 2:\n            return value\n    return 0\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "loops")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	propagate := "execute if score #rf0 loops matches 1 run return run scoreboard players get #r0 loops"
	found := false
	for _, commands := range output.Functions {
		if contains(commands, propagate) {
			found = true
		}
	}
	if !found {
		t.Fatalf("while return propagation missing: %#v", output.Functions)
	}
}

func TestBreakContinueLenAppendInsertAndRemove(t *testing.T) {
	source := "namespace lists\n\ndef test():\n    items = [1, 2]\n    items.append(3)\n    items.insert(1, 8)\n    index = 2\n    items.insert(index, 9)\n    length = len(items)\n    removed = items.remove(1)\n    last = items.remove()\n    total = 0\n    for item in items:\n        if item == 9:\n            continue\n        if item == 3:\n            break\n        total += item\n    while True:\n        break\n    return length + removed + last + total\n"
	program, err := parser.ParseLegacy(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "lists")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	all := []string{}
	for _, commands := range output.Functions {
		all = append(all, commands...)
	}
	if !containsPrefix(all, "data modify storage lists:data lists.v0 append value 0") {
		t.Fatalf("append missing: %#v", all)
	}
	if !contains(all, "data modify storage lists:data lists.v0 insert 1 value 0") {
		t.Fatalf("constant insert missing: %#v", all)
	}
	if !containsSubstring(all, "insert $(index) value 0") {
		t.Fatalf("dynamic insert missing: %#v", all)
	}
	if !containsPrefix(all, "execute store result score ") {
		t.Fatalf("len/remove reads missing: %#v", all)
	}
	if !contains(all, "data remove storage lists:data lists.v0[1]") || !contains(all, "data remove storage lists:data lists.v0[-1]") {
		t.Fatalf("remove missing: %#v", all)
	}
	breakCommands := 0
	for _, command := range all {
		if strings.Contains(command, "matches 1 run return 0") {
			breakCommands++
		}
	}
	if breakCommands < 2 {
		t.Fatalf("break propagation missing: %#v", all)
	}
}

func TestNestedListsCanBeReadWrittenAndDisplayed(t *testing.T) {
	source := "namespace nested\n\ndef test():\n    matrix = [[1, 2], [3, 4]]\n    value = matrix[1][0]\n    say(matrix[0])\n    matrix[0][1] = 8\n    matrix[1] = [9, 10]\n    return matrix[1][0] + value\n"
	program, err := parser.ParseLegacy(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "nested")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	all := []string{}
	for _, commands := range output.Functions {
		all = append(all, commands...)
	}
	for _, wanted := range []string{
		"data modify storage nested:data lists.v0[-1] set value []",
		"execute store result score #t0 nested run data get storage nested:data lists.v0[1][0] 1",
		"data modify storage nested:data scratch.say1 set from storage nested:data lists.v0[0]",
		"execute store result storage nested:data lists.v0[0][1] int 1 run scoreboard players get #c8 nested",
		"data modify storage nested:data lists.v0[1] set value []",
	} {
		if !contains(all, wanted) {
			t.Fatalf("missing %q in %#v", wanted, all)
		}
	}
}

func TestStringStorageDisplayAndComparisonUsesTemporaryObjectives(t *testing.T) {
	source := "namespace words\n\ndef test():\n    left = \"hello\"\n    right = \"hello\"\n    equal = left == right\n    right = \"world\"\n    different = left != right\n    copied = left\n    say(\"value: \", copied, \" equal: \", equal, \" different: \", different)\n"
	program, err := parser.ParseLegacy(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "words")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if !contains(output.Load, "data modify storage words:data strings set value {}") {
		t.Fatalf("string storage not initialized: %#v", output.Load)
	}
	commands := output.Functions["_0"]
	for _, wanted := range []string{
		"data modify storage words:data strings.v0 set value \"hello\"",
		"scoreboard objectives add string_0 dummy",
		"execute store success score #compare string_0 run data modify storage words:data scratch.string_0.left set from storage words:data scratch.string_0.right",
		"scoreboard objectives remove string_0",
		"scoreboard objectives add string_1 dummy",
		"scoreboard objectives remove string_1",
		"data modify storage words:data strings.v4 set from storage words:data strings.v0",
	} {
		if !contains(commands, wanted) {
			t.Fatalf("missing %q in %#v", wanted, commands)
		}
	}
	if !containsSubstring(commands, "{\"nbt\":\"strings.v4\",\"storage\":\"words:data\",\"interpret\":false}") {
		t.Fatalf("string say component missing: %#v", commands)
	}
}

func TestStringJoinStrAndTypedInputsAndGlobals(t *testing.T) {
	source := "namespace typed\n\ndef format(value: int, label: str, items: list):\n    global prefix: str\n    prefix = \"P\"\n    combined = prefix + label + str(value)\n    same = combined == \"Px5\"\n    say(combined)\n    return same + len(items)\n\ndef main():\n    values = [1, 2]\n    return format(5, \"x\", values)\n"
	program, err := parser.ParseLegacy(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "typed")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	mapping := output.FunctionMappings[0]
	if mapping.Parameters[0].IsList || mapping.Parameters[0].IsString || !mapping.Parameters[1].IsString || !mapping.Parameters[2].IsList {
		t.Fatalf("typed parameters = %#v", mapping.Parameters)
	}
	all := []string{}
	for _, commands := range output.Functions {
		all = append(all, commands...)
	}
	if !containsSubstring(all, "set value \"$(value)\"") {
		t.Fatalf("str conversion macro missing: %#v", all)
	}
	if !containsSubstring(all, "set value \"$(left)$(right)\"") {
		t.Fatalf("string join macro missing: %#v", all)
	}
	if !containsSubstring(all, "strings.v") || !containsSubstring(all, "lists.v") {
		t.Fatalf("typed call storage copies missing: %#v", all)
	}
}

func TestTypeTestsBoolConversionAndMixedLists(t *testing.T) {
	source := "namespace types\n\ndef main():\n    number = 2\n    zero = 0\n    text = \"hello\"\n    empty = \"\"\n    items = [1, \"two\", [3]]\n    numeric = number is int\n    not_bool = number is bool\n    bool_value = zero is bool\n    string_value = text is str\n    list_value = items is list\n    first_type = items[0] is int\n    second_type = items[1] is str\n    items.append(\"four\")\n    items.insert(1, \"inserted\")\n    say(items[1], items)\n    return numeric + not_bool + bool_value + string_value + list_value + first_type + second_type + bool(number) + bool(zero) + bool(text) + bool(empty)\n"
	program, err := parser.ParseLegacy(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "types")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	all := []string{}
	for _, commands := range output.Functions {
		all = append(all, commands...)
	}
	for _, wanted := range []string{"matches 0..1", "append value \"\"", "insert 1 value \"\"", "lists.v"} {
		if !containsSubstring(all, wanted) {
			t.Fatalf("missing %q in %#v", wanted, all)
		}
	}
}

func TestDynamicMixedListRuntime(t *testing.T) {
	source := "namespace mixed\n\ndef main():\n    items = [1, \"two\", [3]]\n    index = 1\n    picked = items[index]\n    say(items[index], picked)\n    string_type = items[index] is str\n    for item in items:\n        say(item, item is int, item is str, item is list)\n    removed = items.remove(index)\n    say(removed)\n    return string_type + bool([]) + bool([1])\n"
	program, err := parser.ParseLegacy(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "mixed")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	all := []string{}
	for _, commands := range output.Functions {
		all = append(all, commands...)
	}
	for _, wanted := range []string{"list_types.v", "variant_types.v", "variants.v", "scratch.runtime_type", "data remove storage mixed:data list_types.v"} {
		if !containsSubstring(all, wanted) {
			t.Fatalf("missing %q in %#v", wanted, all)
		}
	}
}

func TestEntityVariablesAndListsUseUUIDObjects(t *testing.T) {
	source := "namespace entities\n\ndef main():\n    target = @s\n    copy = target\n    values = [@s]\n    values.append(@s)\n    values.insert(1, @s)\n    selected = values[0]\n    valid = selected is entity\n    for current in values:\n        say(current)\n    return valid\n"
	program, err := parser.ParseLegacy(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "entities")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	all := []string{}
	for _, commands := range output.Functions {
		all = append(all, commands...)
	}
	for _, wanted := range []string{"UUID", "{type:\"entity\",uuid:[I;0,0,0,0]}", "$tag @s add _entities_$(uuid0)_$(uuid1)_$(uuid2)_$(uuid3)", " add _entities_list_", "list_types.v", "entities.v"} {
		if !containsSubstring(all, wanted) {
			t.Fatalf("missing %q in %#v", wanted, all)
		}
	}
	if !containsSubstring(output.Tick, "_special/entity_list_") {
		t.Fatalf("entity-list tick synchronization missing: %#v", output.Tick)
	}
}

func TestCompileVersion2TypedNumericSubset(t *testing.T) {
	source := "version 2\nnamespace strict\n\ndef add(a: int, b: int) -> int:\n    total: int = a + b\n    return total\n"
	program, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "strict")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if !containsSubstring(output.Functions["_0"], "scoreboard players operation") {
		t.Fatalf("typed function commands = %#v", output.Functions["_0"])
	}
}

func TestCompileVersion2RunsSemanticChecks(t *testing.T) {
	source := "version 2\nnamespace strict\n\ndef main() -> None:\n    value = 5\n"
	program, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	_, err = compiler.Compile(program, "strict")
	if err == nil || !strings.Contains(err.Error(), "must be declared with a type") {
		t.Fatalf("Compile() error = %v", err)
	}
}

func TestCompileVersion2EntitySetUnionIntersectionParametersAndReturn(t *testing.T) {
	source := `namespace strict

def combine(left: set[entity], right: set[entity]) -> set[entity]:
    union: set[entity] = left | right
    shared: set[entity] = left & right
    return union

def use(first: set[entity], second: set[entity]) -> None:
    result: set[entity] = combine(first, second)

def nearby() -> set[entity]:
    entities: set[entity] = @e[type=minecraft:pig]
    return entities
`
	program, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "strict")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	all := append([]string{}, output.Functions["_0"]...)
	all = append(all, output.Functions["_1"]...)
	all = append(all, output.Functions["_2"]...)
	for _, wanted := range []string{
		"tag @e[tag=_strict_set_0] add _strict_set_tmp_0",
		"tag @e[tag=_strict_set_tmp_0] add _strict_set_2",
		"tag @e[tag=_strict_set_tmp_2,tag=_strict_set_tmp_3] add _strict_set_3",
		"tag @e[tag=_strict_set_2] add _strict_return_set_0",
		"tag @e[tag=_strict_return_set_0] add _strict_set_6",
		"tag @e[type=minecraft:pig] add _strict_set_7",
	} {
		if !containsSubstring(all, wanted) {
			t.Fatalf("missing entity-set command %q in %#v", wanted, all)
		}
	}
	if !output.FunctionMappings[0].ReturnsEntitySet || !output.FunctionMappings[0].Parameters[0].IsEntitySet {
		t.Fatalf("entity-set mapping = %#v", output.FunctionMappings[0])
	}
}

func TestCompileVersion2EntitySetOperations(t *testing.T) {
	source := `namespace strict

def operate(group: set[entity], item: entity) -> int:
    group.add(item)
    present: bool = item in group
    group.discard(item)
    group.add(@s)
    group.remove(@s)
    count: int = len(group)
    for found: entity in group:
        say(found)
    group.clear()
    return count + present
`
	program, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "strict")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	all := []string{}
	for _, commands := range output.Functions {
		all = append(all, commands...)
	}
	for _, wanted := range []string{
		"tag @n[tag=_strict_$(uuid0)_$(uuid1)_$(uuid2)_$(uuid3)] add _strict_set_0",
		"if entity @s[tag=_strict_set_0] run scoreboard players set",
		"tag @s add _strict_set_0",
		"tag @s remove _strict_set_0",
		"execute as @e[tag=_strict_set_0] run scoreboard players add",
		"execute as @e[tag=_strict_set_0] if score",
		"tag @e[tag=_strict_set_0] remove _strict_set_0",
	} {
		if !containsSubstring(all, wanted) {
			t.Fatalf("missing entity-set operation %q in %#v", wanted, all)
		}
	}
}

func TestCompileVersion2PrimitiveSetRuntime(t *testing.T) {
	source := `namespace strict

def combine(left: set[int], right: set[int]) -> set[int]:
    union: set[int] = left | right
    shared: set[int] = left & right
    union.add(5)
    union.add(5)
    union.remove(5)
    union.add(5)
    present: bool = 5 in union
    for value: int in shared:
        say(value)
    return union

def use() -> None:
    numbers: set[int] = {1, 2, 3}
    other: set[int] = {3, 4}
    result: set[int] = combine(numbers, other)
    flags: set[bool] = {True, False}
    words: set[str] = {"a", "b"}
    words.add("c")
    words.discard("a")
    has_b: bool = "b" in words
    size: int = len(result)
`
	program, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "strict")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	all := []string{}
	for _, commands := range output.Functions {
		all = append(all, commands...)
	}
	for _, wanted := range []string{
		"items append value {key:\"\",value:0,generation:0}",
		"scratch.set_result.values merge from storage strict:data sets.v1.values",
		"unless data storage strict:data sets.v1.values.\"$(key)\" run data remove",
		"scratch.set_item.generation",
		"set_returns.r0 set from storage strict:data sets.v2",
		"sets.v8 set from storage strict:data set_returns.r0",
		"sets.v10.values.\"b\"",
	} {
		if !containsSubstring(all, wanted) {
			t.Fatalf("missing primitive-set command %q in %#v", wanted, all)
		}
	}
	if !output.FunctionMappings[0].ReturnsPrimitiveSet || !output.FunctionMappings[0].Parameters[0].IsPrimitiveSet {
		t.Fatalf("primitive set mapping = %#v", output.FunctionMappings[0])
	}
}

func TestCompileVersion2ListConcatenation(t *testing.T) {
	source := `namespace strict

def combine(left: list[int], right: list[int]) -> list[int]:
    result: list[int] = left + right + [5]
    result += left
    return result

def use() -> None:
    first: list[int] = [1, 2]
    second: list[int] = [3, 4]
    combined: list[int] = combine(first, second) + first

def nested(a: list[list[int]], b: list[list[int]]) -> list[list[int]]:
    result: list[list[int]] = a + b
    return result
`
	program, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "strict")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	all := []string{}
	for _, commands := range output.Functions {
		all = append(all, commands...)
	}
	for _, wanted := range []string{
		"append from storage strict:data lists.v0[]",
		"append from storage strict:data list_types.v0[]",
		"lists.scratch_leaf_",
		"return_types.r0 set from storage strict:data list_types.v2",
		"append from storage strict:data returns.r0[]",
		"append from storage strict:data return_types.r0[]",
		"data modify storage strict:data lists.v5 set from storage strict:data scratch.list_concat_",
	} {
		if !containsSubstring(all, wanted) {
			t.Fatalf("missing list concatenation command %q in %#v", wanted, all)
		}
	}
}

func TestCompileNBTCompoundsFieldsAndFunctions(t *testing.T) {
	source := `namespace demo

def identity(value: nbt) -> nbt:
    return value

def load():
    data: nbt = {"name": "Alex", "health": 20, "nested": {"active": True}, "mixed": [1, "two"]}
    data["health"] = 18
    health: int = data["health"]
    nested: nbt = data["nested"]
    copy: nbt = identity(data)
    say(copy)
`
	program, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "demo")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	all := []string{}
	for _, commands := range output.Functions {
		all = append(all, commands...)
	}
	for _, wanted := range []string{
		`nbt.v1 set value {}`,
		`nbt.v1."name" set value "Alex"`,
		`nbt.v1."nested"."active" set value 1b`,
		`nbt.v1."mixed" set value []`,
		`data get storage demo:data nbt.v1."health" 1`,
		`nbt_returns.r0 set from storage demo:data nbt.v0`,
		`nbt.v4 set from storage demo:data nbt_returns.r0`,
		`{"nbt":"nbt.v4","storage":"demo:data"}`,
	} {
		if !containsSubstring(all, wanted) {
			t.Fatalf("missing NBT command %q in %#v", wanted, all)
		}
	}
}

func containsSubstring(commands []string, wanted string) bool {
	for _, command := range commands {
		if strings.Contains(command, wanted) {
			return true
		}
	}
	return false
}

func containsPrefix(commands []string, prefix string) bool {
	for _, command := range commands {
		if strings.HasPrefix(command, prefix) {
			return true
		}
	}
	return false
}

func contains(commands []string, wanted string) bool {
	for _, command := range commands {
		if command == wanted {
			return true
		}
	}
	return false
}
