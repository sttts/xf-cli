package main

import (
	"os"

	"github.com/sttts/xf-cli/cmds"
)

func main() {
	if err := cmds.Execute(); err != nil {
		os.Exit(1)
	}
}
