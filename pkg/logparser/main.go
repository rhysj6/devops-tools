package logparser

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"
)

type Stats struct {
	LinesParsed int           `json:"lines_parsed"`
	Duration    time.Duration `json:"duration"`

	PartialMatches  int `json:"partial_matches"`
	CompleteMatches int `json:"complete_matches"`
}

func ParseFromSource(source LogSource, rules []*Rule, maxMatches int, logger *slog.Logger) ([]*ParseMatch, Stats, error) {
	startTime := time.Now()
	logs, err := source.GetLogs()
	if err != nil {
		return nil, Stats{}, fmt.Errorf("failed to get logs from source: %w", err)
	}
	recursiveSource, ok := source.(RecursiveLogSource) // If the source does not support downstream error logs, we can just parse the logs once and return the results.

	if ok {
		rules = append(rules, recursiveSource.GetDownstreamErrorRule())
	}
	matches, stats, err := Parse(logs, rules, maxMatches, logger)

	if !ok {
		stats.Duration = time.Since(startTime)
		return matches, stats, err
	}

	if err != nil {
		return nil, stats, fmt.Errorf("failed to parse logs: %w", err)
	}

	for range max(recursiveSource.GetMaxRecursionDepth(), 3) {
		if len(matches) == 1 && matches[0].Rule == recursiveSource.GetDownstreamErrorRule() {
			logger.Info("found potential downstream failure mention, attempting to get downstream logs", slog.String("content", matches[0].MatchedLines[0].Content))
			downstreamLogs, err := recursiveSource.GetDownstreamErrorLogs(matches[0])
			if err != nil {
				return nil, stats, fmt.Errorf("failed to get downstream error logs: %w", err)
			}
			downstreamMatches, downstreamStats, err := Parse(downstreamLogs, rules, maxMatches, logger)
			if err != nil {
				return nil, stats, fmt.Errorf("failed to parse downstream error logs: %w", err)
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
		if m.Rule != recursiveSource.GetDownstreamErrorRule() {
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

func Parse(r io.ReadCloser, rules []*Rule, maxMatches int, logger *slog.Logger) ([]*ParseMatch, Stats, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() { _ = r.Close() }()
	defer cancel()
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	stats := Stats{}
	reader := bufio.NewReader(r)

	activeMatchers := []*matcher{}
	matches := []*ParseMatch{}

	matchChan := make(chan *ParseMatch, 100)

	lineNo := 0
	for {
		lineNo++
		// Read the line, if the line is more than 4kb then discard it.
		lineBytes, err := reader.ReadSlice('\n')

		if err == io.EOF && len(lineBytes) == 0 {
			lineNo-- // Don't count the EOF as a line
			break
		} else if err == bufio.ErrBufferFull {
			logger.Debug("skipping line as it's more than 4kb", slog.Int("line_number", lineNo))
			// Discard the rest of the line
			_, err := reader.ReadString('\n')
			if err != nil && err != io.EOF && err != bufio.ErrBufferFull {
				return nil, stats, fmt.Errorf("reader error: %w \n Log line number: %v", err, lineNo)
			}
			continue
		} else if err != nil && err != io.EOF {
			return nil, stats, fmt.Errorf("reader error: %w \n Log line number: %v", err, lineNo)
		}

		line := &LogLine{
			Content:    strings.TrimRight(string(lineBytes), "\n"),
			LineNumber: lineNo,
		}

		activeMatchers = purgeInactiveMatchers(lineNo, activeMatchers)
		broadcastLogLine(line, activeMatchers)

		pendingMatchers := initialCheckLine(line, rules)
		for _, m := range pendingMatchers {
			go runMatcher(ctx, m, matchChan)
		}

		stats.PartialMatches = stats.PartialMatches + len(pendingMatchers)

		activeMatchers = append(activeMatchers, pendingMatchers...)
		newMatches := getNewParseMatches(matchChan)
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
	matches = append(matches, getNewParseMatches(matchChan)...)

	stats.LinesParsed = lineNo
	stats.CompleteMatches = len(matches)

	return matches, stats, nil
}
