scoreboard players operation #t0 testpack = #v0 testpack
scoreboard players operation #t0 testpack += #v1 testpack
scoreboard players operation #r0 testpack = #t0 testpack
return run scoreboard players get #r0 testpack
