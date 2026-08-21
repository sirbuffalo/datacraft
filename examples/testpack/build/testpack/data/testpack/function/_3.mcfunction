scoreboard players set #rf3 testpack 0
scoreboard players set #t6 testpack 0
execute if score #t6 testpack matches 0 unless score #v5 testpack matches 0 run function testpack:_19
execute if score #rf3 testpack matches 1 run return run scoreboard players get #r3 testpack
execute if score #t6 testpack matches 0 unless score #v5 testpack matches 0 run scoreboard players set #t6 testpack 1
scoreboard players set #rf3 testpack 1
scoreboard players operation #r3 testpack = #c0 testpack
return run scoreboard players get #r3 testpack
