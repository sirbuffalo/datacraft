scoreboard players operation #t21 testpack = #c2 testpack
scoreboard players operation #t22 testpack = #c3 testpack
scoreboard players operation #v0 testpack = #t21 testpack
scoreboard players operation #v1 testpack = #t22 testpack
function testpack:_0
scoreboard players operation #v16 testpack = #r0 testpack
tellraw @a [{"text":"Testing addition: got "},{"score":{"name":"#v16","objective":"testpack"}},{"text":" expected 5"}]
scoreboard players operation #t23 testpack = #c9 testpack
scoreboard players operation #t23 testpack -= #c4 testpack
scoreboard players operation #v17 testpack = #t23 testpack
tellraw @a [{"text":"Testing subtraction: got "},{"score":{"name":"#v17","objective":"testpack"}},{"text":" expected 5"}]
scoreboard players operation #t24 testpack = #c3 testpack
scoreboard players operation #t24 testpack *= #c4 testpack
scoreboard players operation #v18 testpack = #t24 testpack
tellraw @a [{"text":"Testing multiplication: got "},{"score":{"name":"#v18","objective":"testpack"}},{"text":" expected 12"}]
scoreboard players operation #t25 testpack = #c12 testpack
scoreboard players operation #t25 testpack /= #c3 testpack
scoreboard players operation #v19 testpack = #t25 testpack
tellraw @a [{"text":"Testing division: got "},{"score":{"name":"#v19","objective":"testpack"}},{"text":" expected 4"}]
scoreboard players operation #t26 testpack = #c14 testpack
scoreboard players operation #t26 testpack %= #c5 testpack
scoreboard players operation #v20 testpack = #t26 testpack
tellraw @a [{"text":"Testing remainder: got "},{"score":{"name":"#v20","objective":"testpack"}},{"text":" expected 4"}]
scoreboard players operation #t27 testpack = #c14 testpack
scoreboard players operation #t27 testpack %= #c5 testpack
scoreboard players operation #v21 testpack = #t27 testpack
tellraw @a [{"text":"Testing mod keyword: got "},{"score":{"name":"#v21","objective":"testpack"}},{"text":" expected 4"}]
scoreboard players operation #t28 testpack = #c0 testpack
scoreboard players operation #t28 testpack -= #c5 testpack
scoreboard players operation #v22 testpack = #t28 testpack
tellraw @a [{"text":"Testing unary minus: got "},{"score":{"name":"#v22","objective":"testpack"}},{"text":" expected -5"}]
scoreboard players operation #v23 testpack = #c5 testpack
tellraw @a [{"text":"Testing unary plus: got "},{"score":{"name":"#v23","objective":"testpack"}},{"text":" expected 5"}]
scoreboard players operation #v24 testpack = #c10 testpack
scoreboard players operation #v24 testpack += #c5 testpack
scoreboard players operation #v24 testpack -= #c3 testpack
scoreboard players operation #v24 testpack *= #c2 testpack
scoreboard players operation #v24 testpack /= #c6 testpack
tellraw @a [{"text":"Testing compound assignments: got "},{"score":{"name":"#v24","objective":"testpack"}},{"text":" expected 4"}]
scoreboard players set #t29 testpack 0
execute if score #c5 testpack = #c5 testpack run scoreboard players set #t29 testpack 1
scoreboard players operation #v25 testpack = #t29 testpack
tellraw @a [{"text":"Testing ==: got "},{"score":{"name":"#v25","objective":"testpack"}},{"text":" expected 1"}]
scoreboard players set #t30 testpack 0
execute unless score #c5 testpack = #c4 testpack run scoreboard players set #t30 testpack 1
scoreboard players operation #v26 testpack = #t30 testpack
tellraw @a [{"text":"Testing !=: got "},{"score":{"name":"#v26","objective":"testpack"}},{"text":" expected 1"}]
scoreboard players set #t31 testpack 0
execute if score #c4 testpack < #c5 testpack run scoreboard players set #t31 testpack 1
scoreboard players operation #v27 testpack = #t31 testpack
tellraw @a [{"text":"Testing \u003c: got "},{"score":{"name":"#v27","objective":"testpack"}},{"text":" expected 1"}]
scoreboard players set #t32 testpack 0
execute if score #c5 testpack <= #c5 testpack run scoreboard players set #t32 testpack 1
scoreboard players operation #v28 testpack = #t32 testpack
tellraw @a [{"text":"Testing \u003c=: got "},{"score":{"name":"#v28","objective":"testpack"}},{"text":" expected 1"}]
scoreboard players set #t33 testpack 0
execute if score #c6 testpack > #c5 testpack run scoreboard players set #t33 testpack 1
scoreboard players operation #v29 testpack = #t33 testpack
tellraw @a [{"text":"Testing \u003e: got "},{"score":{"name":"#v29","objective":"testpack"}},{"text":" expected 1"}]
scoreboard players set #t34 testpack 0
execute if score #c5 testpack >= #c5 testpack run scoreboard players set #t34 testpack 1
scoreboard players operation #v30 testpack = #t34 testpack
tellraw @a [{"text":"Testing \u003e=: got "},{"score":{"name":"#v30","objective":"testpack"}},{"text":" expected 1"}]
scoreboard players set #t35 testpack 0
execute unless score #c1 testpack matches 0 unless score #c1 testpack matches 0 run scoreboard players set #t35 testpack 1
scoreboard players operation #v31 testpack = #t35 testpack
tellraw @a [{"text":"Testing and: got "},{"score":{"name":"#v31","objective":"testpack"}},{"text":" expected 1"}]
scoreboard players set #t36 testpack 0
execute unless score #c0 testpack matches 0 run scoreboard players set #t36 testpack 1
execute unless score #c1 testpack matches 0 run scoreboard players set #t36 testpack 1
scoreboard players operation #v32 testpack = #t36 testpack
tellraw @a [{"text":"Testing or: got "},{"score":{"name":"#v32","objective":"testpack"}},{"text":" expected 1"}]
scoreboard players set #t37 testpack 1
execute unless score #c0 testpack matches 0 run scoreboard players set #t37 testpack 0
scoreboard players operation #v33 testpack = #t37 testpack
tellraw @a [{"text":"Testing not: got "},{"score":{"name":"#v33","objective":"testpack"}},{"text":" expected 1"}]
scoreboard players operation #t38 testpack = #c0 testpack
scoreboard players operation #t38 testpack -= #c4 testpack
scoreboard players operation #t39 testpack = #t38 testpack
scoreboard players operation #v5 testpack = #t39 testpack
function testpack:_3
scoreboard players operation #v34 testpack = #r3 testpack
tellraw @a [{"text":"Testing negative truthiness: got "},{"score":{"name":"#v34","objective":"testpack"}},{"text":" expected 1"}]
scoreboard players operation #t40 testpack = #c0 testpack
scoreboard players operation #v5 testpack = #t40 testpack
function testpack:_3
scoreboard players operation #v35 testpack = #r3 testpack
tellraw @a [{"text":"Testing zero falsiness: got "},{"score":{"name":"#v35","objective":"testpack"}},{"text":" expected 0"}]
scoreboard players operation #t41 testpack = #c1 testpack
scoreboard players operation #v2 testpack = #t41 testpack
function testpack:_1
tellraw @a [{"text":"Testing if branch: got "},{"score":{"name":"#r1","objective":"testpack"}},{"text":" expected 10"}]
scoreboard players operation #t42 testpack = #c2 testpack
scoreboard players operation #v2 testpack = #t42 testpack
function testpack:_1
tellraw @a [{"text":"Testing elif branch: got "},{"score":{"name":"#r1","objective":"testpack"}},{"text":" expected 20"}]
scoreboard players operation #t43 testpack = #c3 testpack
scoreboard players operation #v2 testpack = #t43 testpack
function testpack:_1
tellraw @a [{"text":"Testing else branch: got "},{"score":{"name":"#r1","objective":"testpack"}},{"text":" expected 30"}]
scoreboard players operation #t44 testpack = #c1 testpack
scoreboard players operation #t45 testpack = #c1 testpack
scoreboard players operation #v3 testpack = #t44 testpack
scoreboard players operation #v4 testpack = #t45 testpack
function testpack:_2
tellraw @a [{"text":"Testing nested return: got "},{"score":{"name":"#r2","objective":"testpack"}},{"text":" expected 8"}]
function testpack:_8
function testpack:_9
tellraw @a [{"text":"Testing global: got "},{"score":{"name":"#r9","objective":"testpack"}},{"text":" expected 11"}]
data modify storage testpack:data strings.v36 set value "hello"
tellraw @a [{"text":"Testing string variable: got "},{"nbt":"strings.v36","storage":"testpack:data","interpret":false},{"text":" expected hello"}]
data modify storage testpack:data strings.v37 set from storage testpack:data strings.v36
tellraw @a [{"text":"Testing string copy: got "},{"nbt":"strings.v37","storage":"testpack:data","interpret":false},{"text":" expected hello"}]
scoreboard objectives add string_4 dummy
data modify storage testpack:data scratch.string_4.left set from storage testpack:data strings.v36
data modify storage testpack:data scratch.string_4.right set value "hello"
execute store success score #compare string_4 run data modify storage testpack:data scratch.string_4.left set from storage testpack:data scratch.string_4.right
scoreboard players set #t46 testpack 1
execute if score #compare string_4 matches 1 run scoreboard players set #t46 testpack 0
data remove storage testpack:data scratch.string_4
scoreboard objectives remove string_4
scoreboard players operation #v38 testpack = #t46 testpack
tellraw @a [{"text":"Testing string == true: got "},{"score":{"name":"#v38","objective":"testpack"}},{"text":" expected 1"}]
scoreboard objectives add string_5 dummy
data modify storage testpack:data scratch.string_5.left set from storage testpack:data strings.v36
data modify storage testpack:data scratch.string_5.right set value "world"
execute store success score #compare string_5 run data modify storage testpack:data scratch.string_5.left set from storage testpack:data scratch.string_5.right
scoreboard players set #t47 testpack 1
execute if score #compare string_5 matches 1 run scoreboard players set #t47 testpack 0
data remove storage testpack:data scratch.string_5
scoreboard objectives remove string_5
scoreboard players operation #v39 testpack = #t47 testpack
tellraw @a [{"text":"Testing string == false: got "},{"score":{"name":"#v39","objective":"testpack"}},{"text":" expected 0"}]
scoreboard objectives add string_6 dummy
data modify storage testpack:data scratch.string_6.left set from storage testpack:data strings.v36
data modify storage testpack:data scratch.string_6.right set value "world"
execute store success score #compare string_6 run data modify storage testpack:data scratch.string_6.left set from storage testpack:data scratch.string_6.right
scoreboard players set #t48 testpack 0
execute if score #compare string_6 matches 1 run scoreboard players set #t48 testpack 1
data remove storage testpack:data scratch.string_6
scoreboard objectives remove string_6
scoreboard players operation #v40 testpack = #t48 testpack
tellraw @a [{"text":"Testing string !=: got "},{"score":{"name":"#v40","objective":"testpack"}},{"text":" expected 1"}]
data modify storage testpack:data scratch.concat7.left set value "score="
execute store result storage testpack:data scratch.str8.value int 1 run scoreboard players get #c5 testpack
data modify storage testpack:data scratch.str8.destination set value "scratch.concat7.right"
function testpack:_special/string_from_int with storage testpack:data scratch.str8
data remove storage testpack:data scratch.str8
data modify storage testpack:data scratch.concat7.destination set value "strings.v41"
function testpack:_special/string_concat with storage testpack:data scratch.concat7
data remove storage testpack:data scratch.concat7
tellraw @a [{"text":"Testing string join and str(int): got "},{"nbt":"strings.v41","storage":"testpack:data","interpret":false},{"text":" expected score=5"}]
scoreboard objectives add string_9 dummy
data modify storage testpack:data scratch.string_9.left set from storage testpack:data strings.v41
data modify storage testpack:data scratch.string_9.right set value "score=5"
execute store success score #compare string_9 run data modify storage testpack:data scratch.string_9.left set from storage testpack:data scratch.string_9.right
scoreboard players set #t49 testpack 1
execute if score #compare string_9 matches 1 run scoreboard players set #t49 testpack 0
data remove storage testpack:data scratch.string_9
scoreboard objectives remove string_9
scoreboard players operation #v42 testpack = #t49 testpack
tellraw @a [{"text":"Testing joined string comparison: got "},{"score":{"name":"#v42","objective":"testpack"}},{"text":" expected 1"}]
execute store result storage testpack:data scratch.str10.value int 1 run scoreboard players get #c1 testpack
data modify storage testpack:data scratch.str10.destination set value "strings.v43"
function testpack:_special/string_from_int with storage testpack:data scratch.str10
data remove storage testpack:data scratch.str10
tellraw @a [{"text":"Testing str(bool): got "},{"nbt":"strings.v43","storage":"testpack:data","interpret":false},{"text":" expected 1"}]
scoreboard players operation #t50 testpack = #c5 testpack
data modify storage testpack:data strings.v13 set value "value="
scoreboard players operation #v12 testpack = #t50 testpack
function testpack:_10
tellraw @a [{"text":"Testing typed function inputs: got "},{"score":{"name":"#r10","objective":"testpack"}},{"text":" expected 1"}]
function testpack:_11
tellraw @a [{"text":"Testing typed global: got "},{"score":{"name":"#r11","objective":"testpack"}},{"text":" expected 1"}]
scoreboard players set #t51 testpack 0
scoreboard players set #t51 testpack 1
tellraw @a [{"text":"Testing is int: got "},{"score":{"name":"#t51","objective":"testpack"}},{"text":" expected 1"}]
scoreboard players set #t52 testpack 0
execute if score #c1 testpack matches 0..1 run scoreboard players set #t52 testpack 1
tellraw @a [{"text":"Testing bool subset true: got "},{"score":{"name":"#t52","objective":"testpack"}},{"text":" expected 1"}]
scoreboard players set #t53 testpack 0
execute if score #c2 testpack matches 0..1 run scoreboard players set #t53 testpack 1
tellraw @a [{"text":"Testing bool subset false: got "},{"score":{"name":"#t53","objective":"testpack"}},{"text":" expected 0"}]
scoreboard players set #t54 testpack 0
scoreboard players set #t54 testpack 1
tellraw @a [{"text":"Testing bool is int: got "},{"score":{"name":"#t54","objective":"testpack"}},{"text":" expected 1"}]
scoreboard players set #t55 testpack 0
scoreboard players set #t55 testpack 1
tellraw @a [{"text":"Testing is str: got "},{"score":{"name":"#t55","objective":"testpack"}},{"text":" expected 1"}]
data modify storage testpack:data lists.v44 set value []
data modify storage testpack:data list_types.v44 set value []
data modify storage testpack:data lists.v44 append value 0
data modify storage testpack:data list_types.v44 append value "int"
execute store result storage testpack:data lists.v44[-1] int 1 run scoreboard players get #c1 testpack
scoreboard players set #t56 testpack 0
scoreboard players set #t56 testpack 1
tellraw @a [{"text":"Testing is list: got "},{"score":{"name":"#t56","objective":"testpack"}},{"text":" expected 1"}]
scoreboard players set #t57 testpack 0
execute unless score #c0 testpack matches 0 run scoreboard players set #t57 testpack 1
tellraw @a [{"text":"Testing bool(0): got "},{"score":{"name":"#t57","objective":"testpack"}},{"text":" expected 0"}]
scoreboard players operation #t58 testpack = #c0 testpack
scoreboard players operation #t58 testpack -= #c3 testpack
scoreboard players set #t59 testpack 0
execute unless score #t58 testpack matches 0 run scoreboard players set #t59 testpack 1
tellraw @a [{"text":"Testing bool(nonzero): got "},{"score":{"name":"#t59","objective":"testpack"}},{"text":" expected 1"}]
scoreboard objectives add string_11 dummy
data modify storage testpack:data scratch.string_11.left set value ""
data modify storage testpack:data scratch.string_11.right set value ""
execute store success score #compare string_11 run data modify storage testpack:data scratch.string_11.left set from storage testpack:data scratch.string_11.right
scoreboard players set #t60 testpack 0
execute if score #compare string_11 matches 1 run scoreboard players set #t60 testpack 1
data remove storage testpack:data scratch.string_11
scoreboard objectives remove string_11
tellraw @a [{"text":"Testing bool(empty string): got "},{"score":{"name":"#t60","objective":"testpack"}},{"text":" expected 0"}]
scoreboard objectives add string_12 dummy
data modify storage testpack:data scratch.string_12.left set value "hello"
data modify storage testpack:data scratch.string_12.right set value ""
execute store success score #compare string_12 run data modify storage testpack:data scratch.string_12.left set from storage testpack:data scratch.string_12.right
scoreboard players set #t61 testpack 0
execute if score #compare string_12 matches 1 run scoreboard players set #t61 testpack 1
data remove storage testpack:data scratch.string_12
scoreboard objectives remove string_12
tellraw @a [{"text":"Testing bool(string): got "},{"score":{"name":"#t61","objective":"testpack"}},{"text":" expected 1"}]
data modify storage testpack:data lists.v45 set value []
data modify storage testpack:data list_types.v45 set value []
data modify storage testpack:data lists.v45 append value 0
data modify storage testpack:data list_types.v45 append value "int"
execute store result storage testpack:data lists.v45[-1] int 1 run scoreboard players get #c1 testpack
data modify storage testpack:data lists.v45 append value ""
data modify storage testpack:data list_types.v45 append value "str"
data modify storage testpack:data lists.v45[-1] set value "two"
data modify storage testpack:data lists.v45 append value []
data modify storage testpack:data list_types.v45 append value []
data modify storage testpack:data lists.v45[-1] set value []
data modify storage testpack:data list_types.v45[-1] set value []
data modify storage testpack:data lists.v45[-1] append value 0
data modify storage testpack:data list_types.v45[-1] append value "int"
execute store result storage testpack:data lists.v45[-1][-1] int 1 run scoreboard players get #c3 testpack
data modify storage testpack:data lists.v45[-1] append value ""
data modify storage testpack:data list_types.v45[-1] append value "str"
data modify storage testpack:data lists.v45[-1][-1] set value "four"
tellraw @a [{"text":"Testing mixed list: got "},{"nbt":"lists.v45","storage":"testpack:data"},{"text":" expected [1,\"two\",[3,\"four\"]]"}]
data modify storage testpack:data scratch.say62 set from storage testpack:data lists.v45[1]
tellraw @a [{"text":"Testing mixed string item: got "},{"nbt":"scratch.say62","storage":"testpack:data"},{"text":" expected two"}]
scoreboard players set #t63 testpack 0
data modify storage testpack:data scratch.runtime_type13 set from storage testpack:data list_types.v45[1]
data modify storage testpack:data scratch.runtime_type13_expected set value "str"
scoreboard objectives add string_14 dummy
execute store success score #compare string_14 run data modify storage testpack:data scratch.runtime_type13 set from storage testpack:data scratch.runtime_type13_expected
scoreboard players set #t63 testpack 1
execute if score #compare string_14 matches 1 run scoreboard players set #t63 testpack 0
data remove storage testpack:data scratch.runtime_type13
data remove storage testpack:data scratch.runtime_type13_expected
scoreboard objectives remove string_14
tellraw @a [{"text":"Testing mixed item type: got "},{"score":{"name":"#t63","objective":"testpack"}},{"text":" expected 1"}]
data modify storage testpack:data lists.v45 append value ""
data modify storage testpack:data list_types.v45 append value "str"
data modify storage testpack:data lists.v45[-1] set value "five"
tellraw @a [{"text":"Testing append string: got "},{"nbt":"lists.v45","storage":"testpack:data"},{"text":" expected [1,\"two\",[3,\"four\"],\"five\"]"}]
data modify storage testpack:data lists.v45 insert 1 value ""
data modify storage testpack:data list_types.v45 insert 1 value "str"
data modify storage testpack:data lists.v45[1] set value "inserted"
data modify storage testpack:data scratch.say64 set from storage testpack:data lists.v45[1]
tellraw @a [{"text":"Testing insert string: got "},{"nbt":"scratch.say64","storage":"testpack:data"},{"text":" expected inserted"}]
data modify storage testpack:data lists.v46 set value []
data modify storage testpack:data list_types.v46 set value []
data modify storage testpack:data lists.v46 append value 0
data modify storage testpack:data list_types.v46 append value "int"
execute store result storage testpack:data lists.v46[-1] int 1 run scoreboard players get #c7 testpack
data modify storage testpack:data lists.v46 append value ""
data modify storage testpack:data list_types.v46 append value "str"
data modify storage testpack:data lists.v46[-1] set value "dynamic"
data modify storage testpack:data lists.v46 append value []
data modify storage testpack:data list_types.v46 append value []
data modify storage testpack:data lists.v46[-1] set value []
data modify storage testpack:data list_types.v46[-1] set value []
data modify storage testpack:data lists.v46[-1] append value 0
data modify storage testpack:data list_types.v46[-1] append value "int"
execute store result storage testpack:data lists.v46[-1][-1] int 1 run scoreboard players get #c8 testpack
data modify storage testpack:data lists.v46[-1] append value 0
data modify storage testpack:data list_types.v46[-1] append value "int"
execute store result storage testpack:data lists.v46[-1][-1] int 1 run scoreboard players get #c9 testpack
scoreboard players operation #v47 testpack = #c1 testpack
execute store result storage testpack:data scratch.index0 int 1 run scoreboard players get #v47 testpack
function testpack:_27 with storage testpack:data scratch
tellraw @a [{"text":"Testing dynamic string index: got "},{"nbt":"scratch.say65","storage":"testpack:data"},{"text":" expected dynamic"}]
scoreboard players set #t66 testpack 0
execute store result storage testpack:data scratch.index0 int 1 run scoreboard players get #v47 testpack
function testpack:_28 with storage testpack:data scratch
data modify storage testpack:data scratch.runtime_type15_expected set value "str"
scoreboard objectives add string_16 dummy
execute store success score #compare string_16 run data modify storage testpack:data scratch.runtime_type15 set from storage testpack:data scratch.runtime_type15_expected
scoreboard players set #t66 testpack 1
execute if score #compare string_16 matches 1 run scoreboard players set #t66 testpack 0
data remove storage testpack:data scratch.runtime_type15
data remove storage testpack:data scratch.runtime_type15_expected
scoreboard objectives remove string_16
tellraw @a [{"text":"Testing dynamic indexed type: got "},{"score":{"name":"#t66","objective":"testpack"}},{"text":" expected 1"}]
execute store result storage testpack:data scratch.index0 int 1 run scoreboard players get #v47 testpack
function testpack:_29 with storage testpack:data scratch
tellraw @a [{"text":"Testing dynamic value assignment: got "},{"nbt":"variants.v48","storage":"testpack:data"},{"text":" expected dynamic"}]
scoreboard players set #t67 testpack 0
data modify storage testpack:data scratch.runtime_type17 set from storage testpack:data variant_types.v48
data modify storage testpack:data scratch.runtime_type17_expected set value "str"
scoreboard objectives add string_18 dummy
execute store success score #compare string_18 run data modify storage testpack:data scratch.runtime_type17 set from storage testpack:data scratch.runtime_type17_expected
scoreboard players set #t67 testpack 1
execute if score #compare string_18 matches 1 run scoreboard players set #t67 testpack 0
data remove storage testpack:data scratch.runtime_type17
data remove storage testpack:data scratch.runtime_type17_expected
scoreboard objectives remove string_18
tellraw @a [{"text":"Testing dynamic assigned type: got "},{"score":{"name":"#t67","objective":"testpack"}},{"text":" expected 1"}]
tellraw @a [{"text":"Testing mixed iteration follows with 3 value/type lines:"}]
scoreboard players set #t68 testpack 0
scoreboard players set #t70 testpack 0
scoreboard players set #t71 testpack 0
execute store result score #t69 testpack run data get storage testpack:data lists.v46
execute if score #t68 testpack < #t69 testpack run function testpack:_30
data modify storage testpack:data variants.v50 set from storage testpack:data lists.v46[1]
data modify storage testpack:data variant_types.v50 set from storage testpack:data list_types.v46[1]
execute store result score #v50 testpack run data get storage testpack:data lists.v46[1] 1
data remove storage testpack:data lists.v46[1]
data remove storage testpack:data list_types.v46[1]
tellraw @a [{"text":"Testing removed string return: got "},{"nbt":"variants.v50","storage":"testpack:data"},{"text":" expected dynamic"}]
data modify storage testpack:data variants.v51 set from storage testpack:data lists.v46[-1]
data modify storage testpack:data variant_types.v51 set from storage testpack:data list_types.v46[-1]
execute store result score #v51 testpack run data get storage testpack:data lists.v46[-1] 1
data remove storage testpack:data lists.v46[-1]
data remove storage testpack:data list_types.v46[-1]
tellraw @a [{"text":"Testing removed list return: got "},{"nbt":"variants.v51","storage":"testpack:data"},{"text":" expected [8,9]"}]
scoreboard players set #t75 testpack 0
data modify storage testpack:data scratch.runtime_type25 set from storage testpack:data variant_types.v51
data modify storage testpack:data scratch.runtime_type25_expected set value "list"
scoreboard objectives add string_26 dummy
execute store success score #compare string_26 run data modify storage testpack:data scratch.runtime_type25 set from storage testpack:data scratch.runtime_type25_expected
scoreboard players set #t75 testpack 1
execute if score #compare string_26 matches 1 run scoreboard players set #t75 testpack 0
data remove storage testpack:data scratch.runtime_type25
data remove storage testpack:data scratch.runtime_type25_expected
scoreboard objectives remove string_26
tellraw @a [{"text":"Testing removed list type: got "},{"score":{"name":"#t75","objective":"testpack"}},{"text":" expected 1"}]
tellraw @a [{"text":"Testing bool(empty list): got "},{"score":{"name":"#c0","objective":"testpack"}},{"text":" expected 0"}]
tellraw @a [{"text":"Testing bool(nonempty list): got "},{"score":{"name":"#c1","objective":"testpack"}},{"text":" expected 1"}]
execute as @p run function testpack:_33
scoreboard players set #t76 testpack 0
scoreboard players set #t76 testpack 1
tellraw @a [{"text":"Testing entity variable type: got "},{"score":{"name":"#t76","objective":"testpack"}},{"text":" expected 1"}]
data modify storage testpack:data entities.v53 set from storage testpack:data entities.v52
scoreboard players set #t77 testpack 0
scoreboard players set #t77 testpack 1
tellraw @a [{"text":"Testing entity copy type: got "},{"score":{"name":"#t77","objective":"testpack"}},{"text":" expected 1"}]
tag @e[tag=_testpack_list_54] remove _testpack_list_54
data modify storage testpack:data lists.v54 set value []
data modify storage testpack:data list_types.v54 set value []
data modify storage testpack:data lists.v54 append value {}
execute as @p run function testpack:_34
data modify storage testpack:data list_types.v54 append value "entity"
scoreboard players set #t78 testpack 0
data modify storage testpack:data scratch.runtime_type27 set from storage testpack:data list_types.v54[0]
data modify storage testpack:data scratch.runtime_type27_expected set value "entity"
scoreboard objectives add string_28 dummy
execute store success score #compare string_28 run data modify storage testpack:data scratch.runtime_type27 set from storage testpack:data scratch.runtime_type27_expected
scoreboard players set #t78 testpack 1
execute if score #compare string_28 matches 1 run scoreboard players set #t78 testpack 0
data remove storage testpack:data scratch.runtime_type27
data remove storage testpack:data scratch.runtime_type27_expected
scoreboard objectives remove string_28
tellraw @a [{"text":"Testing entity list item type: got "},{"score":{"name":"#t78","objective":"testpack"}},{"text":" expected 1"}]
data modify storage testpack:data lists.v54 append value {}
execute as @p run function testpack:_35
data modify storage testpack:data list_types.v54 append value "entity"
data modify storage testpack:data lists.v54 insert 1 value {}
execute as @p run function testpack:_36
data modify storage testpack:data list_types.v54 insert 1 value "entity"
execute store result score #t79 testpack run data get storage testpack:data lists.v54
tellraw @a [{"text":"Testing entity list mutations: got "},{"score":{"name":"#t79","objective":"testpack"}},{"text":" expected 3"}]
data modify storage testpack:data entities.v55 set from storage testpack:data lists.v54[0]
scoreboard players set #t80 testpack 0
scoreboard players set #t80 testpack 1
tellraw @a [{"text":"Testing entity loaded from list: got "},{"score":{"name":"#t80","objective":"testpack"}},{"text":" expected 1"}]
data modify storage testpack:data lists.v56 set value []
data modify storage testpack:data list_types.v56 set value []
data modify storage testpack:data lists.v56 append value 0
data modify storage testpack:data list_types.v56 append value "int"
execute store result storage testpack:data lists.v56[-1] int 1 run scoreboard players get #c1 testpack
data modify storage testpack:data lists.v56 append value 0
data modify storage testpack:data list_types.v56 append value "int"
execute store result storage testpack:data lists.v56[-1] int 1 run scoreboard players get #c2 testpack
data modify storage testpack:data lists.v56 append value 0
data modify storage testpack:data list_types.v56 append value "int"
execute store result storage testpack:data lists.v56[-1] int 1 run scoreboard players get #c3 testpack
data modify storage testpack:data lists.v6 set from storage testpack:data lists.v56
function testpack:_4
tellraw @a [{"text":"Testing list read: got "},{"score":{"name":"#r4","objective":"testpack"}},{"text":" expected 1"}]
execute store result storage testpack:data lists.v56[0] int 1 run scoreboard players get #c5 testpack
data modify storage testpack:data list_types.v56[0] set value "int"
data modify storage testpack:data scratch.say81 set from storage testpack:data lists.v56[0]
tellraw @a [{"text":"Testing list constant write: got "},{"nbt":"scratch.say81","storage":"testpack:data"},{"text":" expected 5"}]
scoreboard players operation #v57 testpack = #c1 testpack
execute store result storage testpack:data scratch.index0 int 1 run scoreboard players get #v57 testpack
function testpack:_37 with storage testpack:data scratch
function testpack:_38 with storage testpack:data scratch
execute store result storage testpack:data scratch.index0 int 1 run scoreboard players get #v57 testpack
function testpack:_39 with storage testpack:data scratch
tellraw @a [{"text":"Testing list dynamic write/read: got "},{"nbt":"scratch.say82","storage":"testpack:data"},{"text":" expected 9"}]
data modify storage testpack:data lists.v56 append value 0
data modify storage testpack:data list_types.v56 append value "int"
execute store result storage testpack:data lists.v56[-1] int 1 run scoreboard players get #c4 testpack
execute store result score #t83 testpack run data get storage testpack:data lists.v56
tellraw @a [{"text":"Testing append and len: got "},{"score":{"name":"#t83","objective":"testpack"}},{"text":" expected 4"}]
data modify storage testpack:data lists.v56 insert 1 value 0
data modify storage testpack:data list_types.v56 insert 1 value "int"
execute store result storage testpack:data lists.v56[1] int 1 run scoreboard players get #c7 testpack
data modify storage testpack:data scratch.say84 set from storage testpack:data lists.v56[1]
tellraw @a [{"text":"Testing constant insert: got "},{"nbt":"scratch.say84","storage":"testpack:data"},{"text":" expected 7"}]
scoreboard players operation #v58 testpack = #c2 testpack
execute store result storage testpack:data scratch.index int 1 run scoreboard players get #v58 testpack
function testpack:_40 with storage testpack:data scratch
execute store result storage testpack:data scratch.index0 int 1 run scoreboard players get #v58 testpack
function testpack:_41 with storage testpack:data scratch
tellraw @a [{"text":"Testing dynamic insert: got "},{"nbt":"scratch.say85","storage":"testpack:data"},{"text":" expected 8"}]
data modify storage testpack:data scratch.say_remove86 set from storage testpack:data lists.v56[1]
data remove storage testpack:data lists.v56[1]
data remove storage testpack:data list_types.v56[1]
tellraw @a [{"text":"Testing remove(index) value: got "},{"nbt":"scratch.say_remove86","storage":"testpack:data"},{"text":" expected 7"}]
data modify storage testpack:data scratch.say_remove87 set from storage testpack:data lists.v56[-1]
data remove storage testpack:data lists.v56[-1]
data remove storage testpack:data list_types.v56[-1]
tellraw @a [{"text":"Testing remove() value: got "},{"nbt":"scratch.say_remove87","storage":"testpack:data"},{"text":" expected 4"}]
tellraw @a [{"text":"Testing list after removes: got "},{"nbt":"lists.v56","storage":"testpack:data"},{"text":" expected [5,8,9,3]"}]
data modify storage testpack:data lists.v7 set from storage testpack:data lists.v56
function testpack:_5
data modify storage testpack:data lists.v59 set from storage testpack:data returns.r5
tellraw @a [{"text":"Testing list function output: got "},{"nbt":"lists.v59","storage":"testpack:data"},{"text":" expected [5,8,9,3]"}]
data modify storage testpack:data lists.v60 set value []
data modify storage testpack:data list_types.v60 set value []
tellraw @a [{"text":"Testing empty list: got "},{"nbt":"lists.v60","storage":"testpack:data"},{"text":" expected []"}]
data modify storage testpack:data lists.v61 set value []
data modify storage testpack:data list_types.v61 set value []
data modify storage testpack:data lists.v61 append value []
data modify storage testpack:data list_types.v61 append value []
data modify storage testpack:data lists.v61[-1] set value []
data modify storage testpack:data list_types.v61[-1] set value []
data modify storage testpack:data lists.v61[-1] append value 0
data modify storage testpack:data list_types.v61[-1] append value "int"
execute store result storage testpack:data lists.v61[-1][-1] int 1 run scoreboard players get #c1 testpack
data modify storage testpack:data lists.v61[-1] append value 0
data modify storage testpack:data list_types.v61[-1] append value "int"
execute store result storage testpack:data lists.v61[-1][-1] int 1 run scoreboard players get #c2 testpack
data modify storage testpack:data lists.v61 append value []
data modify storage testpack:data list_types.v61 append value []
data modify storage testpack:data lists.v61[-1] set value []
data modify storage testpack:data list_types.v61[-1] set value []
data modify storage testpack:data lists.v61[-1] append value 0
data modify storage testpack:data list_types.v61[-1] append value "int"
execute store result storage testpack:data lists.v61[-1][-1] int 1 run scoreboard players get #c3 testpack
data modify storage testpack:data lists.v61[-1] append value 0
data modify storage testpack:data list_types.v61[-1] append value "int"
execute store result storage testpack:data lists.v61[-1][-1] int 1 run scoreboard players get #c4 testpack
tellraw @a [{"text":"Testing nested list: got "},{"nbt":"lists.v61","storage":"testpack:data"},{"text":" expected [[1,2],[3,4]]"}]
data modify storage testpack:data scratch.say88 set from storage testpack:data lists.v61[0]
tellraw @a [{"text":"Testing nested sublist: got "},{"nbt":"scratch.say88","storage":"testpack:data"},{"text":" expected [1,2]"}]
data modify storage testpack:data scratch.say89 set from storage testpack:data lists.v61[1][1]
tellraw @a [{"text":"Testing chained list read: got "},{"nbt":"scratch.say89","storage":"testpack:data"},{"text":" expected 4"}]
execute store result storage testpack:data lists.v61[0][1] int 1 run scoreboard players get #c8 testpack
data modify storage testpack:data list_types.v61[0][1] set value "int"
data modify storage testpack:data scratch.say90 set from storage testpack:data lists.v61[0][1]
tellraw @a [{"text":"Testing nested list write: got "},{"nbt":"scratch.say90","storage":"testpack:data"},{"text":" expected 8"}]
data modify storage testpack:data lists.v61[1] set value []
data modify storage testpack:data list_types.v61[1] set value []
data modify storage testpack:data lists.v61[1] append value 0
data modify storage testpack:data list_types.v61[1] append value "int"
execute store result storage testpack:data lists.v61[1][-1] int 1 run scoreboard players get #c9 testpack
data modify storage testpack:data lists.v61[1] append value 0
data modify storage testpack:data list_types.v61[1] append value "int"
execute store result storage testpack:data lists.v61[1][-1] int 1 run scoreboard players get #c10 testpack
data modify storage testpack:data scratch.say91 set from storage testpack:data lists.v61[1]
tellraw @a [{"text":"Testing sublist replacement: got "},{"nbt":"scratch.say91","storage":"testpack:data"},{"text":" expected [9,10]"}]
scoreboard players operation #v62 testpack = #c1 testpack
scoreboard players operation #v63 testpack = #c0 testpack
execute store result storage testpack:data scratch.index0 int 1 run scoreboard players get #v62 testpack
execute store result storage testpack:data scratch.index1 int 1 run scoreboard players get #v63 testpack
function testpack:_42 with storage testpack:data scratch
function testpack:_43 with storage testpack:data scratch
execute store result storage testpack:data scratch.index0 int 1 run scoreboard players get #v62 testpack
execute store result storage testpack:data scratch.index1 int 1 run scoreboard players get #v63 testpack
function testpack:_44 with storage testpack:data scratch
tellraw @a [{"text":"Testing dynamic nested write/read: got "},{"nbt":"scratch.say92","storage":"testpack:data"},{"text":" expected 12"}]
scoreboard players operation #v64 testpack = #c0 testpack
data modify storage testpack:data lists.v65 set value []
data modify storage testpack:data list_types.v65 set value []
data modify storage testpack:data lists.v65 append value 0
data modify storage testpack:data list_types.v65 append value "int"
execute store result storage testpack:data lists.v65[-1] int 1 run scoreboard players get #c1 testpack
data modify storage testpack:data lists.v65 append value 0
data modify storage testpack:data list_types.v65 append value "int"
execute store result storage testpack:data lists.v65[-1] int 1 run scoreboard players get #c2 testpack
data modify storage testpack:data lists.v65 append value 0
data modify storage testpack:data list_types.v65 append value "int"
execute store result storage testpack:data lists.v65[-1] int 1 run scoreboard players get #c3 testpack
scoreboard players set #t93 testpack 0
scoreboard players set #t95 testpack 0
scoreboard players set #t96 testpack 0
execute store result score #t94 testpack run data get storage testpack:data lists.v65
execute if score #t93 testpack < #t94 testpack run function testpack:_45
tellraw @a [{"text":"Testing for list: got "},{"score":{"name":"#v64","objective":"testpack"}},{"text":" expected 6"}]
scoreboard players operation #v67 testpack = #c0 testpack
scoreboard players set #t98 testpack 0
scoreboard players set #t99 testpack 0
scoreboard players operation #v68 testpack = #c1 testpack
scoreboard players operation #t97 testpack = #c5 testpack
execute if score #v68 testpack < #t97 testpack run function testpack:_48
tellraw @a [{"text":"Testing range: got "},{"score":{"name":"#v67","objective":"testpack"}},{"text":" expected 10"}]
scoreboard players operation #v69 testpack = #c0 testpack
scoreboard players set #t101 testpack 0
scoreboard players set #t102 testpack 0
scoreboard players operation #v70 testpack = #c5 testpack
scoreboard players operation #t100 testpack = #c0 testpack
execute if score #v70 testpack > #t100 testpack run function testpack:_50
tellraw @a [{"text":"Testing negative range step: got "},{"score":{"name":"#v69","objective":"testpack"}},{"text":" expected 9"}]
scoreboard players operation #v71 testpack = #c0 testpack
scoreboard players set #t103 testpack 0
scoreboard players set #t105 testpack 0
scoreboard players set #t106 testpack 0
execute store result score #t104 testpack run data get storage testpack:data lists.v65
execute if score #t103 testpack < #t104 testpack run function testpack:_52
tellraw @a [{"text":"Testing break in for/if: got "},{"score":{"name":"#v71","objective":"testpack"}},{"text":" expected 3"}]
scoreboard players operation #v73 testpack = #c0 testpack
scoreboard players set #t109 testpack 0
scoreboard players set #t111 testpack 0
scoreboard players set #t112 testpack 0
execute store result score #t110 testpack run data get storage testpack:data lists.v65
execute if score #t109 testpack < #t110 testpack run function testpack:_56
tellraw @a [{"text":"Testing continue in for/if: got "},{"score":{"name":"#v73","objective":"testpack"}},{"text":" expected 4"}]
scoreboard players operation #v75 testpack = #c0 testpack
scoreboard players set #t115 testpack 0
scoreboard players set #t116 testpack 0
function testpack:_60
tellraw @a [{"text":"Testing while: got "},{"score":{"name":"#v75","objective":"testpack"}},{"text":" expected 4"}]
scoreboard players operation #v76 testpack = #c0 testpack
scoreboard players set #t118 testpack 0
scoreboard players set #t119 testpack 0
function testpack:_62
tellraw @a [{"text":"Testing break in while/if: got "},{"score":{"name":"#v76","objective":"testpack"}},{"text":" expected 2"}]
scoreboard players operation #v77 testpack = #c0 testpack
scoreboard players operation #v78 testpack = #c0 testpack
scoreboard players set #t122 testpack 0
scoreboard players set #t123 testpack 0
function testpack:_65
tellraw @a [{"text":"Testing continue in while/if: got "},{"score":{"name":"#v78","objective":"testpack"}},{"text":" expected 8"}]
data modify storage testpack:data lists.v8 set from storage testpack:data lists.v65
function testpack:_6
tellraw @a [{"text":"Testing return inside for: got "},{"score":{"name":"#r6","objective":"testpack"}},{"text":" expected 2"}]
function testpack:_7
tellraw @a [{"text":"Testing return inside while: got "},{"score":{"name":"#r7","objective":"testpack"}},{"text":" expected 3"}]
tellraw @a [{"text":"Testing joined say: got "},{"score":{"name":"#c2","objective":"testpack"}},{"text":"+"},{"score":{"name":"#c3","objective":"testpack"}},{"text":" expected 2+3"}]
tellraw @a {"text":"Testing raw command: got 1 expected 1"}
tellraw @a [{"text":"--- mccomp test pack complete: 94 tests ---"}]
