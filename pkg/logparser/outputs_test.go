package logparser

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestTextOutput(t *testing.T) {

	exampleParseMatch := &ParseMatch{
		Rule: &Rule{
			Name:     "Test rule",
			Solution: "Test solution",
		},
		MatchedLines: []*LogLine{
			{LineNumber: 1, Content: "Test log"},
		},
	}

	tests := []struct {
		name     string
		matches  []*ParseMatch
		stats    Stats
		expected []string
	}{
		{
			name:     "No matches",
			matches:  []*ParseMatch{},
			stats:    Stats{},
			expected: []string{"No matches found"},
		},
		{
			name:    "Matches contain expected fields",
			matches: []*ParseMatch{exampleParseMatch},
			stats:   Stats{},
			expected: []string{
				"Matches:",
				"Matched rule: Test rule",
				"Solution: \nTest solution",
				"1: Test log",
			},
		},
		{
			name:    "Stats display correctly",
			matches: []*ParseMatch{},
			stats: Stats{
				LinesParsed:     1234,
				Duration:        time.Second,
				PartialMatches:  123,
				CompleteMatches: 12,
			},
			expected: []string{
				"Duration:             1s",
				"Partial Matches:      123",
				"Complete Matches:     12",
			},
		},
		{
			name: "Category is displayed when set",
			matches: []*ParseMatch{
				{
					Rule: &Rule{
						Name:     "Test rule",
						Category: "Test category",
						Solution: "Test solution",
					},
					MatchedLines: []*LogLine{
						{LineNumber: 1, Content: "Test log"},
					},
				},
			},
			stats: Stats{
				LinesParsed:     1234,
				Duration:        time.Second,
				PartialMatches:  123,
				CompleteMatches: 12,
			},
			expected: []string{
				"Duration:             1s",
				"Partial Matches:      123",
				"Complete Matches:     12",
				"Category: Test category",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			TextOutput(&buf, tt.matches, tt.stats)
			res := buf.String()
			for _, e := range tt.expected {
				if !strings.Contains(res, e) {
					t.Fatalf("Output doesn't contain '%v' got: %v", e, res)
				}
			}
		})
	}
}
