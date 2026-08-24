package compiler

import (
	"encoding/json"
	"fmt"

	"github.com/sirbuffalo/datacraft/ast"
	"github.com/sirbuffalo/datacraft/token"
)

func (c *compiler) compilePrimitiveSetAssignment(expression ast.Expression, destination uint32, id ast.ScopeID, function *ast.Function) ([]string, error) {
	base := fmt.Sprintf("sets.v%d", destination)
	switch expression := expression.(type) {
	case *ast.Set:
		commands := []string{fmt.Sprintf("data modify storage %s %s set value {values:{},items:[],length:0,next:1}", c.storageName(), base)}
		for _, element := range expression.Elements {
			added, err := c.compilePrimitiveSetAdd(destination, element, id, function)
			if err != nil {
				return nil, err
			}
			commands = append(commands, added...)
		}
		return commands, nil
	case *ast.Identifier:
		source, found := c.resolve(expression.Name, id, function)
		if !found || !c.isPrimitiveSet(source) {
			return nil, Error{Position: expression.Pos, Message: fmt.Sprintf("%q is not a primitive set", expression.Name)}
		}
		if c.primitiveSets[source] != c.primitiveSets[destination] {
			return nil, Error{Position: expression.Pos, Message: "set element types do not match"}
		}
		if source == destination {
			return nil, nil
		}
		return []string{fmt.Sprintf("data modify storage %s %s set from storage %s sets.v%d", c.storageName(), base, c.storageName(), source)}, nil
	case *ast.Call:
		callee, named := callCallee(expression)
		mapping, _, targetObjective, found := c.callTarget(callee)
		if !named || !found || !mapping.ReturnsPrimitiveSet {
			return nil, Error{Position: expression.Pos, Message: "call does not return a primitive set"}
		}
		commands, _, err := c.compileCall(expression, id, function, false)
		if err != nil {
			return nil, err
		}
		return append(commands, fmt.Sprintf("data modify storage %s %s set from storage %s:data set_returns.r%d", c.storageName(), base, targetObjective, mapping.ID)), nil
	case *ast.Binary:
		if expression.Operator != token.Pipe && expression.Operator != token.Ampersand {
			return nil, Error{Position: expression.Pos, Message: "primitive sets only support | and &"}
		}
		left, leftOK := expression.Left.(*ast.Identifier)
		right, rightOK := expression.Right.(*ast.Identifier)
		if !leftOK || !rightOK {
			return nil, Error{Position: expression.Pos, Message: "set union/intersection currently requires set variables"}
		}
		leftID, leftFound := c.resolve(left.Name, id, function)
		rightID, rightFound := c.resolve(right.Name, id, function)
		if !leftFound || !rightFound || !c.isPrimitiveSet(leftID) || !c.isPrimitiveSet(rightID) {
			return nil, Error{Position: expression.Pos, Message: "set operands must be primitive sets"}
		}
		return c.compilePrimitiveSetAlgebra(destination, leftID, rightID, expression.Operator)
	default:
		return nil, Error{Position: expression.Position(), Message: "primitive set expression lowering is not implemented for this expression"}
	}
}

