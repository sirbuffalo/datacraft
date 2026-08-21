scoreboard players set #rf6 testpack 0
scoreboard players set #t8 testpack 0
scoreboard players set #t10 testpack 0
scoreboard players set #t11 testpack 0
execute store result score #t9 testpack run data get storage testpack:data lists.v8
execute if score #t8 testpack < #t9 testpack run function testpack:_20
execute if score #rf6 testpack matches 1 run return run scoreboard players get #r6 testpack
scoreboard players set #rf6 testpack 1
scoreboard players operation #r6 testpack = #c0 testpack
return run scoreboard players get #r6 testpack
