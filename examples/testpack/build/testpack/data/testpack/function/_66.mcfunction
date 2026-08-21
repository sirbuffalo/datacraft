scoreboard players operation #v77 testpack += #c1 testpack
scoreboard players set #t124 testpack 0
execute if score #t124 testpack matches 0 run scoreboard players set #t125 testpack 0
execute if score #t124 testpack matches 0 run execute if score #v77 testpack = #c2 testpack run scoreboard players set #t125 testpack 1
execute if score #t124 testpack matches 0 unless score #t125 testpack matches 0 run function testpack:_67
execute if score #t122 testpack matches 1 run return 0
execute if score #t123 testpack matches 1 run return 0
execute if score #t124 testpack matches 0 unless score #t125 testpack matches 0 run scoreboard players set #t124 testpack 1
scoreboard players operation #v78 testpack += #v77 testpack
