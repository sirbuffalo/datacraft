---
title: Collections
slug: collections
order: 5
website: true
---
# Lists and sets

Collections are typed and may contain nested collections. Lists preserve order; sets preserve uniqueness.

## Lists
```dcraft
values: list[int] = [1, 2, 3]
values[0] = 5
values.append(8)
values.insert(1, 6)
last: int = values.remove()
first: int = values.remove(0)
```

Use `items[index]` for dynamic indexing. Indexed string and list elements can be read, replaced, or removed and returned.

## List operations
`+` joins two compatible lists. `len(values)` returns their length. `list(value)` converts a compatible collection, and nested types such as `list[set[int]]` remain typed.

## Sets
```dcraft
all: set[int] = left | right
common: set[int] = left & right
```

`|` is union and `&` is intersection. The same operators work for entity sets, which are represented by tags rather than storage iteration.

## Set storage
Primitive sets use a JSON object whose keys point to their values with `1b` membership markers. Entity sets use one generated tag applied to every member.

## Iteration
Lists iterate by ordered index. Primitive sets iterate over their stored keys. Entity sets iterate by executing as every entity with the set tag.
