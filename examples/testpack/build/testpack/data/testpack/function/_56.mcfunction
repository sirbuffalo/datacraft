execute store result storage testpack:data scratch.loop_index int 1 run scoreboard players get #t109 testpack
function testpack:_57 with storage testpack:data scratch
function testpack:_58
execute if score #t111 testpack matches 1 run return 0
scoreboard players set #t112 testpack 0
scoreboard players add #t109 testpack 1
execute if score #t109 testpack < #t110 testpack run function testpack:_56
