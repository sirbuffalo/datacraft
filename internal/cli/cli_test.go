package cli

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCheckAndBuildProject(t *testing.T) {
	root := filepath.Join(t.TempDir(), "My Pack")
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"init", "--name", "demo", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("init code = %d, stderr = %s", code, stderr.String())
	}
	for _, name := range []string{"datacraft.toml", filepath.Join("src", "main.dcraft")} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"check", root}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "Checked 1 source file") {
		t.Fatalf("check code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"build", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("build code = %d, stderr = %s", code, stderr.String())
	}
	archivePath := filepath.Join(root, "build", "demo.zip")
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("open build: %v", err)
	}
	defer archive.Close()
	foundMetadata := false
	for _, file := range archive.File {
		foundMetadata = foundMetadata || file.Name == "pack.mcmeta"
	}
	if !foundMetadata {
		t.Fatal("pack.mcmeta is missing from ZIP")
	}
}

func TestBuildReportsSourceErrors(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "bad.dcraft"), []byte("namespace bad\n\ndef bad() -> int:\n    return \"wrong\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"check", root}, &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "function returns str, not int") {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
}

func TestHelpVersionAndUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"version"}, &stdout, &stderr); code != 0 || stdout.String() != "DataCraft 0.1.0\n" {
		t.Fatalf("version code = %d, output = %q", code, stdout.String())
	}
	stdout.Reset()
	if code := Run(nil, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "build [input]") {
		t.Fatalf("help code = %d, output = %q", code, stdout.String())
	}
	if code := Run([]string{"wat"}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("unknown code = %d, stderr = %q", code, stderr.String())
	}
}
