function testpack:_49
execute if score #t98 testpack matches 1 run return 0
scoreboard players set #t99 testpack 0
scoreboard players operation #v68 testpack += #c1 testpack
execute if score #v68 testpack < #t97 testpack run function testpack:_48
