data remove storage testpack:data scratch.entity_uuid
execute store result storage testpack:data scratch.list_index int 1 run scoreboard players get #t127 testpack
function testpack:_special/entity_list_54_loader with storage testpack:data scratch
execute if data storage testpack:data scratch.entity_uuid run function testpack:_special/entity_list_54_item
scoreboard players add #t127 testpack 1
execute if score #t127 testpack < #t128 testpack run function testpack:_special/entity_list_54_controller
