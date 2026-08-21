scoreboard players set #t113 testpack 0
execute if score #t113 testpack matches 0 run scoreboard players set #t114 testpack 0
execute if score #t113 testpack matches 0 run execute if score #v74 testpack = #c2 testpack run scoreboard players set #t114 testpack 1
execute if score #t113 testpack matches 0 unless score #t114 testpack matches 0 run function testpack:_59
execute if score #t111 testpack matches 1 run return 0
execute if score #t112 testpack matches 1 run return 0
execute if score #t113 testpack matches 0 unless score #t114 testpack matches 0 run scoreboard players set #t113 testpack 1
scoreboard players operation #v73 testpack += #v74 testpack
