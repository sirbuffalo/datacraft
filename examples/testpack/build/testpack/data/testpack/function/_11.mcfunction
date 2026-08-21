data modify storage testpack:data strings.v15 set value "global"
scoreboard objectives add string_3 dummy
data modify storage testpack:data scratch.string_3.left set from storage testpack:data strings.v15
data modify storage testpack:data scratch.string_3.right set value "global"
execute store success score #compare string_3 run data modify storage testpack:data scratch.string_3.left set from storage testpack:data scratch.string_3.right
scoreboard players set #t20 testpack 1
execute if score #compare string_3 matches 1 run scoreboard players set #t20 testpack 0
data remove storage testpack:data scratch.string_3
scoreboard objectives remove string_3
scoreboard players operation #r11 testpack = #t20 testpack
return run scoreboard players get #r11 testpack
