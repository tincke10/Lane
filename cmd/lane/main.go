package main

import (
	"os"

	"github.com/tincke10/lane/internal/cmd"
)

var version = "dev"

func main() {
	if len(os.Args) >= 2 && (os.Args[1] == "version" || os.Args[1] == "--version") {
		os.Stdout.WriteString("lane " + version + "\n")
		os.Exit(0)
	}
	os.Exit(cmd.Run(os.Args[1:], os.Stdout, os.Stderr))
}
