$data modify storage testpack:data lists.v56 insert $(index) value 0
$data modify storage testpack:data list_types.v56 insert $(index) value "int"
$execute store result storage testpack:data lists.v56[$(index)] int 1 run scoreboard players get #c8 testpack
