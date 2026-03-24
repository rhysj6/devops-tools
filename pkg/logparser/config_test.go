package logparser

import (
	"testing"
)

func TestCompileRegex(t *testing.T) {
	t.Run("compiles valid regex patterns", func(t *testing.T) {
		lpc := &Config{
			MatchRules: []*MatchRule{
				{
					Name: "test",
					Checks: []LineCheck{
						{RegexText: "test.*pattern", Regex: nil},
					},
				},
			},
		}

		err := lpc.CompileRegex()
		if err != nil {
			t.Fatalf("CompileRegex returned error: %v", err)
		}

		if lpc.MatchRules[0].Checks[0].Regex == nil {
			t.Fatal("Expected regex to be compiled, got nil")
		}
	})

	t.Run("handles empty regex text", func(t *testing.T) {
		lpc := &Config{
			MatchRules: []*MatchRule{
				{
					Name: "test",
					Checks: []LineCheck{
						{RegexText: "", Regex: nil},
					},
				},
			},
		}

		err := lpc.CompileRegex()
		if err != nil {
			t.Fatalf("CompileRegex returned error: %v", err)
		}

		if lpc.MatchRules[0].Checks[0].Regex != nil {
			t.Fatalf("Expected regex to be nil for empty text, got %v", lpc.MatchRules[0].Checks[0].Regex)
		}
	})

	t.Run("returns error for invalid regex", func(t *testing.T) {
		lpc := &Config{
			MatchRules: []*MatchRule{
				{
					Name: "test",
					Checks: []LineCheck{
						{RegexText: "[invalid(regex", Regex: nil},
					},
				},
			},
		}

		err := lpc.CompileRegex()
		if err == nil {
			t.Fatal("Expected error for invalid regex, got nil")
		}
	})

	t.Run("compiles multiple rules and checks", func(t *testing.T) {
		lpc := &Config{
			MatchRules: []*MatchRule{
				{
					Name: "rule1",
					Checks: []LineCheck{
						{RegexText: "error.*", Regex: nil},
						{RegexText: "failed", Regex: nil},
					},
				},
				{
					Name: "rule2",
					Checks: []LineCheck{
						{RegexText: "warning", Regex: nil},
					},
				},
			},
		}

		err := lpc.CompileRegex()
		if err != nil {
			t.Fatalf("CompileRegex returned error: %v", err)
		}

		if lpc.MatchRules[0].Checks[0].Regex == nil || lpc.MatchRules[0].Checks[1].Regex == nil {
			t.Fatal("Expected regexes in rule1 to be compiled")
		}
		if lpc.MatchRules[1].Checks[0].Regex == nil {
			t.Fatal("Expected regex in rule2 to be compiled")
		}
	})
}

func TestLogParserConfig_Output(t *testing.T) {
	t.Run("preserves custom output format", func(t *testing.T) {
		cfg := &Config{
			MatchRules: []*MatchRule{},
			Output:     "json",
			MaxMatches: 10,
		}

		err := cfg.ApplyDefaults()
		if err != nil {
			t.Fatalf("ApplyDefaults returned error: %v", err)
		}

		if cfg.Output != "json" {
			t.Fatalf("Output = %q, want %q", cfg.Output, "json")
		}
	})
}

func TestLogParserConfig_MaxMatches(t *testing.T) {
	t.Run("preserves custom MaxMatches value", func(t *testing.T) {
		lpc := &Config{
			MaxMatches: 42,
		}

		if lpc.MaxMatches != 42 {
			t.Fatalf("MaxMatches = %d, want 42", lpc.MaxMatches)
		}
	})
}

