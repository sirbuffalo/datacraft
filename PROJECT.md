# mccomp project documentation

`mccomp` is an experimental Python-like language and compiler for Minecraft Java Edition data packs. The compiler is written in Go, keeps its core free of filesystem dependencies so it can compile to WebAssembly, and includes a static browser editor that compiles and downloads data-pack ZIP files locally.

This document records the language and compiler work completed so far.

## Language overview

Source files use indentation and colons instead of braces:

```python
namespace demo

def add(a: int, b: int):
    total = a + b
    return total

export def announce(value: int):
    say("The value is ", value)
    return value
```

Indentation uses four spaces. Tabs are rejected.

## Namespaces and scoreboards

Every source namespace owns one scoreboard objective with the same name. For example:

```python
namespace demo
```

generates:

```mcfunction
scoreboard objectives add demo dummy
```

Numeric values are mapped to fake scoreboard players:

- `#v<ID>` stores a variable.
- `#c<VALUE>` stores an integer constant.
- `#t<ID>` stores a temporary expression result.
- `#r<FUNCTION_ID>` stores an internal function result.
- Control-flow helpers use additional reserved fake-player names.

The load function initializes every numeric constant used by the program so the holder for `5` has the score `5`, the holder for `-2` has the score `-2`, and so on.

## Scopes and globals

Every lexical scope receives a deterministic numeric ID. Variables also receive deterministic IDs and are resolved against their local scope, enclosing scopes, and declared globals.

Use `global` inside a function to assign the namespace-global variable:

```python
def update():
    global counter: int
    counter = 5
```

Global type annotations are optional:

```python
global title: str
global values: list
global target: entity
```

## Functions

Every source function becomes an internal Minecraft function named with an underscore and numeric ID:

```text
add -> _0
load -> _1
```

Function parameters have assigned variable locations. Internal numeric return values use Minecraft's `/return` command so nested returns and returns inside conditionals or loops stop execution correctly.

Function calls and return values support integers, strings, and lists. List results are stored in data storage and have a numeric return identifier when required by Minecraft's function return mechanism.

### Optional parameter types

Function input types are optional:

```python
def format(value: int, label: str, items: list, target: entity):
    pass_value = value
```

Supported type names are:

- `int`
- `bool`, which uses integer storage
- `str`
- `list`
- `entity`

### Exported functions

Prefix a function with `export` to create a public wrapper with its real source name:

```python
export def reward(amount: int):
    return amount + 5
```

The internal implementation still has a name such as `_2`. The wrapper accepts normal Minecraft function macros and calls the internal function. Numeric wrappers use `return run function`, returning the internal value through `/return` rather than requiring caller-provided return players or objectives.

## Numeric expressions

The compiler supports:

- Integer literals and negative values
- `+`, `-`, `*`, `/`, and `%`
- `mod` as a readable alias for `%`
- Unary `+` and `-`
- Compound assignment: `+=`, `-=`, `*=`, `/=`
- Comparisons: `==`, `!=`, `<`, `<=`, `>`, `>=`
- Boolean operators: `and`, `or`, `not`

Examples:

```python
total = a + b
remainder = 14 mod 5
valid = total >= 5 and total != 10
```

## Booleans and truthiness

Booleans are stored as integers:

- `False` is `0`.
- `True` is `1`.
- `0` is falsy.
- Every other integer is truthy.

`bool` is a subset of `int`, not a separate scoreboard representation:

```python
0 is bool       # 1
1 is bool       # 1
2 is bool       # 0
True is int     # 1
```

`bool(x)` normalizes values to `0` or `1`:

```python
bool(0)         # 0
bool(-3)        # 1
bool("")        # 0
bool("hello")   # 1
bool([])        # 0
bool([1])       # 1
```

## Runtime type tests

Use `is` to test a value's type:

```python
value is int
value is bool
value is str
value is list
value is entity
```

Type tests work for regular variables, constant list indexes, dynamically indexed mixed-list values, removed values, and mixed-list loop variables. Mixed values carry synchronized runtime type metadata in data storage.

## Control flow

### Conditionals

```python
if value > 10:
    say("large")
elif value == 10:
    say("equal")
else:
    say("small")
```

Conditionals compile to `execute if` and `execute unless` commands plus generated helper functions. Returns inside all branches propagate out of the containing function.

### While loops

