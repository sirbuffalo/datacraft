data modify storage testpack:data scratch.concat0.left set from storage testpack:data strings.v13
execute store result storage testpack:data scratch.str1.value int 1 run scoreboard players get #v12 testpack
data modify storage testpack:data scratch.str1.destination set value "scratch.concat0.right"
function testpack:_special/string_from_int with storage testpack:data scratch.str1
data remove storage testpack:data scratch.str1
data modify storage testpack:data scratch.concat0.destination set value "strings.v14"
function testpack:_special/string_concat with storage testpack:data scratch.concat0
data remove storage testpack:data scratch.concat0
scoreboard objectives add string_2 dummy
data modify storage testpack:data scratch.string_2.left set from storage testpack:data strings.v14
data modify storage testpack:data scratch.string_2.right set value "value=5"
execute store success score #compare string_2 run data modify storage testpack:data scratch.string_2.left set from storage testpack:data scratch.string_2.right
scoreboard players set #t19 testpack 1
execute if score #compare string_2 matches 1 run scoreboard players set #t19 testpack 0
data remove storage testpack:data scratch.string_2
scoreboard objectives remove string_2
scoreboard players operation #r10 testpack = #t19 testpack
return run scoreboard players get #r10 testpack
