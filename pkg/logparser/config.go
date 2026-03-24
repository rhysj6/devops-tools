package logparser

import (
	"fmt"
	"regexp"
)

type Config struct {
	MatchRules    []*MatchRule `mapstructure:"rules"`
	Output        string       `mapstructure:"output"`
	MaxMatches    int          `mapstructure:"maxmatches"`
	MaxLineSizeKB int          `mapstructure:"maxlinesizekb"`
}

func (c *Config) ApplyDefaults() error {
	if c != nil {
		err := c.CompileRegex()
		if err != nil {
			return err
		}

		if c.Output == "" {
			c.Output = "text"
		}

		if c.MaxMatches == 0 {
			c.MaxMatches = 1
		}

		if c.MaxLineSizeKB == 0 {
			c.MaxLineSizeKB = 4
		}
	}

	return nil
}

func (c *Config) CompileRegex() error {
	if c == nil || c.MatchRules == nil {
		return fmt.Errorf("config is nil or has no match rules")
	}
	for i := range c.MatchRules {
		for j := range c.MatchRules[i].Checks {
			if c.MatchRules[i].Checks[j].RegexText != "" {
				rg, err := regexp.Compile(c.MatchRules[i].Checks[j].RegexText)
				if err != nil {
					return err
				}
				c.MatchRules[i].Checks[j].Regex = rg
			}
		}
	}

	return nil
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	err := c.CompileRegex()
	if err != nil {
		return fmt.Errorf("failed to compile regex: %w", err)
	}
	for i, r := range c.MatchRules {
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
