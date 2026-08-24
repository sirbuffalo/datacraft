---
title: Overview
slug: index
order: 1
website: true
---
# A typed language for Minecraft data packs

DataCraft compiles Python-like, indentation-based source into Minecraft Java Edition data-pack functions. Version 2 is the default when a source file has no version header.

## First program
```dcraft
namespace demo

expose def add(a: int, b: int) -> int:
    return a + b
```

Functions compile to internal ID-based functions. `expose` also creates a stable, named wrapper that other packs can call.

## Runtime model
- Each namespace has its own scoreboard.
- Integer and boolean values use scoreboard players.
- Strings, lists, and structured values use command storage.
- Entity values use tags and UUID-backed objects.
- Functions exchange values through reserved runtime locations.

## Source files
DataCraft source files use the `.dcraft` extension. A project may contain several source files under its configured source directory; imports resolve across those files.

## Core rules
- Function parameters and return values are typed.
- `None` is capitalized and only allowed by nullable types.
- A `bool` is a restricted integer whose value is `0` or `1`.
- Collections are homogeneous unless their declared element type permits otherwise.
