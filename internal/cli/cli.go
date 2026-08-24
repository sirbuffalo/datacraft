package cli

import (
	"archive/zip"
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/sirbuffalo/datacraft/datapack"
)

const Version = "0.1.0"

type Config struct {
	Name, Description, Source, Output string
	PackFormat                        int
}

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		printHelp(stdout)
		return 0
	}
	var err error
	switch args[0] {
	case "version", "--version":
		fmt.Fprintf(stdout, "DataCraft %s\n", Version)
		return 0
	case "build":
		err = runBuild(args[1:], stdout, stderr, false)
	case "check":
		err = runBuild(args[1:], stdout, stderr, true)
	case "init":
		err = runInit(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "error: unknown command %q\n\n", args[0])
		printHelp(stderr)
		return 2
	}
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return 0
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, `DataCraft — compile typed source into a Minecraft data pack

Usage: datacraft <command> [options]

Commands:
  init [directory]    Create a DataCraft project
  build [input]       Compile a .dcraft file or project directory
  check [input]       Validate without writing output
  version             Print the version
  help                Show this help

Build options:
  -o, --output PATH          Output directory or .zip file
  -n, --name NAME            Pack name for a single-file build
      --description TEXT     Pack description
      --pack-format NUMBER   Minecraft pack format (default 48)
  -q, --quiet                Suppress success output

Projects use datacraft.toml for defaults. CLI options override the config.
Imports resolve from all .dcraft files under the configured source directory.
`)
}

type buildOptions struct {
	output, name, description string
	packFormat                int
	quiet                     bool
}

func runBuild(args []string, stdout, stderr io.Writer, checkOnly bool) error {
	set := flag.NewFlagSet("build", flag.ContinueOnError)
	set.SetOutput(stderr)
	var o buildOptions
	set.StringVar(&o.output, "o", "", "output path")
	set.StringVar(&o.output, "output", "", "output path")
	set.StringVar(&o.name, "n", "", "pack name")
	set.StringVar(&o.name, "name", "", "pack name")
	set.StringVar(&o.description, "description", "", "pack description")
	set.IntVar(&o.packFormat, "pack-format", 0, "Minecraft pack format")
	set.BoolVar(&o.quiet, "q", false, "quiet")
	set.BoolVar(&o.quiet, "quiet", false, "quiet")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() > 1 {
		return fmt.Errorf("expected at most one input path")
	}
	input := "."
	if set.NArg() == 1 {
		input = set.Arg(0)
	}
	absInput, err := filepath.Abs(input)
	if err != nil {
		return err
	}
	info, err := os.Stat(absInput)
	if err != nil {
		return fmt.Errorf("input %s: %w", input, err)
	}
	root := absInput
	if !info.IsDir() {
		root = filepath.Dir(absInput)
	}
	config, err := readConfig(filepath.Join(root, "datacraft.toml"))
	if err != nil {
		return err
	}
	if o.name == "" {
		o.name = config.Name
	}
	if o.description == "" {
		o.description = config.Description
	}
	if o.packFormat == 0 {
		o.packFormat = config.PackFormat
	}
	if o.packFormat == 0 {
		o.packFormat = 48
	}
	buildInput := absInput
	if info.IsDir() && config.Source != "" {
		buildInput = filepath.Join(root, filepath.FromSlash(config.Source))
	}
	pack, count, err := compileInput(buildInput, datapack.Config{PackName: o.name, Description: o.description, PackFormat: o.packFormat})
	if err != nil {
		return err
	}
	if checkOnly {
		if !o.quiet {
			fmt.Fprintf(stdout, "Checked %d source file(s); %d data-pack file(s) would be generated.\n", count, len(pack.Files))
		}
		return nil
	}
	if o.output == "" {
		o.output = config.Output
	}
	if o.output == "" {
		name := o.name
		if name == "" {
			name = filepath.Base(root)
		}
		o.output = filepath.Join("build", name+".zip")
	}
	if !filepath.IsAbs(o.output) {
		o.output = filepath.Join(root, o.output)
	}
	if err = writePack(pack, o.output); err != nil {
		return err
	}
	if !o.quiet {
		fmt.Fprintf(stdout, "Built %d source file(s) into %s (%d files).\n", count, o.output, len(pack.Files))
	}
	return nil
}

func readConfig(filename string) (Config, error) {
	contents, err := os.ReadFile(filename)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	return parseConfig(filename, string(contents))
}

