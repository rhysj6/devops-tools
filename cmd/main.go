package cmd

import (
	"github.com/spf13/cobra"
)

func GetCommand(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "devops-tools",
		Short:   "A set of devops cli tools",
		Aliases: []string{"do", "dot"},
		Version: version,
	}

	rootCmd.PersistentFlags().StringP("config", "c", "", "Path to config file")
	_ = rootCmd.MarkFlagFilename("config", "yaml")

	addLogParserCommands(rootCmd)

	return rootCmd
}
