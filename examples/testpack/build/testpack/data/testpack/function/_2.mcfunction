scoreboard players set #rf2 testpack 0
scoreboard players set #t4 testpack 0
execute if score #t4 testpack matches 0 unless score #v3 testpack matches 0 run function testpack:_17
execute if score #rf2 testpack matches 1 run return run scoreboard players get #r2 testpack
execute if score #t4 testpack matches 0 unless score #v3 testpack matches 0 run scoreboard players set #t4 testpack 1
scoreboard players set #rf2 testpack 1
scoreboard players operation #r2 testpack = #c0 testpack
return run scoreboard players get #r2 testpack
