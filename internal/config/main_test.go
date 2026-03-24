package config

import (
	"os"
	"testing"

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
func TestParseSlogLevel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "debug", input: "debug", want: "DEBUG"},
		{name: "warn", input: "warn", want: "WARN"},
		{name: "warning", input: "warning", want: "WARN"},
		{name: "error", input: "error", want: "ERROR"},
		{name: "default to info", input: "", want: "INFO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSlogLevel(tt.input).String()
			if got != tt.want {
				t.Fatalf("ParseSlogLevel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
