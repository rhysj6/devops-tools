package pfp

import (
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
