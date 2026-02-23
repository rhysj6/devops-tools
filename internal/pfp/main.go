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

type LogSource interface {
	// Get the logs to be parsed. This should return an io.ReadCloser that can be used to read the logs line by line.
	GetLogs() (io.ReadCloser, error)
}

type RecursiveLogSource interface {
	LogSource
	// GetDownstreamFailedBuildRule returns the rule that should match a log line to be considered the results of a failed downstream build.
	GetDownstreamFailedBuildRule() *Rule
	// GetDownstreamFailedBuildLogs returns the logs of a downstream failed build given a ParseMatch that contains the rule that matched the failed downstream build and the log lines that matched that rule.
	GetDownstreamFailedBuildLogs(*ParseMatch) (io.ReadCloser, error)
	// GetMaxRecursionDepth returns the maximum recursion depth for parsing downstream failed builds. This is to prevent infinite recursion in case of circular dependencies between builds. If not implemented, it defaults to 3, meaning that only one level of downstream failed builds will be parsed.
	GetMaxRecursionDepth() int
}

func ParseFromSource(source LogSource, rules []*Rule, maxMatches int) ([]*ParseMatch, Stats, error) {
	logs, err := source.GetLogs()
	if err != nil {
		return nil, Stats{}, fmt.Errorf("failed to get logs from source: %w", err)
	}
	defer logs.Close()
	if !source.SupportDownstreamFailedBuilds() {
		return Parse(logs, rules, maxMatches)
	}
	// TODO: Add support for downstream failed builds.
	return Parse(logs, rules, maxMatches)
}

func Parse(r io.ReadCloser, rules []*Rule, maxMatches int) ([]*ParseMatch, Stats, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer r.Close()
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
