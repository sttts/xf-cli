package main

import (
	"fmt"
	"os"

	"github.com/sttts/xf-cli/cmds"
)

func main() {
	if err := cmds.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
