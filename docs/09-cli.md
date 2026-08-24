---
title: CLI
slug: cli
order: 9
website: true
---
# Command-line compiler

The `datacraft` command creates, checks, and builds DataCraft projects.

## Create a project
```text
datacraft init my-pack
cd my-pack
datacraft check
datacraft build
```

## Configuration
```toml
[pack]
name = "my_pack"
description = "Built with DataCraft"
format = 48

[build]
source = "src"
output = "build/my_pack.zip"
```

Project configuration lives in `datacraft.toml`. Command-line options override its values.

## Build options
- `-o` or `--output` chooses a directory or ZIP file.
- `-n` or `--name` overrides the pack name.
- `--description` overrides the description.
- `--pack-format` selects the Minecraft pack format.
- `-q` or `--quiet` suppresses success output.

## Single files
`datacraft build path/to/main.dcraft` compiles one file. A project build discovers all `.dcraft` files below the configured source directory so imports are available.

## Browser projects
The web editor automatically saves source files and pack settings in IndexedDB. Existing saves from the older `localStorage` format are migrated once. **Save project** downloads `datacraft.toml` and the complete `src` directory as a ZIP. **Open project** restores a previously saved source ZIP without uploading it to a server.

Use **Load example** to replace the editor contents with a known-working typed project that demonstrates functions, lists, conditions, and joined output. The example is immediately saved locally and is ready to compile.

Use the file toolbar to add, rename, or delete source files. Renames must be unique relative paths ending in `.dcraft`. Deletion asks for confirmation, and the editor always keeps at least one source file in the project.
