---
title: Functions
slug: functions
order: 3
website: true
---
# Functions and modules

Functions compile to Minecraft functions with generated ID names such as `_1`. Their inputs and result use reserved runtime locations.

## Typed functions
```dcraft
def multiply(value: int, factor: int) -> int:
    return value * factor
```

Parameters require types in version 2. A function without a return annotation defaults to `None`; use `-> int`, `-> str`, or another explicit type only when it returns a value.

```dcraft
def load():
    say("loaded")
```

## Exposed functions
```dcraft
expose def double(value: int) -> int:
    return value * 2
```

`expose` creates a public wrapper with the source-level name. The wrapper calls the internal function and returns its result with Minecraft `/return`.

## Imports
Functions may be imported from another source module. Imports are resolved across every `.dcraft` file in the project source directory.

## Globals
Use `global name` inside a function before assigning a namespace-global variable. Global declarations are statically checked against their declared type.

## Return behavior
`return` exits the current generated function, including from inside conditionals and loops. Scalar results use `/return`; structured results are transferred through the runtime storage protocol.
