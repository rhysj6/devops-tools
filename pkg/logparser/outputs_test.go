package logparser

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestTextOutput(t *testing.T) {

	exampleParseMatch := &ParseMatch{
		Rule: &MatchRule{
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
					Rule: &MatchRule{
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

func TestJSONOutput(t *testing.T) {
	tests := []struct {
		name    string
		matches []*ParseMatch
		assert  func(t *testing.T, got []byte)
	}{
		{
			name:    "No matches",
			matches: []*ParseMatch{},
			assert: func(t *testing.T, got []byte) {
				t.Helper()
				if strings.TrimSpace(string(got)) != "[]" {
					t.Fatalf("expected empty JSON array, got: %s", string(got))
				}
			},
		},
		{
			name: "Single match includes expected fields",
			matches: []*ParseMatch{
				{
					Rule: &MatchRule{
						Name:     "Rule A",
						Category: "Build",
						Solution: "Do thing",
					},
					MatchedLines: []*LogLine{
						{LineNumber: 12, Content: "error line"},
					},
				},
			},
			assert: func(t *testing.T, got []byte) {
				t.Helper()
				var payload []map[string]any
				if err := json.Unmarshal(got, &payload); err != nil {
					t.Fatalf("expected valid JSON output, got error: %v", err)
				}
				if len(payload) != 1 {
					t.Fatalf("expected 1 match, got %d", len(payload))
				}

				rule, ok := payload[0]["rule"].(map[string]any)
				if !ok {
					t.Fatalf("expected Rule object in payload, got: %#v", payload[0]["rule"])
				}
				if rule["name"] != "Rule A" {
					t.Fatalf("expected Rule.Name to be Rule A, got: %#v", rule["name"])
				}
				if rule["category"] != "Build" {
					t.Fatalf("expected Rule.Category to be Build, got: %#v", rule["category"])
				}

				lines, ok := payload[0]["matchedLines"].([]any)
				if !ok || len(lines) != 1 {
					t.Fatalf("expected one matched line, got: %#v", payload[0]["matchedLines"])
				}
				line, ok := lines[0].(map[string]any)
				if !ok {
					t.Fatalf("expected matched line object, got: %#v", lines[0])
				}
				if line["content"] != "error line" {
					t.Fatalf("expected matched line content to be error line, got: %#v", line["content"])
				}
				if line["lineNumber"] != float64(12) {
					t.Fatalf("expected matched line number to be 12, got: %#v", line["lineNumber"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			JSONOutput(&buf, tt.matches)
			tt.assert(t, buf.Bytes())
		})
	}
}