func (c *compiler) compilePrimitiveSetAlgebra(destination, left, right uint32, operator token.Kind) ([]string, error) {
	storage := c.storageName()
	temp := "scratch.set_result"
	commands := []string{fmt.Sprintf("data modify storage %s %s set from storage %s sets.v%d", storage, temp, storage, left)}
	if operator == token.Pipe {
		commands = append(commands,
			fmt.Sprintf("data modify storage %s %s.values merge from storage %s sets.v%d.values", storage, temp, storage, right),
			fmt.Sprintf("data modify storage %s %s.items append from storage %s sets.v%d.items[]", storage, temp, storage, right),
		)
	} else {
		index, length := c.newTemporary(), c.newTemporary()
		loader, filter, controller := c.reserveInternalFunction(), c.reserveInternalFunction(), c.reserveInternalFunction()
		c.output.Functions[loader] = []string{fmt.Sprintf("$data modify storage %s scratch.set_item set from storage %s %s.items[$(set_index)]", storage, storage, temp)}
		c.output.Functions[filter] = []string{
			fmt.Sprintf("$execute unless data storage %s sets.v%d.values.\"$(key)\" run data remove storage %s %s.values.\"$(key)\"", storage, right, storage, temp),
		}
		c.output.Functions[controller] = []string{
			fmt.Sprintf("execute store result storage %s scratch.set_index int 1 run scoreboard players get %s %s", storage, index.holder, index.objective),
			fmt.Sprintf("function %s:%s with storage %s scratch", c.functionNamespace, loader, storage),
			fmt.Sprintf("function %s:%s with storage %s scratch.set_item", c.functionNamespace, filter, storage),
			fmt.Sprintf("scoreboard players add %s %s 1", index.holder, index.objective),
			fmt.Sprintf("execute if score %s %s < %s %s run function %s:%s", index.holder, index.objective, length.holder, length.objective, c.functionNamespace, controller),
		}
		commands = append(commands,
			fmt.Sprintf("scoreboard players set %s %s 0", index.holder, index.objective),
			fmt.Sprintf("execute store result score %s %s run data get storage %s %s.items", length.holder, length.objective, storage, temp),
			fmt.Sprintf("execute if score %s %s < %s %s run function %s:%s", index.holder, index.objective, length.holder, length.objective, c.functionNamespace, controller),
		)
	}
	next := c.newTemporary()
	commands = append(commands,
		fmt.Sprintf("execute store result storage %s %s.length int 1 run data get storage %s %s.values", storage, temp, storage, temp),
		fmt.Sprintf("execute store result score %s %s run data get storage %s %s.items", next.holder, next.objective, storage, temp),
		fmt.Sprintf("scoreboard players add %s %s 1", next.holder, next.objective),
		fmt.Sprintf("execute store result storage %s %s.next int 1 run scoreboard players get %s %s", storage, temp, next.holder, next.objective),
		fmt.Sprintf("data modify storage %s sets.v%d set from storage %s %s", storage, destination, storage, temp),
		fmt.Sprintf("data remove storage %s %s", storage, temp),
	)
	return commands, nil
}

func (c *compiler) compilePrimitiveSetMethod(call *ast.Call, attribute *ast.Attribute, setID uint32, id ast.ScopeID, function *ast.Function, requireValue bool) ([]string, value, error) {
	if requireValue {
		return nil, value{}, Error{Position: call.Pos, Message: attribute.Name + " does not return a value"}
	}
	if attribute.Name == "clear" {
		if len(call.Arguments) != 0 {
			return nil, value{}, Error{Position: call.Pos, Message: "clear expects no arguments"}
		}
		return []string{fmt.Sprintf("data modify storage %s sets.v%d set value {values:{},items:[],length:0,next:1}", c.storageName(), setID)}, value{}, nil
	}
	if len(call.Arguments) != 1 {
		return nil, value{}, Error{Position: call.Pos, Message: attribute.Name + " expects one value"}
	}
	switch attribute.Name {
	case "add":
		commands, err := c.compilePrimitiveSetAdd(setID, call.Arguments[0], id, function)
		return commands, value{}, err
	case "discard", "remove":
		commands, err := c.compilePrimitiveSetRemove(setID, call.Arguments[0], id, function, attribute.Name == "remove")
		return commands, value{}, err
	default:
		return nil, value{}, Error{Position: attribute.Pos, Message: fmt.Sprintf("unknown set method %q", attribute.Name)}
	}
}

func (c *compiler) compilePrimitiveSetAdd(setID uint32, expression ast.Expression, id ast.ScopeID, function *ast.Function) ([]string, error) {
	commands, key, valuePath, dynamic, err := c.compilePrimitiveSetKey(expression, id, function)
	if err != nil {
		return nil, err
	}
	base := fmt.Sprintf("sets.v%d", setID)
	present := c.newTemporary()
	generation := c.newTemporary()
	operations := []string{
		fmt.Sprintf("scoreboard players set %s %s 0", present.holder, present.objective),
		fmt.Sprintf("execute if data storage %s %s.values.%s run scoreboard players set %s %s 1", c.storageName(), base, key, present.holder, present.objective),
		fmt.Sprintf("execute if score %s %s matches 0 store result score %s %s run data get storage %s %s.next 1", present.holder, present.objective, generation.holder, generation.objective, c.storageName(), base),
		fmt.Sprintf("execute if score %s %s matches 0 run data modify storage %s %s.items append value {key:\"\",value:0,generation:0}", present.holder, present.objective, c.storageName(), base),
		fmt.Sprintf("execute if score %s %s matches 0 run data modify storage %s %s.items[-1].key set from storage %s scratch.set_key", present.holder, present.objective, c.storageName(), base, c.storageName()),
		fmt.Sprintf("execute if score %s %s matches 0 run data modify storage %s %s.items[-1].value set from storage %s %s", present.holder, present.objective, c.storageName(), base, c.storageName(), valuePath),
		fmt.Sprintf("execute if score %s %s matches 0 store result storage %s %s.items[-1].generation int 1 run scoreboard players get %s %s", present.holder, present.objective, c.storageName(), base, generation.holder, generation.objective),
		fmt.Sprintf("execute if score %s %s matches 0 store result storage %s %s.values.%s int 1 run scoreboard players get %s %s", present.holder, present.objective, c.storageName(), base, key, generation.holder, generation.objective),
		fmt.Sprintf("execute if score %s %s matches 0 run scoreboard players add %s %s 1", present.holder, present.objective, generation.holder, generation.objective),
		fmt.Sprintf("execute if score %s %s matches 0 store result storage %s %s.next int 1 run scoreboard players get %s %s", present.holder, present.objective, c.storageName(), base, generation.holder, generation.objective),
		fmt.Sprintf("execute store result storage %s %s.length int 1 run data get storage %s %s.values", c.storageName(), base, c.storageName(), base),
	}
	return c.wrapSetMacros(commands, operations, dynamic), nil
}

