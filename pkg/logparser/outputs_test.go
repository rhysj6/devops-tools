package logparser

import (
	"bytes"
	"strings"
	"testing"
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
		expected []string
	}{
		{
			name:     "No matches",
			matches:  []*ParseMatch{},
			expected: []string{"No matches found"},
		},
		{
			name:    "Matches contain expected fields",
			matches: []*ParseMatch{exampleParseMatch},
			expected: []string{
				"Matches:",
				"Matched rule: Test rule",
				"Solution: \nTest solution",
				"1: Test log",
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
			expected: []string{
				"Matches:",
				"Matched rule: Test rule",
				"Category: Test category",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			TextOutput(&buf, tt.matches)
			res := buf.String()
			for _, e := range tt.expected {
				if !strings.Contains(res, e) {
					t.Fatalf("Output doesn't contain '%v' got: %v", e, res)
				}
			}
		})
	}
}
