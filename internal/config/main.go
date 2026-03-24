package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/rhysj6/devops-tools/pkg/logparser"
	"github.com/rhysj6/devops-tools/pkg/logparser/jenkinssource"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	LogLevel  string                      `mapstructure:"log_level"`
	LogParser *logparser.Config           `mapstructure:"logparser"`
	Jenkins   jenkinssource.JenkinsClient `mapstructure:"jenkins"`
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

	_ = v.BindPFlag("logparser.maxmatches", cmd.Flags().Lookup("max-matches"))
	_ = v.BindPFlag("logparser.output", cmd.Flags().Lookup("output"))
	_ = v.BindEnv("jenkins.url", "DEVOPS_TOOLS_JENKINS_URL", "HUDSON_URL")
	_ = v.BindEnv("jenkins.username")
	_ = v.BindEnv("jenkins.password")

	var config Config

	err := v.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	err = config.LogParser.ApplyDefaults()
	if err != nil {
		return nil, err
	}

	return &config, nil
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