```python
while counter < 5:
    counter += 1
```

Minecraft functions cannot contain conventional jumps, so loop controllers recursively call themselves while the condition remains true.

### For loops

Lists and `range` are supported:

```python
for item in items:
    say(item)

for number in range(1, 5):
    say(number)

for number in range(5, 0, -2):
    say(number)
```

`range` accepts one, two, or three arguments. The step must currently be a nonzero integer literal. Negative steps are supported.

### Break and continue

`break` and `continue` work in `for` and `while` loops, including when nested inside conditionals. They compile to internal scoreboard signals that propagate through generated helper functions.

## Strings

Strings are stored in namespace data storage under paths such as:

```text
strings.v<ID>
```

Supported string behavior includes:

- Literal and variable assignment
- Copying strings
- Equality and inequality comparison
- Joining with `&`
- Converting numbers with `str(x)`
- Passing strings into typed functions
- Storing typed string globals
- Displaying strings with `say`

Examples:

```python
label = "score="
message = label & str(5)
same = message == "score=5"
say(message)
```

String comparison uses temporary data-storage paths and a temporary `string_<ID>` scoreboard objective. The compiler compares NBT values using `execute store success` and `data modify`, converts the result to `0` or `1`, then removes both the scratch data and temporary objective.

## Lists

Lists are stored as NBT lists under:

```text
lists.v<ID>
```

Parallel runtime element types are stored under:

```text
list_types.v<ID>
```

Lists support:

- Empty and nonempty literals
- Numbers, strings, entities, and nested lists
- Constant and dynamic indexing
- Chained nested indexing
- Constant and dynamic numeric writes
- String and entity element writes
- Nested-list replacement
- Function inputs and outputs
- `len(items)`
- `items.append(value)`
- `items.insert(index, value)`
- `items.remove(index)`
- `items.remove()`, which defaults to the last element
- Iteration
- `say(items)` and displaying individual dynamic elements

Example:

```python
items = [1, "two", [3, "four"]]
index = 1
picked = items[index]
say(picked)
say(picked is str)

items.append("five")
items.insert(1, 8)
removed = items.remove()
```

Dynamic mixed values are copied into variant storage paths, while numeric access also loads a scoreboard value for arithmetic. Runtime type metadata travels with indexed, removed, and iterated values.

## Entity values

Entity selectors are valid assignment and list expressions:

```python
target = @s
nearest = @p
pig = @e[type=minecraft:pig,limit=1]
entities = [@p, @e[type=minecraft:pig,limit=1]]
```

An entity value is always represented as a special NBT object:

```snbt
{type:"entity",uuid:[I;0,0,0,0]}
```

The real UUID integer array replaces the zeros at runtime. Entity variables live under paths such as:

```text
entities.v<ID>
```

### UUID tags

Every loaded entity is assigned a deterministic intrinsic tag built from its namespace and four UUID integer components:

```text
_<namespace>_<uuid0>_<uuid1>_<uuid2>_<uuid3>
```

The following shared functions implement registration:

```text
_special/entity_tag_uuid
_special/entity_register_current
_special/entity_register_all
```

`entity_register_current` reads the executing entity's `@s` UUID and applies its UUID tag. It can be called under another selector:

```mcfunction
execute as @e[type=minecraft:pig,limit=1] run function demo:_special/entity_register_current
```

`entity_register_all` executes the registrar as every `@e`. The compiler automatically calls it from the generated tick wrapper.

A macro can locate a registered entity with:

```mcfunction
$effect give @n[tag=_demo_$(uuid0)_$(uuid1)_$(uuid2)_$(uuid3)] speed 10 1
```

### Entity lists

Entity objects can be created, copied, indexed, appended, inserted, removed, and iterated in lists. Each entity-containing list also owns a membership tag:

```text
_<namespace>_list_<LIST_VARIABLE_ID>
```

The generated tick runtime clears and rebuilds each known entity-list membership tag from the UUID objects currently stored in that list. This prevents removal or reordering from permanently leaving stale membership tags.

## `say`

`say` accepts one or more values and joins them into one `tellraw` message:

```python
say("Testing addition: got ", 2 + 3, " expected 5")
say("Items: ", items)
say("Selected: ", items[index])
```

It supports integers, booleans, expressions, strings, lists, nested lists, mixed values, dynamically indexed values, removed values, and entity objects.

