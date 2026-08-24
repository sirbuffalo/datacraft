package datapack

import (
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/sirbuffalo/datacraft/ast"
	"github.com/sirbuffalo/datacraft/compiler"
	"github.com/sirbuffalo/datacraft/parser"
)

// BuildProject compiles a set of source files. Map keys are diagnostic file
// names; imports resolve against each file's namespace declaration.
func BuildProject(sources map[string]string, config Config) (Pack, error) {
	if len(sources) == 0 {
		return Pack{}, fmt.Errorf("project requires at least one source file")
	}
	programs := map[string]*ast.Program{}
	filenames := map[string]string{}
	keys := make([]string, 0, len(sources))
	for filename := range sources {
		keys = append(keys, filename)
	}
	sort.Strings(keys)
	for _, filename := range keys {
		program, err := parser.Parse(sources[filename])
		if err != nil {
			return Pack{}, fmt.Errorf("%s: %w", filename, err)
		}
		if program.Namespace == "" {
			return Pack{}, fmt.Errorf("%s: imported project files require a namespace", filename)
		}
		if previous, exists := filenames[program.Namespace]; exists {
			return Pack{}, fmt.Errorf("namespace %q is declared by both %s and %s", program.Namespace, previous, filename)
		}
		programs[program.Namespace] = program
		filenames[program.Namespace] = filename
	}

	order, err := projectOrder(programs)
	if err != nil {
		return Pack{}, err
	}
	outputs := map[string]compiler.Output{}
	for _, namespace := range order {
		program := programs[namespace]
		imports := map[string]compiler.ImportedFunction{}
		for _, declaration := range program.Imports {
			dependency, exists := programs[declaration.Namespace]
			if !exists {
				return Pack{}, fmt.Errorf("%s:%d:%d: unknown namespace %q", filenames[namespace], declaration.Pos.Line, declaration.Pos.Column, declaration.Namespace)
			}
			dependencyOutput := outputs[declaration.Namespace]
			for _, name := range declaration.Names {
				definition := findFunction(dependency, name)
				mapping, found := findMapping(dependencyOutput, name)
				if definition == nil || !found {
					return Pack{}, fmt.Errorf("%s:%d:%d: namespace %q has no function %q", filenames[namespace], declaration.Pos.Line, declaration.Pos.Column, declaration.Namespace, name)
				}
				if _, duplicate := imports[name]; duplicate {
					return Pack{}, fmt.Errorf("%s:%d:%d: imported name %q is duplicated", filenames[namespace], declaration.Pos.Line, declaration.Pos.Column, name)
				}
				imports[name] = compiler.ImportedFunction{Namespace: declaration.Namespace, Objective: dependencyOutput.ScoreboardName, Definition: definition, Mapping: mapping}
			}
		}
		output, compileErr := compiler.CompileWithImports(program, namespace, imports)
		if compileErr != nil {
			return Pack{}, fmt.Errorf("%s: %w", filenames[namespace], compileErr)
		}
		outputs[namespace] = output
	}

	if config.PackFormat <= 0 {
		return Pack{}, fmt.Errorf("pack format must be positive")
	}
	if config.Description == "" {
		config.Description = "Compiled with DataCraft"
	}
	files := map[string]string{}
	metadata, _ := json.MarshalIndent(map[string]any{"pack": map[string]any{"pack_format": config.PackFormat, "description": config.Description}}, "", "  ")
	files["pack.mcmeta"] = string(metadata) + "\n"
	var loads, ticks []string
	for _, namespace := range order {
		load, tick, emitErr := emitModule(files, namespace, outputs[namespace])
		if emitErr != nil {
			return Pack{}, emitErr
		}
		loads = append(loads, load)
		if tick != "" {
			ticks = append(ticks, tick)
		}
	}
	files[path.Join("data", "minecraft", "tags", "function", "load.json")] = functionTag(loads)
	if len(ticks) > 0 {
		files[path.Join("data", "minecraft", "tags", "function", "tick.json")] = functionTag(ticks)
	}
	return Pack{Files: files}, nil
}

