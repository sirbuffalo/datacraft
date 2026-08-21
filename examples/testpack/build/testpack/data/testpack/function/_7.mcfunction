scoreboard players set #rf7 testpack 0
scoreboard players operation #v10 testpack = #c0 testpack
scoreboard players set #t14 testpack 0
scoreboard players set #t15 testpack 0
function testpack:_24
execute if score #rf7 testpack matches 1 run return run scoreboard players get #r7 testpack
scoreboard players set #rf7 testpack 1
scoreboard players operation #r7 testpack = #c0 testpack
return run scoreboard players get #r7 testpack
