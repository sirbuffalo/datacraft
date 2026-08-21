scoreboard players operation #v76 testpack += #c1 testpack
scoreboard players set #t120 testpack 0
execute if score #t120 testpack matches 0 run scoreboard players set #t121 testpack 0
execute if score #t120 testpack matches 0 run execute if score #v76 testpack = #c2 testpack run scoreboard players set #t121 testpack 1
execute if score #t120 testpack matches 0 unless score #t121 testpack matches 0 run function testpack:_73
execute if score #t118 testpack matches 1 run return 0
execute if score #t119 testpack matches 1 run return 0
execute if score #t120 testpack matches 0 unless score #t121 testpack matches 0 run scoreboard players set #t120 testpack 1
