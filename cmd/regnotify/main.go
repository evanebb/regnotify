package main

import (
	"github.com/evanebb/regnotify/cli"
	"os"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
