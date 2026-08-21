function testpack:_51
execute if score #t101 testpack matches 1 run return 0
scoreboard players set #t102 testpack 0
scoreboard players operation #v70 testpack += #cn2 testpack
execute if score #v70 testpack > #t100 testpack run function testpack:_50
