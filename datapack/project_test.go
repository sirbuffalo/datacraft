package datapack_test

import (
	"strings"
	"testing"

	"github.com/sirbuffalo/datacraft/datapack"
)

func TestBuildProjectImportsAnyFunctionAcrossNamespaces(t *testing.T) {
	sources := map[string]string{
		"math.dcraft": `namespace math

def add(a: int, b: int) -> int:
    return a + b

def identity(values: list[int]) -> list[int]:
    return values
`,
		"main.dcraft": `namespace app

from math import add, identity

expose def run() -> int:
    source: list[int] = [1, 2]
    copied: list[int] = identity(source)
    return add(copied[0], copied[1])
`,
	}
	pack, err := datapack.BuildProject(sources, datapack.Config{PackName: "demo", PackFormat: 48})
	if err != nil {
		t.Fatalf("BuildProject() error = %v", err)
	}
	main := pack.Files["data/app/function/_0.mcfunction"]
	for _, wanted := range []string{
		"data modify storage math:data lists.v2 set from storage app:data lists.v0",
		"function math:_1",
		"data modify storage app:data lists.v1 set from storage math:data returns.r1",
		"scoreboard players operation #v0 math =",
		"scoreboard players operation #v1 math =",
		"function math:_0",
		"return run scoreboard players get #r0 app",
	} {
		if !strings.Contains(main, wanted) {
			t.Fatalf("app function is missing %q:\n%s", wanted, main)
		}
	}
	if _, exposed := pack.Files["data/math/function/add.mcfunction"]; exposed {
		t.Fatal("private imported function unexpectedly gained an exposed wrapper")
	}
	if _, exposed := pack.Files["data/app/function/run.mcfunction"]; !exposed {
		t.Fatal("exposed app wrapper is missing")
	}
	loadTag := pack.Files["data/minecraft/tags/function/load.json"]
	if !strings.Contains(loadTag, `"math:load"`) || !strings.Contains(loadTag, `"app:load"`) {
		t.Fatalf("load tag = %s", loadTag)
	}
}

func TestBuildProjectRejectsCyclesAndMissingFunctions(t *testing.T) {
	_, err := datapack.BuildProject(map[string]string{
		"a.dcraft": "namespace a\nfrom b import use\ndef run() -> None:\n    use()\n",
		"b.dcraft": "namespace b\nfrom a import run\ndef use() -> None:\n    run()\n",
	}, datapack.Config{PackFormat: 48})
	if err == nil || !strings.Contains(err.Error(), "cyclic import") {
		t.Fatalf("cycle error = %v", err)
	}

	_, err = datapack.BuildProject(map[string]string{
		"a.dcraft": "namespace a\nfrom b import missing\ndef run() -> None:\n    missing()\n",
		"b.dcraft": "namespace b\ndef available() -> None:\n    say(1)\n",
	}, datapack.Config{PackFormat: 48})
	if err == nil || !strings.Contains(err.Error(), `has no function "missing"`) {
		t.Fatalf("missing function error = %v", err)
	}
}
