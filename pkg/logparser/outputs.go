package logparser

import (
	"fmt"
	"io"
)

func TextOutput(w io.Writer, matches []*ParseMatch, stats Stats) {
	_, _ = fmt.Fprintln(w, "Stats:")
	_, _ = fmt.Fprintf(w, "Lines parsed:         %v \n", stats.LinesParsed)
	_, _ = fmt.Fprintf(w, "Duration:             %v \n", stats.Duration)
	_, _ = fmt.Fprintf(w, "Partial Matches:      %v \n", stats.PartialMatches)
	_, _ = fmt.Fprintf(w, "Complete Matches:     %v \n", stats.CompleteMatches)

	if len(matches) == 0 {
		_, _ = fmt.Fprintln(w, "No matches found")
		return
	}
	_, _ = fmt.Fprintln(w, "Matches:")

	for _, m := range matches {
		_, _ = fmt.Fprintf(w, "Matched rule: %v \n", m.Rule.Name)
		_, _ = fmt.Fprintf(w, "Solution: \n%v\n\n", m.Rule.Solution)
		_, _ = fmt.Fprintf(w, "Log extract: \n\n")
		for _, l := range m.MatchedLines {
			_, _ = fmt.Fprintf(w, "%v: %v \n", l.LineNumber, l.Content)
		}
	}
}
