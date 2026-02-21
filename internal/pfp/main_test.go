package pfp

import (
	"regexp"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	t.Run("calculates stats correctly", func(t *testing.T) {
		input := "line1\nline2\nline3\n"
		reader := strings.NewReader(input)

		_, stats, err := Parse(reader, []*Rule{}, 10)
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		if stats.LinesParsed != 4 {
			t.Fatalf("LinesParsed = %d, want 4", stats.LinesParsed)
		}
		if stats.Duration == 0 {
			t.Fatal("Duration should be non-zero")
		}
	})

	t.Run("returns no matches when rules are empty", func(t *testing.T) {
		input := "line1\nline2\n"
		reader := strings.NewReader(input)

		matches, _, err := Parse(reader, []*Rule{}, 10)
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		if len(matches) != 0 {
			t.Fatalf("Expected 0 matches, got %d", len(matches))
		}
	})

	t.Run("respects maxMatches as soft limit", func(t *testing.T) {
		// maxMatches is a soft limit: when exceeded, parsing stops but active
		// goroutines are allowed to finish, so final count may exceed maxMatches
		// instead we check that not too many lines were parsed
		matching := strings.Repeat("match line \n", 3)
		notMatching := strings.Repeat("test line\n", 100)

		reader := strings.NewReader(matching + notMatching)

		rule := &Rule{
			Checks: []LineMatcher{
				{RegexText: "match", Regex: regexp.MustCompile("match")},
			},
		}

		matches, stats, err := Parse(reader, []*Rule{rule}, 2)
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		if stats.LinesParsed >= 103 {
			t.Fatalf("Expected early exit, parsed %d lines", stats.LinesParsed)
		}

		if len(matches) == 0 {
			t.Fatal("Expected matches to be collected")
		}
	})

	t.Run("handles reader with no newlines", func(t *testing.T) {
		input := "single line"
		reader := strings.NewReader(input)

		_, stats, err := Parse(reader, []*Rule{}, 10)
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		if stats.LinesParsed < 1 {
			t.Fatalf("LinesParsed = %d, want at least 1", stats.LinesParsed)
		}
	})

	t.Run("handles empty reader", func(t *testing.T) {
		reader := strings.NewReader("")

		matches, stats, err := Parse(reader, []*Rule{}, 10)
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		if len(matches) != 0 {
			t.Fatalf("Expected 0 matches, got %d", len(matches))
		}
		if stats.LinesParsed != 1 {
			t.Fatalf("LinesParsed = %d, want 1", stats.LinesParsed)
		}
	})

	t.Run("measures duration", func(t *testing.T) {
		input := strings.Repeat("line\n", 10)
		reader := strings.NewReader(input)

		_, stats, err := Parse(reader, []*Rule{}, 100)
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		if stats.Duration <= 0 {
			t.Fatalf("Duration should be positive, got %v", stats.Duration)
		}
	})
}
