$data modify storage testpack:data variants.v48 set from storage testpack:data lists.v46[$(index0)]
$data modify storage testpack:data variant_types.v48 set from storage testpack:data list_types.v46[$(index0)]
$execute store result score #v48 testpack run data get storage testpack:data lists.v46[$(index0)] 1
