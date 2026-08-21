scoreboard players set #t5 testpack 0
execute if score #t5 testpack matches 0 unless score #v4 testpack matches 0 run function testpack:_18
execute if score #rf2 testpack matches 1 run return run scoreboard players get #r2 testpack
execute if score #t5 testpack matches 0 unless score #v4 testpack matches 0 run scoreboard players set #t5 testpack 1
