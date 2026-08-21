execute store result storage testpack:data scratch.loop_index int 1 run scoreboard players get #t8 testpack
function testpack:_21 with storage testpack:data scratch
function testpack:_22
execute if score #rf6 testpack matches 1 run return run scoreboard players get #r6 testpack
execute if score #t10 testpack matches 1 run return 0
scoreboard players set #t11 testpack 0
scoreboard players add #t8 testpack 1
execute if score #t8 testpack < #t9 testpack run function testpack:_20
