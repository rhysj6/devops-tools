package pfp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"time"
)

type Stats struct {
	LinesParsed int           `json:"lines_parsed"`
	Duration    time.Duration `json:"duration"`

	PartialMatches  int `json:"partial_matches"`
	CompleteMatches int `json:"complete_matches"`
}

func Parse(r io.Reader, rules []*Rule, maxMatches int) ([]*ParseMatch, Stats, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stats := Stats{}
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
			go RunMatcher(ctx, m, matchChan)
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
