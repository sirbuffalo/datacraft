package basic_test

import (
	"testing"

	"mccomp/compiler"
	"mccomp/parser"
)

const source = `namespace example

def load():
    global counter
    counter = 0
    limit = 5
    doubled = limit * 2
    result = doubled + 1
    /say Example pack loaded
    return result

def tick():
    global counter
    counter += 1
    reached_limit = counter >= 5
    return reached_limit

export def reset():
    global counter
    counter = 0
`

func TestExampleCompiles(t *testing.T) {
	program, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := compiler.Compile(program, "example")
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(output.Load) == 0 {
		t.Fatal("generated load function is empty")
	}
	if len(output.Functions["_0"]) == 0 || len(output.Functions["_1"]) == 0 {
		t.Fatal("example functions were not generated")
	}
	if !output.FunctionMappings[2].Exported || output.FunctionMappings[2].GeneratedName != "_2" {
		t.Fatal("exported reset function was not mapped")
	}
}
