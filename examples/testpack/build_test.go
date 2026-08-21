package testpack

import (
	"os"
	"path/filepath"
	"testing"

	"mccomp/datapack"
)

func TestSourceBuilds(t *testing.T) {
	source, err := os.ReadFile("testpack.mccomp")
	if err != nil {
		t.Fatal(err)
	}
	pack, err := datapack.Build(string(source), datapack.Config{PackName: "testpack", PackFormat: 48})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(pack.Files) == 0 {
		t.Fatal("data pack has no files")
	}
	for name, contents := range pack.Files {
		path := filepath.Join("build", "testpack", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
