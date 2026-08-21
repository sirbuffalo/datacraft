// Package scope discovers lexical scopes and assigns stable IDs to them.
package scope

import "mccomp/ast"

type Kind string

const (
	Global    Kind = "global"
	Function  Kind = "function"
	IfBody    Kind = "if"
	ElifBody  Kind = "elif"
	ElseBody  Kind = "else"
	ForBody   Kind = "for"
	WhileBody Kind = "while"
)

// Scope is one lexical variable namespace. Parent is nil only for the global
// scope. IDs are unique within one compilation and assigned in source order.
type Scope struct {
	ID             ast.ScopeID
	Kind           Kind
	ScoreboardName string
	Parent         *Scope
	Children       []*Scope
}

type Result struct {
	Root *Scope
	ByID map[ast.ScopeID]*Scope
}

// Assign walks a parsed program depth-first and records a unique ID and
// the shared namespace scoreboard on every lexical scope.
func Assign(program *ast.Program, namespace string) Result {
	a := assigner{namespace: namespace, next: 1, byID: make(map[ast.ScopeID]*Scope)}
	root := &Scope{ID: 0, Kind: Global, ScoreboardName: namespace}
	program.ScopeID = root.ID
	a.byID[root.ID] = root

	for _, function := range program.Functions {
		functionScope := a.child(root, Function)
		function.ScopeID = functionScope.ID
		a.statements(function.Body, functionScope)
	}

	return Result{Root: root, ByID: a.byID}
}

type assigner struct {
	namespace string
	next      ast.ScopeID
	byID      map[ast.ScopeID]*Scope
}

func (a *assigner) child(parent *Scope, kind Kind) *Scope {
	s := &Scope{
		ID:             a.next,
		Kind:           kind,
		ScoreboardName: a.namespace,
		Parent:         parent,
	}
	a.next++
	parent.Children = append(parent.Children, s)
	a.byID[s.ID] = s
	return s
}

func (a *assigner) statements(statements []ast.Statement, parent *Scope) {
	for _, statement := range statements {
		switch statement := statement.(type) {
		case *ast.If:
			body := a.child(parent, IfBody)
			statement.BodyScopeID = body.ID
			a.statements(statement.Body, body)

			for i := range statement.Elifs {
				elif := &statement.Elifs[i]
				elifScope := a.child(parent, ElifBody)
				elif.ScopeID = elifScope.ID
				a.statements(elif.Body, elifScope)
			}

			if len(statement.Else) > 0 {
				elseScope := a.child(parent, ElseBody)
				statement.ElseScopeID = elseScope.ID
				a.statements(statement.Else, elseScope)
			}
		case *ast.For:
			body := a.child(parent, ForBody)
			statement.ScopeID = body.ID
			a.statements(statement.Body, body)
		case *ast.While:
			body := a.child(parent, WhileBody)
			statement.ScopeID = body.ID
			a.statements(statement.Body, body)
		}
	}
}
