$data modify storage testpack:data variants.v74 set from storage testpack:data lists.v65[$(loop_index)]
$data modify storage testpack:data variant_types.v74 set from storage testpack:data list_types.v65[$(loop_index)]
$execute store result score #v74 testpack run data get storage testpack:data lists.v65[$(loop_index)] 1
