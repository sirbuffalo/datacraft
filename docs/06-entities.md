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
Use `uuid: list[int] = target["UUID"]` to read the entity's four-integer UUID. Other NBT lists can use a compatible typed list such as `effects: list[nbt] = target["active_effects"]`. Scalar and nested paths use `int`, `bool`, `str`, or `nbt`. Fields are writable with assignments such as `target["Health"] = 10` and `target["Brain"]["memories"] = data`. Property access such as `target.UUID` is not supported.
