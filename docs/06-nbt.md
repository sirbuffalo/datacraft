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
uuid: list[int] = target["UUID"]
health: int = target["Health"]
name: str = target["CustomName"]
memories: nbt = target["Brain"]["memories"]
effects: list[nbt] = target["active_effects"]
```

DataCraft resolves the entity through its registered UUID tag and emits `data get entity` or `data modify ... set from entity` as appropriate for the destination type. Any NBT list or array can use `list[int]`, `list[bool]`, `list[str]`, or `list[nbt]`; DataCraft generates matching list metadata dynamically for indexing and iteration. Entity NBT keys must be string literals.

Entity NBT fields are writable using the same bracket-only paths:

```dcraft
target["Health"] = 10
target["CustomName"] = "Alex"
target["Brain"]["memories"] = {"home": [1, 2, 3]}
target["active_effects"] = effects
```

The assigned value may be an NBT-compatible literal or an `int`, `bool`, `str`, `list`, or `nbt` variable. DataCraft stages the value in temporary storage, resolves the entity by its registered UUID tag, writes the field, and removes the temporary data. Entity NBT keys must be string literals, and compound fields still use bracket syntax rather than property syntax.

Typed NBT lists may contain compound literals, nested compounds and lists, or existing `nbt` variables:

```dcraft
speed: nbt = {"id": "speed", "duration": 200}
effects: list[nbt] = [
    speed,
    {"id": "strength", "duration": 100, "flags": [True, False]}
]
```

Each element is stored as a compound and receives `nbt` runtime type metadata, so it remains usable when indexed or iterated.
