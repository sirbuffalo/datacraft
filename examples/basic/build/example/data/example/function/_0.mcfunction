scoreboard players operation #v0 example = #c0 example
scoreboard players operation #v1 example = #c5 example
scoreboard players operation #t0 example = #v1 example
scoreboard players operation #t0 example *= #c2 example
scoreboard players operation #v2 example = #t0 example
scoreboard players operation #t1 example = #v2 example
scoreboard players operation #t1 example += #c1 example
scoreboard players operation #v3 example = #t1 example
say Example pack loaded
scoreboard players operation #r0 example = #v3 example
return run scoreboard players get #r0 example
