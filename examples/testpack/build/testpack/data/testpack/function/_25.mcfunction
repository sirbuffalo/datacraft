scoreboard players operation #v10 testpack += #c1 testpack
scoreboard players set #t16 testpack 0
execute if score #t16 testpack matches 0 run scoreboard players set #t17 testpack 0
execute if score #t16 testpack matches 0 run execute if score #v10 testpack = #c3 testpack run scoreboard players set #t17 testpack 1
execute if score #t16 testpack matches 0 unless score #t17 testpack matches 0 run function testpack:_26
execute if score #rf7 testpack matches 1 run return run scoreboard players get #r7 testpack
execute if score #t14 testpack matches 1 run return 0
execute if score #t15 testpack matches 1 run return 0
execute if score #t16 testpack matches 0 unless score #t17 testpack matches 0 run scoreboard players set #t16 testpack 1
