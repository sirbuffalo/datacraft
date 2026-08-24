package docsgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateWebsitePagesAndSkipRepositoryOnlyPages(t *testing.T) {
	source := t.TempDir()
	output := t.TempDir()
	write := func(name, contents string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(source, name), []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("overview.md", "---\ntitle: Overview\nslug: index\norder: 1\nwebsite: true\n---\n# Intro\n\n## Example\nUse `code`.\n\n```dcraft\nsay(1)\n```\n")
	write("private.md", "---\ntitle: Structure\nslug: structure\norder: 2\nwebsite: false\n---\n# Private\n")

	if err := Generate(source, output); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(filepath.Join(output, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	generated := string(contents)
	for _, expected := range []string{"Generated from docs/*.md", "<code>code</code>", "say(1)", `aria-current="page"`} {
		if !strings.Contains(generated, expected) {
			t.Errorf("generated page does not contain %q", expected)
		}
	}
	if _, err := os.Stat(filepath.Join(output, "structure.html")); !os.IsNotExist(err) {
		t.Fatalf("repository-only page was generated: %v", err)
	}
}

func TestGenerateRejectsIncompleteFrontMatter(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "bad.md"), []byte("# Missing metadata\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Generate(source, t.TempDir()); err == nil {
		t.Fatal("Generate succeeded with missing front matter")
	}
}
