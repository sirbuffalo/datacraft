scoreboard players operation #v0 example += #c1 example
scoreboard players set #t2 example 0
execute if score #v0 example >= #c5 example run scoreboard players set #t2 example 1
scoreboard players operation #v4 example = #t2 example
scoreboard players operation #r1 example = #v4 example
return run scoreboard players get #r1 example
