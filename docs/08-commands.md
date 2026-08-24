---
title: Commands
slug: commands
order: 8
website: true
---
# Raw Minecraft commands

A line beginning with `/` is emitted as a Minecraft command from inside a function.

## Basic command
```dcraft
def ignite():
    /give @s minecraft:tnt 1
```

The browser editor highlights command names, selectors, resources, and DataCraft expressions.

## Interpolation
Use `${variable}` to insert a typed DataCraft variable into a command:

```dcraft
def give_tnt(count: int, target: entity):
    /give ${target} minecraft:tnt ${count}
```

DataCraft generates a Minecraft macro helper, copies each value into temporary command storage, invokes the helper with `function ... with storage`, and removes the temporary data afterward. Integer, boolean, string, list, NBT, and singular entity variables are supported. An entity becomes a selector for its registered UUID tag.

Only variable names are accepted inside `${...}`. Existing Minecraft `$(name)` text is left unchanged. Sets cannot be inserted into a single command because they can represent several entities or values.

## Safety
The compiler rejects missing variables, malformed interpolation, and values that cannot be represented safely. Generated helper functions clean temporary scoreboard and storage values after their operation.
