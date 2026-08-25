# DataCraft

DataCraft is a statically typed, Python-like language that compiles to Minecraft Java Edition data packs. It provides functions, expressions, control flow, typed collections, entity values, imports, and raw Minecraft commands while generating the scoreboards, storage, tags, and helper functions needed at runtime.

```dcraft
namespace demo

expose def add(a: int, b: int) -> int:
    return a + b

expose def load():
    values: list[int] = [2, 3, 5]
    result: int = add(values[0], values[1])
    if result == 5:
        say("DataCraft loaded: ", result)
```

## Features

- Required parameter types and statically checked return types
- Omitted function return annotations default to `None`
- `int`, `bool`, `str`, `entity`, nullable values, lists, and sets
- Storage-backed NBT compounds with JSON-style literals and bracket access
- `if`, `elif`, `else`, `for`, `while`, `break`, and `continue`
- Internal ID-based functions and stable public wrappers using `expose`
- Multi-file projects with imports
- Entity values and entity sets backed by generated tags
- Typed entity NBT reads such as `target["UUID"]`
- Raw Minecraft commands and syntax highlighting
- Typed `${variable}` interpolation in raw Minecraft commands
- Native Go CLI and a static WebAssembly browser editor
- Project ZIP import/export and IndexedDB autosave in the browser

## Install the CLI

DataCraft currently requires Go to install from source:

```sh
go install github.com/sirbuffalo/datacraft/cmd/datacraft@latest
```

For local development:

```sh
git clone https://github.com/sirbuffalo/datacraft.git
cd datacraft
go install ./cmd/datacraft
```

## Create and build a project

```sh
datacraft init my-pack
cd my-pack
datacraft check
datacraft build
```

`init` creates this structure:

```text
my-pack/
├── datacraft.toml
├── src/
│   └── main.dcraft
└── build/
    └── my_pack.zip
```

The configuration file controls pack metadata and build paths:

```toml
[pack]
name = "my_pack"
description = "Built with DataCraft"
format = 48

[build]
source = "src"
output = "build/my_pack.zip"
```

Useful commands:

```text
datacraft init [directory]    Create a project
datacraft check [input]       Type-check without writing output
datacraft build [input]       Compile a file or project into a data pack
datacraft version             Print the compiler version
datacraft help                Show all options
```

## Browser editor

Build the WebAssembly compiler and serve the static application:

```sh
GOOS=js GOARCH=wasm go build -o web/compiler.wasm ./web/wasm
python3 -m http.server 8000 -d web
```

Open [http://localhost:8000](http://localhost:8000). The editor runs entirely in the browser and includes multiple files, syntax highlighting, compilation, data-pack downloads, source-project ZIPs, examples, and IndexedDB persistence.

## Documentation

- [Overview](docs/01-overview.md)
- [Types](docs/02-types.md)
- [Functions and modules](docs/03-functions.md)
- [Control flow](docs/04-control-flow.md)
- [Lists and sets](docs/05-collections.md)
- [Entities](docs/06-entities.md)
- [NBT compounds](docs/06-nbt.md)
- [Built-ins and operators](docs/07-builtins.md)
- [Raw Minecraft commands](docs/08-commands.md)
- [CLI and browser projects](docs/09-cli.md)
- [Implementation status](docs/10-status.md)
- [Repository and project structure](docs/20-project-structure.md)

The Markdown files under `docs/` are canonical. Generate the website documentation after editing them:

```sh
go run ./cmd/docsgen
```

Pages marked `website: true` are written to `web/docs/`. Contributor material marked `website: false` remains Markdown-only.

## Development

Run native tests and confirm that all shared packages remain WebAssembly-compatible:

```sh
go test ./...
GOOS=js GOARCH=wasm go build ./...
go run ./cmd/docsgen
git diff --check
```

The compiler pipeline is lexer → parser → semantic checker → command compiler → data-pack assembly. Native-only commands use Go build constraints so the compiler core can also run in WebAssembly.
