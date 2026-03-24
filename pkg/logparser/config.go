package logparser

import (
	"fmt"
	"regexp"
)

type Config struct {
	Rules         []*Rule `mapstructure:"rules"`
	Output        string  `mapstructure:"output"`
	MaxMatches    int     `mapstructure:"maxmatches"`
	MaxLineSizeKB int     `mapstructure:"maxlinesizekb"`
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

func (c *Config) Validate() error {
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
