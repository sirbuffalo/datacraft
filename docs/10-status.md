---
title: Status
slug: status
order: 10
website: true
---
# Language status

DataCraft is under active development. The compiler, CLI, WebAssembly build, web editor, typed collections, control flow, modules, and entity runtime are usable and covered by automated tests.

## Implemented
- Typed functions, globals, constants, and readonly bindings
- Integers, booleans, strings, lists, sets, entities, and nullable values
- Branches, loops, break, continue, and nested returns
- Imports and exposed function wrappers
- Native CLI and browser compiler
- Minecraft command and language syntax highlighting

## Still evolving
- Broader command interpolation coverage
- More diagnostics with source spans and suggestions
- Runtime performance and generated-command optimization
- Additional Minecraft-version compatibility testing

## Compatibility
The compiler core remains compatible with Go WebAssembly. Native-only entry points use build constraints so the same compiler packages build for both native and browser targets.
