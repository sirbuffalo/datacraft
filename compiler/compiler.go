// Package compiler lowers the parsed language into Minecraft commands.
package compiler

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/sirbuffalo/datacraft/ast"
	"github.com/sirbuffalo/datacraft/compiler/scope"
	"github.com/sirbuffalo/datacraft/semantic"
	"github.com/sirbuffalo/datacraft/token"
)

type Variable struct {
	ID      uint32
	ScopeID ast.ScopeID
	Name    string
	Holder  string
}

type Function struct {
	ID                  uint32
	Name                string
	GeneratedName       string
	Exposed             bool
	Parameters          []Parameter
	ReturnsValue        bool
	ReturnsList         bool
	ReturnsEntitySet    bool
	ReturnsPrimitiveSet bool
	ReturnsNBT          bool
	ReturnHolder        string
}

type Parameter struct {
	Name           string
	VariableID     uint32
	Holder         string
	IsList         bool
	IsString       bool
	IsEntitySet    bool
	IsPrimitiveSet bool
	IsNBT          bool
}

type ImportedFunction struct {
	Namespace  string
	Objective  string
	Definition *ast.Function
	Mapping    Function
}

type Output struct {
	// Load contains commands for the generated load.mcfunction entry point.
	Load             []string
	Functions        map[string][]string
	FunctionNames    map[string]string
	FunctionMappings []Function
	Variables        []Variable
	ScoreboardName   string
	Tick             []string
}

type Error struct {
	Position token.Position
	Message  string
}

func (e Error) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Position.Line, e.Position.Column, e.Message)
}

// Compile lowers assignments and scalar expressions to the source namespace's
// scoreboard. It only consumes and returns in-memory values, so it is safe for
// js/wasm.
func Compile(program *ast.Program, functionNamespace string) (Output, error) {
	return CompileWithImports(program, functionNamespace, nil)
}

func CompileWithImports(program *ast.Program, functionNamespace string, imports map[string]ImportedFunction) (Output, error) {
	importedDefinitions := make(map[string]*ast.Function, len(imports))
	for name, imported := range imports {
		importedDefinitions[name] = imported.Definition
	}
	if err := semantic.CheckWithImports(program, importedDefinitions); err != nil {
		return Output{}, err
	}
	objective := program.Namespace
	if objective == "" {
		objective = functionNamespace
	}
	scopes := scope.Assign(program, objective)
	c := compiler{
		functionNamespace: functionNamespace,
		objective:         objective,
		scopes:            scopes,
		constants:         make(map[int64]struct{}),
		declared:          make(map[ast.ScopeID]map[string]uint32),
		globals:           make(map[*ast.Function]map[string]struct{}),
		globalTypes:       make(map[*ast.Function]map[string]string),
		listVariables:     make(map[uint32]struct{}),
		listDepth:         make(map[uint32]int),
		stringVariables:   make(map[uint32]struct{}),
		variableTypes:     make(map[uint32]string),
		listElementTypes:  make(map[uint32][]string),
		listDeclaredTypes: make(map[uint32]string),
		variantVariables:  make(map[uint32]struct{}),
		entityVariables:   make(map[uint32]struct{}),
		entityLists:       make(map[uint32]struct{}),
		entitySets:        make(map[uint32]struct{}),
		primitiveSets:     make(map[uint32]string),
		nbtVariables:      make(map[uint32]struct{}),
		functionIndexes:   make(map[string]int),
		functionsByName:   make(map[string]*ast.Function),
		imports:           imports,
		output: Output{
			Functions:      make(map[string][]string),
			FunctionNames:  make(map[string]string),
			ScoreboardName: objective,
		},
	}
	globalStatements := make([]ast.Statement, len(program.Globals))
	for index, declaration := range program.Globals {
		globalStatements[index] = declaration
	}
	c.collectStatements(globalStatements, program.ScopeID, nil)

	for id, function := range program.Functions {
		if _, exists := c.output.FunctionNames[function.Name]; exists {
			return Output{}, Error{Position: function.Pos, Message: fmt.Sprintf("duplicate function %q", function.Name)}
		}
		generatedName := "_" + strconv.Itoa(id)
		c.output.FunctionNames[function.Name] = generatedName
		c.functionIndexes[function.Name] = id
		c.functionsByName[function.Name] = function
		c.output.FunctionMappings = append(c.output.FunctionMappings, Function{
			ID: uint32(id), Name: function.Name, GeneratedName: generatedName,
			Exposed: function.Exposed, ReturnsValue: functionReturnsValue(function.Body),
			ReturnHolder: returnHolder(uint32(id)),
		})
	}
	c.nextFunctionID = uint32(len(program.Functions))
	for _, function := range program.Functions {
		c.collectFunction(function)
	}
	for _, function := range program.Functions {
		mapping := &c.output.FunctionMappings[c.functionIndexes[function.Name]]
		mapping.ReturnsList = c.functionReturnsList(function)
		mapping.ReturnsEntitySet = isEntitySetType(function.ReturnType)
		mapping.ReturnsPrimitiveSet = isPrimitiveSetType(function.ReturnType)
		mapping.ReturnsNBT = function.ReturnType != nil && function.ReturnType.Name == "nbt"
	}
	for range program.Functions {
		for _, function := range program.Functions {
			c.inferListCallArguments(function.Body, function.ScopeID, function)
			c.markListCallResults(function.Body, function.ScopeID, function)
			mapping := &c.output.FunctionMappings[c.functionIndexes[function.Name]]
			mapping.ReturnsList = c.functionReturnsList(function)
		}
	}
	for _, function := range program.Functions {
		mapping := &c.output.FunctionMappings[c.functionIndexes[function.Name]]
		for _, name := range function.Parameters {
			variableID := c.declared[function.ScopeID][name]
			mapping.Parameters = append(mapping.Parameters, Parameter{
				Name: name, VariableID: variableID, Holder: variableHolder(variableID), IsList: c.isList(variableID), IsString: c.isString(variableID), IsEntitySet: c.isEntitySet(variableID), IsPrimitiveSet: c.isPrimitiveSet(variableID), IsNBT: c.isNBT(variableID),
			})
		}
	}
	for _, function := range program.Functions {
		var returnSignal *value
		var commands []string
		if statementsHaveNestedReturn(function.Body) {
			signal := value{holder: returnSignalHolder(uint32(c.functionIndexes[function.Name])), objective: c.objective}
			returnSignal = &signal
			commands = append(commands, fmt.Sprintf("scoreboard players set %s %s 0", signal.holder, signal.objective))
		}
		compiled, err := c.compileStatements(function.Body, function.ScopeID, function, returnSignal, nil, nil)
		if err != nil {
			return Output{}, err
		}
		commands = append(commands, compiled...)
		c.output.Functions[c.output.FunctionNames[function.Name]] = commands
	}
	for _, declaration := range program.Globals {
		compiled, err := c.compileAssignment(declaration, program.ScopeID, nil)
		if err != nil {
			return Output{}, err
		}
		c.globalInitializers = append(c.globalInitializers, compiled...)
	}
	register := c.ensureEntityRuntime()
	registerAll := c.specialFunction("entity_register_all", []string{fmt.Sprintf("execute as @e run function %s:%s", c.functionNamespace, register)})
	c.output.Tick = []string{fmt.Sprintf("function %s:%s", c.functionNamespace, registerAll)}
	c.buildEntityListSync()
	c.buildLoad()
	return c.output, nil
}

type compiler struct {
	functionNamespace  string
	objective          string
	scopes             scope.Result
	constants          map[int64]struct{}
	declared           map[ast.ScopeID]map[string]uint32
	globals            map[*ast.Function]map[string]struct{}
	globalTypes        map[*ast.Function]map[string]string
	listVariables      map[uint32]struct{}
	listDepth          map[uint32]int
	stringVariables    map[uint32]struct{}
	variableTypes      map[uint32]string
	listElementTypes   map[uint32][]string
	listDeclaredTypes  map[uint32]string
	variantVariables   map[uint32]struct{}
	entityVariables    map[uint32]struct{}
	entityLists        map[uint32]struct{}
	entitySets         map[uint32]struct{}
	primitiveSets      map[uint32]string
	nbtVariables       map[uint32]struct{}
	functionIndexes    map[string]int
	functionsByName    map[string]*ast.Function
	imports            map[string]ImportedFunction
	nextVariable       uint32
	nextFunctionID     uint32
	temporary          uint64
	stringTemporary    uint64
	entityTemporary    uint64
	commandTemporary   uint64
	globalInitializers []string
	output             Output
}

type value struct {
	holder    string
	objective string
}

func (c *compiler) collectFunction(function *ast.Function) {
	c.globals[function] = make(map[string]struct{})
	c.globalTypes[function] = make(map[string]string)
	c.collectGlobals(function.Body, function)
	c.ensureScope(function.ScopeID)
	for name := range c.globals[function] {
		variableID := c.declare(0, name)
		c.applyDeclaredType(variableID, c.globalTypes[function][name])
	}
	for _, parameter := range function.Parameters {
		variableID := c.declare(function.ScopeID, parameter)
		if ref := function.Types[parameter]; ref != nil {
			c.applyTypeRef(variableID, ref)
		} else {
			c.applyDeclaredType(variableID, function.ParameterTypes[parameter])
		}
	}
	c.collectStatements(function.Body, function.ScopeID, function)
}

func (c *compiler) collectGlobals(statements []ast.Statement, function *ast.Function) {
	for _, statement := range statements {
		switch statement := statement.(type) {
		case *ast.Global:
			for _, name := range statement.Names {
				c.globals[function][name] = struct{}{}
				if statement.Types[name] != "" {
					c.globalTypes[function][name] = statement.Types[name]
				}
			}
		case *ast.If:
			c.collectGlobals(statement.Body, function)
			for _, branch := range statement.Elifs {
				c.collectGlobals(branch.Body, function)
			}
			c.collectGlobals(statement.Else, function)
		case *ast.For:
			c.collectGlobals(statement.Body, function)
		case *ast.While:
			c.collectGlobals(statement.Body, function)
		}
	}
}

func (c *compiler) applyDeclaredType(variableID uint32, typeName string) {
	switch typeName {
	case "":
	case "int", "bool":
		c.variableTypes[variableID] = "int"
	case "str":
		c.stringVariables[variableID] = struct{}{}
		c.variableTypes[variableID] = "str"
	case "nbt":
		c.nbtVariables[variableID] = struct{}{}
		c.variableTypes[variableID] = "nbt"
	case "list":
		c.listVariables[variableID] = struct{}{}
		c.variableTypes[variableID] = "list"
		if c.listDepth[variableID] == 0 {
			c.listDepth[variableID] = 1
		}
	case "entity":
		c.entityVariables[variableID] = struct{}{}
		c.variableTypes[variableID] = "entity"
	}
}

func (c *compiler) applyTypeRef(variableID uint32, ref *ast.TypeRef) {
	if ref == nil {
		return
	}
	if ref.Name == "set" && ref.Element != nil && ref.Element.Name == "entity" {
		c.entitySets[variableID] = struct{}{}
		c.variableTypes[variableID] = "set"
		return
	}
	if ref.Name == "set" && ref.Element != nil && (ref.Element.Name == "int" || ref.Element.Name == "bool" || ref.Element.Name == "str") {
		c.primitiveSets[variableID] = ref.Element.Name
		c.variableTypes[variableID] = "set"
		return
	}
	if ref.Name == "list" && ref.Element != nil && ref.Element.Name == "entity" {
		c.entityLists[variableID] = struct{}{}
		c.listElementTypes[variableID] = []string{"entity"}
	}
	if ref.Name == "list" && ref.Element != nil {
		c.listDeclaredTypes[variableID] = ref.Element.Name
	}
	c.applyDeclaredType(variableID, ref.Name)
}

func (c *compiler) collectStatements(statements []ast.Statement, id ast.ScopeID, function *ast.Function) {
	c.ensureScope(id)
	for _, statement := range statements {
		switch statement := statement.(type) {
		case *ast.Assignment:
			targetScope := id
			if _, global := c.globals[function][statement.Name]; global {
				targetScope = 0
			}
			variableID, exists := c.resolve(statement.Name, targetScope, function)
			if !exists {
				variableID = c.declare(targetScope, statement.Name)
			}
			c.applyTypeRef(variableID, statement.DeclaredType)
			if _, list := statement.Value.(*ast.List); list {
				c.listVariables[variableID] = struct{}{}
				c.variableTypes[variableID] = "list"
				depth := expressionListDepth(statement.Value) + len(statement.Indices)
				if depth > c.listDepth[variableID] {
					c.listDepth[variableID] = depth
				}
				literal := statement.Value.(*ast.List)
				types := make([]string, len(literal.Elements))
				for index, element := range literal.Elements {
					types[index] = c.expressionType(element, id, function)
					if types[index] == "entity" {
						c.entityLists[variableID] = struct{}{}
					}
				}
				c.listElementTypes[variableID] = types
			}
			if _, stringValue := statement.Value.(*ast.String); stringValue {
				c.stringVariables[variableID] = struct{}{}
				c.variableTypes[variableID] = "str"
			}
			if _, entityValue := statement.Value.(*ast.EntitySelector); entityValue {
				c.entityVariables[variableID] = struct{}{}
				c.variableTypes[variableID] = "entity"
			}
			if source, copied := statement.Value.(*ast.Identifier); copied {
				if sourceID, found := c.resolve(source.Name, id, function); found {
					if c.isString(sourceID) {
						c.stringVariables[variableID] = struct{}{}
					}
					if _, entity := c.entityVariables[sourceID]; entity {
						c.entityVariables[variableID] = struct{}{}
						c.variableTypes[variableID] = "entity"
					}
				}
			}
			if c.isStringExpression(statement.Value, id, function) {
				c.stringVariables[variableID] = struct{}{}
				c.variableTypes[variableID] = "str"
			}
			if inferred := c.expressionType(statement.Value, id, function); inferred != "" {
				c.variableTypes[variableID] = inferred
			}
			if indexed, ok := statement.Value.(*ast.Index); ok {
				if root, indices := compilerIndexedRoot(indexed); root != nil {
					if sourceID, found := c.resolve(root.Name, id, function); found && c.isList(sourceID) {
						dynamic := false
						for _, index := range indices {
							if _, constant := integerLiteral(index); !constant {
								dynamic = true
							}
						}
						kind := c.expressionType(indexed, id, function)
						heterogeneous := false
						allEntities := len(c.listElementTypes[sourceID]) > 0
						for _, elementType := range c.listElementTypes[sourceID] {
							if elementType != "int" {
								heterogeneous = true
							}
							if elementType != "entity" {
								allEntities = false
							}
						}
						if kind == "entity" || (dynamic && allEntities) {
							c.entityVariables[variableID] = struct{}{}
							c.variableTypes[variableID] = "entity"
						} else if (dynamic && heterogeneous) || kind == "str" || kind == "list" {
							c.variantVariables[variableID] = struct{}{}
							delete(c.variableTypes, variableID)
						}
					}
				}
			}
			if call, ok := statement.Value.(*ast.Call); ok {
				if attribute, ok := call.Callee.(*ast.Attribute); ok && attribute.Name == "remove" {
					c.variantVariables[variableID] = struct{}{}
					delete(c.variableTypes, variableID)
				}
			}
			if statement.Index != nil && !c.isNBT(variableID) && c.variableTypes[variableID] != "entity" {
				c.listVariables[variableID] = struct{}{}
			}
			c.markListUses(statement.Value, id, function)
			c.markListUses(statement.Index, id, function)
			for _, index := range statement.Indices {
				c.markListUses(index, id, function)
			}
		case *ast.ExpressionStatement:
			c.markListUses(statement.Expression, id, function)
		case *ast.If:
			c.markListUses(statement.Condition, id, function)
			c.collectStatements(statement.Body, statement.BodyScopeID, function)
			for _, branch := range statement.Elifs {
				c.collectStatements(branch.Body, branch.ScopeID, function)
			}
			if len(statement.Else) > 0 {
				c.collectStatements(statement.Else, statement.ElseScopeID, function)
			}
		case *ast.For:
			c.markListUses(statement.Iterable, id, function)
			mixedList := false
			if iterable, ok := statement.Iterable.(*ast.Identifier); ok {
				if variableID, found := c.resolve(iterable.Name, id, function); found {
					if c.isEntitySet(variableID) {
						loopID := c.declare(statement.ScopeID, statement.Variable)
						c.entityVariables[loopID] = struct{}{}
						c.variableTypes[loopID] = "entity"
						c.collectStatements(statement.Body, statement.ScopeID, function)
						continue
					}
					if c.isPrimitiveSet(variableID) {
						loopID := c.declare(statement.ScopeID, statement.Variable)
						c.applyTypeRef(loopID, statement.VariableType)
						c.collectStatements(statement.Body, statement.ScopeID, function)
						continue
					}
					c.listVariables[variableID] = struct{}{}
					mixedList = true
					allEntities := len(c.listElementTypes[variableID]) > 0
					for _, kind := range c.listElementTypes[variableID] {
						if kind != "entity" {
							allEntities = false
						}
					}
					if allEntities {
						loopID := c.declare(statement.ScopeID, statement.Variable)
						c.entityVariables[loopID] = struct{}{}
						c.variableTypes[loopID] = "entity"
					}
				}
			}
			loopID := c.declare(statement.ScopeID, statement.Variable)
			if mixedList {
				c.variantVariables[loopID] = struct{}{}
			}
			c.collectStatements(statement.Body, statement.ScopeID, function)
		case *ast.While:
			c.markListUses(statement.Condition, id, function)
			c.collectStatements(statement.Body, statement.ScopeID, function)
		}
	}
}

