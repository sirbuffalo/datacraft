function testpack:_special/entity_register_current
data modify storage testpack:data entities.v52 set value {type:"entity",uuid:[I;0,0,0,0]}
data modify storage testpack:data entities.v52.uuid set from storage testpack:data scratch.entity_uuid
