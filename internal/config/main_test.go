package config

import (
	"os"
	"testing"

	"github.com/rhysj6/devops-tools/pkg/logparser"
	"github.com/spf13/cobra"
)

func TestLoadConfig(t *testing.T) {
	t.Run("loads config with environment variables", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("config", "", "")
		cmd.Flags().String("max-matches", "", "")
		cmd.Flags().String("output", "", "")

		t.Setenv("DEVOPS_TOOLS_JENKINS_URL", "https://jenkins.example.com")
		t.Setenv("DEVOPS_TOOLS_JENKINS_USERNAME", "testuser")
		t.Setenv("DEVOPS_TOOLS_JENKINS_PASSWORD", "testpass")

		cfg, err := LoadConfig(cmd)
		if err != nil {
			t.Fatalf("LoadConfig returned error: %v", err)
		}

		if cfg.Jenkins.URL != "https://jenkins.example.com" {
			t.Fatalf("Jenkins URL = %q, want %q", cfg.Jenkins.URL, "https://jenkins.example.com")
		}
		if cfg.Jenkins.Username != "testuser" {
			t.Fatalf("Jenkins Username = %q, want %q", cfg.Jenkins.Username, "testuser")
		}
		if cfg.Jenkins.Password != "testpass" {
			t.Fatalf("Jenkins Password = %q, want %q", cfg.Jenkins.Password, "testpass")
		}
	})

	t.Run("prefers DEVOPS_TOOLS_JENKINS_URL over HUDSON_URL", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("config", "", "")
		cmd.Flags().String("max-matches", "", "")
		cmd.Flags().String("output", "", "")

		t.Setenv("DEVOPS_TOOLS_JENKINS_URL", "https://primary.jenkins.com")
		t.Setenv("HUDSON_URL", "https://fallback.jenkins.com")

		cfg, err := LoadConfig(cmd)
		if err != nil {
			t.Fatalf("LoadConfig returned error: %v", err)
		}

		if cfg.Jenkins.URL != "https://primary.jenkins.com" {
			t.Fatalf("Jenkins URL = %q, want %q", cfg.Jenkins.URL, "https://primary.jenkins.com")
		}
	})

	t.Run("falls back to HUDSON_URL when DEVOPS_TOOLS_JENKINS_URL not set", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("config", "", "")
		cmd.Flags().String("max-matches", "", "")
		cmd.Flags().String("output", "", "")

		// Unset DEVOPS_TOOLS_JENKINS_URL if it exists
		_ = os.Unsetenv("DEVOPS_TOOLS_JENKINS_URL")
		t.Setenv("HUDSON_URL", "https://fallback.jenkins.com")

		cfg, err := LoadConfig(cmd)
		if err != nil {
			t.Fatalf("LoadConfig returned error: %v", err)
		}

		if cfg.Jenkins.URL != "https://fallback.jenkins.com" {
			t.Fatalf("Jenkins URL = %q, want %q", cfg.Jenkins.URL, "https://fallback.jenkins.com")
		}
	})

	t.Run("handles missing config file gracefully", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("config", "", "")
		cmd.Flags().String("max-matches", "", "")
		cmd.Flags().String("output", "", "")

		cfg, err := LoadConfig(cmd)
		if err != nil {
			t.Fatalf("LoadConfig returned error: %v", err)
		}

		// Should not be nil even with missing config file
		if cfg == nil {
			t.Fatal("Expected Config to be returned, got nil")
		}
	})
}

