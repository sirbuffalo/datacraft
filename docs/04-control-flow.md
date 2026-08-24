---
title: Control flow
slug: control-flow
order: 4
website: true
---
# Branches and loops

DataCraft lowers control flow into generated Minecraft functions and `execute if` commands.

## Conditions
```dcraft
if score > 10:
    say("high")
elif score:
    say("positive")
else:
    say("zero")
```

For integers, zero is false and every other value is true. Empty strings and collections are false when converted with `bool`.

## For loops
```dcraft
for item in items:
    say(item)
```

A loop compiles into a helper function that recursively schedules its next iteration.

## While loops
```dcraft
while remaining > 0:
    remaining = remaining - 1
```

The condition is reevaluated before each recursive iteration.

## Loop control
`break` exits the nearest loop. `continue` skips to its next iteration. Returns propagate correctly through generated loop and branch helpers.
