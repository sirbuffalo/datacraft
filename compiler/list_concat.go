package compiler

import (
	"fmt"
	"strconv"

	"github.com/sirbuffalo/datacraft/ast"
	"github.com/sirbuffalo/datacraft/token"
)

func (c *compiler) compileListConcatenationAssignment(statement *ast.Assignment, destination uint32, id ast.ScopeID, function *ast.Function) ([]string, error) {
	if statement.Operator != token.Assign && statement.Operator != token.PlusAssign {
		return nil, Error{Position: statement.Pos, Message: "list concatenation supports '=' and '+='"}
	}
	temporaryID := c.temporary
	c.temporary++
	values := "scratch.list_concat_" + strconv.FormatUint(temporaryID, 10)
	types := "scratch.list_concat_types_" + strconv.FormatUint(temporaryID, 10)
	commands := []string{fmt.Sprintf("data modify storage %s %s set value []", c.storageName(), values), fmt.Sprintf("data modify storage %s %s set value []", c.storageName(), types)}
	if statement.Operator == token.PlusAssign {
		commands = append(commands, c.appendListPaths(values, types, fmt.Sprintf("lists.v%d", destination), fmt.Sprintf("list_types.v%d", destination))...)
	}
	appended, err := c.appendListExpression(statement.Value, values, types, id, function, destination)
	if err != nil {
		return nil, err
	}
	commands = append(commands, appended...)
	commands = append(commands,
		fmt.Sprintf("data modify storage %s lists.v%d set from storage %s %s", c.storageName(), destination, c.storageName(), values),
		fmt.Sprintf("data modify storage %s list_types.v%d set from storage %s %s", c.storageName(), destination, c.storageName(), types),
		fmt.Sprintf("data remove storage %s %s", c.storageName(), values), fmt.Sprintf("data remove storage %s %s", c.storageName(), types))
	return commands, nil
}

func (c *compiler) appendListExpression(expression ast.Expression, values, types string, id ast.ScopeID, function *ast.Function, destination uint32) ([]string, error) {
	switch expression := expression.(type) {
	case *ast.Binary:
		if expression.Operator != token.Plus {
			return nil, Error{Position: expression.Pos, Message: "lists only support '+' concatenation"}
		}
		left, err := c.appendListExpression(expression.Left, values, types, id, function, destination)
		if err != nil {
			return nil, err
		}
		right, err := c.appendListExpression(expression.Right, values, types, id, function, destination)
		return append(left, right...), err
	case *ast.Identifier:
		source, found := c.resolve(expression.Name, id, function)
		if !found || !c.isList(source) {
			return nil, Error{Position: expression.Pos, Message: fmt.Sprintf("%q is not a list", expression.Name)}
		}
		if _, entities := c.entityLists[source]; entities {
			c.entityLists[destination] = struct{}{}
		}
		return c.appendListPaths(values, types, fmt.Sprintf("lists.v%d", source), fmt.Sprintf("list_types.v%d", source)), nil
	case *ast.List:
		if listContainsEntity(expression) {
			c.entityLists[destination] = struct{}{}
		}
		leafID := c.temporary
		c.temporary++
		leafValues := "lists.scratch_leaf_" + strconv.FormatUint(leafID, 10)
		leafTypes := "list_types.scratch_leaf_" + strconv.FormatUint(leafID, 10)
		commands, err := c.compileListValueAtPath(expression, leafValues, id, function)
		if err != nil {
			return nil, err
		}
		commands = append(commands, c.appendListPaths(values, types, leafValues, leafTypes)...)
		commands = append(commands, fmt.Sprintf("data remove storage %s %s", c.storageName(), leafValues), fmt.Sprintf("data remove storage %s %s", c.storageName(), leafTypes))
		return commands, nil
	case *ast.Call:
		name, named := callCallee(expression)
		calleeID, found := c.functionIndexes[name]
		if !named || !found || !c.output.FunctionMappings[calleeID].ReturnsList {
			return nil, Error{Position: expression.Pos, Message: "call does not return a list"}
		}
		commands, _, err := c.compileCall(expression, id, function, false)
		if err != nil {
			return nil, err
		}
		return append(commands, c.appendListPaths(values, types, fmt.Sprintf("returns.r%d", calleeID), fmt.Sprintf("return_types.r%d", calleeID))...), nil
	default:
		return nil, Error{Position: expression.Position(), Message: "expected a list expression"}
	}
}

func (c *compiler) appendListPaths(destinationValues, destinationTypes, sourceValues, sourceTypes string) []string {
	return []string{fmt.Sprintf("data modify storage %s %s append from storage %s %s[]", c.storageName(), destinationValues, c.storageName(), sourceValues), fmt.Sprintf("data modify storage %s %s append from storage %s %s[]", c.storageName(), destinationTypes, c.storageName(), sourceTypes)}
}
