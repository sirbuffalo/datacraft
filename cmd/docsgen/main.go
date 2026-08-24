//go:build !js

package main

import (
	"fmt"
	"os"

	"github.com/sirbuffalo/datacraft/internal/docsgen"
)

func main() {
	if err := docsgen.Generate("docs", "web/docs"); err != nil {
		fmt.Fprintln(os.Stderr, "documentation generation failed:", err)
		os.Exit(1)
	}
}