func (c *compiler) compilePrimitiveSetRemove(setID uint32, expression ast.Expression, id ast.ScopeID, function *ast.Function, strict bool) ([]string, error) {
	commands, key, _, dynamic, err := c.compilePrimitiveSetKey(expression, id, function)
	if err != nil {
		return nil, err
	}
	base := fmt.Sprintf("sets.v%d", setID)
	ops := []string{}
	if strict {
		ops = append(ops, fmt.Sprintf("execute unless data storage %s %s.values.%s run return fail", c.storageName(), base, key))
	}
	ops = append(ops, fmt.Sprintf("data remove storage %s %s.values.%s", c.storageName(), base, key), fmt.Sprintf("execute store result storage %s %s.length int 1 run data get storage %s %s.values", c.storageName(), base, c.storageName(), base))
	return c.wrapSetMacros(commands, ops, dynamic), nil
}

func (c *compiler) compilePrimitiveSetMembership(expression ast.Expression, setID uint32, id ast.ScopeID, function *ast.Function) ([]string, value, error) {
	commands, key, _, dynamic, err := c.compilePrimitiveSetKey(expression, id, function)
	if err != nil {
		return nil, value{}, err
	}
	result := c.newTemporary()
	operation := []string{
		fmt.Sprintf("scoreboard players set %s %s 0", result.holder, result.objective),
		fmt.Sprintf("execute if data storage %s sets.v%d.values.%s run scoreboard players set %s %s 1", c.storageName(), setID, key, result.holder, result.objective),
	}
	return c.wrapSetMacros(commands, operation, dynamic), result, nil
}

func (c *compiler) compilePrimitiveSetKey(expression ast.Expression, id ast.ScopeID, function *ast.Function) ([]string, string, string, bool, error) {
	path := "scratch.set_value"
	switch value := expression.(type) {
	case *ast.Integer:
		return []string{fmt.Sprintf("data modify storage %s %s set value %d", c.storageName(), path, value.Value), fmt.Sprintf("data modify storage %s scratch.set_key set value \"%d\"", c.storageName(), value.Value)}, fmt.Sprintf("\"%d\"", value.Value), path, false, nil
	case *ast.Boolean:
		n := 0
		if value.Value {
			n = 1
		}
		return []string{fmt.Sprintf("data modify storage %s %s set value %d", c.storageName(), path, n), fmt.Sprintf("data modify storage %s scratch.set_key set value \"%d\"", c.storageName(), n)}, fmt.Sprintf("\"%d\"", n), path, false, nil
	case *ast.String:
		encoded, _ := json.Marshal(value.Value)
		return []string{fmt.Sprintf("data modify storage %s %s set value %s", c.storageName(), path, encoded), fmt.Sprintf("data modify storage %s scratch.set_key set value %s", c.storageName(), encoded)}, string(encoded), path, false, nil
	}
	if c.isStringExpression(expression, id, function) {
		commands, err := c.compileStringExpressionToPath(expression, path, id, function)
		if err != nil {
			return nil, "", "", false, err
		}
		commands = append(commands, fmt.Sprintf("data modify storage %s scratch.set_key set from storage %s %s", c.storageName(), c.storageName(), path))
		return commands, "\"$(set_key)\"", path, true, nil
	}
	compiled, result, err := c.compileExpression(expression, id, function)
	if err != nil {
		return nil, "", "", false, err
	}
	compiled = append(compiled, fmt.Sprintf("execute store result storage %s %s int 1 run scoreboard players get %s %s", c.storageName(), path, result.holder, result.objective), fmt.Sprintf("execute store result storage %s scratch.set_key int 1 run scoreboard players get %s %s", c.storageName(), result.holder, result.objective))
	return compiled, "\"$(set_key)\"", path, true, nil
}