func (c *compiler) markListUses(expression ast.Expression, id ast.ScopeID, function *ast.Function) {
	if expression == nil {
		return
	}
	switch expression := expression.(type) {
	case *ast.Index:
		if target, ok := expression.Target.(*ast.Identifier); ok {
			if variableID, found := c.resolve(target.Name, id, function); found {
				if !c.isNBT(variableID) && c.variableTypes[variableID] != "entity" {
					c.listVariables[variableID] = struct{}{}
				}
			}
		}
		c.markListUses(expression.Index, id, function)
	case *ast.Unary:
		c.markListUses(expression.Right, id, function)
	case *ast.Binary:
		c.markListUses(expression.Left, id, function)
		c.markListUses(expression.Right, id, function)
	case *ast.Call:
		c.markListUses(expression.Callee, id, function)
		for _, argument := range expression.Arguments {
			c.markListUses(argument, id, function)
		}
	case *ast.List:
		for _, element := range expression.Elements {
			c.markListUses(element, id, function)
		}
	case *ast.Attribute:
		if target, ok := expression.Target.(*ast.Identifier); ok {
			if variableID, found := c.resolve(target.Name, id, function); found {
				if !c.isEntitySet(variableID) && !c.isPrimitiveSet(variableID) && !c.isNBT(variableID) {
					c.listVariables[variableID] = struct{}{}
				}
			}
		}
	}
}

func (c *compiler) isList(variableID uint32) bool {
	_, ok := c.listVariables[variableID]
	return ok
}

func (c *compiler) isString(variableID uint32) bool {
	_, ok := c.stringVariables[variableID]
	return ok
}

func (c *compiler) isNBT(variableID uint32) bool {
	_, ok := c.nbtVariables[variableID]
	return ok
}

func (c *compiler) isEntitySet(variableID uint32) bool {
	_, ok := c.entitySets[variableID]
	return ok
}

func isEntitySetType(ref *ast.TypeRef) bool {
	return ref != nil && ref.Name == "set" && ref.Element != nil && ref.Element.Name == "entity"
}

func (c *compiler) isPrimitiveSet(variableID uint32) bool {
	_, ok := c.primitiveSets[variableID]
	return ok
}

func isPrimitiveSetType(ref *ast.TypeRef) bool {
	return ref != nil && ref.Name == "set" && ref.Element != nil && (ref.Element.Name == "int" || ref.Element.Name == "bool" || ref.Element.Name == "str")
}

func (c *compiler) entitySetTag(variableID uint32) string {
	return fmt.Sprintf("_%s_set_%d", c.objective, variableID)
}

func (c *compiler) entitySetReturnTag(functionID uint32) string {
	return fmt.Sprintf("_%s_return_set_%d", c.objective, functionID)
}

func entitySetTagFor(objective string, variableID uint32) string {
	return fmt.Sprintf("_%s_set_%d", objective, variableID)
}

func entitySetReturnTagFor(objective string, functionID uint32) string {
	return fmt.Sprintf("_%s_return_set_%d", objective, functionID)
}

func (c *compiler) callTarget(name string) (Function, string, string, bool) {
	if index, ok := c.functionIndexes[name]; ok {
		return c.output.FunctionMappings[index], c.functionNamespace, c.objective, true
	}
	if imported, ok := c.imports[name]; ok {
		objective := imported.Objective
		if objective == "" {
			objective = imported.Namespace
		}
		return imported.Mapping, imported.Namespace, objective, true
	}
	return Function{}, "", "", false
}

func (c *compiler) functionReturnsList(function *ast.Function) bool {
	var visit func([]ast.Statement) bool
	visit = func(statements []ast.Statement) bool {
		for _, statement := range statements {
			switch statement := statement.(type) {
			case *ast.Return:
				if identifier, ok := statement.Value.(*ast.Identifier); ok {
					if variableID, found := c.resolve(identifier.Name, function.ScopeID, function); found && c.isList(variableID) {
						return true
					}
				}
			case *ast.If:
				if visit(statement.Body) || visit(statement.Else) {
					return true
				}
				for _, branch := range statement.Elifs {
					if visit(branch.Body) {
						return true
					}
				}
			case *ast.For:
				if visit(statement.Body) {
					return true
				}
			case *ast.While:
				if visit(statement.Body) {
					return true
				}
			}
		}
		return false
	}
	return visit(function.Body)
}

func (c *compiler) markListCallResults(statements []ast.Statement, id ast.ScopeID, function *ast.Function) {
	for _, statement := range statements {
		switch statement := statement.(type) {
		case *ast.Assignment:
			call, ok := statement.Value.(*ast.Call)
			callee, named := callCallee(call)
			if ok && named {
				if index, found := c.functionIndexes[callee]; found && c.output.FunctionMappings[index].ReturnsList {
					if variableID, resolved := c.resolve(statement.Name, id, function); resolved {
						c.listVariables[variableID] = struct{}{}
					}
				}
			}
		case *ast.If:
			c.markListCallResults(statement.Body, statement.BodyScopeID, function)
			for _, branch := range statement.Elifs {
				c.markListCallResults(branch.Body, branch.ScopeID, function)
			}
			c.markListCallResults(statement.Else, statement.ElseScopeID, function)
		case *ast.For:
			c.markListCallResults(statement.Body, statement.ScopeID, function)
		case *ast.While:
			c.markListCallResults(statement.Body, statement.ScopeID, function)
		}
	}
}

func callCallee(call *ast.Call) (string, bool) {
	if call == nil {
		return "", false
	}
	identifier, ok := call.Callee.(*ast.Identifier)
	if !ok {
		return "", false
	}
	return identifier.Name, true
}

func (c *compiler) inferListCallArguments(statements []ast.Statement, id ast.ScopeID, function *ast.Function) {
	var expression func(ast.Expression)
	expression = func(expr ast.Expression) {
		switch expr := expr.(type) {
		case *ast.Call:
			if name, ok := callCallee(expr); ok {
				callee := c.functionsByName[name]
				if callee != nil {
					for i, argument := range expr.Arguments {
						identifier, named := argument.(*ast.Identifier)
						if named && i < len(callee.Parameters) {
							if variableID, found := c.resolve(identifier.Name, id, function); found && c.isList(variableID) {
								c.listVariables[c.declared[callee.ScopeID][callee.Parameters[i]]] = struct{}{}
							}
						}
					}
				}
			}
			for _, argument := range expr.Arguments {
				expression(argument)
			}
		case *ast.Index:
			expression(expr.Target)
			expression(expr.Index)
		case *ast.Unary:
			expression(expr.Right)
		case *ast.Binary:
			expression(expr.Left)
			expression(expr.Right)
		case *ast.List:
			for _, item := range expr.Elements {
				expression(item)
			}
		}
	}
	for _, statement := range statements {
		switch statement := statement.(type) {
		case *ast.Assignment:
			expression(statement.Value)
			expression(statement.Index)
		case *ast.Return:
			expression(statement.Value)
		case *ast.ExpressionStatement:
			expression(statement.Expression)
		case *ast.If:
			expression(statement.Condition)
			c.inferListCallArguments(statement.Body, statement.BodyScopeID, function)
			for _, branch := range statement.Elifs {
				c.inferListCallArguments(branch.Body, branch.ScopeID, function)
			}
			c.inferListCallArguments(statement.Else, statement.ElseScopeID, function)
		case *ast.For:
			expression(statement.Iterable)
			c.inferListCallArguments(statement.Body, statement.ScopeID, function)
		case *ast.While:
			expression(statement.Condition)
			c.inferListCallArguments(statement.Body, statement.ScopeID, function)
		}
	}
}

func (c *compiler) compileStatements(statements []ast.Statement, id ast.ScopeID, function *ast.Function, returnSignal, breakSignal, continueSignal *value) ([]string, error) {
	var commands []string
	for _, statement := range statements {
		switch statement := statement.(type) {
		case *ast.Global:
			continue
		case *ast.Assignment:
			compiled, err := c.compileAssignment(statement, id, function)
			if err != nil {
				return nil, err
			}
			commands = append(commands, compiled...)
		case *ast.ExpressionStatement:
			var compiled []string
			var err error
			if call, ok := statement.Expression.(*ast.Call); ok {
				compiled, _, err = c.compileCall(call, id, function, false)
			} else {
				compiled, _, err = c.compileExpression(statement.Expression, id, function)
			}
			if err != nil {
				return nil, err
			}
			commands = append(commands, compiled...)
		case *ast.Command:
			compiled, err := c.compileCommand(statement, id, function)
			if err != nil {
				return nil, err
			}
			commands = append(commands, compiled...)
		case *ast.If:
			compiled, err := c.compileIf(statement, id, function, returnSignal, breakSignal, continueSignal)
			if err != nil {
				return nil, err
			}
			commands = append(commands, compiled...)
		case *ast.For:
			compiled, err := c.compileFor(statement, id, function, returnSignal)
			if err != nil {
				return nil, err
			}
			commands = append(commands, compiled...)
		case *ast.While:
			compiled, err := c.compileWhile(statement, function, returnSignal)
			if err != nil {
				return nil, err
			}
			commands = append(commands, compiled...)
		case *ast.Break:
			if breakSignal == nil {
				return nil, Error{Position: statement.Pos, Message: "break can only be used inside a loop"}
			}
			commands = append(commands, fmt.Sprintf("scoreboard players set %s %s 1", breakSignal.holder, breakSignal.objective), "return 0")
		case *ast.Continue:
			if continueSignal == nil {
				return nil, Error{Position: statement.Pos, Message: "continue can only be used inside a loop"}
			}
			commands = append(commands, fmt.Sprintf("scoreboard players set %s %s 1", continueSignal.holder, continueSignal.objective), "return 0")
		case *ast.Return:
			if returnSignal != nil {
				commands = append(commands, fmt.Sprintf("scoreboard players set %s %s 1", returnSignal.holder, returnSignal.objective))
			}
			if statement.Value == nil {
				commands = append(commands, "return 0")
				continue
			}
			mapping := c.output.FunctionMappings[c.functionIndexes[function.Name]]
			if mapping.ReturnsNBT {
				identifier, ok := statement.Value.(*ast.Identifier)
				if !ok {
					return nil, Error{Position: statement.Value.Position(), Message: "NBT return currently requires an nbt variable"}
				}
				variableID, found := c.resolve(identifier.Name, id, function)
				if !found || !c.isNBT(variableID) {
					return nil, Error{Position: identifier.Pos, Message: "return value is not an nbt variable"}
				}
				commands = append(commands, fmt.Sprintf("data modify storage %s nbt_returns.r%d set from storage %s nbt.v%d", c.storageName(), mapping.ID, c.storageName(), variableID), "return 0")
				continue
			}
			if mapping.ReturnsEntitySet {
				compiled, err := c.compileEntitySetExpression(statement.Value, c.entitySetReturnTag(mapping.ID), id, function)
				if err != nil {
					return nil, err
				}
				commands = append(commands, compiled...)
				commands = append(commands, "return 0")
				continue
			}
			if mapping.ReturnsPrimitiveSet {
				identifier, ok := statement.Value.(*ast.Identifier)
				if !ok {
					return nil, Error{Position: statement.Value.Position(), Message: "primitive set return currently requires a set variable"}
				}
				setID, found := c.resolve(identifier.Name, id, function)
				if !found || !c.isPrimitiveSet(setID) {
					return nil, Error{Position: identifier.Pos, Message: "return value is not a primitive set"}
				}
				commands = append(commands, fmt.Sprintf("data modify storage %s set_returns.r%d set from storage %s sets.v%d", c.storageName(), mapping.ID, c.storageName(), setID), "return 0")
				continue
			}
			if identifier, ok := statement.Value.(*ast.Identifier); ok {
				if variableID, found := c.resolve(identifier.Name, id, function); found && c.isList(variableID) {
					c.constants[int64(variableID)] = struct{}{}
					commands = append(commands, fmt.Sprintf("data modify storage %s returns.r%d set from storage %s lists.v%d", c.storageName(), mapping.ID, c.storageName(), variableID))
					commands = append(commands, fmt.Sprintf("data modify storage %s return_types.r%d set from storage %s list_types.v%d", c.storageName(), mapping.ID, c.storageName(), variableID))
					returned := value{holder: mapping.ReturnHolder, objective: c.objective}
					commands = append(commands, scoreboardOperation(returned, "=", value{holder: constantHolder(int64(variableID)), objective: c.objective}))
					commands = append(commands, fmt.Sprintf("return run scoreboard players get %s %s", returned.holder, returned.objective))
					continue
				}
			}
			compiled, result, err := c.compileExpression(statement.Value, id, function)
			if err != nil {
				return nil, err
			}
			commands = append(commands, compiled...)
			returned := value{holder: mapping.ReturnHolder, objective: c.objective}
			commands = append(commands, scoreboardOperation(returned, "=", result))
			commands = append(commands, fmt.Sprintf("return run scoreboard players get %s %s", returned.holder, returned.objective))
		default:
			return nil, Error{Position: statement.Position(), Message: "control-flow compilation is not implemented yet"}
		}
	}
	return commands, nil
}

func (c *compiler) compileCommand(statement *ast.Command, id ast.ScopeID, function *ast.Function) ([]string, error) {
	text := statement.Text
	if !strings.Contains(text, "${") {
		return []string{text}, nil
	}
	path := "scratch.command_" + strconv.FormatUint(c.commandTemporary, 10)
	c.commandTemporary++
	var setup []string
	var rendered strings.Builder
	for cursor, slot := 0, 0; cursor < len(text); slot++ {
		start := strings.Index(text[cursor:], "${")
		if start < 0 {
			rendered.WriteString(text[cursor:])
			break
		}
		start += cursor
		rendered.WriteString(text[cursor:start])
		end := strings.IndexByte(text[start+2:], '}')
		if end < 0 {
			return nil, Error{Position: statement.Pos, Message: "unterminated command interpolation"}
		}
		end += start + 2
		name := strings.TrimSpace(text[start+2 : end])
		if !validInterpolationName(name) {
			return nil, Error{Position: statement.Pos, Message: "command interpolation expects a variable name"}
		}
		variableID, found := c.resolve(name, id, function)
		if !found {
			return nil, Error{Position: statement.Pos, Message: fmt.Sprintf("undefined command variable %q", name)}
		}
		field := fmt.Sprintf("m%d", slot)
		switch {
		case c.variableTypes[variableID] == "entity":
			for part := 0; part < 4; part++ {
				setup = append(setup, fmt.Sprintf("data modify storage %s %s.%s_%d set from storage %s entities.v%d.uuid[%d]", c.storageName(), path, field, part, c.storageName(), variableID, part))
			}
			rendered.WriteString(fmt.Sprintf("@n[tag=_%s_$(%s_0)_$(%s_1)_$(%s_2)_$(%s_3)]", c.objective, field, field, field, field))
		case c.isString(variableID):
			setup = append(setup, fmt.Sprintf("data modify storage %s %s.%s set from storage %s strings.v%d", c.storageName(), path, field, c.storageName(), variableID))
			rendered.WriteString("$(" + field + ")")
		case c.isNBT(variableID):
			setup = append(setup, fmt.Sprintf("data modify storage %s %s.%s set from storage %s nbt.v%d", c.storageName(), path, field, c.storageName(), variableID))
			rendered.WriteString("$(" + field + ")")
		case c.isList(variableID):
			setup = append(setup, fmt.Sprintf("data modify storage %s %s.%s set from storage %s lists.v%d", c.storageName(), path, field, c.storageName(), variableID))
			rendered.WriteString("$(" + field + ")")
		case c.isEntitySet(variableID) || c.isPrimitiveSet(variableID):
			return nil, Error{Position: statement.Pos, Message: fmt.Sprintf("set variable %q cannot be interpolated into one command", name)}
		default:
			setup = append(setup, fmt.Sprintf("execute store result storage %s %s.%s int 1 run scoreboard players get %s %s", c.storageName(), path, field, variableHolder(variableID), c.objective))
			rendered.WriteString("$(" + field + ")")
		}
		cursor = end + 1
	}
	helper := c.reserveInternalFunction()
	c.output.Functions[helper] = []string{"$" + rendered.String()}
	setup = append(setup,
		fmt.Sprintf("function %s:%s with storage %s %s", c.functionNamespace, helper, c.storageName(), path),
		fmt.Sprintf("data remove storage %s %s", c.storageName(), path),
	)
	return setup, nil
}

