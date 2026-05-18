package main

import (
	"os"

	"github.com/xpadev-net/gh-auto-switch/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
