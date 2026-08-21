data modify storage testpack:data scratch.entity_uuid set from entity @s UUID
execute store result storage testpack:data scratch.uuid0 int 1 run data get storage testpack:data scratch.entity_uuid[0]
execute store result storage testpack:data scratch.uuid1 int 1 run data get storage testpack:data scratch.entity_uuid[1]
execute store result storage testpack:data scratch.uuid2 int 1 run data get storage testpack:data scratch.entity_uuid[2]
execute store result storage testpack:data scratch.uuid3 int 1 run data get storage testpack:data scratch.entity_uuid[3]
function testpack:_special/entity_tag_uuid with storage testpack:data scratch
