package scope_test

import (
	"testing"

	"mccomp/ast"
	"mccomp/compiler/scope"
	"mccomp/parser"
)

func TestAssignEveryScopeInSourceOrder(t *testing.T) {
	source := "namespace demo\n\ndef run(items):\n    if ready():\n        for item in items:\n            use(item)\n    elif waiting():\n        wait()\n    else:\n        stop()\n\ndef other():\n    return 1\n"
	program, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	result := scope.Assign(program, "demo")
	if program.ScopeID != 0 {
		t.Fatalf("global scope = %d, want 0", program.ScopeID)
	}
	if program.Functions[0].ScopeID != 1 {
		t.Fatalf("run scope = %d, want 1", program.Functions[0].ScopeID)
	}

	conditional := program.Functions[0].Body[0].(*ast.If)
	loop := conditional.Body[0].(*ast.For)
	if conditional.BodyScopeID != 2 || loop.ScopeID != 3 {
		t.Fatalf("if/for scopes = %d/%d, want 2/3", conditional.BodyScopeID, loop.ScopeID)
	}
	if conditional.Elifs[0].ScopeID != 4 || conditional.ElseScopeID != 5 {
		t.Fatalf("elif/else scopes = %d/%d, want 4/5", conditional.Elifs[0].ScopeID, conditional.ElseScopeID)
	}
	if program.Functions[1].ScopeID != 6 {
		t.Fatalf("other scope = %d, want 6", program.Functions[1].ScopeID)
	}
	if len(result.ByID) != 7 {
		t.Fatalf("scope count = %d, want 7", len(result.ByID))
	}
	if result.ByID[3].Parent.ID != 2 {
		t.Fatalf("for parent = %d, want 2", result.ByID[3].Parent.ID)
	}
	for id, current := range result.ByID {
		if current.ScoreboardName != "demo" {
			t.Fatalf("scope %d scoreboard = %q, want demo", id, current.ScoreboardName)
		}
	}
}

func TestAssignIsDeterministic(t *testing.T) {
	program, err := parser.Parse("def run(items):\n    for item in items:\n        use(item)\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	first := scope.Assign(program, "example")
	second := scope.Assign(program, "example")
	if len(first.ByID) != len(second.ByID) || program.Functions[0].ScopeID != 1 {
		t.Fatal("reassigning scopes changed the result")
	}
}
