$data modify storage testpack:data variants.v9 set from storage testpack:data lists.v8[$(loop_index)]
$data modify storage testpack:data variant_types.v9 set from storage testpack:data list_types.v8[$(loop_index)]
$execute store result score #v9 testpack run data get storage testpack:data lists.v8[$(loop_index)] 1
