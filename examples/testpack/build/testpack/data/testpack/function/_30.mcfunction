execute store result storage testpack:data scratch.loop_index int 1 run scoreboard players get #t68 testpack
function testpack:_31 with storage testpack:data scratch
function testpack:_32
execute if score #t70 testpack matches 1 run return 0
scoreboard players set #t71 testpack 0
scoreboard players add #t68 testpack 1
execute if score #t68 testpack < #t69 testpack run function testpack:_30
