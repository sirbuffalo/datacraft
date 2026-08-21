function testpack:_special/entity_register_current
tag @s add _testpack_list_54
data modify storage testpack:data lists.v54[1] set value {type:"entity",uuid:[I;0,0,0,0]}
data modify storage testpack:data lists.v54[1].uuid set from storage testpack:data scratch.entity_uuid
