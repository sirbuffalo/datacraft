data modify storage testpack:data returns.r5 set from storage testpack:data lists.v7
scoreboard players operation #r5 testpack = #c7 testpack
return run scoreboard players get #r5 testpack