func validInterpolationName(name string) bool {
	if name == "" || !((name[0] >= 'a' && name[0] <= 'z') || (name[0] >= 'A' && name[0] <= 'Z') || name[0] == '_') {
		return false
	}
	for index := 1; index < len(name); index++ {
		character := name[index]
		if !((character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || character == '_') {
			return false
		}
	}
	return true
}

func (c *compiler) compileIf(statement *ast.If, parentID ast.ScopeID, function *ast.Function, returnSignal, breakSignal, continueSignal *value) ([]string, error) {
	matched := c.newTemporary()
	commands := []string{fmt.Sprintf("scoreboard players set %s %s 0", matched.holder, matched.objective)}

	conditionCommands, condition, err := c.compileExpression(statement.Condition, parentID, function)
	if err != nil {
		return nil, err
	}
	bodyName := c.reserveInternalFunction()
	bodyCommands, err := c.compileStatements(statement.Body, statement.BodyScopeID, function, returnSignal, breakSignal, continueSignal)
	if err != nil {
		return nil, err
	}
	c.output.Functions[bodyName] = bodyCommands
	commands = append(commands, gatedCommands(matched, conditionCommands)...)
	commands = append(commands, executeTruthy(matched, condition, "function "+c.functionNamespace+":"+bodyName))
	commands = appendReturnPropagation(commands, returnSignal, c.functionReturnValue(function))
	commands = appendBreakPropagation(commands, breakSignal)
	commands = appendContinuePropagation(commands, continueSignal)
	commands = append(commands, executeTruthy(matched, condition, "scoreboard players set "+matched.holder+" "+matched.objective+" 1"))

	for _, branch := range statement.Elifs {
		branchCommands, branchCondition, compileErr := c.compileExpression(branch.Condition, parentID, function)
		if compileErr != nil {
			return nil, compileErr
		}
		branchName := c.reserveInternalFunction()
		body, compileErr := c.compileStatements(branch.Body, branch.ScopeID, function, returnSignal, breakSignal, continueSignal)
		if compileErr != nil {
			return nil, compileErr
		}
		c.output.Functions[branchName] = body
		commands = append(commands, gatedCommands(matched, branchCommands)...)
		commands = append(commands, executeTruthy(matched, branchCondition, "function "+c.functionNamespace+":"+branchName))
		commands = appendReturnPropagation(commands, returnSignal, c.functionReturnValue(function))
		commands = appendBreakPropagation(commands, breakSignal)
		commands = appendContinuePropagation(commands, continueSignal)
		commands = append(commands, executeTruthy(matched, branchCondition, "scoreboard players set "+matched.holder+" "+matched.objective+" 1"))
	}

	if len(statement.Else) > 0 {
		elseName := c.reserveInternalFunction()
		body, compileErr := c.compileStatements(statement.Else, statement.ElseScopeID, function, returnSignal, breakSignal, continueSignal)
		if compileErr != nil {
			return nil, compileErr
		}
		c.output.Functions[elseName] = body
		commands = append(commands, fmt.Sprintf("execute if score %s %s matches 0 run function %s:%s", matched.holder, matched.objective, c.functionNamespace, elseName))
		commands = appendReturnPropagation(commands, returnSignal, c.functionReturnValue(function))
		commands = appendBreakPropagation(commands, breakSignal)
		commands = appendContinuePropagation(commands, continueSignal)
	}

	return commands, nil
}

func (c *compiler) compileFor(statement *ast.For, parentID ast.ScopeID, function *ast.Function, returnSignal *value) ([]string, error) {
	if identifier, ok := statement.Iterable.(*ast.Identifier); ok {
		collectionID, found := c.resolve(identifier.Name, parentID, function)
		if !found {
			return nil, Error{Position: identifier.Pos, Message: fmt.Sprintf("undefined collection %q", identifier.Name)}
		}
		if c.isEntitySet(collectionID) {
			return c.compileEntitySetFor(statement, collectionID, function, returnSignal)
		}
		if c.isPrimitiveSet(collectionID) {
			return c.compilePrimitiveSetFor(statement, collectionID, function, returnSignal)
		}
		if !c.isList(collectionID) {
			return nil, Error{Position: identifier.Pos, Message: fmt.Sprintf("%q is not iterable", identifier.Name)}
		}
		return c.compileListFor(statement, collectionID, function, returnSignal)
	}
	if call, ok := statement.Iterable.(*ast.Call); ok {
		if name, named := callCallee(call); named && name == "range" {
			return c.compileRangeFor(statement, call, function, returnSignal)
		}
	}
	return nil, Error{Position: statement.Iterable.Position(), Message: "for loops support list variables and range(...)"}
}

func (c *compiler) compileEntitySetFor(statement *ast.For, setID uint32, function *ast.Function, returnSignal *value) ([]string, error) {
	loopVariableID, ok := c.resolve(statement.Variable, statement.ScopeID, function)
	if !ok {
		return nil, Error{Position: statement.Pos, Message: "undefined loop variable"}
	}
	c.entityVariables[loopVariableID] = struct{}{}
	c.variableTypes[loopVariableID] = "entity"
	bodyName := c.reserveInternalFunction()
	runnerName := c.reserveInternalFunction()
	breakSignal := c.newTemporary()
	continueSignal := c.newTemporary()
	body, err := c.compileStatements(statement.Body, statement.ScopeID, function, returnSignal, &breakSignal, &continueSignal)
	if err != nil {
		return nil, err
	}
	c.output.Functions[bodyName] = body
	capture := c.compileEntityCapture("@s", fmt.Sprintf("entities.v%d", loopVariableID))
	runner := append([]string{}, capture...)
	runner = append(runner, "function "+c.functionNamespace+":"+bodyName)
	runner = appendReturnPropagation(runner, returnSignal, c.functionReturnValue(function))
	c.output.Functions[runnerName] = runner
	commands := []string{
		fmt.Sprintf("scoreboard players set %s %s 0", breakSignal.holder, breakSignal.objective),
		fmt.Sprintf("scoreboard players set %s %s 0", continueSignal.holder, continueSignal.objective),
		fmt.Sprintf("execute as @e[tag=%s] if score %s %s matches 0 run function %s:%s", c.entitySetTag(setID), breakSignal.holder, breakSignal.objective, c.functionNamespace, runnerName),
	}
	commands = appendReturnPropagation(commands, returnSignal, c.functionReturnValue(function))
	return commands, nil
}

func (c *compiler) compileListFor(statement *ast.For, listID uint32, function *ast.Function, returnSignal *value) ([]string, error) {
	loopVariableID, ok := c.resolve(statement.Variable, statement.ScopeID, function)
	if !ok {
		return nil, Error{Position: statement.Pos, Message: "undefined loop variable"}
	}
	index := c.newTemporary()
	length := c.newTemporary()
	controllerName := c.reserveInternalFunction()
	loaderName := c.reserveInternalFunction()
	bodyName := c.reserveInternalFunction()
	breakSignal := c.newTemporary()
	continueSignal := c.newTemporary()
	body, err := c.compileStatements(statement.Body, statement.ScopeID, function, returnSignal, &breakSignal, &continueSignal)
	if err != nil {
		return nil, err
	}
	c.output.Functions[bodyName] = body
	valuePath := fmt.Sprintf("variants.v%d", loopVariableID)
	if _, entity := c.entityVariables[loopVariableID]; entity {
		valuePath = fmt.Sprintf("entities.v%d", loopVariableID)
	}
	c.output.Functions[loaderName] = []string{
		fmt.Sprintf("$data modify storage %s %s set from storage %s lists.v%d[$(loop_index)]", c.storageName(), valuePath, c.storageName(), listID),
		fmt.Sprintf("$data modify storage %s variant_types.v%d set from storage %s list_types.v%d[$(loop_index)]", c.storageName(), loopVariableID, c.storageName(), listID),
		fmt.Sprintf("$execute store result score %s %s run data get storage %s lists.v%d[$(loop_index)] 1", variableHolder(loopVariableID), c.objective, c.storageName(), listID),
	}
	controller := []string{
		fmt.Sprintf("execute store result storage %s scratch.loop_index int 1 run scoreboard players get %s %s", c.storageName(), index.holder, index.objective),
		fmt.Sprintf("function %s:%s with storage %s scratch", c.functionNamespace, loaderName, c.storageName()),
		"function " + c.functionNamespace + ":" + bodyName,
	}
	controller = appendReturnPropagation(controller, returnSignal, c.functionReturnValue(function))
	controller = appendBreakPropagation(controller, &breakSignal)
	controller = append(controller, fmt.Sprintf("scoreboard players set %s %s 0", continueSignal.holder, continueSignal.objective))
	controller = append(controller, fmt.Sprintf("scoreboard players add %s %s 1", index.holder, index.objective))
	controller = append(controller, fmt.Sprintf("execute if score %s %s < %s %s run function %s:%s", index.holder, index.objective, length.holder, length.objective, c.functionNamespace, controllerName))
	c.output.Functions[controllerName] = controller
	commands := []string{
		fmt.Sprintf("scoreboard players set %s %s 0", index.holder, index.objective),
		fmt.Sprintf("scoreboard players set %s %s 0", breakSignal.holder, breakSignal.objective),
		fmt.Sprintf("scoreboard players set %s %s 0", continueSignal.holder, continueSignal.objective),
		fmt.Sprintf("execute store result score %s %s run data get storage %s lists.v%d", length.holder, length.objective, c.storageName(), listID),
		fmt.Sprintf("execute if score %s %s < %s %s run function %s:%s", index.holder, index.objective, length.holder, length.objective, c.functionNamespace, controllerName),
	}
	commands = appendReturnPropagation(commands, returnSignal, c.functionReturnValue(function))
	return commands, nil
}

func (c *compiler) compileRangeFor(statement *ast.For, call *ast.Call, function *ast.Function, returnSignal *value) ([]string, error) {
	if len(call.Arguments) < 1 || len(call.Arguments) > 3 {
		return nil, Error{Position: call.Pos, Message: "range expects 1, 2, or 3 arguments"}
	}
	startExpr := ast.Expression(&ast.Integer{Pos: call.Pos, Value: 0})
	stopExpr := call.Arguments[0]
	step := int64(1)
	if len(call.Arguments) >= 2 {
		startExpr, stopExpr = call.Arguments[0], call.Arguments[1]
	}
	if len(call.Arguments) == 3 {
		literal, ok := integerLiteral(call.Arguments[2])
		if !ok || literal == 0 {
			return nil, Error{Position: call.Arguments[2].Position(), Message: "range step must be a nonzero integer literal"}
		}
		step = literal
	}
	startCommands, start, err := c.compileExpression(startExpr, statement.ScopeID, function)
	if err != nil {
		return nil, err
	}
	stopCommands, stop, err := c.compileExpression(stopExpr, statement.ScopeID, function)
	if err != nil {
		return nil, err
	}
	loopID, _ := c.resolve(statement.Variable, statement.ScopeID, function)
	target := c.variableValue(loopID)
	stableStop := c.newTemporary()
	commands := append(startCommands, stopCommands...)
	commands = append(commands, scoreboardOperation(target, "=", start), scoreboardOperation(stableStop, "=", stop))
	c.constants[step] = struct{}{}
	controllerName := c.reserveInternalFunction()
	bodyName := c.reserveInternalFunction()
	breakSignal := c.newTemporary()
	continueSignal := c.newTemporary()
	body, err := c.compileStatements(statement.Body, statement.ScopeID, function, returnSignal, &breakSignal, &continueSignal)
	if err != nil {
		return nil, err
	}
	c.output.Functions[bodyName] = body
	comparison := "<"
	if step < 0 {
		comparison = ">"
	}
	controller := []string{"function " + c.functionNamespace + ":" + bodyName}
	controller = appendReturnPropagation(controller, returnSignal, c.functionReturnValue(function))
	controller = appendBreakPropagation(controller, &breakSignal)
	controller = append(controller, fmt.Sprintf("scoreboard players set %s %s 0", continueSignal.holder, continueSignal.objective))
	controller = append(controller, scoreboardOperation(target, "+=", value{holder: constantHolder(step), objective: c.objective}))
	controller = append(controller, fmt.Sprintf("execute if score %s %s %s %s %s run function %s:%s", target.holder, target.objective, comparison, stableStop.holder, stableStop.objective, c.functionNamespace, controllerName))
	c.output.Functions[controllerName] = controller
	commands = append(commands, fmt.Sprintf("execute if score %s %s %s %s %s run function %s:%s", target.holder, target.objective, comparison, stableStop.holder, stableStop.objective, c.functionNamespace, controllerName))
	commands = append([]string{fmt.Sprintf("scoreboard players set %s %s 0", breakSignal.holder, breakSignal.objective), fmt.Sprintf("scoreboard players set %s %s 0", continueSignal.holder, continueSignal.objective)}, commands...)
	commands = appendReturnPropagation(commands, returnSignal, c.functionReturnValue(function))
	return commands, nil
}

func integerLiteral(expression ast.Expression) (int64, bool) {
	if integer, ok := expression.(*ast.Integer); ok {
		return integer.Value, true
	}
	if unary, ok := expression.(*ast.Unary); ok && unary.Operator == token.Minus {
		if integer, ok := unary.Right.(*ast.Integer); ok {
			return -integer.Value, true
		}
	}
	return 0, false
}

func (c *compiler) compileWhile(statement *ast.While, function *ast.Function, returnSignal *value) ([]string, error) {
	controllerName := c.reserveInternalFunction()
	bodyName := c.reserveInternalFunction()
	breakSignal := c.newTemporary()
	continueSignal := c.newTemporary()
	body, err := c.compileStatements(statement.Body, statement.ScopeID, function, returnSignal, &breakSignal, &continueSignal)
	if err != nil {
		return nil, err
	}
	c.output.Functions[bodyName] = body
	conditionCommands, condition, err := c.compileExpression(statement.Condition, statement.ScopeID, function)
	if err != nil {
		return nil, err
	}
	controller := append([]string{}, conditionCommands...)
	controller = append(controller, fmt.Sprintf("execute unless score %s %s matches 0 run function %s:%s", condition.holder, condition.objective, c.functionNamespace, bodyName))
	controller = appendReturnPropagation(controller, returnSignal, c.functionReturnValue(function))
	controller = appendBreakPropagation(controller, &breakSignal)
	controller = append(controller, fmt.Sprintf("scoreboard players set %s %s 0", continueSignal.holder, continueSignal.objective))
	controller = append(controller, fmt.Sprintf("execute unless score %s %s matches 0 run function %s:%s", condition.holder, condition.objective, c.functionNamespace, controllerName))
	c.output.Functions[controllerName] = controller
	commands := []string{fmt.Sprintf("scoreboard players set %s %s 0", breakSignal.holder, breakSignal.objective), fmt.Sprintf("scoreboard players set %s %s 0", continueSignal.holder, continueSignal.objective), "function " + c.functionNamespace + ":" + controllerName}
	commands = appendReturnPropagation(commands, returnSignal, c.functionReturnValue(function))
	return commands, nil
}

func (c *compiler) functionReturnValue(function *ast.Function) value {
	mapping := c.output.FunctionMappings[c.functionIndexes[function.Name]]
	return value{holder: mapping.ReturnHolder, objective: c.objective}
}

func appendReturnPropagation(commands []string, signal *value, returned value) []string {
	if signal == nil {
		return commands
	}
	return append(commands, fmt.Sprintf("execute if score %s %s matches 1 run return run scoreboard players get %s %s", signal.holder, signal.objective, returned.holder, returned.objective))
}

func appendBreakPropagation(commands []string, signal *value) []string {
	if signal == nil {
		return commands
	}
	return append(commands, fmt.Sprintf("execute if score %s %s matches 1 run return 0", signal.holder, signal.objective))
}

func appendContinuePropagation(commands []string, signal *value) []string {
	if signal == nil {
		return commands
	}
	return append(commands, fmt.Sprintf("execute if score %s %s matches 1 run return 0", signal.holder, signal.objective))
}

func (c *compiler) reserveInternalFunction() string {
	name := "_" + strconv.FormatUint(uint64(c.nextFunctionID), 10)
	c.nextFunctionID++
	return name
}

func (c *compiler) specialFunction(name string, commands []string) string {
	path := "_special/" + name
	if _, exists := c.output.Functions[path]; !exists {
		c.output.Functions[path] = commands
	}
	return path
}

func gatedCommands(matched value, commands []string) []string {
	gated := make([]string, 0, len(commands))
	for _, command := range commands {
		gated = append(gated, fmt.Sprintf("execute if score %s %s matches 0 run %s", matched.holder, matched.objective, command))
	}
	return gated
}

func executeTruthy(matched, condition value, command string) string {
	return fmt.Sprintf(
		"execute if score %s %s matches 0 unless score %s %s matches 0 run %s",
		matched.holder, matched.objective, condition.holder, condition.objective, command,
	)
}

func (c *compiler) compileAssignment(statement *ast.Assignment, id ast.ScopeID, function *ast.Function) ([]string, error) {
	variableID, ok := c.resolve(statement.Name, id, function)
	if !ok {
		return nil, Error{Position: statement.Pos, Message: fmt.Sprintf("undefined variable %q", statement.Name)}
	}
	if statement.Index != nil {
		if c.isNBT(variableID) {
			return c.compileNBTFieldAssignment(statement, variableID, id, function)
		}
		return c.compileListItemAssignment(statement, variableID, id, function)
	}
	if _, none := statement.Value.(*ast.NoneLiteral); none {
		return c.compileNoneAssignment(variableID), nil
	}
	if indexed, ok := statement.Value.(*ast.Index); ok {
		if entityID, path, found := c.entityIndexPath(indexed, id, function); found {
			return c.compileEntityDataReadAssignment(variableID, entityID, path), nil
		}
	}
	if c.isNBT(variableID) {
		if statement.Operator != token.Assign {
			return nil, Error{Position: statement.Pos, Message: "NBT assignment only supports '='"}
		}
		if call, ok := statement.Value.(*ast.Call); ok {
			if calleeName, named := callCallee(call); named {
				if mapping, _, targetObjective, found := c.callTarget(calleeName); found && mapping.ReturnsNBT {
					commands, _, err := c.compileCall(call, id, function, false)
					if err != nil {
						return nil, err
					}
					commands = append(commands, fmt.Sprintf("data modify storage %s nbt.v%d set from storage %s nbt_returns.r%d", c.storageName(), variableID, targetObjective+":data", mapping.ID))
					return commands, nil
				}
			}
		}
		return c.compileNBTValueToPath(statement.Value, fmt.Sprintf("nbt.v%d", variableID), id, function)
	}
	if indexed, ok := statement.Value.(*ast.Index); ok {
		if path, sourceNBT, found := c.nbtIndexPath(indexed, id, function); found {
			return c.compileNBTReadAssignment(variableID, path, sourceNBT), nil
		}
	}
	if c.isEntitySet(variableID) {
		if statement.Operator != token.Assign {
			return nil, Error{Position: statement.Pos, Message: "entity-set assignment only supports '='"}
		}
		if call, ok := statement.Value.(*ast.Call); ok {
			if calleeName, named := callCallee(call); named {
				if mapping, _, targetObjective, found := c.callTarget(calleeName); found && mapping.ReturnsEntitySet {
					commands, _, err := c.compileCall(call, id, function, false)
					if err != nil {
						return nil, err
					}
					destination := c.entitySetTag(variableID)
					source := entitySetReturnTagFor(targetObjective, mapping.ID)
					commands = append(commands,
						fmt.Sprintf("tag @e[tag=%s] remove %s", destination, destination),
						fmt.Sprintf("tag @e[tag=%s] add %s", source, destination),
					)
					return commands, nil
				}
			}
		}
		return c.compileEntitySetExpression(statement.Value, c.entitySetTag(variableID), id, function)
	}
	if c.isPrimitiveSet(variableID) {
		if statement.Operator != token.Assign {
			return nil, Error{Position: statement.Pos, Message: "set assignment only supports '='"}
		}
		return c.compilePrimitiveSetAssignment(statement.Value, variableID, id, function)
	}
	if c.isList(variableID) {
		binary, binaryValue := statement.Value.(*ast.Binary)
		if statement.Operator == token.PlusAssign || (binaryValue && binary.Operator == token.Plus) {
			return c.compileListConcatenationAssignment(statement, variableID, id, function)
		}
	}
	if selector, ok := statement.Value.(*ast.EntitySelector); ok {
		if statement.Operator != token.Assign {
			return nil, Error{Position: statement.Pos, Message: "entity assignment only supports '='"}
		}
		return c.compileEntityCapture(selector.Value, fmt.Sprintf("entities.v%d", variableID)), nil
	}
	if source, ok := statement.Value.(*ast.Identifier); ok {
		if sourceID, found := c.resolve(source.Name, id, function); found {
			if _, entity := c.entityVariables[sourceID]; entity {
				return []string{fmt.Sprintf("data modify storage %s entities.v%d set from storage %s entities.v%d", c.storageName(), variableID, c.storageName(), sourceID)}, nil
			}
		}
	}
	if indexed, ok := statement.Value.(*ast.Index); ok {
		if _, entity := c.entityVariables[variableID]; entity {
			root, indices := compilerIndexedRoot(indexed)
			if root == nil {
				return nil, Error{Position: indexed.Pos, Message: "entity list indexing requires a list"}
			}
			listID, found := c.resolve(root.Name, id, function)
			if !found || !c.isList(listID) {
				return nil, Error{Position: root.Pos, Message: "entity list indexing requires a list"}
			}
			commands, sourcePath, dynamic, err := c.compileIndexPath(fmt.Sprintf("lists.v%d", listID), indices, id, function)
			if err != nil {
				return nil, err
			}
			copyCommand := fmt.Sprintf("data modify storage %s entities.v%d set from storage %s %s", c.storageName(), variableID, c.storageName(), sourcePath)
			if dynamic {
				helper := c.reserveInternalFunction()
				c.output.Functions[helper] = []string{"$" + copyCommand}
				commands = append(commands, fmt.Sprintf("function %s:%s with storage %s scratch", c.functionNamespace, helper, c.storageName()))
			} else {
				commands = append(commands, copyCommand)
			}
			return commands, nil
		}
	}
	if indexed, ok := statement.Value.(*ast.Index); ok {
		if _, variant := c.variantVariables[variableID]; variant {
			root, indices := compilerIndexedRoot(indexed)
			if root == nil {
				return nil, Error{Position: indexed.Pos, Message: "indexed value requires a list"}
			}
			listID, found := c.resolve(root.Name, id, function)
			if !found || !c.isList(listID) {
				return nil, Error{Position: root.Pos, Message: "indexed value requires a list"}
			}
			commands, sourcePath, dynamic, err := c.compileIndexPath(fmt.Sprintf("lists.v%d", listID), indices, id, function)
			if err != nil {
				return nil, err
			}
			_, typePath, _, _, err := c.runtimeTypePath(indexed, id, function)
			if err != nil {
				return nil, err
			}
			copies := []string{
				fmt.Sprintf("data modify storage %s variants.v%d set from storage %s %s", c.storageName(), variableID, c.storageName(), sourcePath),
				fmt.Sprintf("data modify storage %s variant_types.v%d set from storage %s %s", c.storageName(), variableID, c.storageName(), typePath),
				fmt.Sprintf("execute store result score %s %s run data get storage %s %s 1", variableHolder(variableID), c.objective, c.storageName(), sourcePath),
			}
			if dynamic {
				helper := c.reserveInternalFunction()
				for i := range copies {
					copies[i] = "$" + copies[i]
				}
				c.output.Functions[helper] = copies
				commands = append(commands, fmt.Sprintf("function %s:%s with storage %s scratch", c.functionNamespace, helper, c.storageName()))
			} else {
				commands = append(commands, copies...)
			}
			return commands, nil
		}
	}
	if call, ok := statement.Value.(*ast.Call); ok {
		if attribute, ok := call.Callee.(*ast.Attribute); ok && attribute.Name == "remove" {
			return c.compileRemoveToVariant(call, attribute, variableID, id, function)
		}
	}
	if c.isStringExpression(statement.Value, id, function) {
		if statement.Operator != token.Assign {
			return nil, Error{Position: statement.Pos, Message: "string assignment only supports '='"}
		}
		return c.compileStringExpressionToPath(statement.Value, fmt.Sprintf("strings.v%d", variableID), id, function)
	}
	if text, ok := statement.Value.(*ast.String); ok {
		if statement.Operator != token.Assign {
			return nil, Error{Position: statement.Pos, Message: "string assignment only supports '='"}
		}
		encoded, _ := json.Marshal(text.Value)
		return []string{fmt.Sprintf("data modify storage %s strings.v%d set value %s", c.storageName(), variableID, encoded)}, nil
	}
	if source, ok := statement.Value.(*ast.Identifier); ok {
		if sourceID, found := c.resolve(source.Name, id, function); found && c.isString(sourceID) {
			if statement.Operator != token.Assign {
				return nil, Error{Position: statement.Pos, Message: "string assignment only supports '='"}
			}
			return []string{fmt.Sprintf("data modify storage %s strings.v%d set from storage %s strings.v%d", c.storageName(), variableID, c.storageName(), sourceID)}, nil
		}
	}
	if call, ok := statement.Value.(*ast.Call); ok {
		if calleeName, named := callCallee(call); named {
			if mapping, _, targetObjective, found := c.callTarget(calleeName); found && mapping.ReturnsList {
				if statement.Operator != token.Assign {
					return nil, Error{Position: statement.Pos, Message: "list return values only support '=' assignment"}
				}
				commands, _, err := c.compileCall(call, id, function, false)
				if err != nil {
					return nil, err
				}
				targetStorage := targetObjective + ":data"
				commands = append(commands, fmt.Sprintf("data modify storage %s lists.v%d set from storage %s returns.r%d", c.storageName(), variableID, targetStorage, mapping.ID))
				commands = append(commands, fmt.Sprintf("data modify storage %s list_types.v%d set from storage %s return_types.r%d", c.storageName(), variableID, targetStorage, mapping.ID))
				return commands, nil
			}
		}
	}
	if list, ok := statement.Value.(*ast.List); ok {
		if statement.Operator != token.Assign {
			return nil, Error{Position: statement.Pos, Message: "list assignment only supports '='"}
		}
		return c.compileListValueAtPath(list, fmt.Sprintf("lists.v%d", variableID), id, function)
	}
	commands, right, err := c.compileExpression(statement.Value, id, function)
	if err != nil {
		return nil, err
	}
	if _, list := c.listVariables[variableID]; list {
		return nil, Error{Position: statement.Pos, Message: "a list variable must be assigned a list value"}
	}
	target := c.variableValue(variableID)
	operation := map[token.Kind]string{
		token.Assign: "=", token.PlusAssign: "+=", token.MinusAssign: "-=",
		token.StarAssign: "*=", token.SlashAssign: "/=",
	}[statement.Operator]
	commands = append(commands, scoreboardOperation(target, operation, right))
	return commands, nil
}

func (c *compiler) compileNoneAssignment(variableID uint32) []string {
	storage := c.storageName()
	switch {
	case c.isEntitySet(variableID):
		tag := c.entitySetTag(variableID)
		return []string{fmt.Sprintf("tag @e[tag=%s] remove %s", tag, tag)}
	case c.isPrimitiveSet(variableID):
		return []string{fmt.Sprintf("data remove storage %s sets.v%d", storage, variableID)}
	case c.isList(variableID):
		return []string{
			fmt.Sprintf("data remove storage %s lists.v%d", storage, variableID),
			fmt.Sprintf("data remove storage %s list_types.v%d", storage, variableID),
		}
	case c.isString(variableID):
		return []string{fmt.Sprintf("data remove storage %s strings.v%d", storage, variableID)}
	case c.isNBT(variableID):
		return []string{fmt.Sprintf("data remove storage %s nbt.v%d", storage, variableID)}
	case c.variableTypes[variableID] == "entity":
		return []string{fmt.Sprintf("data remove storage %s entities.v%d", storage, variableID)}
	default:
		return []string{fmt.Sprintf("scoreboard players reset %s %s", variableHolder(variableID), c.objective)}
	}
}

func (c *compiler) nbtIndexPath(indexed *ast.Index, id ast.ScopeID, function *ast.Function) (string, uint32, bool) {
	root, indices := compilerIndexedRoot(indexed)
	if root == nil {
		return "", 0, false
	}
	variableID, found := c.resolve(root.Name, id, function)
	if !found || !c.isNBT(variableID) {
		return "", 0, false
	}
	path := fmt.Sprintf("nbt.v%d", variableID)
	for _, index := range indices {
		key, ok := index.(*ast.String)
		if !ok {
			return "", 0, false
		}
		encoded, _ := json.Marshal(key.Value)
		path += "." + string(encoded)
	}
	return path, variableID, true
}

func (c *compiler) entityIndexPath(indexed *ast.Index, id ast.ScopeID, function *ast.Function) (uint32, string, bool) {
	root, indices := compilerIndexedRoot(indexed)
	if root == nil {
		return 0, "", false
	}
	variableID, found := c.resolve(root.Name, id, function)
	if !found || c.variableTypes[variableID] != "entity" {
		return 0, "", false
	}
	path := ""
	for _, index := range indices {
		key, ok := index.(*ast.String)
		if !ok {
			return 0, "", false
		}
		encoded, _ := json.Marshal(key.Value)
		if path != "" {
			path += "."
		}
		path += string(encoded)
	}
	return variableID, path, path != ""
}

func (c *compiler) compileEntityDataReadAssignment(destination, entityID uint32, path string) []string {
	storage := c.storageName()
	scratch := "scratch.entity_read_" + strconv.FormatUint(c.entityTemporary, 10)
	c.entityTemporary++
	commands := make([]string, 0, 7)
	for part := 0; part < 4; part++ {
		commands = append(commands, fmt.Sprintf("data modify storage %s %s.uuid%d set from storage %s entities.v%d.uuid[%d]", storage, scratch, part, storage, entityID, part))
	}
	selector := fmt.Sprintf("@n[tag=_%s_$(uuid0)_$(uuid1)_$(uuid2)_$(uuid3)]", c.objective)
	var read string
	var after []string
	switch {
	case c.isList(destination):
		read = fmt.Sprintf("execute as %s run data modify storage %s lists.v%d set from entity @s %s", selector, storage, destination, path)
		elementType := c.listDeclaredTypes[destination]
		if elementType == "" {
			elementType = "nbt"
		}
		encodedType, _ := json.Marshal(elementType)
		remaining := c.newTemporary()
		metadataHelper := c.reserveInternalFunction()
		c.output.Functions[metadataHelper] = []string{
			fmt.Sprintf("execute if score %s %s matches 1.. run data modify storage %s list_types.v%d append value %s", remaining.holder, remaining.objective, storage, destination, encodedType),
			fmt.Sprintf("execute if score %s %s matches 1.. run scoreboard players remove %s %s 1", remaining.holder, remaining.objective, remaining.holder, remaining.objective),
			fmt.Sprintf("execute if score %s %s matches 1.. run function %s:%s", remaining.holder, remaining.objective, c.functionNamespace, metadataHelper),
		}
		after = append(after,
			fmt.Sprintf("data modify storage %s list_types.v%d set value []", storage, destination),
			fmt.Sprintf("execute store result score %s %s run data get storage %s lists.v%d", remaining.holder, remaining.objective, storage, destination),
			fmt.Sprintf("function %s:%s", c.functionNamespace, metadataHelper),
		)
	case c.isNBT(destination):
		read = fmt.Sprintf("execute as %s run data modify storage %s nbt.v%d set from entity @s %s", selector, storage, destination, path)
	case c.isString(destination):
		read = fmt.Sprintf("execute as %s run data modify storage %s strings.v%d set from entity @s %s", selector, storage, destination, path)
	default:
		read = fmt.Sprintf("execute as %s store result score %s %s run data get entity @s %s 1", selector, variableHolder(destination), c.objective, path)
	}
	helper := c.reserveInternalFunction()
	c.output.Functions[helper] = []string{"$" + read}
	commands = append(commands,
		fmt.Sprintf("function %s:%s with storage %s %s", c.functionNamespace, helper, storage, scratch),
	)
	commands = append(commands, after...)
	commands = append(commands, fmt.Sprintf("data remove storage %s %s", storage, scratch))
	return commands
}

func (c *compiler) compileNBTFieldAssignment(statement *ast.Assignment, variableID uint32, id ast.ScopeID, function *ast.Function) ([]string, error) {
	if statement.Operator != token.Assign {
		return nil, Error{Position: statement.Pos, Message: "NBT fields only support '=' assignment"}
	}
	path := fmt.Sprintf("nbt.v%d", variableID)
	indices := statement.Indices
	if len(indices) == 0 {
		indices = []ast.Expression{statement.Index}
	}
	for _, index := range indices {
		key, ok := index.(*ast.String)
		if !ok {
			return nil, Error{Position: index.Position(), Message: "NBT keys must be string literals"}
		}
		encoded, _ := json.Marshal(key.Value)
		path += "." + string(encoded)
	}
	return c.compileNBTValueToPath(statement.Value, path, id, function)
}

func (c *compiler) compileNBTReadAssignment(variableID uint32, sourcePath string, _ uint32) []string {
	storage := c.storageName()
	switch {
	case c.isNBT(variableID):
		return []string{fmt.Sprintf("data modify storage %s nbt.v%d set from storage %s %s", storage, variableID, storage, sourcePath)}
	case c.isString(variableID):
		return []string{fmt.Sprintf("data modify storage %s strings.v%d set from storage %s %s", storage, variableID, storage, sourcePath)}
	default:
		return []string{fmt.Sprintf("execute store result score %s %s run data get storage %s %s 1", variableHolder(variableID), c.objective, storage, sourcePath)}
	}
}

func (c *compiler) compileNBTValueToPath(expression ast.Expression, path string, id ast.ScopeID, function *ast.Function) ([]string, error) {
	storage := c.storageName()
	switch expression := expression.(type) {
	case *ast.NBT:
		commands := []string{fmt.Sprintf("data modify storage %s %s set value {}", storage, path)}
		for _, field := range expression.Fields {
			encoded, _ := json.Marshal(field.Key)
			compiled, err := c.compileNBTValueToPath(field.Value, path+"."+string(encoded), id, function)
			if err != nil {
				return nil, err
			}
			commands = append(commands, compiled...)
		}
		return commands, nil
	case *ast.Set:
		if len(expression.Elements) == 0 {
			return []string{fmt.Sprintf("data modify storage %s %s set value {}", storage, path)}, nil
		}
	case *ast.List:
		commands := []string{fmt.Sprintf("data modify storage %s %s set value []", storage, path)}
		for _, element := range expression.Elements {
			commands = append(commands, fmt.Sprintf("data modify storage %s %s append value 0", storage, path))
			compiled, err := c.compileNBTValueToPath(element, path+"[-1]", id, function)
			if err != nil {
				return nil, err
			}
			commands = append(commands, compiled...)
		}
		return commands, nil
	case *ast.String:
		encoded, _ := json.Marshal(expression.Value)
		return []string{fmt.Sprintf("data modify storage %s %s set value %s", storage, path, encoded)}, nil
	case *ast.Integer:
		return []string{fmt.Sprintf("data modify storage %s %s set value %d", storage, path, expression.Value)}, nil
	case *ast.Boolean:
		value := 0
		if expression.Value {
			value = 1
		}
		return []string{fmt.Sprintf("data modify storage %s %s set value %db", storage, path, value)}, nil
	case *ast.Identifier:
		variableID, found := c.resolve(expression.Name, id, function)
		if !found {
			return nil, Error{Position: expression.Pos, Message: fmt.Sprintf("undefined variable %q", expression.Name)}
		}
		source := ""
		switch {
		case c.isNBT(variableID):
			source = fmt.Sprintf("nbt.v%d", variableID)
		case c.isString(variableID):
			source = fmt.Sprintf("strings.v%d", variableID)
		case c.isList(variableID):
			source = fmt.Sprintf("lists.v%d", variableID)
		}
		if source != "" {
			return []string{fmt.Sprintf("data modify storage %s %s set from storage %s %s", storage, path, storage, source)}, nil
		}
		return []string{fmt.Sprintf("execute store result storage %s %s int 1 run scoreboard players get %s %s", storage, path, variableHolder(variableID), c.objective)}, nil
	case *ast.Index:
		if source, _, found := c.nbtIndexPath(expression, id, function); found {
			return []string{fmt.Sprintf("data modify storage %s %s set from storage %s %s", storage, path, storage, source)}, nil
		}
	}
	return nil, Error{Position: expression.Position(), Message: "unsupported NBT value"}
}

func (c *compiler) compileEntitySetExpression(expression ast.Expression, destination string, id ast.ScopeID, function *ast.Function) ([]string, error) {
	switch expression := expression.(type) {
	case *ast.EntitySelector:
		return []string{
			fmt.Sprintf("tag @e[tag=%s] remove %s", destination, destination),
			fmt.Sprintf("tag %s add %s", expression.Value, destination),
		}, nil
	case *ast.Identifier:
		variableID, found := c.resolve(expression.Name, id, function)
		if !found || !c.isEntitySet(variableID) {
			return nil, Error{Position: expression.Pos, Message: fmt.Sprintf("%q is not an entity set", expression.Name)}
		}
		source := c.entitySetTag(variableID)
		if source == destination {
			return nil, nil
		}
		return []string{
			fmt.Sprintf("tag @e[tag=%s] remove %s", destination, destination),
			fmt.Sprintf("tag @e[tag=%s] add %s", source, destination),
		}, nil
	case *ast.Set:
		commands := []string{fmt.Sprintf("tag @e[tag=%s] remove %s", destination, destination)}
		for _, element := range expression.Elements {
			selector, ok := element.(*ast.EntitySelector)
			if !ok {
				return nil, Error{Position: element.Position(), Message: "entity-set literals currently require entity selectors"}
			}
			commands = append(commands, fmt.Sprintf("tag %s add %s", selector.Value, destination))
		}
		return commands, nil
	case *ast.Binary:
		if expression.Operator != token.Pipe && expression.Operator != token.Ampersand {
			return nil, Error{Position: expression.Pos, Message: "entity sets only support | and &"}
		}
		leftTag := fmt.Sprintf("_%s_set_tmp_%d", c.objective, c.entityTemporary)
		c.entityTemporary++
		rightTag := fmt.Sprintf("_%s_set_tmp_%d", c.objective, c.entityTemporary)
		c.entityTemporary++
		left, err := c.compileEntitySetExpression(expression.Left, leftTag, id, function)
		if err != nil {
			return nil, err
		}
		right, err := c.compileEntitySetExpression(expression.Right, rightTag, id, function)
		if err != nil {
			return nil, err
		}
		commands := append(left, right...)
		commands = append(commands, fmt.Sprintf("tag @e[tag=%s] remove %s", destination, destination))
		if expression.Operator == token.Pipe {
			commands = append(commands,
				fmt.Sprintf("tag @e[tag=%s] add %s", leftTag, destination),
				fmt.Sprintf("tag @e[tag=%s] add %s", rightTag, destination),
			)
		} else {
			commands = append(commands, fmt.Sprintf("tag @e[tag=%s,tag=%s] add %s", leftTag, rightTag, destination))
		}
		commands = append(commands,
			fmt.Sprintf("tag @e[tag=%s] remove %s", leftTag, leftTag),
			fmt.Sprintf("tag @e[tag=%s] remove %s", rightTag, rightTag),
		)
		return commands, nil
	default:
		return nil, Error{Position: expression.Position(), Message: "expected an entity-set expression"}
	}
}

func (c *compiler) compileListItemAssignment(statement *ast.Assignment, variableID uint32, id ast.ScopeID, function *ast.Function) ([]string, error) {
	if _, list := c.listVariables[variableID]; !list {
		return nil, Error{Position: statement.Pos, Message: fmt.Sprintf("%q is not a list", statement.Name)}
	}
	if statement.Operator != token.Assign {
		return nil, Error{Position: statement.Pos, Message: "list items currently only support '=' assignment"}
	}
	indices := statement.Indices
	if len(indices) == 0 {
		indices = []ast.Expression{statement.Index}
	}
	commands, path, dynamic, err := c.compileIndexPath(fmt.Sprintf("lists.v%d", variableID), indices, id, function)
	if err != nil {
		return nil, err
	}
	if list, ok := statement.Value.(*ast.List); ok {
		if dynamic {
			return nil, Error{Position: statement.Pos, Message: "assigning a nested list currently requires constant indexes"}
		}
		nested, err := c.compileListValueAtPath(list, path, id, function)
		return append(commands, nested...), err
	}
	if selector, ok := statement.Value.(*ast.EntitySelector); ok {
		if dynamic {
			return nil, Error{Position: statement.Pos, Message: "assigning an entity through a dynamic list index is not supported yet"}
		}
		commands = append(commands, c.compileEntityCapture(selector.Value, path)...)
		typePath := strings.Replace(path, "lists.", "list_types.", 1)
		commands = append(commands, fmt.Sprintf("data modify storage %s %s set value \"entity\"", c.storageName(), typePath))
		return commands, nil
	}
	if c.isStringExpression(statement.Value, id, function) {
		if dynamic {
			return nil, Error{Position: statement.Pos, Message: "assigning a string through a dynamic list index is not supported yet"}
		}
		written, err := c.compileStringExpressionToPath(statement.Value, path, id, function)
		commands = append(commands, written...)
		typePath := strings.Replace(path, "lists.", "list_types.", 1)
		commands = append(commands, fmt.Sprintf("data modify storage %s %s set value \"str\"", c.storageName(), typePath))
		return commands, err
	}
	valueCommands, item, err := c.compileExpression(statement.Value, id, function)
	if err != nil {
		return nil, err
	}
	commands = append(commands, valueCommands...)
	write := fmt.Sprintf("execute store result storage %s %s int 1 run scoreboard players get %s %s", c.storageName(), path, item.holder, item.objective)
	if dynamic {
		helper := c.reserveInternalFunction()
		c.output.Functions[helper] = []string{"$" + write}
		commands = append(commands, fmt.Sprintf("function %s:%s with storage %s scratch", c.functionNamespace, helper, c.storageName()))
	} else {
		commands = append(commands, write)
	}
	typePath := strings.Replace(path, "lists.", "list_types.", 1)
	typeWrite := fmt.Sprintf("data modify storage %s %s set value \"int\"", c.storageName(), typePath)
	if dynamic {
		helper := c.reserveInternalFunction()
		c.output.Functions[helper] = []string{"$" + typeWrite}
		commands = append(commands, fmt.Sprintf("function %s:%s with storage %s scratch", c.functionNamespace, helper, c.storageName()))
	} else {
		commands = append(commands, typeWrite)
	}
	return commands, nil
}

func (c *compiler) compileListValueAtPath(list *ast.List, path string, id ast.ScopeID, function *ast.Function) ([]string, error) {
	typePath := strings.Replace(path, "lists.", "list_types.", 1)
	commands := []string{
		fmt.Sprintf("data modify storage %s %s set value []", c.storageName(), path),
		fmt.Sprintf("data modify storage %s %s set value []", c.storageName(), typePath),
	}
	if listID, ok := listIDFromPath(path); ok && !strings.Contains(path, "[") && listContainsEntity(list) {
		c.entityLists[listID] = struct{}{}
		tag := c.entityListTag(listID)
		commands = append([]string{fmt.Sprintf("tag @e[tag=%s] remove %s", tag, tag)}, commands...)
	}
	for _, element := range list.Elements {
		if selector, ok := element.(*ast.EntitySelector); ok {
			commands = append(commands, fmt.Sprintf("data modify storage %s %s append value {}", c.storageName(), path))
			commands = append(commands, c.compileEntityCapture(selector.Value, path+"[-1]")...)
			commands = append(commands, fmt.Sprintf("data modify storage %s %s append value \"entity\"", c.storageName(), typePath))
			continue
		}
		if nested, ok := element.(*ast.List); ok {
			commands = append(commands, fmt.Sprintf("data modify storage %s %s append value []", c.storageName(), path))
			commands = append(commands, fmt.Sprintf("data modify storage %s %s append value []", c.storageName(), typePath))
			nestedCommands, err := c.compileListValueAtPath(nested, path+"[-1]", id, function)
			if err != nil {
				return nil, err
			}
			commands = append(commands, nestedCommands...)
			continue
		}
		if c.isStringExpression(element, id, function) {
			commands = append(commands, fmt.Sprintf("data modify storage %s %s append value \"\"", c.storageName(), path))
			commands = append(commands, fmt.Sprintf("data modify storage %s %s append value \"str\"", c.storageName(), typePath))
			compiled, err := c.compileStringExpressionToPath(element, path+"[-1]", id, function)
			if err != nil {
				return nil, err
			}
			commands = append(commands, compiled...)
			continue
		}
		compiled, item, err := c.compileExpression(element, id, function)
		if err != nil {
			return nil, err
		}
		commands = append(commands, compiled...)
		commands = append(commands, fmt.Sprintf("data modify storage %s %s append value 0", c.storageName(), path))
		commands = append(commands, fmt.Sprintf("data modify storage %s %s append value \"int\"", c.storageName(), typePath))
		commands = append(commands, fmt.Sprintf("execute store result storage %s %s[-1] int 1 run scoreboard players get %s %s", c.storageName(), path, item.holder, item.objective))
	}
	return commands, nil
}

func listContainsEntity(list *ast.List) bool {
	for _, element := range list.Elements {
		if _, ok := element.(*ast.EntitySelector); ok {
			return true
		}
		if nested, ok := element.(*ast.List); ok && listContainsEntity(nested) {
			return true
		}
	}
	return false
}

func (c *compiler) entityUUIDTagMacro() string {
	return fmt.Sprintf("_%s_$(uuid0)_$(uuid1)_$(uuid2)_$(uuid3)", c.objective)
}

func (c *compiler) compileEntityCapture(selector, destination string) []string {
	capture := c.reserveInternalFunction()
	register := c.ensureEntityRuntime()
	commands := []string{fmt.Sprintf("function %s:%s", c.functionNamespace, register)}
	if listID, ok := listIDFromPath(destination); ok {
		c.entityLists[listID] = struct{}{}
		commands = append(commands, fmt.Sprintf("tag @s add %s", c.entityListTag(listID)))
	}
	commands = append(commands, fmt.Sprintf("data modify storage %s %s set value {type:\"entity\",uuid:[I;0,0,0,0]}", c.storageName(), destination))
	commands = append(commands, fmt.Sprintf("data modify storage %s %s.uuid set from storage %s scratch.entity_uuid", c.storageName(), destination, c.storageName()))
	c.output.Functions[capture] = commands
	return []string{fmt.Sprintf("execute as %s run function %s:%s", selector, c.functionNamespace, capture)}
}

func listIDFromPath(path string) (uint32, bool) {
	if !strings.HasPrefix(path, "lists.v") {
		return 0, false
	}
	rest := strings.TrimPrefix(path, "lists.v")
	end := strings.IndexByte(rest, '[')
	if end >= 0 {
		rest = rest[:end]
	}
	value, err := strconv.ParseUint(rest, 10, 32)
	return uint32(value), err == nil
}

func (c *compiler) entityListTag(listID uint32) string {
	return fmt.Sprintf("_%s_list_%d", c.objective, listID)
}

func (c *compiler) ensureEntityRuntime() string {
	tagger := c.specialFunction("entity_tag_uuid", []string{"$tag @s add " + c.entityUUIDTagMacro()})
	registerCommands := []string{fmt.Sprintf("data modify storage %s scratch.entity_uuid set from entity @s UUID", c.storageName())}
	for index := 0; index < 4; index++ {
		registerCommands = append(registerCommands, fmt.Sprintf("execute store result storage %s scratch.uuid%d int 1 run data get storage %s scratch.entity_uuid[%d]", c.storageName(), index, c.storageName(), index))
	}
	registerCommands = append(registerCommands, fmt.Sprintf("function %s:%s with storage %s scratch", c.functionNamespace, tagger, c.storageName()))
	return c.specialFunction("entity_register_current", registerCommands)
}

func (c *compiler) buildEntityListSync() {
	ids := make([]int, 0, len(c.entityLists))
	for id := range c.entityLists {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	for _, rawID := range ids {
		listID := uint32(rawID)
		index, length := c.newTemporary(), c.newTemporary()
		start := fmt.Sprintf("_special/entity_list_%d", listID)
		controller := fmt.Sprintf("_special/entity_list_%d_controller", listID)
		loader := fmt.Sprintf("_special/entity_list_%d_loader", listID)
		item := fmt.Sprintf("_special/entity_list_%d_item", listID)
		tag := c.entityListTag(listID)
		addTag := c.specialFunction(fmt.Sprintf("entity_list_%d_add", listID), []string{fmt.Sprintf("$tag @n[tag=%s] add %s", c.entityUUIDTagMacro(), tag)})
		itemCommands := []string{}
		for part := 0; part < 4; part++ {
			itemCommands = append(itemCommands, fmt.Sprintf("execute store result storage %s scratch.uuid%d int 1 run data get storage %s scratch.entity_uuid[%d]", c.storageName(), part, c.storageName(), part))
		}
		itemCommands = append(itemCommands, fmt.Sprintf("function %s:%s with storage %s scratch", c.functionNamespace, addTag, c.storageName()))
		c.output.Functions[item] = itemCommands
		c.output.Functions[loader] = []string{fmt.Sprintf("$data modify storage %s scratch.entity_uuid set from storage %s lists.v%d[$(list_index)].uuid", c.storageName(), c.storageName(), listID)}
		c.output.Functions[controller] = []string{
			fmt.Sprintf("data remove storage %s scratch.entity_uuid", c.storageName()),
			fmt.Sprintf("execute store result storage %s scratch.list_index int 1 run scoreboard players get %s %s", c.storageName(), index.holder, index.objective),
			fmt.Sprintf("function %s:%s with storage %s scratch", c.functionNamespace, loader, c.storageName()),
			fmt.Sprintf("execute if data storage %s scratch.entity_uuid run function %s:%s", c.storageName(), c.functionNamespace, item),
			fmt.Sprintf("scoreboard players add %s %s 1", index.holder, index.objective),
			fmt.Sprintf("execute if score %s %s < %s %s run function %s:%s", index.holder, index.objective, length.holder, length.objective, c.functionNamespace, controller),
		}
		c.output.Functions[start] = []string{
			fmt.Sprintf("tag @e[tag=%s] remove %s", tag, tag),
			fmt.Sprintf("scoreboard players set %s %s 0", index.holder, index.objective),
			fmt.Sprintf("execute store result score %s %s run data get storage %s lists.v%d", length.holder, length.objective, c.storageName(), listID),
			fmt.Sprintf("execute if score %s %s < %s %s run function %s:%s", index.holder, index.objective, length.holder, length.objective, c.functionNamespace, controller),
		}
		c.output.Tick = append(c.output.Tick, fmt.Sprintf("function %s:%s", c.functionNamespace, start))
	}
}

func (c *compiler) compileIndexPath(base string, indices []ast.Expression, id ast.ScopeID, function *ast.Function) ([]string, string, bool, error) {
	var commands []string
	path := base
	dynamic := false
	for i, expression := range indices {
		if literal, ok := integerLiteral(expression); ok {
			path += "[" + strconv.FormatInt(literal, 10) + "]"
			continue
		}
		compiled, index, err := c.compileExpression(expression, id, function)
		if err != nil {
			return nil, "", false, err
		}
		commands = append(commands, compiled...)
		key := "index" + strconv.Itoa(i)
		commands = append(commands, fmt.Sprintf("execute store result storage %s scratch.%s int 1 run scoreboard players get %s %s", c.storageName(), key, index.holder, index.objective))
		path += "[$(" + key + ")]"
		dynamic = true
	}
	return commands, path, dynamic, nil
}

func expressionListDepth(expression ast.Expression) int {
	list, ok := expression.(*ast.List)
	if !ok {
		return 0
	}
	depth := 1
	for _, element := range list.Elements {
		if nested := 1 + expressionListDepth(element); nested > depth {
			depth = nested
		}
	}
	return depth
}

func (c *compiler) compileExpression(expression ast.Expression, id ast.ScopeID, function *ast.Function) ([]string, value, error) {
	switch expression := expression.(type) {
	case *ast.Integer:
		c.constants[expression.Value] = struct{}{}
		return nil, value{holder: constantHolder(expression.Value), objective: c.objective}, nil
	case *ast.Boolean:
		number := int64(0)
		if expression.Value {
			number = 1
		}
		c.constants[number] = struct{}{}
		return nil, value{holder: constantHolder(number), objective: c.objective}, nil
	case *ast.Identifier:
		variableID, ok := c.resolve(expression.Name, id, function)
		if !ok {
			return nil, value{}, Error{Position: expression.Pos, Message: fmt.Sprintf("undefined variable %q", expression.Name)}
		}
		if _, list := c.listVariables[variableID]; list {
			return nil, value{}, Error{Position: expression.Pos, Message: fmt.Sprintf("list %q must be indexed", expression.Name)}
		}
		if c.isString(variableID) {
			return nil, value{}, Error{Position: expression.Pos, Message: fmt.Sprintf("string %q can only be displayed, assigned, or compared", expression.Name)}
		}
		return nil, c.variableValue(variableID), nil
	case *ast.Index:
		return c.compileListIndex(expression, id, function)
	case *ast.Unary:
		commands, right, err := c.compileExpression(expression.Right, id, function)
		if err != nil {
			return nil, value{}, err
		}
		if expression.Operator == token.Plus {
			return commands, right, nil
		}
		temporary := c.newTemporary()
		if expression.Operator == token.Minus {
			c.constants[0] = struct{}{}
			commands = append(commands, scoreboardOperation(temporary, "=", value{constantHolder(0), c.objective}))
			commands = append(commands, scoreboardOperation(temporary, "-=", right))
			return commands, temporary, nil
		}
		if expression.Operator == token.Not {
			commands = append(commands, fmt.Sprintf("scoreboard players set %s %s 1", temporary.holder, temporary.objective))
			commands = append(commands, fmt.Sprintf("execute unless score %s %s matches 0 run scoreboard players set %s %s 0", right.holder, right.objective, temporary.holder, temporary.objective))
			return commands, temporary, nil
		}
	case *ast.Binary:
		if expression.Operator == token.Is {
			return c.compileTypeTest(expression, id, function)
		}
		if expression.Operator == token.In {
			return c.compileEntitySetMembership(expression, id, function)
		}
		return c.compileBinary(expression, id, function)
	case *ast.Call:
		return c.compileCall(expression, id, function, true)
	}
	return nil, value{}, Error{Position: expression.Position(), Message: fmt.Sprintf("expression %T is not supported yet", expression)}
}

func (c *compiler) compileTypeTest(expression *ast.Binary, id ast.ScopeID, function *ast.Function) ([]string, value, error) {
	typeName, ok := expression.Right.(*ast.Identifier)
	if !ok || (typeName.Name != "bool" && typeName.Name != "int" && typeName.Name != "str" && typeName.Name != "list" && typeName.Name != "entity" && typeName.Name != "nbt") {
		return nil, value{}, Error{Position: expression.Right.Position(), Message: "is expects bool, int, str, list, entity, or nbt"}
	}
	result := c.newTemporary()
	commands := []string{fmt.Sprintf("scoreboard players set %s %s 0", result.holder, result.objective)}
	if runtimeCommands, runtimePath, dynamic, found, err := c.runtimeTypePath(expression.Left, id, function); err != nil {
		return nil, value{}, err
	} else if found {
		commands = append(commands, runtimeCommands...)
		actualPath := "scratch.runtime_type" + strconv.FormatUint(c.stringTemporary, 10)
		c.stringTemporary++
		copyCommand := fmt.Sprintf("data modify storage %s %s set from storage %s %s", c.storageName(), actualPath, c.storageName(), runtimePath)
		if dynamic {
			helper := c.reserveInternalFunction()
			c.output.Functions[helper] = []string{"$" + copyCommand}
			commands = append(commands, fmt.Sprintf("function %s:%s with storage %s scratch", c.functionNamespace, helper, c.storageName()))
		} else {
			commands = append(commands, copyCommand)
		}
		wanted := typeName.Name
		if wanted == "bool" {
			wanted = "int"
		}
		expectedPath := actualPath + "_expected"
		encodedWanted, _ := json.Marshal(wanted)
		commands = append(commands, fmt.Sprintf("data modify storage %s %s set value %s", c.storageName(), expectedPath, encodedWanted))
		objective := "string_" + strconv.FormatUint(c.stringTemporary, 10)
		c.stringTemporary++
		commands = append(commands, "scoreboard objectives add "+objective+" dummy")
		commands = append(commands, fmt.Sprintf("execute store success score #compare %s run data modify storage %s %s set from storage %s %s", objective, c.storageName(), actualPath, c.storageName(), expectedPath))
		commands = append(commands, fmt.Sprintf("scoreboard players set %s %s 1", result.holder, result.objective))
		commands = append(commands, fmt.Sprintf("execute if score #compare %s matches 1 run scoreboard players set %s %s 0", objective, result.holder, result.objective))
		if typeName.Name == "bool" {
			compiled, number, compileErr := c.compileExpression(expression.Left, id, function)
			if compileErr != nil {
				return nil, value{}, compileErr
			}
			commands = append(commands, compiled...)
			commands = append(commands, fmt.Sprintf("execute unless score %s %s matches 0..1 run scoreboard players set %s %s 0", number.holder, number.objective, result.holder, result.objective))
		}
		commands = append(commands, fmt.Sprintf("data remove storage %s %s", c.storageName(), actualPath))
		commands = append(commands, fmt.Sprintf("data remove storage %s %s", c.storageName(), expectedPath))
		commands = append(commands, "scoreboard objectives remove "+objective)
		return commands, result, nil
	}
	actual := c.expressionType(expression.Left, id, function)
	if typeName.Name == "bool" {
		if actual != "int" {
			return commands, result, nil
		}
		compiled, number, err := c.compileExpression(expression.Left, id, function)
		if err != nil {
			return nil, value{}, err
		}
		commands = append(commands, compiled...)
		commands = append(commands, fmt.Sprintf("execute if score %s %s matches 0..1 run scoreboard players set %s %s 1", number.holder, number.objective, result.holder, result.objective))
		return commands, result, nil
	}
	wanted := typeName.Name
	if actual == wanted {
		commands = append(commands, fmt.Sprintf("scoreboard players set %s %s 1", result.holder, result.objective))
	}
	return commands, result, nil
}

func (c *compiler) runtimeTypePath(expression ast.Expression, id ast.ScopeID, function *ast.Function) ([]string, string, bool, bool, error) {
	if identifier, ok := expression.(*ast.Identifier); ok {
		if variableID, found := c.resolve(identifier.Name, id, function); found {
			if _, variant := c.variantVariables[variableID]; variant {
				return nil, fmt.Sprintf("variant_types.v%d", variableID), false, true, nil
			}
		}
	}
	if indexed, ok := expression.(*ast.Index); ok {
		root, indices := compilerIndexedRoot(indexed)
		if root != nil {
			if variableID, found := c.resolve(root.Name, id, function); found && c.isList(variableID) {
				commands, path, dynamic, err := c.compileIndexPath(fmt.Sprintf("list_types.v%d", variableID), indices, id, function)
				return commands, path, dynamic, true, err
			}
		}
	}
	return nil, "", false, false, nil
}

func (c *compiler) expressionType(expression ast.Expression, id ast.ScopeID, function *ast.Function) string {
	switch expression := expression.(type) {
	case *ast.Integer, *ast.Boolean, *ast.Unary:
		return "int"
	case *ast.String:
		return "str"
	case *ast.EntitySelector:
		return "entity"
	case *ast.List:
		return "list"
	case *ast.NBT:
		return "nbt"
	case *ast.Identifier:
		if variableID, found := c.resolve(expression.Name, id, function); found {
			if kind := c.variableTypes[variableID]; kind != "" {
				return kind
			}
			if c.isList(variableID) {
				return "list"
			}
			if c.isString(variableID) {
				return "str"
			}
			return "int"
		}
	case *ast.Index:
		root, indices := compilerIndexedRoot(expression)
		if root != nil && len(indices) == 1 {
			if variableID, found := c.resolve(root.Name, id, function); found {
				if literal, ok := integerLiteral(indices[0]); ok && literal >= 0 && int(literal) < len(c.listElementTypes[variableID]) {
					return c.listElementTypes[variableID][literal]
				}
			}
		}
		return "int"
	case *ast.Binary:
		if expression.Operator == token.Plus && c.isStringExpression(expression.Left, id, function) && c.isStringExpression(expression.Right, id, function) {
			return "str"
		}
		return "int"
	case *ast.Call:
		if name, ok := callCallee(expression); ok && name == "str" {
			return "str"
		}
	}
	return ""
}

func (c *compiler) compileListIndex(expression *ast.Index, id ast.ScopeID, function *ast.Function) ([]string, value, error) {
	target, indices := compilerIndexedRoot(expression)
	if target == nil {
		return nil, value{}, Error{Position: expression.Pos, Message: "list indexing requires a list variable"}
	}
	variableID, ok := c.resolve(target.Name, id, function)
	if !ok {
		return nil, value{}, Error{Position: target.Pos, Message: fmt.Sprintf("undefined variable %q", target.Name)}
	}
	if _, list := c.listVariables[variableID]; !list {
		return nil, value{}, Error{Position: target.Pos, Message: fmt.Sprintf("%q is not a list", target.Name)}
	}
	if depth := c.listDepth[variableID]; depth > 0 && len(indices) < depth {
		return nil, value{}, Error{Position: expression.Pos, Message: "sublist must be indexed again or passed to say"}
	}
	result := c.newTemporary()
	commands, path, dynamic, err := c.compileIndexPath(fmt.Sprintf("lists.v%d", variableID), indices, id, function)
	if err != nil {
		return nil, value{}, err
	}
	read := fmt.Sprintf("execute store result score %s %s run data get storage %s %s 1", result.holder, result.objective, c.storageName(), path)
	if dynamic {
		helperName := c.reserveInternalFunction()
		c.output.Functions[helperName] = []string{"$" + read}
		commands = append(commands, fmt.Sprintf("function %s:%s with storage %s scratch", c.functionNamespace, helperName, c.storageName()))
	} else {
		commands = append(commands, read)
	}
	return commands, result, nil
}

func compilerIndexedRoot(indexed *ast.Index) (*ast.Identifier, []ast.Expression) {
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

func (c *compiler) storageName() string {
	return c.objective + ":data"
}

func (c *compiler) compileBinary(expression *ast.Binary, id ast.ScopeID, function *ast.Function) ([]string, value, error) {
	leftString := c.isStringExpression(expression.Left, id, function)
	rightString := c.isStringExpression(expression.Right, id, function)
	if leftString || rightString {
		if !leftString || !rightString {
			return nil, value{}, Error{Position: expression.Pos, Message: "cannot compare a string with a non-string value"}
		}
		if expression.Operator != token.Equal && expression.Operator != token.NotEqual {
			return nil, value{}, Error{Position: expression.Pos, Message: "strings only support == and != comparisons"}
		}
		return c.compileStringComparison(expression, id, function)
	}
	leftCommands, left, err := c.compileExpression(expression.Left, id, function)
	if err != nil {
		return nil, value{}, err
	}
	rightCommands, right, err := c.compileExpression(expression.Right, id, function)
	if err != nil {
		return nil, value{}, err
	}
	commands := append(leftCommands, rightCommands...)
	temporary := c.newTemporary()

	if operation, ok := map[token.Kind]string{
		token.Plus: "+=", token.Minus: "-=", token.Star: "*=", token.Slash: "/=", token.Percent: "%=",
	}[expression.Operator]; ok {
		commands = append(commands, scoreboardOperation(temporary, "=", left))
		commands = append(commands, scoreboardOperation(temporary, operation, right))
		return commands, temporary, nil
	}

	commands = append(commands, fmt.Sprintf("scoreboard players set %s %s 0", temporary.holder, temporary.objective))
	if expression.Operator == token.And {
		commands = append(commands, fmt.Sprintf("execute unless score %s %s matches 0 unless score %s %s matches 0 run scoreboard players set %s %s 1", left.holder, left.objective, right.holder, right.objective, temporary.holder, temporary.objective))
		return commands, temporary, nil
	}
	if expression.Operator == token.Or {
		commands = append(commands, fmt.Sprintf("execute unless score %s %s matches 0 run scoreboard players set %s %s 1", left.holder, left.objective, temporary.holder, temporary.objective))
		commands = append(commands, fmt.Sprintf("execute unless score %s %s matches 0 run scoreboard players set %s %s 1", right.holder, right.objective, temporary.holder, temporary.objective))
		return commands, temporary, nil
	}

	comparison := map[token.Kind]string{
		token.Equal: "=", token.Less: "<", token.LessEqual: "<=",
		token.Greater: ">", token.GreaterEqual: ">=",
	}[expression.Operator]
	if expression.Operator == token.NotEqual {
		commands = append(commands, fmt.Sprintf("execute unless score %s %s = %s %s run scoreboard players set %s %s 1", left.holder, left.objective, right.holder, right.objective, temporary.holder, temporary.objective))
		return commands, temporary, nil
	}
	if comparison != "" {
		commands = append(commands, fmt.Sprintf("execute if score %s %s %s %s %s run scoreboard players set %s %s 1", left.holder, left.objective, comparison, right.holder, right.objective, temporary.holder, temporary.objective))
		return commands, temporary, nil
	}
	return nil, value{}, Error{Position: expression.Pos, Message: fmt.Sprintf("operator %s is not supported", expression.Operator)}
}

func (c *compiler) isStringExpression(expression ast.Expression, id ast.ScopeID, function *ast.Function) bool {
	if _, ok := expression.(*ast.String); ok {
		return true
	}
	if identifier, ok := expression.(*ast.Identifier); ok {
		if variableID, found := c.resolve(identifier.Name, id, function); found {
			return c.isString(variableID)
		}
	}
	if _, ok := expression.(*ast.Index); ok {
		return c.expressionType(expression, id, function) == "str"
	}
	if binary, ok := expression.(*ast.Binary); ok && binary.Operator == token.Plus {
		return c.isStringExpression(binary.Left, id, function) && c.isStringExpression(binary.Right, id, function)
	}
	if call, ok := expression.(*ast.Call); ok {
		if name, named := callCallee(call); named && name == "str" {
			return true
		}
	}
	return false
}

func (c *compiler) compileStringExpressionToPath(expression ast.Expression, path string, id ast.ScopeID, function *ast.Function) ([]string, error) {
	switch expression := expression.(type) {
	case *ast.String:
		encoded, _ := json.Marshal(expression.Value)
		return []string{fmt.Sprintf("data modify storage %s %s set value %s", c.storageName(), path, encoded)}, nil
	case *ast.Identifier:
		variableID, found := c.resolve(expression.Name, id, function)
		if !found || !c.isString(variableID) {
			return nil, Error{Position: expression.Pos, Message: fmt.Sprintf("%q is not a string", expression.Name)}
		}
		return []string{fmt.Sprintf("data modify storage %s %s set from storage %s strings.v%d", c.storageName(), path, c.storageName(), variableID)}, nil
	case *ast.Index:
		root, indices := compilerIndexedRoot(expression)
		if root == nil {
			return nil, Error{Position: expression.Pos, Message: "string list indexing requires a list variable"}
		}
		variableID, found := c.resolve(root.Name, id, function)
		if !found || !c.isList(variableID) || c.expressionType(expression, id, function) != "str" {
			return nil, Error{Position: expression.Pos, Message: "indexed value is not a string"}
		}
		commands, source, dynamic, err := c.compileIndexPath(fmt.Sprintf("lists.v%d", variableID), indices, id, function)
		if err != nil {
			return nil, err
		}
		copyCommand := fmt.Sprintf("data modify storage %s %s set from storage %s %s", c.storageName(), path, c.storageName(), source)
		if dynamic {
			helper := c.reserveInternalFunction()
			c.output.Functions[helper] = []string{"$" + copyCommand}
			commands = append(commands, fmt.Sprintf("function %s:%s with storage %s scratch", c.functionNamespace, helper, c.storageName()))
		} else {
			commands = append(commands, copyCommand)
		}
		return commands, nil
	case *ast.Call:
		name, named := callCallee(expression)
		if !named || name != "str" || len(expression.Arguments) != 1 {
			return nil, Error{Position: expression.Pos, Message: "str expects 1 integer or boolean argument"}
		}
		commands, number, err := c.compileExpression(expression.Arguments[0], id, function)
		if err != nil {
			return nil, err
		}
		temporaryID := c.stringTemporary
		c.stringTemporary++
		base := "scratch.str" + strconv.FormatUint(temporaryID, 10)
		commands = append(commands, fmt.Sprintf("execute store result storage %s %s.value int 1 run scoreboard players get %s %s", c.storageName(), base, number.holder, number.objective))
		encodedPath, _ := json.Marshal(path)
		commands = append(commands, fmt.Sprintf("data modify storage %s %s.destination set value %s", c.storageName(), base, encodedPath))
		helper := c.specialFunction("string_from_int", []string{fmt.Sprintf("$data modify storage %s $(destination) set value \"$(value)\"", c.storageName())})
		commands = append(commands, fmt.Sprintf("function %s:%s with storage %s %s", c.functionNamespace, helper, c.storageName(), base))
		commands = append(commands, fmt.Sprintf("data remove storage %s %s", c.storageName(), base))
		return commands, nil
	case *ast.Binary:
		if expression.Operator != token.Plus {
			break
		}
		temporaryID := c.stringTemporary
		c.stringTemporary++
		base := "scratch.concat" + strconv.FormatUint(temporaryID, 10)
		left, err := c.compileStringExpressionToPath(expression.Left, base+".left", id, function)
		if err != nil {
			return nil, err
		}
		right, err := c.compileStringExpressionToPath(expression.Right, base+".right", id, function)
		if err != nil {
			return nil, err
		}
		commands := append(left, right...)
		encodedPath, _ := json.Marshal(path)
		commands = append(commands, fmt.Sprintf("data modify storage %s %s.destination set value %s", c.storageName(), base, encodedPath))
		helper := c.specialFunction("string_concat", []string{fmt.Sprintf("$data modify storage %s $(destination) set value \"$(left)$(right)\"", c.storageName())})
		commands = append(commands, fmt.Sprintf("function %s:%s with storage %s %s", c.functionNamespace, helper, c.storageName(), base))
		commands = append(commands, fmt.Sprintf("data remove storage %s %s", c.storageName(), base))
		return commands, nil
	}
	return nil, Error{Position: expression.Position(), Message: "expected a string expression"}
}

func (c *compiler) compileStringComparison(expression *ast.Binary, id ast.ScopeID, function *ast.Function) ([]string, value, error) {
	comparisonID := c.stringTemporary
	c.stringTemporary++
	objective := "string_" + strconv.FormatUint(comparisonID, 10)
	base := "scratch." + objective
	commands := []string{"scoreboard objectives add " + objective + " dummy"}
	for i, operand := range []ast.Expression{expression.Left, expression.Right} {
		path := base + ".left"
		if i == 1 {
			path = base + ".right"
		}
		compiled, err := c.compileStringExpressionToPath(operand, path, id, function)
		if err != nil {
			return nil, value{}, err
		}
		commands = append(commands, compiled...)
	}
	commands = append(commands, fmt.Sprintf("execute store success score #compare %s run data modify storage %s %s.left set from storage %s %s.right", objective, c.storageName(), base, c.storageName(), base))
	result := c.newTemporary()
	defaultValue, changedValue := 1, 0
	if expression.Operator == token.NotEqual {
		defaultValue, changedValue = 0, 1
	}
	commands = append(commands, fmt.Sprintf("scoreboard players set %s %s %d", result.holder, result.objective, defaultValue))
	commands = append(commands, fmt.Sprintf("execute if score #compare %s matches 1 run scoreboard players set %s %s %d", objective, result.holder, result.objective, changedValue))
	commands = append(commands, fmt.Sprintf("data remove storage %s %s", c.storageName(), base))
	commands = append(commands, "scoreboard objectives remove "+objective)
	return commands, result, nil
}

func (c *compiler) compileBool(call *ast.Call, id ast.ScopeID, function *ast.Function) ([]string, value, error) {
	if len(call.Arguments) != 1 {
		return nil, value{}, Error{Position: call.Pos, Message: "bool expects 1 argument"}
	}
	argument := call.Arguments[0]
	if list, ok := argument.(*ast.List); ok {
		number := int64(0)
		if len(list.Elements) > 0 {
			number = 1
		}
		c.constants[number] = struct{}{}
		return nil, value{holder: constantHolder(number), objective: c.objective}, nil
	}
	if c.isStringExpression(argument, id, function) {
		comparison := &ast.Binary{Pos: call.Pos, Left: argument, Operator: token.NotEqual, Right: &ast.String{Pos: call.Pos, Value: ""}}
		return c.compileStringComparison(comparison, id, function)
	}
	if identifier, ok := argument.(*ast.Identifier); ok {
		if variableID, found := c.resolve(identifier.Name, id, function); found && c.isList(variableID) {
			length := c.newTemporary()
			result := c.newTemporary()
			commands := []string{
				fmt.Sprintf("execute store result score %s %s run data get storage %s lists.v%d", length.holder, length.objective, c.storageName(), variableID),
				fmt.Sprintf("scoreboard players set %s %s 0", result.holder, result.objective),
				fmt.Sprintf("execute unless score %s %s matches 0 run scoreboard players set %s %s 1", length.holder, length.objective, result.holder, result.objective),
			}
			return commands, result, nil
		}
	}
	commands, number, err := c.compileExpression(argument, id, function)
	if err != nil {
		return nil, value{}, err
	}
	result := c.newTemporary()
	commands = append(commands, fmt.Sprintf("scoreboard players set %s %s 0", result.holder, result.objective))
	commands = append(commands, fmt.Sprintf("execute unless score %s %s matches 0 run scoreboard players set %s %s 1", number.holder, number.objective, result.holder, result.objective))
	return commands, result, nil
}

func (c *compiler) compileCall(call *ast.Call, id ast.ScopeID, function *ast.Function, requireValue bool) ([]string, value, error) {
	if attribute, ok := call.Callee.(*ast.Attribute); ok {
		if identifier, targetOK := attribute.Target.(*ast.Identifier); targetOK {
			if variableID, found := c.resolve(identifier.Name, id, function); found && c.isEntitySet(variableID) {
				return c.compileEntitySetMethod(call, attribute, variableID, id, function, requireValue)
			}
			if variableID, found := c.resolve(identifier.Name, id, function); found && c.isPrimitiveSet(variableID) {
				return c.compilePrimitiveSetMethod(call, attribute, variableID, id, function, requireValue)
			}
		}
		return c.compileListMethod(call, attribute, id, function, requireValue)
	}
	callee, ok := call.Callee.(*ast.Identifier)
	if !ok {
		return nil, value{}, Error{Position: call.Pos, Message: "function name must be an identifier"}
	}
	if callee.Name == "bool" {
		return c.compileBool(call, id, function)
	}
	if callee.Name == "say" {
		if requireValue {
			return nil, value{}, Error{Position: call.Pos, Message: "say does not return a value"}
		}
		return c.compileSay(call, id, function)
	}
	if callee.Name == "len" {
		if len(call.Arguments) != 1 {
			return nil, value{}, Error{Position: call.Pos, Message: "len expects 1 argument"}
		}
		identifier, ok := call.Arguments[0].(*ast.Identifier)
		if !ok {
			return nil, value{}, Error{Position: call.Pos, Message: "len expects a list variable"}
		}
		collectionID, found := c.resolve(identifier.Name, id, function)
		if !found {
			return nil, value{}, Error{Position: identifier.Pos, Message: fmt.Sprintf("undefined collection %q", identifier.Name)}
		}
		result := c.newTemporary()
		if c.isEntitySet(collectionID) {
			return []string{
				fmt.Sprintf("scoreboard players set %s %s 0", result.holder, result.objective),
				fmt.Sprintf("execute as @e[tag=%s] run scoreboard players add %s %s 1", c.entitySetTag(collectionID), result.holder, result.objective),
			}, result, nil
		}
		if c.isPrimitiveSet(collectionID) {
			return []string{fmt.Sprintf("execute store result score %s %s run data get storage %s sets.v%d.length 1", result.holder, result.objective, c.storageName(), collectionID)}, result, nil
		}
		if c.isNBT(collectionID) {
			return []string{fmt.Sprintf("execute store result score %s %s run data get storage %s nbt.v%d", result.holder, result.objective, c.storageName(), collectionID)}, result, nil
		}
		if !c.isList(collectionID) {
			return nil, value{}, Error{Position: identifier.Pos, Message: fmt.Sprintf("%q is not a collection", identifier.Name)}
		}
		return []string{fmt.Sprintf("execute store result score %s %s run data get storage %s lists.v%d", result.holder, result.objective, c.storageName(), collectionID)}, result, nil
	}
	mapping, targetNamespace, targetObjective, ok := c.callTarget(callee.Name)
	if !ok {
		return nil, value{}, Error{Position: callee.Pos, Message: fmt.Sprintf("undefined function %q", callee.Name)}
	}
	targetStorage := targetObjective + ":data"
	if len(call.Arguments) != len(mapping.Parameters) {
		return nil, value{}, Error{
			Position: call.Pos,
			Message:  fmt.Sprintf("function %q expects %d arguments, got %d", callee.Name, len(mapping.Parameters), len(call.Arguments)),
		}
	}

	var commands []string
	arguments := make([]value, 0, len(call.Arguments))
	for i, argument := range call.Arguments {
		if mapping.Parameters[i].IsNBT {
			identifier, ok := argument.(*ast.Identifier)
			if !ok {
				return nil, value{}, Error{Position: argument.Position(), Message: "NBT argument must be an nbt variable"}
			}
			sourceID, found := c.resolve(identifier.Name, id, function)
			if !found || !c.isNBT(sourceID) {
				return nil, value{}, Error{Position: identifier.Pos, Message: fmt.Sprintf("%q is not an nbt variable", identifier.Name)}
			}
			commands = append(commands, fmt.Sprintf("data modify storage %s nbt.v%d set from storage %s nbt.v%d", targetStorage, mapping.Parameters[i].VariableID, c.storageName(), sourceID))
			arguments = append(arguments, value{})
			continue
		}
		if mapping.Parameters[i].IsPrimitiveSet {
			identifier, ok := argument.(*ast.Identifier)
			if !ok {
				return nil, value{}, Error{Position: argument.Position(), Message: "primitive set argument must be a set variable"}
			}
			sourceID, found := c.resolve(identifier.Name, id, function)
			if !found || !c.isPrimitiveSet(sourceID) {
				return nil, value{}, Error{Position: identifier.Pos, Message: fmt.Sprintf("%q is not a primitive set", identifier.Name)}
			}
			commands = append(commands, fmt.Sprintf("data modify storage %s sets.v%d set from storage %s sets.v%d", targetStorage, mapping.Parameters[i].VariableID, c.storageName(), sourceID))
			arguments = append(arguments, value{})
			continue
		}
		if mapping.Parameters[i].IsEntitySet {
			compiled, err := c.compileEntitySetExpression(argument, entitySetTagFor(targetObjective, mapping.Parameters[i].VariableID), id, function)
			if err != nil {
				return nil, value{}, err
			}
			commands = append(commands, compiled...)
			arguments = append(arguments, value{})
			continue
		}
		if mapping.Parameters[i].IsString {
			argumentPath := fmt.Sprintf("scratch.import_string_%d", i)
			compiled, err := c.compileStringExpressionToPath(argument, argumentPath, id, function)
			if err != nil {
				return nil, value{}, err
			}
			commands = append(commands, compiled...)
			commands = append(commands, fmt.Sprintf("data modify storage %s strings.v%d set from storage %s %s", targetStorage, mapping.Parameters[i].VariableID, c.storageName(), argumentPath))
			commands = append(commands, fmt.Sprintf("data remove storage %s %s", c.storageName(), argumentPath))
			arguments = append(arguments, value{})
			continue
		}
		if mapping.Parameters[i].IsList {
			identifier, ok := argument.(*ast.Identifier)
			if !ok {
				return nil, value{}, Error{Position: argument.Position(), Message: "list argument must be a list variable"}
			}
			variableID, found := c.resolve(identifier.Name, id, function)
			if !found || !c.isList(variableID) {
				return nil, value{}, Error{Position: identifier.Pos, Message: fmt.Sprintf("%q is not a list", identifier.Name)}
			}
			commands = append(commands,
				fmt.Sprintf("data modify storage %s lists.v%d set from storage %s lists.v%d", targetStorage, mapping.Parameters[i].VariableID, c.storageName(), variableID),
				fmt.Sprintf("data modify storage %s list_types.v%d set from storage %s list_types.v%d", targetStorage, mapping.Parameters[i].VariableID, c.storageName(), variableID),
			)
			arguments = append(arguments, value{})
			continue
		}
		compiled, result, err := c.compileExpression(argument, id, function)
		if err != nil {
			return nil, value{}, err
		}
		commands = append(commands, compiled...)
		stable := c.newTemporary()
		commands = append(commands, scoreboardOperation(stable, "=", result))
		arguments = append(arguments, stable)
	}
	for i, argument := range arguments {
		if mapping.Parameters[i].IsList || mapping.Parameters[i].IsString || mapping.Parameters[i].IsEntitySet || mapping.Parameters[i].IsPrimitiveSet || mapping.Parameters[i].IsNBT {
			continue
		}
		parameter := value{holder: mapping.Parameters[i].Holder, objective: targetObjective}
		commands = append(commands, scoreboardOperation(parameter, "=", argument))
	}
	commands = append(commands, "function "+targetNamespace+":"+mapping.GeneratedName)
	returned := value{holder: mapping.ReturnHolder, objective: targetObjective}
	if mapping.ReturnsList {
		if requireValue {
			return nil, value{}, Error{Position: call.Pos, Message: fmt.Sprintf("list returned by %q must be assigned to a variable", callee.Name)}
		}
		return commands, value{}, nil
	}
	if mapping.ReturnsEntitySet {
		if requireValue {
			return nil, value{}, Error{Position: call.Pos, Message: fmt.Sprintf("entity set returned by %q must be assigned to a variable", callee.Name)}
		}
		return commands, value{}, nil
	}
	if mapping.ReturnsPrimitiveSet {
		if requireValue {
			return nil, value{}, Error{Position: call.Pos, Message: fmt.Sprintf("primitive set returned by %q must be assigned to a variable", callee.Name)}
		}
		return commands, value{}, nil
	}
	if mapping.ReturnsNBT {
		if requireValue {
			return nil, value{}, Error{Position: call.Pos, Message: fmt.Sprintf("NBT returned by %q must be assigned to a variable", callee.Name)}
		}
		return commands, value{}, nil
	}
	if requireValue && !mapping.ReturnsValue {
		return nil, value{}, Error{Position: call.Pos, Message: fmt.Sprintf("function %q does not return a value", callee.Name)}
	}
	return commands, returned, nil
}

func (c *compiler) compileRemoveToVariant(call *ast.Call, attribute *ast.Attribute, destination uint32, id ast.ScopeID, function *ast.Function) ([]string, error) {
	target, ok := attribute.Target.(*ast.Identifier)
	if !ok || len(call.Arguments) > 1 {
		return nil, Error{Position: call.Pos, Message: "remove expects 0 or 1 indexes"}
	}
	listID, found := c.resolve(target.Name, id, function)
	if !found || !c.isList(listID) {
		return nil, Error{Position: target.Pos, Message: "remove requires a list"}
	}
	path := "-1"
	var commands []string
	dynamic := false
	if len(call.Arguments) == 1 {
		if literal, ok := integerLiteral(call.Arguments[0]); ok {
			path = strconv.FormatInt(literal, 10)
		} else {
			compiled, index, err := c.compileExpression(call.Arguments[0], id, function)
			if err != nil {
				return nil, err
			}
			commands = append(commands, compiled...)
			commands = append(commands, fmt.Sprintf("execute store result storage %s scratch.index int 1 run scoreboard players get %s %s", c.storageName(), index.holder, index.objective))
			path, dynamic = "$(index)", true
		}
	}
	operations := []string{
		fmt.Sprintf("data modify storage %s variants.v%d set from storage %s lists.v%d[%s]", c.storageName(), destination, c.storageName(), listID, path),
		fmt.Sprintf("data modify storage %s variant_types.v%d set from storage %s list_types.v%d[%s]", c.storageName(), destination, c.storageName(), listID, path),
		fmt.Sprintf("execute store result score %s %s run data get storage %s lists.v%d[%s] 1", variableHolder(destination), c.objective, c.storageName(), listID, path),
		fmt.Sprintf("data remove storage %s lists.v%d[%s]", c.storageName(), listID, path),
		fmt.Sprintf("data remove storage %s list_types.v%d[%s]", c.storageName(), listID, path),
	}
	if dynamic {
		helper := c.reserveInternalFunction()
		for i := range operations {
			operations[i] = "$" + operations[i]
		}
		c.output.Functions[helper] = operations
		commands = append(commands, fmt.Sprintf("function %s:%s with storage %s scratch", c.functionNamespace, helper, c.storageName()))
	} else {
		commands = append(commands, operations...)
	}
	return commands, nil
}

func (c *compiler) compileEntitySetMethod(call *ast.Call, attribute *ast.Attribute, setID uint32, id ast.ScopeID, function *ast.Function, requireValue bool) ([]string, value, error) {
	if requireValue {
		return nil, value{}, Error{Position: call.Pos, Message: attribute.Name + " does not return a value"}
	}
	tag := c.entitySetTag(setID)
	if attribute.Name == "clear" {
		if len(call.Arguments) != 0 {
			return nil, value{}, Error{Position: call.Pos, Message: "clear expects no arguments"}
		}
		return []string{fmt.Sprintf("tag @e[tag=%s] remove %s", tag, tag)}, value{}, nil
	}
	if attribute.Name != "add" && attribute.Name != "discard" && attribute.Name != "remove" {
		return nil, value{}, Error{Position: attribute.Pos, Message: fmt.Sprintf("unknown entity-set method %q", attribute.Name)}
	}
	if len(call.Arguments) != 1 {
		return nil, value{}, Error{Position: call.Pos, Message: attribute.Name + " expects one entity"}
	}
	operation := "add"
	if attribute.Name == "discard" || attribute.Name == "remove" {
		operation = "remove"
	}
	commands, selector, dynamic, err := c.compileEntitySelector(call.Arguments[0], id, function)
	if err != nil {
		return nil, value{}, err
	}
	operations := []string{}
	if attribute.Name == "remove" {
		operations = append(operations, "execute as "+selector+" unless entity @s[tag="+tag+"] run return fail")
	}
	operations = append(operations, fmt.Sprintf("tag %s %s %s", selector, operation, tag))
	if dynamic {
		helper := c.reserveInternalFunction()
		for index := range operations {
			operations[index] = "$" + operations[index]
		}
		c.output.Functions[helper] = operations
		commands = append(commands, fmt.Sprintf("function %s:%s with storage %s scratch", c.functionNamespace, helper, c.storageName()))
	} else {
		commands = append(commands, operations...)
	}
	return commands, value{}, nil
}

func (c *compiler) compileEntitySetMembership(expression *ast.Binary, id ast.ScopeID, function *ast.Function) ([]string, value, error) {
	set, ok := expression.Right.(*ast.Identifier)
	if !ok {
		return nil, value{}, Error{Position: expression.Right.Position(), Message: "membership requires an entity-set variable"}
	}
	setID, found := c.resolve(set.Name, id, function)
	if found && c.isPrimitiveSet(setID) {
		return c.compilePrimitiveSetMembership(expression.Left, setID, id, function)
	}
	if !found || !c.isEntitySet(setID) {
		return nil, value{}, Error{Position: set.Pos, Message: fmt.Sprintf("%q is not an entity set", set.Name)}
	}
	commands, selector, dynamic, err := c.compileEntitySelector(expression.Left, id, function)
	if err != nil {
		return nil, value{}, err
	}
	result := c.newTemporary()
	commands = append(commands, fmt.Sprintf("scoreboard players set %s %s 0", result.holder, result.objective))
	check := fmt.Sprintf("execute as %s if entity @s[tag=%s] run scoreboard players set %s %s 1", selector, c.entitySetTag(setID), result.holder, result.objective)
	if dynamic {
		helper := c.reserveInternalFunction()
		c.output.Functions[helper] = []string{"$" + check}
		commands = append(commands, fmt.Sprintf("function %s:%s with storage %s scratch", c.functionNamespace, helper, c.storageName()))
	} else {
		commands = append(commands, check)
	}
	return commands, result, nil
}

func (c *compiler) compileEntitySelector(expression ast.Expression, id ast.ScopeID, function *ast.Function) ([]string, string, bool, error) {
	if selector, ok := expression.(*ast.EntitySelector); ok {
		return nil, selector.Value, false, nil
	}
	identifier, ok := expression.(*ast.Identifier)
	if !ok {
		return nil, "", false, Error{Position: expression.Position(), Message: "expected an entity selector or entity variable"}
	}
	variableID, found := c.resolve(identifier.Name, id, function)
	if !found {
		return nil, "", false, Error{Position: identifier.Pos, Message: fmt.Sprintf("undefined entity %q", identifier.Name)}
	}
	if _, entity := c.entityVariables[variableID]; !entity {
		return nil, "", false, Error{Position: identifier.Pos, Message: fmt.Sprintf("%q is not an entity", identifier.Name)}
	}
	commands := []string{fmt.Sprintf("data modify storage %s scratch.entity_uuid set from storage %s entities.v%d.uuid", c.storageName(), c.storageName(), variableID)}
	for part := 0; part < 4; part++ {
		commands = append(commands, fmt.Sprintf("execute store result storage %s scratch.uuid%d int 1 run data get storage %s scratch.entity_uuid[%d]", c.storageName(), part, c.storageName(), part))
	}
	return commands, "@n[tag=" + c.entityUUIDTagMacro() + "]", true, nil
}

func (c *compiler) compileListMethod(call *ast.Call, attribute *ast.Attribute, id ast.ScopeID, function *ast.Function, requireValue bool) ([]string, value, error) {
	target, ok := attribute.Target.(*ast.Identifier)
	if !ok {
		return nil, value{}, Error{Position: attribute.Pos, Message: "list method requires a list variable"}
	}
	listID, found := c.resolve(target.Name, id, function)
	if !found || !c.isList(listID) {
		return nil, value{}, Error{Position: target.Pos, Message: fmt.Sprintf("%q is not a list", target.Name)}
	}
	switch attribute.Name {
	case "append":
		if requireValue {
			return nil, value{}, Error{Position: call.Pos, Message: "append does not return a value"}
		}
		if len(call.Arguments) != 1 {
			return nil, value{}, Error{Position: call.Pos, Message: "append expects 1 argument"}
		}
		if selector, ok := call.Arguments[0].(*ast.EntitySelector); ok {
			commands := []string{fmt.Sprintf("data modify storage %s lists.v%d append value {}", c.storageName(), listID)}
			commands = append(commands, c.compileEntityCapture(selector.Value, fmt.Sprintf("lists.v%d[-1]", listID))...)
			commands = append(commands, fmt.Sprintf("data modify storage %s list_types.v%d append value \"entity\"", c.storageName(), listID))
			return commands, value{}, nil
		}
		if c.isStringExpression(call.Arguments[0], id, function) {
			commands := []string{fmt.Sprintf("data modify storage %s lists.v%d append value \"\"", c.storageName(), listID), fmt.Sprintf("data modify storage %s list_types.v%d append value \"str\"", c.storageName(), listID)}
			written, err := c.compileStringExpressionToPath(call.Arguments[0], fmt.Sprintf("lists.v%d[-1]", listID), id, function)
			return append(commands, written...), value{}, err
		}
		if nested, ok := call.Arguments[0].(*ast.List); ok {
			commands := []string{fmt.Sprintf("data modify storage %s lists.v%d append value []", c.storageName(), listID), fmt.Sprintf("data modify storage %s list_types.v%d append value []", c.storageName(), listID)}
			written, err := c.compileListValueAtPath(nested, fmt.Sprintf("lists.v%d[-1]", listID), id, function)
			return append(commands, written...), value{}, err
		}
		commands, item, err := c.compileExpression(call.Arguments[0], id, function)
		if err != nil {
			return nil, value{}, err
		}
		commands = append(commands, fmt.Sprintf("data modify storage %s lists.v%d append value 0", c.storageName(), listID))
		commands = append(commands, fmt.Sprintf("data modify storage %s list_types.v%d append value \"int\"", c.storageName(), listID))
		commands = append(commands, fmt.Sprintf("execute store result storage %s lists.v%d[-1] int 1 run scoreboard players get %s %s", c.storageName(), listID, item.holder, item.objective))
		return commands, value{}, nil
	case "insert":
		if requireValue {
			return nil, value{}, Error{Position: call.Pos, Message: "insert does not return a value"}
		}
		if len(call.Arguments) != 2 {
			return nil, value{}, Error{Position: call.Pos, Message: "insert expects index and value"}
		}
		if selector, ok := call.Arguments[1].(*ast.EntitySelector); ok {
			literal, constant := integerLiteral(call.Arguments[0])
			if !constant {
				return nil, value{}, Error{Position: call.Pos, Message: "inserting entities currently requires a constant index"}
			}
			commands := []string{fmt.Sprintf("data modify storage %s lists.v%d insert %d value {}", c.storageName(), listID, literal)}
			commands = append(commands, c.compileEntityCapture(selector.Value, fmt.Sprintf("lists.v%d[%d]", listID, literal))...)
			commands = append(commands, fmt.Sprintf("data modify storage %s list_types.v%d insert %d value \"entity\"", c.storageName(), listID, literal))
			return commands, value{}, nil
		}
		if c.isStringExpression(call.Arguments[1], id, function) {
			literal, constant := integerLiteral(call.Arguments[0])
			if !constant {
				return nil, value{}, Error{Position: call.Pos, Message: "inserting strings currently requires a constant index"}
			}
			commands := []string{fmt.Sprintf("data modify storage %s lists.v%d insert %d value \"\"", c.storageName(), listID, literal), fmt.Sprintf("data modify storage %s list_types.v%d insert %d value \"str\"", c.storageName(), listID, literal)}
			written, err := c.compileStringExpressionToPath(call.Arguments[1], fmt.Sprintf("lists.v%d[%d]", listID, literal), id, function)
			return append(commands, written...), value{}, err
		}
		commands, item, err := c.compileExpression(call.Arguments[1], id, function)
		if err != nil {
			return nil, value{}, err
		}
		if literal, ok := integerLiteral(call.Arguments[0]); ok {
			commands = append(commands, fmt.Sprintf("data modify storage %s lists.v%d insert %d value 0", c.storageName(), listID, literal))
			commands = append(commands, fmt.Sprintf("data modify storage %s list_types.v%d insert %d value \"int\"", c.storageName(), listID, literal))
			commands = append(commands, fmt.Sprintf("execute store result storage %s lists.v%d[%d] int 1 run scoreboard players get %s %s", c.storageName(), listID, literal, item.holder, item.objective))
			return commands, value{}, nil
		}
		indexCommands, index, err := c.compileExpression(call.Arguments[0], id, function)
		if err != nil {
			return nil, value{}, err
		}
		commands = append(commands, indexCommands...)
		commands = append(commands, fmt.Sprintf("execute store result storage %s scratch.index int 1 run scoreboard players get %s %s", c.storageName(), index.holder, index.objective))
		helper := c.reserveInternalFunction()
		c.output.Functions[helper] = []string{
			fmt.Sprintf("$data modify storage %s lists.v%d insert $(index) value 0", c.storageName(), listID),
			fmt.Sprintf("$data modify storage %s list_types.v%d insert $(index) value \"int\"", c.storageName(), listID),
			fmt.Sprintf("$execute store result storage %s lists.v%d[$(index)] int 1 run scoreboard players get %s %s", c.storageName(), listID, item.holder, item.objective),
		}
		commands = append(commands, fmt.Sprintf("function %s:%s with storage %s scratch", c.functionNamespace, helper, c.storageName()))
		return commands, value{}, nil
	case "remove":
		if len(call.Arguments) > 1 {
			return nil, value{}, Error{Position: call.Pos, Message: "remove expects 0 or 1 arguments"}
		}
		result := c.newTemporary()
		path := "-1"
		var commands []string
		if len(call.Arguments) == 1 {
			if literal, ok := call.Arguments[0].(*ast.Integer); ok {
				path = strconv.FormatInt(literal.Value, 10)
			} else {
				indexCommands, index, err := c.compileExpression(call.Arguments[0], id, function)
				if err != nil {
					return nil, value{}, err
				}
				commands = append(commands, indexCommands...)
				commands = append(commands, fmt.Sprintf("execute store result storage %s scratch.index int 1 run scoreboard players get %s %s", c.storageName(), index.holder, index.objective))
				helper := c.reserveInternalFunction()
				c.output.Functions[helper] = []string{
					fmt.Sprintf("$execute store result score %s %s run data get storage %s lists.v%d[$(index)] 1", result.holder, result.objective, c.storageName(), listID),
					fmt.Sprintf("$data remove storage %s lists.v%d[$(index)]", c.storageName(), listID),
					fmt.Sprintf("$data remove storage %s list_types.v%d[$(index)]", c.storageName(), listID),
				}
				commands = append(commands, fmt.Sprintf("function %s:%s with storage %s scratch", c.functionNamespace, helper, c.storageName()))
				return commands, result, nil
			}
		}
		commands = append(commands, fmt.Sprintf("execute store result score %s %s run data get storage %s lists.v%d[%s] 1", result.holder, result.objective, c.storageName(), listID, path))
		commands = append(commands, fmt.Sprintf("data remove storage %s lists.v%d[%s]", c.storageName(), listID, path))
		commands = append(commands, fmt.Sprintf("data remove storage %s list_types.v%d[%s]", c.storageName(), listID, path))
		return commands, result, nil
	default:
		return nil, value{}, Error{Position: attribute.Pos, Message: fmt.Sprintf("unknown list method %q", attribute.Name)}
	}
}

func (c *compiler) compileSay(call *ast.Call, id ast.ScopeID, function *ast.Function) ([]string, value, error) {
	if len(call.Arguments) == 0 {
		return nil, value{}, Error{Position: call.Pos, Message: "say expects at least 1 argument"}
	}
	var commands []string
	components := make([]string, 0, len(call.Arguments))
	for _, argument := range call.Arguments {
		if text, ok := argument.(*ast.String); ok {
			encoded, _ := json.Marshal(text.Value)
			components = append(components, fmt.Sprintf("{\"text\":%s}", encoded))
			continue
		}
		if identifier, ok := argument.(*ast.Identifier); ok {
			if variableID, found := c.resolve(identifier.Name, id, function); found {
				if _, entity := c.entityVariables[variableID]; entity {
					components = append(components, fmt.Sprintf("{\"nbt\":\"entities.v%d\",\"storage\":\"%s\"}", variableID, c.storageName()))
					continue
				}
				if _, variant := c.variantVariables[variableID]; variant {
					components = append(components, fmt.Sprintf("{\"nbt\":\"variants.v%d\",\"storage\":\"%s\"}", variableID, c.storageName()))
					continue
				}
				if c.isList(variableID) {
					components = append(components, fmt.Sprintf("{\"nbt\":\"lists.v%d\",\"storage\":\"%s\"}", variableID, c.storageName()))
					continue
				}
				if c.isNBT(variableID) {
					components = append(components, fmt.Sprintf("{\"nbt\":\"nbt.v%d\",\"storage\":\"%s\"}", variableID, c.storageName()))
					continue
				}
				if c.isString(variableID) {
					components = append(components, fmt.Sprintf("{\"nbt\":\"strings.v%d\",\"storage\":\"%s\",\"interpret\":false}", variableID, c.storageName()))
					continue
				}
			}
		}
		if indexed, ok := argument.(*ast.Index); ok {
			if sourcePath, _, found := c.nbtIndexPath(indexed, id, function); found {
				components = append(components, fmt.Sprintf("{\"nbt\":%q,\"storage\":%q}", sourcePath, c.storageName()))
				continue
			}
			root, indices := compilerIndexedRoot(indexed)
			if root != nil {
				if variableID, found := c.resolve(root.Name, id, function); found && c.isList(variableID) {
					path := "scratch.say" + strconv.FormatUint(c.temporary, 10)
					c.temporary++
					indexCommands, sourcePath, dynamic, err := c.compileIndexPath(fmt.Sprintf("lists.v%d", variableID), indices, id, function)
					if err != nil {
						return nil, value{}, err
					}
					commands = append(commands, indexCommands...)
					copyCommand := fmt.Sprintf("data modify storage %s %s set from storage %s %s", c.storageName(), path, c.storageName(), sourcePath)
					if dynamic {
						helper := c.reserveInternalFunction()
						c.output.Functions[helper] = []string{"$" + copyCommand}
						commands = append(commands, fmt.Sprintf("function %s:%s with storage %s scratch", c.functionNamespace, helper, c.storageName()))
					} else {
						commands = append(commands, copyCommand)
					}
					components = append(components, fmt.Sprintf("{\"nbt\":\"%s\",\"storage\":\"%s\"}", path, c.storageName()))
					continue
				}
			}
		}
		if method, ok := argument.(*ast.Call); ok {
			if attribute, ok := method.Callee.(*ast.Attribute); ok && attribute.Name == "remove" {
				target, targetOK := attribute.Target.(*ast.Identifier)
				listID, found := uint32(0), false
				if targetOK {
					listID, found = c.resolve(target.Name, id, function)
				}
				if !found || !c.isList(listID) || len(method.Arguments) > 1 {
					return nil, value{}, Error{Position: method.Pos, Message: "remove expects a list and 0 or 1 indexes"}
				}
				path := "-1"
				dynamic := false
				if len(method.Arguments) == 1 {
					if literal, ok := integerLiteral(method.Arguments[0]); ok {
						path = strconv.FormatInt(literal, 10)
					} else {
						indexCommands, index, err := c.compileExpression(method.Arguments[0], id, function)
						if err != nil {
							return nil, value{}, err
						}
						commands = append(commands, indexCommands...)
						commands = append(commands, fmt.Sprintf("execute store result storage %s scratch.index int 1 run scoreboard players get %s %s", c.storageName(), index.holder, index.objective))
						path = "$(index)"
						dynamic = true
					}
				}
				resultPath := "scratch.say_remove" + strconv.FormatUint(c.temporary, 10)
				c.temporary++
				removeCommands := []string{
					fmt.Sprintf("data modify storage %s %s set from storage %s lists.v%d[%s]", c.storageName(), resultPath, c.storageName(), listID, path),
					fmt.Sprintf("data remove storage %s lists.v%d[%s]", c.storageName(), listID, path),
					fmt.Sprintf("data remove storage %s list_types.v%d[%s]", c.storageName(), listID, path),
				}
				if dynamic {
					helper := c.reserveInternalFunction()
					for i := range removeCommands {
						removeCommands[i] = "$" + removeCommands[i]
					}
					c.output.Functions[helper] = removeCommands
					commands = append(commands, fmt.Sprintf("function %s:%s with storage %s scratch", c.functionNamespace, helper, c.storageName()))
				} else {
					commands = append(commands, removeCommands...)
				}
				components = append(components, fmt.Sprintf("{\"nbt\":\"%s\",\"storage\":\"%s\"}", resultPath, c.storageName()))
				continue
			}
		}
		if list, ok := argument.(*ast.List); ok {
			path := "scratch.say" + strconv.FormatUint(c.temporary, 10)
			c.temporary++
			commands = append(commands, fmt.Sprintf("data modify storage %s %s set value []", c.storageName(), path))
			for _, element := range list.Elements {
				compiled, item, err := c.compileExpression(element, id, function)
				if err != nil {
					return nil, value{}, err
				}
				commands = append(commands, compiled...)
				commands = append(commands, fmt.Sprintf("data modify storage %s %s append value 0", c.storageName(), path))
				commands = append(commands, fmt.Sprintf("execute store result storage %s %s[-1] int 1 run scoreboard players get %s %s", c.storageName(), path, item.holder, item.objective))
			}
			components = append(components, fmt.Sprintf("{\"nbt\":\"%s\",\"storage\":\"%s\"}", path, c.storageName()))
			continue
		}
		if c.isStringExpression(argument, id, function) {
			path := "scratch.say_string" + strconv.FormatUint(c.stringTemporary, 10)
			c.stringTemporary++
			compiled, err := c.compileStringExpressionToPath(argument, path, id, function)
			if err != nil {
				return nil, value{}, err
			}
			commands = append(commands, compiled...)
			components = append(components, fmt.Sprintf("{\"nbt\":\"%s\",\"storage\":\"%s\",\"interpret\":false}", path, c.storageName()))
			continue
		}
		compiled, result, err := c.compileExpression(argument, id, function)
		if err != nil {
			return nil, value{}, err
		}
		commands = append(commands, compiled...)
		components = append(components, fmt.Sprintf("{\"score\":{\"name\":\"%s\",\"objective\":\"%s\"}}", result.holder, result.objective))
	}
	commands = append(commands, "tellraw @a ["+strings.Join(components, ",")+"]")
	return commands, value{}, nil
}

func (c *compiler) resolve(name string, id ast.ScopeID, function *ast.Function) (uint32, bool) {
	if _, global := c.globals[function][name]; global {
		variableID, ok := c.declared[0][name]
		return variableID, ok
	}
	current := c.scopes.ByID[id]
	for current != nil {
		if variableID, ok := c.declared[current.ID][name]; ok {
			return variableID, true
		}
		current = current.Parent
	}
	return 0, false
}

func (c *compiler) declare(id ast.ScopeID, name string) uint32 {
	c.ensureScope(id)
	if existing, ok := c.declared[id][name]; ok {
		return existing
	}
	variableID := c.nextVariable
	c.nextVariable++
	c.declared[id][name] = variableID
	c.output.Variables = append(c.output.Variables, Variable{
		ID: variableID, ScopeID: id, Name: name, Holder: variableHolder(variableID),
	})
	return variableID
}

func (c *compiler) ensureScope(id ast.ScopeID) {
	if c.declared[id] == nil {
		c.declared[id] = make(map[string]uint32)
	}
}

func (c *compiler) variableValue(id uint32) value {
	return value{holder: variableHolder(id), objective: c.objective}
}

func (c *compiler) newTemporary() value {
	holder := "#t" + strconv.FormatUint(c.temporary, 10)
	c.temporary++
	return value{holder: holder, objective: c.objective}
}

func (c *compiler) buildLoad() {
	c.output.Load = append(c.output.Load, "scoreboard objectives add "+c.objective+" dummy")
	if len(c.listVariables) > 0 {
		c.output.Load = append(c.output.Load, "data modify storage "+c.storageName()+" lists set value {}")
	}
	if len(c.stringVariables) > 0 {
		c.output.Load = append(c.output.Load, "data modify storage "+c.storageName()+" strings set value {}")
	}
	constants := make([]int64, 0, len(c.constants))
	for constant := range c.constants {
		constants = append(constants, constant)
	}
	sort.Slice(constants, func(i, j int) bool { return constants[i] < constants[j] })
	for _, constant := range constants {
		c.output.Load = append(c.output.Load, fmt.Sprintf("scoreboard players set %s %s %d", constantHolder(constant), c.objective, constant))
	}
	c.output.Load = append(c.output.Load, c.globalInitializers...)
}

func variableHolder(id uint32) string {
	return "#v" + strconv.FormatUint(uint64(id), 10)
}

func constantHolder(number int64) string {
	if number < 0 {
		return "#cn" + strconv.FormatInt(-number, 10)
	}
	return "#c" + strconv.FormatInt(number, 10)
}

func returnHolder(id uint32) string {
	return "#r" + strconv.FormatUint(uint64(id), 10)
}

func returnSignalHolder(id uint32) string {
	return "#rf" + strconv.FormatUint(uint64(id), 10)
}

// statementsHaveNestedReturn reports whether a return is inside control flow.
// Those returns execute in generated helper functions and need to be relayed.
func statementsHaveNestedReturn(statements []ast.Statement) bool {
	for _, statement := range statements {
		switch statement := statement.(type) {
		case *ast.If:
			if statementsReturn(statement.Body) || statementsReturn(statement.Else) {
				return true
			}
			for _, branch := range statement.Elifs {
				if statementsReturn(branch.Body) {
					return true
				}
			}
		case *ast.For:
			if statementsReturn(statement.Body) {
				return true
			}
		case *ast.While:
			if statementsReturn(statement.Body) {
				return true
			}
		}
	}
	return false
}

func statementsReturn(statements []ast.Statement) bool {
	for _, statement := range statements {
		switch statement := statement.(type) {
		case *ast.Return:
			return true
		case *ast.If:
			if statementsReturn(statement.Body) || statementsReturn(statement.Else) {
				return true
			}
			for _, branch := range statement.Elifs {
				if statementsReturn(branch.Body) {
					return true
				}
			}
		case *ast.For:
			if statementsReturn(statement.Body) {
				return true
			}
		case *ast.While:
			if statementsReturn(statement.Body) {
				return true
			}
		}
	}
	return false
}

func functionReturnsValue(statements []ast.Statement) bool {
	for _, statement := range statements {
		switch statement := statement.(type) {
		case *ast.Return:
			if statement.Value != nil {
				return true
			}
		case *ast.If:
			if functionReturnsValue(statement.Body) || functionReturnsValue(statement.Else) {
				return true
			}
			for _, branch := range statement.Elifs {
				if functionReturnsValue(branch.Body) {
					return true
				}
			}
		case *ast.For:
			if functionReturnsValue(statement.Body) {
				return true
			}
		case *ast.While:
			if functionReturnsValue(statement.Body) {
				return true
			}
		}
	}
	return false
}

func scoreboardOperation(target value, operation string, source value) string {
	return fmt.Sprintf("scoreboard players operation %s %s %s %s %s", target.holder, target.objective, operation, source.holder, source.objective)
}
