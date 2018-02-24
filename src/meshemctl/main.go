package main

import (
	"github.com/rerorero/meshem/src/meshemctl/command"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "meshemctl",
		Short: "meshem is an example implementation of service mesh.",
	}
)

func init() {
	rootCmd.AddCommand(command.NewVersionCommand())
	rootCmd.AddCommand(command.NewServiceCommand())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		command.ExitWithError(err)
	}
}
