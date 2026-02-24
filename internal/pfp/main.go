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
	startTime := time.Now()
	logs, err := source.GetLogs()
	if err != nil {
		return nil, Stats{}, fmt.Errorf("failed to get logs from source: %w", err)
	}
	recursiveSource, ok := source.(RecursiveLogSource) // If the source does not support downstream failed builds, we can just parse the logs once and return the results.

	if ok {
		rules = append(rules, recursiveSource.GetDownstreamFailedBuildRule())
	}
	matches, stats, err := Parse(logs, rules, maxMatches)

	if !ok {
		stats.Duration = time.Since(startTime)
		return matches, stats, err
	}

	if err != nil {
		return nil, stats, fmt.Errorf("failed to parse logs: %w", err)
	}

	for range max(recursiveSource.GetMaxRecursionDepth(), 3) {
		if len(matches) == 1 && matches[0].Rule == recursiveSource.GetDownstreamFailedBuildRule() {
			downstreamLogs, err := recursiveSource.GetDownstreamFailedBuildLogs(matches[0])
			if err != nil {
				return nil, stats, fmt.Errorf("failed to get downstream failed build logs: %w", err)
			}
			downstreamMatches, downstreamStats, err := Parse(downstreamLogs, rules, maxMatches)
			if err != nil {
				return nil, stats, fmt.Errorf("failed to parse downstream failed build logs: %w", err)
			}
			matches = append(matches, downstreamMatches...)
			stats.PartialMatches += downstreamStats.PartialMatches
			stats.CompleteMatches += downstreamStats.CompleteMatches
			stats.LinesParsed += downstreamStats.LinesParsed
		} else {
			break
		}
	}

	customMatches := []*ParseMatch{}
	for _, m := range matches {
		if m.Rule != recursiveSource.GetDownstreamFailedBuildRule() {
			customMatches = append(customMatches, m)
		}
	}

	stats.Duration = time.Since(startTime)

	// Filter out the downstream failure mentions if there are recognised matches in the final logs.
	if len(customMatches) > 0 {
		stats.CompleteMatches = len(customMatches)
		return customMatches, stats, nil
	}

	return matches, stats, nil
}

func Parse(r io.ReadCloser, rules []*Rule, maxMatches int) ([]*ParseMatch, Stats, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer r.Close()
	defer cancel()
	stats := Stats{}
	scanner := bufio.NewScanner(r)

	activeMatchers := []*Matcher{}
	matches := []*ParseMatch{}

	matchChan := make(chan *ParseMatch, 100)

	lineNo := 0
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

	stats.LinesParsed = lineNo
	stats.CompleteMatches = len(matches)

	return matches, stats, nil
}
