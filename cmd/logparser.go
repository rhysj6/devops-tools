package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/rhysj6/devops-tools/internal/config"
	"github.com/rhysj6/devops-tools/pkg/logparser"
	"github.com/rhysj6/devops-tools/pkg/logparser/filesource"
	"github.com/rhysj6/devops-tools/pkg/logparser/jenkinssource"
	"github.com/spf13/cobra"
)

func addLogParserCommands(rootCmd *cobra.Command) {
	logParserCmd := &cobra.Command{
		Use:     "logparser",
		Aliases: []string{"lp"},
		Short:   "Pipeline failure parser",
		Long:    `Reads in the log and parses it against a set of rules in the config, will return the first matching true`,
	}

	logParserCmd.PersistentFlags().StringP("output", "o", "text", "output format (json|text)")

	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig(cmd)
			if err != nil {
				return err
			}
			return cfg.LogParser.Validate()
		},
		SilenceUsage: true,
	}
	logParserCmd.AddCommand(validateCmd)

	fileParseCmd := &cobra.Command{
		Use:   "file [path]",
		Short: "Read in a file for failure parsing",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogParser(cmd, "file", args)
		},
	}
	logParserCmd.AddCommand(fileParseCmd)

	jenkinsParseCmd := &cobra.Command{
		Use:   "jenkins [url|job_name] [build_no]",
		Short: "Reads logs from Jenkins for failure parsing",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogParser(cmd, "jenkins", args)
		},
	}
	logParserCmd.AddCommand(jenkinsParseCmd)

	rootCmd.AddCommand(logParserCmd)
}

func runLogParser(cmd *cobra.Command, source string, args []string) error {
	cfg, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}
	if cfg.LogParser == nil {
		return fmt.Errorf("logparser config is not set")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: config.ParseSlogLevel(cfg.LogLevel),
	}))

	var logSource logparser.LogSource

	switch source {
	case "file":
		logSource = filesource.NewFileLogSource(args[0])
	case "jenkins":
		if cfg.Jenkins.URL == "" {
			return fmt.Errorf("the Jenkins URL is not set")
		}
		logSource, err = jenkinssource.NewJenkinsLogSource(cfg.Jenkins, args)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported source: %s", source)
	}

	matches, stats, err := logparser.ParseFromSource(logSource, cfg.LogParser.Rules, cfg.LogParser.MaxMatches, logger)

	logger.Info("stats",
		slog.Any("lines_parsed", stats.LinesParsed),
		slog.Any("duration", stats.Duration),
		slog.Any("partial_matches", stats.PartialMatches),
		slog.Any("complete_matches", stats.CompleteMatches),
	)

	outputFormat, _ := cmd.Flags().GetString("output")
	switch outputFormat {
	case "json":
		logparser.JSONOutput(os.Stdout, matches)
	default:
		logparser.TextOutput(os.Stdout, matches)
	}

	return err
}
