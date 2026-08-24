---
title: Types
slug: types
order: 2
website: true
---
# Static types

DataCraft checks values before generating commands. Function signatures require types; local variable annotations are supported where an explicit declaration is useful.

## Scalar types
- `int` stores a scoreboard integer.
- `bool` is an integer constrained to `0` or `1`.
- `str` stores text in command storage.
- `entity` represents one selected entity.
- `None` represents absence in a nullable type.

## Collection types
- `list[T]` is an ordered storage list.
- `set[int]`, `set[bool]`, and `set[str]` use keyed storage objects.
- `set[entity]` uses an entity tag.
- Nested types such as `list[set[int]]` are supported.

## Type tests
Use `x is int`, `x is bool`, `x is str`, or `x is list`. Because boolean is a subset of integer, `x is bool` specifically checks whether the value is `0` or `1`.

## Nullable values
A nullable type may hold `None`. Singular entity selection uses `@n` semantics and a limit of one; a missing match becomes `None` when the type permits it.

## Immutability
`const` values are fully immutable. `readonly` prevents reassignment while allowing the underlying value according to its declared collection rules.
