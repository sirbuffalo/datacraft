# Basic DataCraft example

This example demonstrates:

- a pack namespace;
- local and global variables;
- integer constants;
- arithmetic and comparison expressions;
- compound assignment;
- generated `load` and `tick` entry points;
- deterministic function mappings such as `load -> _0` and `tick -> _1`;
- exposed wrapper functions such as `reset.mcfunction -> _2`;
- a raw Minecraft command;
- returning a scoreboard value.

Compiling with the pack name `example` creates these objectives in the generated
load function:

```mcfunction
scoreboard objectives add example dummy
```

Each source namespace owns one objective named after the namespace. Constants,
variables, and temporary values for that namespace all live in its objective.
Constants use a `#c` holder prefix:

```mcfunction
scoreboard players set #c0 example 0
scoreboard players set #c1 example 1
scoreboard players set #c2 example 2
scoreboard players set #c5 example 5
```

Internal functions read parameters from their assigned `#v<ID>` holders and
write return values to `#r<FUNCTION_ID>`. Exposed wrappers accept parameters as
Minecraft function macros. Returning wrappers use `return run function` so the
exact integer returned by the internal function is returned directly.
