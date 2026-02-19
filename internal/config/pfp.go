package config

import (
	"log"
	"regexp"

	"github.com/rhysj6/devops-tools/internal/pfp"
)

type Config struct {
	Pfp *LogParserConfig `mapstructure:"pfp"`
}

func (c *Config) SetupConfig() error {
	if c.Pfp != nil {
		err := c.Pfp.CompileRegex()
		if err != nil {
			return err
		}

		if c.Pfp.Output == "" {
			c.Pfp.Output = "text"
		}

		if c.Pfp.MaxMatches == 0 {
			c.Pfp.MaxMatches = 1
		}
	}

	return nil
}

type LogParserConfig struct {
	Rules      []*pfp.Rule   `mapstructure:"rules"`
	Output     string        `mapstructure:"output"`
	MaxMatches int           `mapstructure:"maxmatches"`
	Jenkins    JenkinsConfig `mapstructure:"jenkins"`
}

type JenkinsConfig struct {
	Url      string `mapstructure:"url"`      // DEVOPS_TOOLS_PFP_JENKINS_URL
	Username string `mapstructure:"username"` // DEVOPS_TOOLS_PFP_JENKINS_USERNAME
	Password string `mapstructure:"password"` // DEVOPS_TOOLS_PFP_JENKINS_PASSWORD
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
		log.Fatal(err)
	}
	for _, r := range c.Rules {
		if len(r.Checks) == 0 {
			log.Fatal("Error validating config, a rule has 0 checks")
		}
	}

	return nil
}