## Raw Minecraft commands

A source line beginning with `/` is copied into the generated function without the leading slash:

```python
/tellraw @a {"text":"Hello from a raw command"}
```

General `${expression}` interpolation in raw commands was designed but intentionally postponed while mixed-list and entity execution were completed. It is not yet part of the current parser/compiler behavior.

## Shared special functions

Frequently repeated macro operations are emitted once under `_special/` and reused:

```text
_special/string_from_int
_special/string_concat
_special/entity_tag_uuid
_special/entity_register_current
_special/entity_register_all
```

Entity-list synchronization adds list-specific special functions for loading UUID objects, finding their entities, and rebuilding membership tags. Sharing these helpers reduces duplicate generated `.mcfunction` content.

## Generated load and tick behavior

The generated load function:

- Creates the namespace scoreboard objective.
- Initializes all integer constants.
- Calls the user's `load()` function when one exists.

The generated `_tick.mcfunction`:

- Registers every loaded entity with its UUID tag.
- Synchronizes entity-list membership tags.
- Calls the user's `tick()` function when one exists.

Minecraft load and tick function tags point to the generated wrappers.

## Compiler architecture

The project is divided into small Go packages:

- `token/`: token kinds, operators, and keywords.
- `lexer/`: indentation-aware, WASM-compatible tokenization.
- `ast/`: program, statement, and expression nodes.
- `parser/`: recursive-descent parser for the Python-like syntax.
- `compiler/scope/`: deterministic scope IDs.
- `compiler/`: semantic collection, variable/function mappings, type tracking, and Minecraft command generation.
- `datapack/`: in-memory data-pack file generation.
- `web/wasm/`: JavaScript/WebAssembly bridge.
- `web/`: static editor and download interface.
- `examples/basic/`: small compiler example.
- `examples/testpack/`: comprehensive in-game diagnostic pack.

The compiler and datapack builder operate on strings and in-memory maps rather than reading or writing files. Native tests and the browser frontend decide how generated files are saved.

## WebAssembly editor

The static browser editor includes:

- Go compiler compiled to WebAssembly
- In-browser data-pack compilation
- ZIP generation and download
- CodeMirror editor
- Syntax highlighting for language keywords, types, selectors, literals, comments, operators, and raw Minecraft commands
- No backend requirement

Build the browser compiler from the project root:

```sh
GOOS=js GOARCH=wasm go build -o web/compiler.wasm ./web/wasm
```

Serve the static files:

```sh
cd web
python3 -m http.server 8000
```

Then open `http://localhost:8000/`.

## Building and testing

Run all Go tests:

```sh
go test ./...
```

Check that every package also builds for WebAssembly:

```sh
GOOS=js GOARCH=wasm go build ./...
```

Rebuild the browser module:

```sh
GOOS=js GOARCH=wasm go build -o web/compiler.wasm ./web/wasm
```

The project currently passes native Go tests and a full WebAssembly build.

## Test data pack

The diagnostic source is:

```text
examples/testpack/testpack.mccomp
```

The generated artifacts are:

```text
examples/testpack/build/testpack/
examples/testpack/build/testpack.zip
```

The current pack contains 94 diagnostic tests covering arithmetic, comparisons, truthiness, conditionals, returns, globals, strings, joins, typing, lists, nested and mixed lists, mutation, loops, break/continue, entities, UUID objects, and entity lists.

Run it in Minecraft with:

```mcfunction
/reload
/function testpack:run_tests
```

Every diagnostic line prints a `got` value and an `expected` value.

## Current limitations and likely next work

- General raw-command interpolation such as `/give @s tnt ${number}` is not implemented yet.
- Entity selectors should resolve to one entity when stored as one entity variable; selectors matching many entities cannot form one unambiguous object.
- Entity-list tick synchronization currently focuses on direct entity objects in known lists.
- Dynamic string/entity insertion has more restrictions than numeric insertion in some paths.
- Full static type checking and user-facing type errors are still intentionally lightweight.
- Imports and multiple source files are not implemented.
- Dictionaries or general objects other than the internal entity object are not implemented.
- Return-type annotations are not implemented.

The strongest next feature is command interpolation that understands value kinds. Numeric expressions can become macro numbers, strings can become macro text, entity objects can become UUID-tag selectors, and entity lists can become selectors using their generated membership tags.
