package semantic_test

import (
	"strings"
	"testing"

	"github.com/sirbuffalo/datacraft/parser"
	"github.com/sirbuffalo/datacraft/semantic"
)

func TestVersion2TypedProgram(t *testing.T) {
	source := `version 2
namespace strict

const LIMIT: int = 5

def total(values: readonly list[int]) -> int:
    result: int = 0
    for value: int in values:
        result += value
    return result

def main() -> None:
    values: list[int] = [1, 2, 3]
    unique: set[int] = {1, 2, 3}
    possible: entity? = @n[type=minecraft:pig]
    say(total(values), len(unique), possible is None)
`
	program, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if err = semantic.Check(program); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
}

func TestOmittedReturnTypeMeansNone(t *testing.T) {
	program, err := parser.Parse(`namespace demo

def load():
    say("loaded")
`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if err = semantic.Check(program); err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	checkError(t, `namespace demo

def invalid():
    return 5
`, "function returns int, not None")
}

func TestNBTCompoundsUseTypedBracketAccess(t *testing.T) {
	program, err := parser.Parse(`namespace demo

def load():
    data: nbt = {"name": "Alex", "health": 20, "nested": {"active": True}, "mixed": [1, "two"]}
    data["health"] = 18
    health: int = data["health"]
    nested: nbt = data["nested"]
    empty: nbt = {}
    is_compound: bool = data is nbt
    say(data)
`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if err = semantic.Check(program); err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	checkError(t, `def bad():
    data: nbt = {"value": 1}
    key: str = "value"
    result: int = data[key]
`, "NBT keys must be string literals")
	checkError(t, `def bad():
    data: nbt = {"missing": None}
`, "None and entities are not allowed")
}

func TestEntityNBTUsesTypedBracketReads(t *testing.T) {
	program, err := parser.Parse(`namespace demo

def inspect():
    target: entity? = @s
    uuid: list[int] = target["UUID"]
    health: int = target["Health"]
    name: str = target["CustomName"]
    memory: nbt = target["Brain"]["memories"]
    effects: list[nbt] = target["active_effects"]
`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if err = semantic.Check(program); err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	checkError(t, `def bad():
    target: entity? = @s
    key: str = "UUID"
    uuid: list[int] = target[key]
`, "entity NBT keys must be string literals")
}

func TestEntityNBTWritesRespectGlobal(t *testing.T) {
	checkError(t, `namespace demo

target: entity? = @s

def update():
    target["Health"] = 10
`, "requires 'global target'")
}

func TestVersion2NamespaceGlobalWritesRequireGlobal(t *testing.T) {
	checkError(t, `namespace strict

counter: int = 0

def update() -> None:
    counter = 1
`, "requires 'global counter'")

	program, err := parser.Parse(`namespace strict

counter: int = 0

def update() -> None:
    global counter
    counter = 1
`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if err = semantic.Check(program); err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	checkError(t, `namespace strict

counter: int = 0

def update() -> None:
    global missing
`, "namespace global \"missing\" is not declared")
}

func TestVersion2RejectsMixedList(t *testing.T) {
	checkError(t, `version 2
namespace strict

def main() -> None:
    values: list[int] = [1, "two"]
`, "list element is str, expected int")
}

func TestVersion2RejectsConstAndReadonlyMutation(t *testing.T) {
	checkError(t, `version 2
namespace strict

def main() -> None:
    const values: list[int] = [1]
    values.append(2)
`, "cannot mutate constant collection")

	checkError(t, `version 2
namespace strict

def inspect(values: readonly list[int]) -> None:
    values.remove()
`, "cannot mutate readonly collection")
}

func TestVersion2RequiresSingularEntitySelector(t *testing.T) {
	checkError(t, `version 2
namespace strict

def main() -> None:
    target: entity? = @e[type=minecraft:pig]
`, "may select multiple entities")
}

func TestVersion2CollectionAndStringOperators(t *testing.T) {
	source := `namespace strict

def combine(words: list[str], more: list[str], left: set[int], right: set[int]) -> None:
    text: str = "hello" + " world"
    all_words: list[str] = words + more
    union: set[int] = left | right
    intersection: set[int] = left & right
`
	program, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if err = semantic.Check(program); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
}

func TestVersion2RejectsWrongJoinOperators(t *testing.T) {
	checkError(t, `def bad(left: set[int], right: set[int]) -> None:
    value: set[int] = left + right
`, "arithmetic requires int or bool operands")

	checkError(t, `def bad(left: list[int], right: list[str]) -> None:
    value: list[int] = left + right
`, "+ requires lists with the same element type")

	checkError(t, `def bad(left: str, right: str) -> None:
    value: str = left & right
`, "& requires sets with the same element type")
}

func TestVersion2EntitySetOperations(t *testing.T) {
	source := `def operate(group: set[entity], item: entity) -> int:
    group.add(item)
    present: bool = item in group
    group.discard(item)
    group.remove(item)
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
	if err = semantic.Check(program); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
}

func checkError(t *testing.T, source, wanted string) {
	t.Helper()
	program, err := parser.Parse(source)
	if err == nil {
		err = semantic.Check(program)
	}
	if err == nil || !strings.Contains(err.Error(), wanted) {
		t.Fatalf("error = %v, want containing %q", err, wanted)
	}
}
