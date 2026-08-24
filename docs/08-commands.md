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
Command interpolation is designed to turn typed values into the representation a command expects. Numeric values become macro inputs and entity values become generated tag selectors.

## Safety
The type checker rejects interpolation that cannot be represented safely. Generated helper functions clean temporary scoreboard and storage values after their operation.