func TestSetupConfig(t *testing.T) {
	t.Run("sets default output to text", func(t *testing.T) {
		cfg := &Config{
			LogParser: &LogParserConfig{
				Rules:      []*logparser.Rule{},
				Output:     "",
				MaxMatches: 1,
			},
		}

		err := cfg.SetupConfig()
		if err != nil {
			t.Fatalf("SetupConfig returned error: %v", err)
		}

		if cfg.LogParser.Output != "text" {
			t.Fatalf("Output = %q, want %q", cfg.LogParser.Output, "text")
		}
	})

	t.Run("sets default MaxMatches to 1", func(t *testing.T) {
		cfg := &Config{
			LogParser: &LogParserConfig{
				Rules:      []*logparser.Rule{},
				Output:     "text",
				MaxMatches: 0,
			},
		}

		err := cfg.SetupConfig()
		if err != nil {
			t.Fatalf("SetupConfig returned error: %v", err)
		}

		if cfg.LogParser.MaxMatches != 1 {
			t.Fatalf("MaxMatches = %d, want 1", cfg.LogParser.MaxMatches)
		}
	})

	t.Run("preserves non-default values", func(t *testing.T) {
		cfg := &Config{
			LogLevel: "debug",
			LogParser: &LogParserConfig{
				Rules:      []*logparser.Rule{},
				Output:     "json",
				MaxMatches: 5,
			},
		}

		err := cfg.SetupConfig()
		if err != nil {
			t.Fatalf("SetupConfig returned error: %v", err)
		}

		if cfg.LogParser.Output != "json" {
			t.Fatalf("Output = %q, want %q", cfg.LogParser.Output, "json")
		}
		if cfg.LogParser.MaxMatches != 5 {
			t.Fatalf("MaxMatches = %d, want 5", cfg.LogParser.MaxMatches)
		}
		if cfg.LogLevel != "debug" {
			t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
		}
	})

	t.Run("handles nil LogParser gracefully", func(t *testing.T) {
		cfg := &Config{
			LogParser: nil,
		}

		err := cfg.SetupConfig()
		if err != nil {
			t.Fatalf("SetupConfig returned error: %v", err)
		}
	})
}

func TestSetupConfig_Integration(t *testing.T) {
	t.Run("setup with complex rules and regex", func(t *testing.T) {
		cfg := &Config{
			LogParser: &LogParserConfig{
				Rules: []*logparser.Rule{
					{
						Name: "errors",
						Checks: []logparser.LineMatcher{
							{RegexText: "ERROR", Regex: nil},
							{RegexText: "CRITICAL", Regex: nil},
						},
					},
					{
						Name: "warnings",
						Checks: []logparser.LineMatcher{
							{RegexText: "WARN", Regex: nil},
						},
					},
				},
				Output:     "json",
				MaxMatches: 50,
			},
		}

		err := cfg.SetupConfig()
		if err != nil {
			t.Fatalf("SetupConfig returned error: %v", err)
		}

		if cfg.LogParser.Output != "json" {
			t.Fatalf("Output not preserved")
		}
		if cfg.LogParser.MaxMatches != 50 {
			t.Fatalf("MaxMatches not preserved")
		}
		if cfg.LogParser.Rules[0].Checks[0].Regex == nil {
			t.Fatal("Regex should be compiled")
		}
	})

	t.Run("setup returns error on invalid regex", func(t *testing.T) {
		cfg := &Config{
			LogParser: &LogParserConfig{
				Rules: []*logparser.Rule{
					{
						Name: "bad",
						Checks: []logparser.LineMatcher{
							{RegexText: "[unclosed", Regex: nil},
						},
					},
				},
				Output:     "text",
				MaxMatches: 1,
			},
		}

		err := cfg.SetupConfig()
		if err == nil {
			t.Fatal("Expected error for invalid regex in setup")
		}
	})
}

func TestLoadConfig_Integration(t *testing.T) {
	t.Run("config loading with flags", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("config", "", "config file path")
		cmd.Flags().Int("max-matches", 0, "")
		cmd.Flags().String("output", "", "")

		t.Setenv("DEVOPS_TOOLS_JENKINS_URL", "https://jenkins.test.com")

		cfg, err := LoadConfig(cmd)
		if err != nil {
			t.Fatalf("LoadConfig returned error: %v", err)
		}

		if cfg == nil {
			t.Fatal("Config should not be nil")
		}

		if cfg.Jenkins.URL != "https://jenkins.test.com" {
			t.Fatalf("Jenkins URL not loaded from env")
		}
	})
}
