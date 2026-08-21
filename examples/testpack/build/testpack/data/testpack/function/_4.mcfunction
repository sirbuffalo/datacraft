execute store result score #t7 testpack run data get storage testpack:data lists.v6[0] 1
scoreboard players operation #r4 testpack = #t7 testpack
return run scoreboard players get #r4 testpack
