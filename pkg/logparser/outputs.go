package logparser

import (
	"encoding/json"
	"fmt"
	"io"
)

// TextOutput writes a human-readable summary of matches to the provided writer.
func TextOutput(w io.Writer, matches []*ParseMatch) {
	if len(matches) == 0 {
		_, _ = fmt.Fprintln(w, "No matches found")
		return
	}
	_, _ = fmt.Fprintln(w, "Matches:")

	for _, m := range matches {
		_, _ = fmt.Fprintf(w, "Matched rule: %v \n", m.Rule.Name)
		if m.Rule.Category != "" {
			_, _ = fmt.Fprintf(w, "Category: %v \n", m.Rule.Category)
		}
		_, _ = fmt.Fprintf(w, "Solution: \n%v\n\n", m.Rule.Solution)
		_, _ = fmt.Fprintf(w, "Log extract: \n\n")
		for _, l := range m.MatchedLines {
			_, _ = fmt.Fprintf(w, "%v: %v \n", l.LineNumber, l.Content)
		}
	}
}

// JSONOutput writes a JSON array of matches to the provided writer.
func JSONOutput(w io.Writer, matches []*ParseMatch) {
	jsonBytes, err := json.MarshalIndent(matches, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(w, "Error marshalling matches to JSON: %v", err)
		return
	}
	_, _ = w.Write(jsonBytes)
}
