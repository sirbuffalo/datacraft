---
title: NBT
slug: nbt
order: 6
website: true
---
# NBT compounds

The `nbt` type stores JSON-style compound data in the namespace's Minecraft command storage. It may contain integers, booleans, strings, lists, and nested NBT compounds. Minecraft NBT has no null value, so `None` and entities cannot be embedded in a compound.

## Literals
```dcraft
profile: nbt = {"name": "Alex", "health": 20, "active": True, "position": [10, 64, -4], "settings": {"particles": False}}
empty: nbt = {}
```

NBT arrays may contain different NBT-compatible value kinds. Keys must be string literals.

## Bracket access
```dcraft
profile["health"] = 18
health: int = profile["health"]
settings: nbt = profile["settings"]
particles: bool = profile["settings"]["particles"]
```

NBT does not support property syntax: use `profile["health"]`, never `profile.health`. Field reads need an expected type, normally supplied by a typed variable declaration.

## Functions
```dcraft
def update(profile: nbt) -> nbt:
    profile["active"] = True
    return profile
```

Whole NBT variables can be passed to and returned from functions. Structured return data uses runtime storage because Minecraft `/return` itself only carries an integer result.

## Output and length
`say(profile)` displays the compound, while `say(profile["name"])` displays a selected field. `len(profile)` returns the number of keys in the compound.

## Entity data
Entity values expose their Minecraft NBT through the same bracket-only syntax:

```dcraft
target: entity? = @s
uuid: nbt = target["UUID"]
health: int = target["Health"]
name: str = target["CustomName"]
memories: nbt = target["Brain"]["memories"]
```

DataCraft resolves the entity through its registered UUID tag and emits `data get entity` or `data modify ... set from entity` as appropriate for the destination type. Entity NBT paths are currently read-only, their keys must be string literals, and the destination must be `int`, `bool`, `str`, or `nbt`.
