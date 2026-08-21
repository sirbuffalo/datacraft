tag @e[tag=_testpack_list_54] remove _testpack_list_54
scoreboard players set #t127 testpack 0
execute store result score #t128 testpack run data get storage testpack:data lists.v54
execute if score #t127 testpack < #t128 testpack run function testpack:_special/entity_list_54_controller
