---
title: Project structure
slug: project-structure
order: 20
website: false
---
# Repository and project structure

This contributor-oriented page is intentionally excluded from the generated website. The other files in this directory are the canonical sources for `web/docs`.

## User project layout
```text
my-pack/
├── datacraft.toml
├── src/
│   ├── main.dcraft
│   └── utilities.dcraft
└── build/
    └── my_pack.zip
```

`datacraft.toml` selects the pack metadata, source directory, and build output. Keep generated data packs out of source control unless they are intentional fixtures.

## Compiler repository
```text
ast/                 parsed syntax tree
lexer/               source tokenization
parser/              indentation-aware parser
semantic/            type and symbol checking
compiler/            Minecraft command lowering
compiler/scope/      IDs, variables, and scopes
datapack/            pack assembly and project builds
internal/cli/        native CLI implementation
internal/docsgen/    Markdown-to-HTML documentation generator
cmd/datacraft/       native CLI entry point
cmd/docsgen/         documentation generator entry point
docs/                canonical Markdown documentation
web/                 static editor and generated website docs
web/wasm/            WebAssembly entry point
examples/            example and integration-test packs
```

## Compilation pipeline
Source passes through lexer, parser, semantic analysis, compiler lowering, and data-pack assembly. The CLI and WebAssembly entry point both call the same compiler packages.

## Documentation workflow
Edit Markdown in `docs`, then run:

```text
go run ./cmd/docsgen
```

Pages with `website: true` are regenerated under `web/docs`. Pages with `website: false`, including this guide, remain repository-only. Never edit generated HTML directly.

## Verification
Before committing compiler or documentation changes, run:

```text
go test ./...
GOOS=js GOARCH=wasm go build ./...
go run ./cmd/docsgen
git diff --check
```
