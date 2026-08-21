scoreboard players set #t126 testpack 0
execute if score #v77 testpack < #c4 testpack run scoreboard players set #t126 testpack 1
execute unless score #t126 testpack matches 0 run function testpack:_75
execute if score #t122 testpack matches 1 run return 0
scoreboard players set #t123 testpack 0
execute unless score #t126 testpack matches 0 run function testpack:_74
