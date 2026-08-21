scoreboard players set #t117 testpack 0
execute if score #v75 testpack < #c4 testpack run scoreboard players set #t117 testpack 1
execute unless score #t117 testpack matches 0 run function testpack:_61
execute if score #t115 testpack matches 1 run return 0
scoreboard players set #t116 testpack 0
execute unless score #t117 testpack matches 0 run function testpack:_60
