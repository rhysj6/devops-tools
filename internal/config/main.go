package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rhysj6/devops-tools/internal/jenkins"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	Pfp     *LogParserConfig      `mapstructure:"pfp"`
	Jenkins jenkins.JenkinsClient `mapstructure:"jenkins"`
}

func LoadConfig(cmd *cobra.Command) (*Config, error) {
	v := viper.New()

	cfgFlag := cmd.Flags().Lookup("config")
	if cfgFlag != nil && cfgFlag.Value.String() != "" {
		cfgPath := cfgFlag.Value.String()
		v.SetConfigFile(cfgPath)
	} else {
		home, _ := os.UserHomeDir()
		v.AddConfigPath(filepath.Join(home, ".devops-tools"))
		v.AddConfigPath(".")
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}

	v.SetEnvPrefix("DEVOPS_TOOLS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config: %w", err)
		}
	}

	v.BindPFlag("pfp.maxmatches", cmd.Flags().Lookup("max-matches"))
	v.BindPFlag("pfp.output", cmd.Flags().Lookup("output"))
	v.BindEnv("jenkins.url", "DEVOPS_TOOLS_JENKINS_URL", "HUDSON_URL")
	v.BindEnv("jenkins.username")
	v.BindEnv("jenkins.password", "DEVOPS_TOOLS_JENKINS_PASSWORD")

	var config Config

	err := v.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	err = config.SetupConfig()
	if err != nil {
		return nil, err
	}

	return &config, nil
}
