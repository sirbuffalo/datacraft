scoreboard players set #t72 testpack 0
data modify storage testpack:data scratch.runtime_type19 set from storage testpack:data variant_types.v49
data modify storage testpack:data scratch.runtime_type19_expected set value "int"
scoreboard objectives add string_20 dummy
execute store success score #compare string_20 run data modify storage testpack:data scratch.runtime_type19 set from storage testpack:data scratch.runtime_type19_expected
scoreboard players set #t72 testpack 1
execute if score #compare string_20 matches 1 run scoreboard players set #t72 testpack 0
data remove storage testpack:data scratch.runtime_type19
data remove storage testpack:data scratch.runtime_type19_expected
scoreboard objectives remove string_20
scoreboard players set #t73 testpack 0
data modify storage testpack:data scratch.runtime_type21 set from storage testpack:data variant_types.v49
data modify storage testpack:data scratch.runtime_type21_expected set value "str"
scoreboard objectives add string_22 dummy
execute store success score #compare string_22 run data modify storage testpack:data scratch.runtime_type21 set from storage testpack:data scratch.runtime_type21_expected
scoreboard players set #t73 testpack 1
execute if score #compare string_22 matches 1 run scoreboard players set #t73 testpack 0
data remove storage testpack:data scratch.runtime_type21
data remove storage testpack:data scratch.runtime_type21_expected
scoreboard objectives remove string_22
scoreboard players set #t74 testpack 0
data modify storage testpack:data scratch.runtime_type23 set from storage testpack:data variant_types.v49
data modify storage testpack:data scratch.runtime_type23_expected set value "list"
scoreboard objectives add string_24 dummy
execute store success score #compare string_24 run data modify storage testpack:data scratch.runtime_type23 set from storage testpack:data scratch.runtime_type23_expected
scoreboard players set #t74 testpack 1
execute if score #compare string_24 matches 1 run scoreboard players set #t74 testpack 0
data remove storage testpack:data scratch.runtime_type23
data remove storage testpack:data scratch.runtime_type23_expected
scoreboard objectives remove string_24
tellraw @a [{"text":"Mixed iteration value "},{"nbt":"variants.v49","storage":"testpack:data"},{"text":" types int/str/list: "},{"score":{"name":"#t72","objective":"testpack"}},{"text":"/"},{"score":{"name":"#t73","objective":"testpack"}},{"text":"/"},{"score":{"name":"#t74","objective":"testpack"}}]
