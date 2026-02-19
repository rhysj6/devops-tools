package pfp

import (
	"fmt"
	"io"
)

func TextOutput(w io.Writer, matches []*ParseMatch, stats Stats) {
	fmt.Println(w, "Stats:")
	fmt.Fprintf(w, "Lines parsed:         %v \n", stats.LinesParsed)
	fmt.Fprintf(w, "Duration:             %v \n", stats.Duration)
	fmt.Fprintf(w, "Partial Matches:      %v \n", stats.PartialMatches)
	fmt.Fprintf(w, "Complete Matches:     %v \n", stats.CompleteMatches)

	if len(matches) == 0 {
		fmt.Fprintln(w, "No matches found")
		return
	}
	fmt.Fprintln(w, "Matches:")

	for _, m := range matches {
		fmt.Fprintf(w, "Matched rule: %v \n", m.Rule.Name)
		fmt.Fprintf(w, "Solution: \n%v\n\n", m.Rule.Solution)
		fmt.Fprintf(w, "Log excerpt: \n\n")
		for _, l := range m.MatchedLines {
			fmt.Fprintf(w, "%v: %v \n", l.LineNumber, l.Content)
		}
	}
}
