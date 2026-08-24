---
title: Built-ins
slug: builtins
order: 7
website: true
---
# Standard operations

Built-ins compile to scoreboard, storage, text-component, and runtime-helper commands.

## Output
`say(value, ...)` joins its arguments into one message. It supports strings, integers, booleans, lists, and expressions.

## Conversion
- `str(value)` converts an integer or boolean to text.
- `bool(value)` returns `0` for zero, an empty string, or an empty list; otherwise it returns `1`.
- `list(value)` creates a compatible typed list.
- `set(value)` creates a compatible typed set.

## Collection helpers
- `len(value)` returns a string, list, or set length.
- `append(value)` adds to the end of a list.
- `insert(index, value)` inserts into a list.
- `remove(index)` removes and returns an element; the index defaults to the end.

## Operators
Arithmetic supports `+`, `-`, `*`, `/`, and `%`. `+` also joins strings and lists. Set union uses `|`; set intersection uses `&`.
