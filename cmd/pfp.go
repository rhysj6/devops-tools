package cmd

import (
	"fmt"
	"os"

	"github.com/rhysj6/devops-tools/internal/config"
	"github.com/rhysj6/devops-tools/internal/pfp"
	"github.com/rhysj6/devops-tools/pkg/pfp/filesource"
	"github.com/spf13/cobra"
)

func addPfpCommands(rootCmd *cobra.Command) {
	pfpCmd := &cobra.Command{
		Use:   "pfp",
		Short: "Pipeline failure parser",
		Long:  `Reads in the log and parses it against a set of rules in the config, will return the first matching true`,
	}

	pfpCmd.Flags().StringP("output", "o", "text", "output format (json|text)")

	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig(cmd)
			if err != nil {
				return err
			}
			return cfg.Pfp.Validate()
		},
		SilenceUsage: true,
	}
	pfpCmd.AddCommand(validateCmd)

	fileParseCmd := &cobra.Command{
		Use:   "file [path]",
		Short: "Read in a file for failure parsing",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPfp(cmd, "file", args)
		},
	}
	pfpCmd.AddCommand(fileParseCmd)

	jenkinsParseCmd := &cobra.Command{
		Use:   "jenkins [url|job_name] [build_no]",
		Short: "Reads logs from Jenkins for failure parsing",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPfp(cmd, "jenkins", args)
		},
	}
	pfpCmd.AddCommand(jenkinsParseCmd)

	rootCmd.AddCommand(pfpCmd)
}

func runPfp(cmd *cobra.Command, source string, args []string) error {
	cfg, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}
	if cfg.Pfp == nil {
		return fmt.Errorf("pfp config is not set")
	}

	var logSource pfp.LogSource

	switch source {
	case "file":
		logSource = filesource.NewFileLogSource(args[0])
	case "jenkins":
		if cfg.Jenkins.URL == "" {
			return fmt.Errorf("the Jenkins URL is not set")
		}
		logSource, err = pfp.NewJenkinsLogSource(cfg.Jenkins, args)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported source: %s", source)
	}

	matches, stats, err := pfp.ParseFromSource(logSource, cfg.Pfp.Rules, cfg.Pfp.MaxMatches)
	pfp.TextOutput(os.Stdout, matches, stats)
	return err
}
