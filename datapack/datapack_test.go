package datapack_test

import (
	"strings"
	"testing"

	"mccomp/datapack"
)

func TestBuildDataPack(t *testing.T) {
	source := "namespace demo\n\ndef load():\n    value = 5\n\ndef tick():\n    value = 1 + 2\n\nexport def reward(amount):\n    return amount + 5\n"
	pack, err := datapack.Build(source, datapack.Config{PackName: "demo", PackFormat: 48})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	load := pack.Files["data/demo/function/load.mcfunction"]
	if !strings.Contains(load, "scoreboard objectives add demo dummy") {
		t.Fatalf("load function does not create pack objective:\n%s", load)
	}
	if !strings.Contains(load, "scoreboard players set #c5 demo 5") {
		t.Fatalf("load function does not initialize constant 5:\n%s", load)
	}
	if _, ok := pack.Files["data/minecraft/tags/function/load.json"]; !ok {
		t.Fatal("load function tag was not generated")
	}
	if _, ok := pack.Files["data/minecraft/tags/function/tick.json"]; !ok {
		t.Fatal("tick function tag was not generated")
	}
	if _, ok := pack.Files["data/demo/function/_0.mcfunction"]; !ok {
		t.Fatal("mapped load function _0 was not generated")
	}
	if _, ok := pack.Files["data/demo/function/_1.mcfunction"]; !ok {
		t.Fatal("mapped tick function _1 was not generated")
	}
	if !strings.Contains(pack.Files["data/minecraft/tags/function/tick.json"], "demo:_tick") {
		t.Fatal("tick tag does not use runtime tick wrapper")
	}
	tick := pack.Files["data/demo/function/_tick.mcfunction"]
	if !strings.Contains(tick, "function demo:_special/entity_register_all") || !strings.Contains(tick, "function demo:_1") {
		t.Fatalf("tick wrapper does not run entity registration and user tick:\n%s", tick)
	}
	wantWrapper := "$scoreboard players set #v2 demo $(amount)\nreturn run function demo:_2\n"
	if wrapper := pack.Files["data/demo/function/reward.mcfunction"]; wrapper != wantWrapper {
		t.Fatalf("export wrapper = %q, want %q", wrapper, wantWrapper)
	}
}

func TestPackNameDoesNotReplaceNamespaceScoreboard(t *testing.T) {
	pack, err := datapack.Build("namespace logic\n\nexport def value():\n    return 5\n", datapack.Config{PackName: "bundle", PackFormat: 48})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	load := pack.Files["data/bundle/function/load.mcfunction"]
	if !strings.Contains(load, "scoreboard objectives add logic dummy") {
		t.Fatalf("load function = %q", load)
	}
	wrapper := pack.Files["data/bundle/function/value.mcfunction"]
	if !strings.Contains(wrapper, "return run function bundle:_0") {
		t.Fatalf("wrapper = %q", wrapper)
	}
}
