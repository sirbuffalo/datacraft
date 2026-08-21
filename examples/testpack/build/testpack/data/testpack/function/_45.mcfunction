execute store result storage testpack:data scratch.loop_index int 1 run scoreboard players get #t93 testpack
function testpack:_46 with storage testpack:data scratch
function testpack:_47
execute if score #t95 testpack matches 1 run return 0
scoreboard players set #t96 testpack 0
scoreboard players add #t93 testpack 1
execute if score #t93 testpack < #t94 testpack run function testpack:_45
