package pfp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"time"
)

type Stats struct {
	TotalLines  int           `json:"total_lines"`
	LinesParsed int           `json:"lines_parsed"`
	Duration    time.Duration `json:"duration"`

	PartialMatches  int `json:"partial_matches"`
	CompleteMatches int `json:"complete_matches"`
}

func Parse(r io.Reader, rules []*Rule, maxMatches int) ([]*ParseMatch, Stats, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stats := Stats{
		PartialMatches: 0,
	}
	startTime := time.Now()
	scanner := bufio.NewScanner(r)

	activeMatchers := []*Matcher{}
	matches := []*ParseMatch{}

	matchChan := make(chan *ParseMatch, 100)

	lineNo := 1
	for scanner.Scan() {
		lineNo++
		line := &LogLine{
			Content:    scanner.Text(),
			LineNumber: lineNo,
		}

		if err := scanner.Err(); err != nil {
			return nil, stats, fmt.Errorf("Scanner error: %w \n Log line number %v:", err, lineNo)
		}

		activeMatchers = PurgeInactiveMatchers(lineNo, activeMatchers)
		BroadcastLogLine(line, activeMatchers)

		pendingMatchers := InitialCheckLine(line, rules)
		for _, m := range pendingMatchers {
			go RunMatcher(m, matchChan, ctx)
		}

		stats.PartialMatches = stats.PartialMatches + len(pendingMatchers)

		activeMatchers = append(activeMatchers, pendingMatchers...)
		newMatches := GetNewParseMatches(matchChan)
		if len(newMatches) > 0 {
			matches = append(matches, newMatches...)
			if len(matches) > maxMatches {
				cancel()
				break
			}
		}
	}

	// Wait for any remaining matchers to finish
	for _, m := range activeMatchers {
		<-m.DoneChannel
	}
	matches = append(matches, GetNewParseMatches(matchChan)...)

	stats.Duration = time.Since(startTime)
	stats.LinesParsed = lineNo
	stats.CompleteMatches = len(matches)

	return matches, stats, nil
}

func TextOutput(matches []*ParseMatch, stats Stats) {
	fmt.Println("Stats:")
	fmt.Printf("Lines parsed:         %v \n", stats.LinesParsed)
	fmt.Printf("Duration:             %v \n", stats.Duration)
	fmt.Printf("Partial Matches:      %v \n", stats.PartialMatches)
	fmt.Printf("Complete Matches:     %v \n", stats.CompleteMatches)

	if len(matches) == 0 {
		fmt.Println("No matches found")
		return
	}
	fmt.Println("Matches:")

	for _, m := range matches {
		fmt.Printf("Matched rule: %v \n", m.Rule.Name)
		fmt.Printf("Solution: \n%v\n\n", m.Rule.Solution)
		fmt.Printf("Log excerpt: \n\n")
		for _, l := range m.MatchedLines {
			fmt.Printf("%v: %v \n", l.LineNumber, l.Content)
		}
	}
}
