scoreboard players set #t107 testpack 0
execute if score #t107 testpack matches 0 run scoreboard players set #t108 testpack 0
execute if score #t107 testpack matches 0 run execute if score #v72 testpack = #c3 testpack run scoreboard players set #t108 testpack 1
execute if score #t107 testpack matches 0 unless score #t108 testpack matches 0 run function testpack:_55
execute if score #t105 testpack matches 1 run return 0
execute if score #t106 testpack matches 1 run return 0
execute if score #t107 testpack matches 0 unless score #t108 testpack matches 0 run scoreboard players set #t107 testpack 1
scoreboard players operation #v71 testpack += #v72 testpack
