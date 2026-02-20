package config

import (
	"testing"

	"github.com/rhysj6/devops-tools/internal/pfp"
)

func TestCompileRegex(t *testing.T) {
	t.Run("compiles valid regex patterns", func(t *testing.T) {
		lpc := &LogParserConfig{
			Rules: []*pfp.Rule{
				{
					Name: "test",
					Checks: []pfp.LineMatcher{
						{RegexText: "test.*pattern", Regex: nil},
					},
				},
			},
		}

		err := lpc.CompileRegex()
		if err != nil {
			t.Fatalf("CompileRegex returned error: %v", err)
		}

		if lpc.Rules[0].Checks[0].Regex == nil {
			t.Fatal("Expected regex to be compiled, got nil")
		}
	})

	t.Run("handles empty regex text", func(t *testing.T) {
		lpc := &LogParserConfig{
			Rules: []*pfp.Rule{
				{
					Name: "test",
					Checks: []pfp.LineMatcher{
						{RegexText: "", Regex: nil},
					},
				},
			},
		}

		err := lpc.CompileRegex()
		if err != nil {
			t.Fatalf("CompileRegex returned error: %v", err)
		}

		if lpc.Rules[0].Checks[0].Regex != nil {
			t.Fatalf("Expected regex to be nil for empty text, got %v", lpc.Rules[0].Checks[0].Regex)
		}
	})

	t.Run("returns error for invalid regex", func(t *testing.T) {
		lpc := &LogParserConfig{
			Rules: []*pfp.Rule{
				{
					Name: "test",
					Checks: []pfp.LineMatcher{
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
		lpc := &LogParserConfig{
			Rules: []*pfp.Rule{
				{
					Name: "rule1",
					Checks: []pfp.LineMatcher{
						{RegexText: "error.*", Regex: nil},
						{RegexText: "failed", Regex: nil},
					},
				},
				{
					Name: "rule2",
					Checks: []pfp.LineMatcher{
						{RegexText: "warning", Regex: nil},
					},
				},
			},
		}

		err := lpc.CompileRegex()
		if err != nil {
			t.Fatalf("CompileRegex returned error: %v", err)
		}

		if lpc.Rules[0].Checks[0].Regex == nil || lpc.Rules[0].Checks[1].Regex == nil {
			t.Fatal("Expected regexes in rule1 to be compiled")
		}
		if lpc.Rules[1].Checks[0].Regex == nil {
			t.Fatal("Expected regex in rule2 to be compiled")
		}
	})
}

func TestLogParserConfig_Output(t *testing.T) {
	t.Run("preserves custom output format", func(t *testing.T) {
		cfg := &Config{
			Pfp: &LogParserConfig{
				Rules:      []*pfp.Rule{},
				Output:     "json",
				MaxMatches: 10,
			},
		}

		err := cfg.SetupConfig()
		if err != nil {
			t.Fatalf("SetupConfig returned error: %v", err)
		}

		if cfg.Pfp.Output != "json" {
			t.Fatalf("Output = %q, want %q", cfg.Pfp.Output, "json")
		}
	})
}

func TestLogParserConfig_MaxMatches(t *testing.T) {
	t.Run("preserves custom MaxMatches value", func(t *testing.T) {
		lpc := &LogParserConfig{
			MaxMatches: 42,
		}

		if lpc.MaxMatches != 42 {
			t.Fatalf("MaxMatches = %d, want 42", lpc.MaxMatches)
		}
	})
}

func TestCompileRegex_EdgeCases(t *testing.T) {
	t.Run("skips compilation for empty regex text", func(t *testing.T) {
		lpc := &LogParserConfig{
			Rules: []*pfp.Rule{
				{
					Name: "mixed",
					Checks: []pfp.LineMatcher{
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

		if lpc.Rules[0].Checks[0].Regex == nil {
			t.Fatal("First regex should be compiled")
		}
		if lpc.Rules[0].Checks[1].Regex != nil {
			t.Fatal("Empty regex should remain nil")
		}
		if lpc.Rules[0].Checks[2].Regex == nil {
			t.Fatal("Third regex should be compiled")
		}
	})

	t.Run("handles complex regex patterns", func(t *testing.T) {
		lpc := &LogParserConfig{
			Rules: []*pfp.Rule{
				{
					Name: "complex",
					Checks: []pfp.LineMatcher{
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

		for i, check := range lpc.Rules[0].Checks {
			if check.Regex == nil {
				t.Fatalf("Regex %d should be compiled", i)
			}
		}
	})
}

func TestLogParserConfig_RuleStructure(t *testing.T) {
	t.Run("handles rules with single check", func(t *testing.T) {
		lpc := &LogParserConfig{
			Rules: []*pfp.Rule{
				{
					Name: "single",
					Checks: []pfp.LineMatcher{
						{RegexText: "error", Regex: nil},
					},
				},
			},
		}

		err := lpc.CompileRegex()
		if err != nil {
			t.Fatalf("CompileRegex returned error: %v", err)
		}

		if len(lpc.Rules[0].Checks) != 1 {
			t.Fatalf("Expected 1 check, got %d", len(lpc.Rules[0].Checks))
		}
	})

	t.Run("handles rules with many checks", func(t *testing.T) {
		checks := make([]pfp.LineMatcher, 10)
		for i := range 10 {
			checks[i] = pfp.LineMatcher{RegexText: "pattern", Regex: nil}
		}

		lpc := &LogParserConfig{
			Rules: []*pfp.Rule{
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

		if len(lpc.Rules[0].Checks) != 10 {
			t.Fatalf("Expected 10 checks, got %d", len(lpc.Rules[0].Checks))
		}

		for i, check := range lpc.Rules[0].Checks {
			if check.Regex == nil {
				t.Fatalf("Check %d regex should be compiled", i)
			}
		}
	})

	t.Run("handles mixed empty and non-empty regex patterns", func(t *testing.T) {
		lpc := &LogParserConfig{
			Rules: []*pfp.Rule{
				{
					Name: "mixed",
					Checks: []pfp.LineMatcher{
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

		if lpc.Rules[0].Checks[0].Regex != nil {
			t.Fatal("Empty regex at position 0 should be nil")
		}
		if lpc.Rules[0].Checks[1].Regex == nil {
			t.Fatal("Non-empty regex at position 1 should be compiled")
		}
		if lpc.Rules[0].Checks[2].Regex != nil {
			t.Fatal("Empty regex at position 2 should be nil")
		}
		if lpc.Rules[0].Checks[3].Regex == nil {
			t.Fatal("Non-empty regex at position 3 should be compiled")
		}
	})
}

func TestValidate(t *testing.T) {
	t.Run("returns nil for valid config with rules and checks", func(t *testing.T) {
		lpc := &LogParserConfig{
			Rules: []*pfp.Rule{
				{
					Name: "error-rule",
					Checks: []pfp.LineMatcher{
						{RegexText: "ERROR", Regex: nil},
						{RegexText: "FATAL", Regex: nil},
					},
				},
				{
					Name: "warning-rule",
					Checks: []pfp.LineMatcher{
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
		if lpc.Rules[0].Checks[0].Regex == nil {
			t.Fatal("First rule's first check regex should be compiled")
		}
		if lpc.Rules[1].Checks[0].Regex == nil {
			t.Fatal("Second rule's check regex should be compiled")
		}
	})

	t.Run("returns error for rule with 0 checks", func(t *testing.T) {
		lpc := &LogParserConfig{
			Rules: []*pfp.Rule{
				{
					Name:   "empty-rule",
					Checks: []pfp.LineMatcher{},
				},
			},
		}

		err := lpc.Validate()
		if err == nil {
			t.Fatal("Expected error for rule with 0 checks, got nil")
		}
	})

	t.Run("returns error for invalid regex in rules", func(t *testing.T) {
		lpc := &LogParserConfig{
			Rules: []*pfp.Rule{
				{
					Name: "invalid-rule",
					Checks: []pfp.LineMatcher{
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
		lpc := &LogParserConfig{
			Rules: []*pfp.Rule{
				{
					Name: "valid-rule",
					Checks: []pfp.LineMatcher{
						{RegexText: "ERROR", Regex: nil},
					},
				},
				{
					Name:   "empty-rule",
					Checks: []pfp.LineMatcher{},
				},
			},
		}

		err := lpc.Validate()
		if err == nil {
			t.Fatal("Expected error for second rule with 0 checks, got nil")
		}
	})
}
