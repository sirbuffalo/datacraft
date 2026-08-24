// Package datapack turns source code into an in-memory Minecraft data pack.
package datapack

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/sirbuffalo/datacraft/compiler"
	"github.com/sirbuffalo/datacraft/parser"
)

type Config struct {
	PackName    string
	Description string
	PackFormat  int
}

type Pack struct {
	Files map[string]string
}

// Build parses and compiles source into paths and file contents. It performs no
// filesystem access, allowing callers to use it in native Go and WebAssembly.
func Build(source string, config Config) (Pack, error) {
	program, err := parser.Parse(source)
	if err != nil {
		return Pack{}, err
	}
	if config.PackName == "" {
		config.PackName = program.Namespace
	}
	if config.PackName == "" {
		return Pack{}, fmt.Errorf("pack name is required")
	}
	if config.PackFormat <= 0 {
		return Pack{}, fmt.Errorf("pack format must be positive")
	}
	if config.Description == "" {
		config.Description = "Compiled with DataCraft"
	}

	output, err := compiler.Compile(program, config.PackName)
	if err != nil {
		return Pack{}, err
	}
	files := make(map[string]string)
	metadata, err := json.MarshalIndent(map[string]any{
		"pack": map[string]any{
			"pack_format": config.PackFormat,
			"description": config.Description,
		},
	}, "", "  ")
	if err != nil {
		return Pack{}, err
	}
	files["pack.mcmeta"] = string(metadata) + "\n"

	loadCommands := append([]string{}, output.Load...)
	if userLoad, ok := output.FunctionNames["load"]; ok {
		mapping := output.FunctionMappings[functionIndex(output.FunctionMappings, "load")]
		if len(mapping.Parameters) > 0 {
			return Pack{}, fmt.Errorf("load function cannot have parameters")
		}
		loadCommands = append(loadCommands, "function "+config.PackName+":"+userLoad)
	}
	files[path.Join("data", config.PackName, "function", "load.mcfunction")] = functionText(loadCommands)
	files[path.Join("data", "minecraft", "tags", "function", "load.json")] = tag(config.PackName + ":load")

	for name, commands := range output.Functions {
		files[path.Join("data", config.PackName, "function", name+".mcfunction")] = functionText(commands)
	}
	for _, function := range output.FunctionMappings {
		if !function.Exposed || function.Name == "load" {
			continue
		}
		if strings.HasPrefix(function.Name, "_") {
			return Pack{}, fmt.Errorf("exposed function %q uses the reserved internal prefix '_'", function.Name)
		}
		wrapper := make([]string, 0, len(function.Parameters)+2)
		for _, parameter := range function.Parameters {
			if parameter.IsList {
				wrapper = append(wrapper, fmt.Sprintf("$data modify storage %s:data lists.v%d set value $(%s)", output.ScoreboardName, parameter.VariableID, parameter.Name))
			} else if parameter.IsString {
				wrapper = append(wrapper, fmt.Sprintf("$data modify storage %s:data strings.v%d set value $(%s)", output.ScoreboardName, parameter.VariableID, parameter.Name))
			} else {
				wrapper = append(wrapper, "$scoreboard players set "+parameter.Holder+" "+output.ScoreboardName+" $("+parameter.Name+")")
			}
		}
		if function.ReturnsValue {
			wrapper = append(wrapper, "return run function "+config.PackName+":"+function.GeneratedName)
		} else {
			wrapper = append(wrapper, "function "+config.PackName+":"+function.GeneratedName)
		}
		files[path.Join("data", config.PackName, "function", function.Name+".mcfunction")] = functionText(wrapper)
	}
	tickCommands := append([]string{}, output.Tick...)
	if tick, ok := output.FunctionNames["tick"]; ok {
		tickCommands = append(tickCommands, "function "+config.PackName+":"+tick)
	}
	if len(tickCommands) > 0 {
		files[path.Join("data", config.PackName, "function", "_tick.mcfunction")] = functionText(tickCommands)
		files[path.Join("data", "minecraft", "tags", "function", "tick.json")] = tag(config.PackName + ":_tick")
	}
	return Pack{Files: files}, nil
}

func functionIndex(functions []compiler.Function, name string) int {
	for i := range functions {
		if functions[i].Name == name {
			return i
		}
	}
	return -1
}

func functionText(commands []string) string {
	if len(commands) == 0 {
		return ""
	}
	return strings.Join(commands, "\n") + "\n"
}

func tag(function string) string {
	encoded, _ := json.MarshalIndent(map[string]any{"values": []string{function}}, "", "  ")
	return string(encoded) + "\n"
}
