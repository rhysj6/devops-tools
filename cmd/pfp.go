package cmd

import (
	"bufio"
	"io"
	"os"

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

	var r io.Reader

	switch source {
	case "file":
		f, e := loadFile(args[0])
		if e != nil {
			return e
		}
		defer f.Close()
		r = bufio.NewReader(f)
	}

	m, s, e := pfp.Parse(r, cfg.Pfp.Rules, cfg.Pfp.MaxMatches)
	pfp.TextOutput(os.Stdout, m, s)

	if e != nil {
		return e
	}
	return nil
}