func TestCompileRegex_EdgeCases(t *testing.T) {
	t.Run("skips compilation for empty regex text", func(t *testing.T) {
		lpc := &Config{
			MatchRules: []*MatchRule{
				{
					Name: "mixed",
					Checks: []LineCheck{
						{RegexText: "valid", Regex: nil},
						{RegexText: "", Regex: nil},
						{RegexText: "also.*valid", Regex: nil},
					},
				},
			},
		}

		err := lpc.CompileRegex()
		if err != nil {
			t.Fatalf("CompileRegex returned error: %v", err)
		}

		if lpc.MatchRules[0].Checks[0].Regex == nil {
			t.Fatal("First regex should be compiled")
		}
		if lpc.MatchRules[0].Checks[1].Regex != nil {
			t.Fatal("Empty regex should remain nil")
		}
		if lpc.MatchRules[0].Checks[2].Regex == nil {
			t.Fatal("Third regex should be compiled")
		}
	})

	t.Run("handles complex regex patterns", func(t *testing.T) {
		lpc := &Config{
			MatchRules: []*MatchRule{
				{
					Name: "complex",
					Checks: []LineCheck{
						{RegexText: `^(?P<level>ERROR|WARN|INFO):\s+(?P<msg>.+)$`, Regex: nil},
						{RegexText: `\b(?:\d{1,3}\.){3}\d{1,3}\b`, Regex: nil},
						{RegexText: `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`, Regex: nil},
					},
				},
			},
		}

		err := lpc.CompileRegex()
		if err != nil {
			t.Fatalf("CompileRegex returned error: %v", err)
		}

		for i, check := range lpc.MatchRules[0].Checks {
			if check.Regex == nil {
				t.Fatalf("Regex %d should be compiled", i)
			}
		}
	})
}

func TestLogParserConfig_RuleStructure(t *testing.T) {
	t.Run("handles rules with single check", func(t *testing.T) {
		lpc := &Config{
			MatchRules: []*MatchRule{
				{
					Name: "single",
					Checks: []LineCheck{
						{RegexText: "error", Regex: nil},
					},
				},
			},
		}

		err := lpc.CompileRegex()
		if err != nil {
			t.Fatalf("CompileRegex returned error: %v", err)
		}

		if len(lpc.MatchRules[0].Checks) != 1 {
			t.Fatalf("Expected 1 check, got %d", len(lpc.MatchRules[0].Checks))
		}
	})

	t.Run("handles rules with many checks", func(t *testing.T) {
		checks := make([]LineCheck, 10)
		for i := range 10 {
			checks[i] = LineCheck{RegexText: "pattern", Regex: nil}
		}

		lpc := &Config{
			MatchRules: []*MatchRule{
				{
					Name:   "many",
					Checks: checks,
				},
			},
		}

		err := lpc.CompileRegex()
		if err != nil {
			t.Fatalf("CompileRegex returned error: %v", err)
		}

		if len(lpc.MatchRules[0].Checks) != 10 {
			t.Fatalf("Expected 10 checks, got %d", len(lpc.MatchRules[0].Checks))
		}

		for i, check := range lpc.MatchRules[0].Checks {
			if check.Regex == nil {
				t.Fatalf("Check %d regex should be compiled", i)
			}
		}
	})

	t.Run("handles mixed empty and non-empty regex patterns", func(t *testing.T) {
		lpc := &Config{
			MatchRules: []*MatchRule{
				{
					Name: "mixed",
					Checks: []LineCheck{
						{RegexText: "", Regex: nil},
						{RegexText: "pattern", Regex: nil},
						{RegexText: "", Regex: nil},
						{RegexText: "another", Regex: nil},
					},
				},
			},
		}

		err := lpc.CompileRegex()
		if err != nil {
			t.Fatalf("CompileRegex returned error: %v", err)
		}

		if lpc.MatchRules[0].Checks[0].Regex != nil {
			t.Fatal("Empty regex at position 0 should be nil")
		}
		if lpc.MatchRules[0].Checks[1].Regex == nil {
			t.Fatal("Non-empty regex at position 1 should be compiled")
		}
		if lpc.MatchRules[0].Checks[2].Regex != nil {
			t.Fatal("Empty regex at position 2 should be nil")
		}
		if lpc.MatchRules[0].Checks[3].Regex == nil {
			t.Fatal("Non-empty regex at position 3 should be compiled")
		}
	})
}

