scoreboard players set #t18 testpack 0
execute if score #v10 testpack < #c4 testpack run scoreboard players set #t18 testpack 1
execute unless score #t18 testpack matches 0 run function testpack:_25
execute if score #rf7 testpack matches 1 run return run scoreboard players get #r7 testpack
execute if score #t14 testpack matches 1 run return 0
scoreboard players set #t15 testpack 0
execute unless score #t18 testpack matches 0 run function testpack:_24
