package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/rhysj6/devops-tools/internal/config"
	"github.com/rhysj6/devops-tools/internal/pfp"
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
			err = cfg.Pfp.Validate()
			if err != nil {
				return err
			}
			return nil
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

func loadFile(path string) (io.ReadCloser, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func runPfp(cmd *cobra.Command, source string, args []string) error {
	cfg, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}
	if cfg.Pfp == nil {
		return fmt.Errorf("pfp config is not set")
	}

	var rc io.ReadCloser
	var re io.Reader

	switch source {
	case "file":
		rc, err = loadFile(args[0])
		if err != nil {
			return err
		}
	case "jenkins":
		if cfg.Jenkins.Url == "" {
			return fmt.Errorf("Jenkins URL is not set")
		}
		var jobName string
		var buildNumber int
		if len(args) == 1 {
			jobName, buildNumber, err = cfg.Jenkins.GetJobNameAndNumberFromUrl(args[0])
			if err != nil {
				return err
			}
		} else {
			jobName = args[0]
			buildNumber, err = strconv.Atoi(args[1])
			if err != nil {
				return err
			}
		}
		rc, err = cfg.Jenkins.GetBuildLogs(jobName, buildNumber)
		if err != nil {
			return err
		}
	}

	defer rc.Close()
	re = bufio.NewReader(rc)

	m, s, e := pfp.Parse(re, cfg.Pfp.Rules, cfg.Pfp.MaxMatches)
	pfp.TextOutput(os.Stdout, m, s)

	if e != nil {
		return e
	}
	return nil
}
