package main

import (
	"os"

	"github.com/symbolsecurity/cli/internal/commands"
)

func main() {
	if err := commands.Execute(os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