func TestValidate(t *testing.T) {
	t.Run("returns nil for valid config with rules and checks", func(t *testing.T) {
		lpc := &Config{
			MatchRules: []*MatchRule{
				{
					Name: "error-rule",
					Checks: []LineCheck{
						{RegexText: "ERROR", Regex: nil},
						{RegexText: "FATAL", Regex: nil},
					},
					MaxLines: 5,
				},
				{
					Name: "warning-rule",
					Checks: []LineCheck{
						{RegexText: "WARN", Regex: nil},
					},
				},
			},
		}

		err := lpc.Validate()
		if err != nil {
			t.Fatalf("Validate returned error for valid config: %v", err)
		}

		// Verify regexes were compiled
		if lpc.MatchRules[0].Checks[0].Regex == nil {
			t.Fatal("First rule's first check regex should be compiled")
		}
		if lpc.MatchRules[1].Checks[0].Regex == nil {
			t.Fatal("Second rule's check regex should be compiled")
		}
	})

	t.Run("returns error for rule with 0 checks", func(t *testing.T) {
		lpc := &Config{
			MatchRules: []*MatchRule{
				{
					Name:   "empty-rule",
					Checks: []LineCheck{},
				},
			},
		}

		err := lpc.Validate()
		if err == nil {
			t.Fatal("Expected error for rule with 0 checks, got nil")
		}
	})

	t.Run("returns error for invalid regex in rules", func(t *testing.T) {
		lpc := &Config{
			MatchRules: []*MatchRule{
				{
					Name: "invalid-rule",
					Checks: []LineCheck{
						{RegexText: "[unclosed", Regex: nil},
					},
				},
			},
		}

		err := lpc.Validate()
		if err == nil {
			t.Fatal("Expected error for invalid regex, got nil")
		}
	})

	t.Run("returns error when multiple rules and one has empty checks", func(t *testing.T) {
		lpc := &Config{
			MatchRules: []*MatchRule{
				{
					Name: "valid-rule",
					Checks: []LineCheck{
						{RegexText: "ERROR", Regex: nil},
					},
				},
				{
					Name:   "empty-rule",
					Checks: []LineCheck{},
				},
			},
		}

		err := lpc.Validate()
		if err == nil {
			t.Fatal("Expected error for second rule with 0 checks, got nil")
		}
	})

	t.Run("returns error for rule with checks but no maxlines set", func(t *testing.T) {
		lpc := &Config{
			MatchRules: []*MatchRule{
				{
					Name:   "testing rule",
					Checks: []LineCheck{{}, {}}, // Just some empty checks to trigger the error
				},
			},
		}

		err := lpc.Validate()
		// Check error message contains expected text
		if err == nil {
			t.Fatal("Expected error for rule with checks but no maxlines set, got nil")
		}
		expectedErrMsg := "rule 0 (testing rule) has multiple checks but maxlines is not set"
		if err.Error() != expectedErrMsg {
			t.Fatalf("Expected error message %q, got %q", expectedErrMsg, err.Error())
		}
	})
}

func TestApplyDefaults(t *testing.T) {
	t.Run("applies default values when fields are zero", func(t *testing.T) {
		cfg := &Config{}

		err := cfg.ApplyDefaults()
		if err != nil {
			t.Fatalf("ApplyDefaults returned error: %v", err)
		}

		if cfg.Output != "text" {
			t.Fatalf("Expected default Output to be 'text', got %q", cfg.Output)
		}
		if cfg.MaxMatches != 1 {
			t.Fatalf("Expected default MaxMatches to be 1, got %d", cfg.MaxMatches)
		}
		if cfg.MaxLineSizeKB != 4 {
			t.Fatalf("Expected default MaxLineSizeKB to be 4, got %d", cfg.MaxLineSizeKB)
		}
	})

	t.Run("preserves non-zero values and compiles regex", func(t *testing.T) {
		cfg := &Config{
			MatchRules: []*MatchRule{
				{
					Name: "test",
					Checks: []LineCheck{
						{RegexText: "pattern", Regex: nil},
					},
				},
			},
			Output:        "json",
			MaxMatches:    5,
			MaxLineSizeKB: 10,
		}

		err := cfg.ApplyDefaults()
		if err != nil {
			t.Fatalf("ApplyDefaults returned error: %v", err)
		}

		if cfg.Output != "json" {
			t.Fatalf("Expected Output to be 'json', got %q", cfg.Output)
		}
		if cfg.MaxMatches != 5 {
			t.Fatalf("Expected MaxMatches to be 5, got %d", cfg.MaxMatches)
		}
		if cfg.MaxLineSizeKB != 10 {
			t.Fatalf("Expected MaxLineSizeKB to be 10, got %d", cfg.MaxLineSizeKB)
		}
		if cfg.MatchRules[0].Checks[0].Regex == nil {
			t.Fatal("Expected regex to be compiled, got nil")
		}
	})

	t.Run("handles nil Config gracefully", func(t *testing.T) {
		var cfg *Config = nil
		err := cfg.ApplyDefaults()
		if err != nil {
			t.Fatalf("ApplyDefaults returned error for nil Config: %v", err)
		}
	})
}
