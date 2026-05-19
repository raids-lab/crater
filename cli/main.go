package main

import (
	"os"

	"github.com/raids-lab/crater/cli/cmd"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "__complete" {
		os.Exit(cmd.RunCompleteFast(os.Args[2:]))
	}
	cmd.Execute()
}
