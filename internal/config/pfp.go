package config

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/rhysj6/devops-tools/pkg/logparser"
)

func (c *Config) SetupConfig() error {
	if c.LogParser != nil {
		err := c.LogParser.CompileRegex()
		if err != nil {
			return err
		}

		if c.LogParser.Output == "" {
			c.LogParser.Output = "text"
		}

		if c.LogParser.MaxMatches == 0 {
			c.LogParser.MaxMatches = 1
		}
	}

	return nil
}

type LogParserConfig struct {
	Rules      []*logparser.Rule `mapstructure:"rules"`
	Output     string            `mapstructure:"output"`
	MaxMatches int               `mapstructure:"maxmatches"`
}

func ParseSlogLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (c *LogParserConfig) CompileRegex() error {
	for i := range c.Rules {
		for j := range c.Rules[i].Checks {
			if c.Rules[i].Checks[j].RegexText != "" {
				rg, err := regexp.Compile(c.Rules[i].Checks[j].RegexText)
				if err != nil {
					return err
				}
				c.Rules[i].Checks[j].Regex = rg
			}
		}
	}

	return nil
}

func (c *LogParserConfig) Validate() error {
	err := c.CompileRegex()
	if err != nil {
		return fmt.Errorf("failed to compile regex: %w", err)
	}
	for i, r := range c.Rules {
		if len(r.Checks) == 0 {
			return fmt.Errorf("rule %d (%q) has 0 checks", i, r.Name)
		} else {
			if r.MaxLines == 0 && len(r.Checks) > 1 {
				return fmt.Errorf("rule %d (%v) has multiple checks but maxlines is not set", i, r.Name)
			}
		}
	}

	return nil
}