func projectOrder(programs map[string]*ast.Program) ([]string, error) {
	state := map[string]uint8{}
	var order, stack []string
	var visit func(string) error
	visit = func(namespace string) error {
		switch state[namespace] {
		case 1:
			return fmt.Errorf("cyclic import: %s -> %s", strings.Join(stack, " -> "), namespace)
		case 2:
			return nil
		}
		program, exists := programs[namespace]
		if !exists {
			return fmt.Errorf("unknown namespace %q", namespace)
		}
		state[namespace] = 1
		stack = append(stack, namespace)
		for _, declaration := range program.Imports {
			if err := visit(declaration.Namespace); err != nil {
				return err
			}
		}
		stack = stack[:len(stack)-1]
		state[namespace] = 2
		order = append(order, namespace)
		return nil
	}
	names := make([]string, 0, len(programs))
	for name := range programs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := visit(name); err != nil {
			return nil, err
		}
	}
	return order, nil
}

func findFunction(program *ast.Program, name string) *ast.Function {
	for _, function := range program.Functions {
		if function.Name == name {
			return function
		}
	}
	return nil
}

func findMapping(output compiler.Output, name string) (compiler.Function, bool) {
	for _, mapping := range output.FunctionMappings {
		if mapping.Name == name {
			return mapping, true
		}
	}
	return compiler.Function{}, false
}

func emitModule(files map[string]string, namespace string, output compiler.Output) (string, string, error) {
	loadCommands := append([]string{}, output.Load...)
	if userLoad, ok := output.FunctionNames["load"]; ok {
		mapping, _ := findMapping(output, "load")
		if len(mapping.Parameters) > 0 {
			return "", "", fmt.Errorf("namespace %s: load function cannot have parameters", namespace)
		}
		loadCommands = append(loadCommands, "function "+namespace+":"+userLoad)
	}
	files[path.Join("data", namespace, "function", "load.mcfunction")] = functionText(loadCommands)
	for name, commands := range output.Functions {
		files[path.Join("data", namespace, "function", name+".mcfunction")] = functionText(commands)
	}
	for _, function := range output.FunctionMappings {
		if !function.Exposed || function.Name == "load" {
			continue
		}
		wrapper := make([]string, 0, len(function.Parameters)+2)
		for _, parameter := range function.Parameters {
			switch {
			case parameter.IsList:
				wrapper = append(wrapper, fmt.Sprintf("$data modify storage %s:data lists.v%d set value $(%s)", output.ScoreboardName, parameter.VariableID, parameter.Name))
			case parameter.IsString:
				wrapper = append(wrapper, fmt.Sprintf("$data modify storage %s:data strings.v%d set value $(%s)", output.ScoreboardName, parameter.VariableID, parameter.Name))
			default:
				wrapper = append(wrapper, "$scoreboard players set "+parameter.Holder+" "+output.ScoreboardName+" $("+parameter.Name+")")
			}
		}
		if function.ReturnsValue {
			wrapper = append(wrapper, "return run function "+namespace+":"+function.GeneratedName)
		} else {
			wrapper = append(wrapper, "function "+namespace+":"+function.GeneratedName)
		}
		files[path.Join("data", namespace, "function", function.Name+".mcfunction")] = functionText(wrapper)
	}
	tickCommands := append([]string{}, output.Tick...)
	if tick, ok := output.FunctionNames["tick"]; ok {
		tickCommands = append(tickCommands, "function "+namespace+":"+tick)
	}
	tickName := ""
	if len(tickCommands) > 0 {
		files[path.Join("data", namespace, "function", "_tick.mcfunction")] = functionText(tickCommands)
		tickName = namespace + ":_tick"
	}
	return namespace + ":load", tickName, nil
}

func functionTag(values []string) string {
	encoded, _ := json.MarshalIndent(map[string]any{"values": values}, "", "  ")
	return string(encoded) + "\n"
}