func (c *compiler) wrapSetMacros(commands, operations []string, dynamic bool) []string {
	if !dynamic {
		return append(commands, operations...)
	}
	helper := c.reserveInternalFunction()
	for i := range operations {
		operations[i] = "$" + operations[i]
	}
	c.output.Functions[helper] = operations
	return append(commands, fmt.Sprintf("function %s:%s with storage %s scratch", c.functionNamespace, helper, c.storageName()))
}

func (c *compiler) compilePrimitiveSetFor(statement *ast.For, setID uint32, function *ast.Function, returnSignal *value) ([]string, error) {
	loopID, ok := c.resolve(statement.Variable, statement.ScopeID, function)
	if !ok {
		return nil, Error{Position: statement.Pos, Message: "undefined set loop variable"}
	}
	index, length := c.newTemporary(), c.newTemporary()
	currentGeneration, itemGeneration := c.newTemporary(), c.newTemporary()
	breakSignal, continueSignal := c.newTemporary(), c.newTemporary()
	bodyName, itemName, loaderName, controllerName := c.reserveInternalFunction(), c.reserveInternalFunction(), c.reserveInternalFunction(), c.reserveInternalFunction()
	body, err := c.compileStatements(statement.Body, statement.ScopeID, function, returnSignal, &breakSignal, &continueSignal)
	if err != nil {
		return nil, err
	}
	c.output.Functions[bodyName] = body
	loadValue := fmt.Sprintf("execute store result score %s %s run data get storage %s scratch.set_item.value 1", variableHolder(loopID), c.objective, c.storageName())
	if c.primitiveSets[setID] == "str" {
		c.stringVariables[loopID] = struct{}{}
		loadValue = fmt.Sprintf("data modify storage %s strings.v%d set from storage %s scratch.set_item.value", c.storageName(), loopID, c.storageName())
	}
	c.output.Functions[itemName] = []string{
		fmt.Sprintf("scoreboard players set %s %s -1", currentGeneration.holder, currentGeneration.objective),
		fmt.Sprintf("$execute store result score %s %s run data get storage %s sets.v%d.values.\"$(key)\" 1", currentGeneration.holder, currentGeneration.objective, c.storageName(), setID),
		fmt.Sprintf("execute store result score %s %s run data get storage %s scratch.set_item.generation 1", itemGeneration.holder, itemGeneration.objective, c.storageName()),
		fmt.Sprintf("execute if score %s %s = %s %s run %s", currentGeneration.holder, currentGeneration.objective, itemGeneration.holder, itemGeneration.objective, loadValue),
		fmt.Sprintf("execute if score %s %s = %s %s run function %s:%s", currentGeneration.holder, currentGeneration.objective, itemGeneration.holder, itemGeneration.objective, c.functionNamespace, bodyName),
	}
	c.output.Functions[loaderName] = []string{fmt.Sprintf("$data modify storage %s scratch.set_item set from storage %s sets.v%d.items[$(set_index)]", c.storageName(), c.storageName(), setID)}
	controller := []string{
		fmt.Sprintf("execute store result storage %s scratch.set_index int 1 run scoreboard players get %s %s", c.storageName(), index.holder, index.objective),
		fmt.Sprintf("function %s:%s with storage %s scratch", c.functionNamespace, loaderName, c.storageName()),
		fmt.Sprintf("function %s:%s with storage %s scratch.set_item", c.functionNamespace, itemName, c.storageName()),
	}
	controller = appendReturnPropagation(controller, returnSignal, c.functionReturnValue(function))
	controller = appendBreakPropagation(controller, &breakSignal)
	controller = append(controller, fmt.Sprintf("scoreboard players set %s %s 0", continueSignal.holder, continueSignal.objective), fmt.Sprintf("scoreboard players add %s %s 1", index.holder, index.objective), fmt.Sprintf("execute if score %s %s < %s %s run function %s:%s", index.holder, index.objective, length.holder, length.objective, c.functionNamespace, controllerName))
	c.output.Functions[controllerName] = controller
	commands := []string{
		fmt.Sprintf("scoreboard players set %s %s 0", index.holder, index.objective), fmt.Sprintf("scoreboard players set %s %s 0", breakSignal.holder, breakSignal.objective), fmt.Sprintf("scoreboard players set %s %s 0", continueSignal.holder, continueSignal.objective),
		fmt.Sprintf("execute store result score %s %s run data get storage %s sets.v%d.items", length.holder, length.objective, c.storageName(), setID),
		fmt.Sprintf("execute if score %s %s < %s %s run function %s:%s", index.holder, index.objective, length.holder, length.objective, c.functionNamespace, controllerName),
	}
	return appendReturnPropagation(commands, returnSignal, c.functionReturnValue(function)), nil
}
