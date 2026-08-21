scoreboard players set #t12 testpack 0
execute if score #t12 testpack matches 0 run scoreboard players set #t13 testpack 0
execute if score #t12 testpack matches 0 run execute if score #v9 testpack = #c2 testpack run scoreboard players set #t13 testpack 1
execute if score #t12 testpack matches 0 unless score #t13 testpack matches 0 run function testpack:_23
execute if score #rf6 testpack matches 1 run return run scoreboard players get #r6 testpack
execute if score #t10 testpack matches 1 run return 0
execute if score #t11 testpack matches 1 run return 0
execute if score #t12 testpack matches 0 unless score #t13 testpack matches 0 run scoreboard players set #t12 testpack 1
