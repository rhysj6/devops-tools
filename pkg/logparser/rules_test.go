package logparser

import (
	"regexp"
	"testing"
)

func TestRuleGetNeededLineCount(t *testing.T) {
	tests := []struct {
		name     string
		rule     Rule
		expected int
	}{
		{
			name:     "Handles single check",
			rule:     Rule{Checks: []LineMatcher{{Contains: "Hello?"}}, MaxLines: 100},
			expected: 1,
		},
		{
			name:     "Handles more checks than limit",
			rule:     Rule{Checks: []LineMatcher{{}}, MaxLines: 0},
			expected: 1,
		},
		{
			name:     "Handles max lines",
			rule:     Rule{Checks: []LineMatcher{{Contains: "Something"}, {Contains: "Something"}}, MaxLines: 17},
			expected: 17,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.rule.getNeededLineCount()

			if r != tt.expected {
				t.Fatalf("Expected %v lines got %v", tt.expected, r)
			}
		})
	}

}

func TestCheckLine(t *testing.T) {
	re1, err := regexp.Compile(`ERROR:`)
	if err != nil {
		t.Fatal("Error compiling regex")
	}
	re2, err := regexp.Compile(`ERROR:Hi`)
	if err != nil {
		t.Fatal("Error compiling regex")
	}

	tests := []struct {
		name     string
		lm       LineMatcher
		expected bool
	}{
		{
			name:     "Handle match contains with no regex",
			lm:       LineMatcher{Contains: "ERROR"},
			expected: true,
		},
		{
			name:     "Handle no match contains with no regex",
			lm:       LineMatcher{Contains: "INFO"},
			expected: false,
		},
		{
			name:     "Handle match contains with regex",
			lm:       LineMatcher{Contains: "ERROR", Regex: re1},
			expected: true,
		},
		{
			name:     "Handle no match contains with regex",
			lm:       LineMatcher{Contains: "ERROR", Regex: re2},
			expected: false,
		},
		{
			name:     "Handle match regex only",
			lm:       LineMatcher{Regex: re1},
			expected: true,
		},
		{
			name:     "Handle no match regex only",
			lm:       LineMatcher{Regex: re2},
			expected: false,
		},
		{
			name:     "Handle no values ",
			lm:       LineMatcher{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line := "ERROR: Something failed"

			r := tt.lm.CheckLine(line)

			if r != tt.expected {
				t.Fatalf("Expected %t but got %t", tt.expected, r)
			}
		})
	}
}
