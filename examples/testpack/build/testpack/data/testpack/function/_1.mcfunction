scoreboard players set #rf1 testpack 0
scoreboard players set #t1 testpack 0
execute if score #t1 testpack matches 0 run scoreboard players set #t2 testpack 0
execute if score #t1 testpack matches 0 run execute if score #v2 testpack = #c1 testpack run scoreboard players set #t2 testpack 1
execute if score #t1 testpack matches 0 unless score #t2 testpack matches 0 run function testpack:_14
execute if score #rf1 testpack matches 1 run return run scoreboard players get #r1 testpack
execute if score #t1 testpack matches 0 unless score #t2 testpack matches 0 run scoreboard players set #t1 testpack 1
execute if score #t1 testpack matches 0 run scoreboard players set #t3 testpack 0
execute if score #t1 testpack matches 0 run execute if score #v2 testpack = #c2 testpack run scoreboard players set #t3 testpack 1
execute if score #t1 testpack matches 0 unless score #t3 testpack matches 0 run function testpack:_15
execute if score #rf1 testpack matches 1 run return run scoreboard players get #r1 testpack
execute if score #t1 testpack matches 0 unless score #t3 testpack matches 0 run scoreboard players set #t1 testpack 1
execute if score #t1 testpack matches 0 run function testpack:_16
execute if score #rf1 testpack matches 1 run return run scoreboard players get #r1 testpack
