package main

import (
	"fmt"
	"os"

	"github.com/rhysj6/devops-tools/cmd"
)

func main() {
	rootCmd := cmd.GetCommand()
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
