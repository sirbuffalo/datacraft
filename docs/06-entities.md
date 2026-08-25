---
title: Entities
slug: entities
order: 6
website: true
---
# Entity values

Entity values connect typed variables to Minecraft selectors without repeatedly storing large selectors in commands.

## Singular entities
```dcraft
pig: entity = @e[type=minecraft:pig]
```

Braces are not required around a selector. A singular entity uses limit-one selection and replaces the previous variable tag whenever it is assigned.

## Identity
The runtime registers the currently executing entity, gives it a generated namespace tag, and stores its UUID in an entity object. This lets an entity survive storage in lists and function calls.

## Entity sets
```dcraft
pigs: set[entity] = @e[type=minecraft:pig]
```

An entity set is a generated tag shared by every member. Union, intersection, insertion, removal, membership, and iteration operate directly on tags.

## Command selectors
When an entity value is interpolated into a raw command, DataCraft emits an appropriate tag selector. Runtime helpers centralize common register, resolve, retag, and cleanup operations.

## Reading entity NBT
Use bracket access such as `target["UUID"]` or `target["Brain"]["memories"]`. The result must be assigned to a typed `int`, `bool`, `str`, or `nbt` variable. Property access such as `target.UUID` is not supported.
