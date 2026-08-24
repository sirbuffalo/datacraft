//go:build !js

package main

import (
	"os"

	"github.com/sirbuffalo/datacraft/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
