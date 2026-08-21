execute store result storage testpack:data scratch.loop_index int 1 run scoreboard players get #t103 testpack
function testpack:_53 with storage testpack:data scratch
function testpack:_54
execute if score #t105 testpack matches 1 run return 0
scoreboard players set #t106 testpack 0
scoreboard players add #t103 testpack 1
execute if score #t103 testpack < #t104 testpack run function testpack:_52