func parseConfig(filename, contents string) (Config, error) {
	var config Config
	section := ""
	scanner := bufio.NewScanner(strings.NewReader(contents))
	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		if strings.HasPrefix(text, "[") && strings.HasSuffix(text, "]") {
			section = strings.TrimSpace(text[1 : len(text)-1])
			if section != "pack" && section != "build" {
				return Config{}, fmt.Errorf("%s:%d: unknown section [%s]", filename, line, section)
			}
			continue
		}
		key, raw, found := strings.Cut(text, "=")
		if !found {
			return Config{}, fmt.Errorf("%s:%d: expected key = value", filename, line)
		}
		key, raw = strings.TrimSpace(key), strings.TrimSpace(raw)
		switch section + "." + key {
		case "pack.name":
			config.Name, found = tomlString(raw)
		case "pack.description":
			config.Description, found = tomlString(raw)
		case "pack.format":
			parsed, parseErr := strconv.Atoi(raw)
			config.PackFormat, found = parsed, parseErr == nil
		case "build.source":
			config.Source, found = tomlString(raw)
		case "build.output":
			config.Output, found = tomlString(raw)
		default:
			return Config{}, fmt.Errorf("%s:%d: unknown setting %s.%s", filename, line, section, key)
		}
		if !found {
			return Config{}, fmt.Errorf("%s:%d: invalid value for %s", filename, line, key)
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func tomlString(value string) (string, bool) {
	decoded, err := strconv.Unquote(value)
	return decoded, err == nil
}

func compileInput(input string, config datapack.Config) (datapack.Pack, int, error) {
	info, err := os.Stat(input)
	if err != nil {
		return datapack.Pack{}, 0, fmt.Errorf("source path %s: %w", input, err)
	}
	if !info.IsDir() {
		if filepath.Ext(input) != ".dcraft" {
			return datapack.Pack{}, 0, fmt.Errorf("source file must use .dcraft")
		}
		contents, err := os.ReadFile(input)
		if err != nil {
			return datapack.Pack{}, 0, err
		}
		pack, err := datapack.Build(string(contents), config)
		return pack, 1, err
	}
	sources, err := discoverSources(input)
	if err != nil {
		return datapack.Pack{}, 0, err
	}
	pack, err := datapack.BuildProject(sources, config)
	return pack, len(sources), err
}

func discoverSources(root string) (map[string]string, error) {
	sources := map[string]string{}
	err := filepath.WalkDir(root, func(filename string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if filename != root && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "build") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(filename) != ".dcraft" {
			return nil
		}
		contents, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, filename)
		if err != nil {
			return err
		}
		sources[filepath.ToSlash(relative)] = string(contents)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(sources) == 0 {
		return nil, fmt.Errorf("no .dcraft files found under %s", root)
	}
	return sources, nil
}

func writePack(pack datapack.Pack, output string) error {
	if strings.EqualFold(filepath.Ext(output), ".zip") {
		return writeZIP(pack.Files, output)
	}
	for name, contents := range pack.Files {
		filename := filepath.Join(output, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filename, []byte(contents), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func writeZIP(files map[string]string, output string) error {
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		file, err := writer.Create(name)
		if err != nil {
			return err
		}
		if _, err = io.WriteString(file, files[name]); err != nil {
			return err
		}
	}
	if err := writer.Close(); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(output), ".datacraft-*.zip")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if _, err = temporary.Write(buffer.Bytes()); err != nil {
		temporary.Close()
		return err
	}
	if err = temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, output)
}

func runInit(args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("init", flag.ContinueOnError)
	set.SetOutput(stderr)
	name := set.String("name", "", "project namespace")
	format := set.Int("pack-format", 48, "Minecraft pack format")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() > 1 {
		return fmt.Errorf("expected at most one directory")
	}
	target := "."
	if set.NArg() == 1 {
		target = set.Arg(0)
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	projectName := *name
	if projectName == "" {
		projectName = sanitizeNamespace(filepath.Base(absTarget))
	}
	if projectName == "" {
		projectName = "example"
	}
	configPath := filepath.Join(absTarget, "datacraft.toml")
	sourcePath := filepath.Join(absTarget, "src", "main.dcraft")
	for _, filename := range []string{configPath, sourcePath} {
		if _, err := os.Stat(filename); err == nil {
			return fmt.Errorf("refusing to overwrite %s", filename)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err = os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		return err
	}
	config := fmt.Sprintf("[pack]\nname = %q\ndescription = %q\nformat = %d\n\n[build]\nsource = %q\noutput = %q\n", projectName, "Built with DataCraft", *format, "src", "build/"+projectName+".zip")
	if err = os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		return err
	}
	source := fmt.Sprintf("namespace %s\n\nexpose def load() -> None:\n    say(\"%s loaded\")\n", projectName, projectName)
	if err = os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Created DataCraft project in %s\n\nNext:\n  cd %s\n  datacraft check\n  datacraft build\n", absTarget, target)
	return nil
}

func sanitizeNamespace(value string) string {
	value = strings.ToLower(value)
	var result strings.Builder
	for _, character := range value {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' || character == '_' || character == '-' || character == '.' {
			result.WriteRune(character)
		} else if result.Len() > 0 {
			result.WriteByte('_')
		}
	}
	return strings.Trim(result.String(), "_.-")
}
