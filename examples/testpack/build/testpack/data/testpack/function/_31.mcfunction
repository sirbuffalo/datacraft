$data modify storage testpack:data variants.v49 set from storage testpack:data lists.v46[$(loop_index)]
$data modify storage testpack:data variant_types.v49 set from storage testpack:data list_types.v46[$(loop_index)]
$execute store result score #v49 testpack run data get storage testpack:data lists.v46[$(loop_index)] 1
