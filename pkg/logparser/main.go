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

// Stats reports aggregate metrics from a parsing run.
type Stats struct {
	LinesParsed int           `json:"lines_parsed"`
	Duration    time.Duration `json:"duration"`

	PartialMatches  int `json:"partial_matches"`
	CompleteMatches int `json:"complete_matches"`
}

type LogParser struct {
	Rules      []*Rule
	MaxMatches int
	logger     *slog.Logger
	ctx        context.Context
}

type ParserOption func(*LogParser)

func WithRules(rules []*Rule) ParserOption {
	return func(lp *LogParser) {
		lp.Rules = rules
	}
}

func WithMaxMatches(max int) ParserOption {
	return func(lp *LogParser) {
		lp.MaxMatches = max
	}
}

func WithLogger(logger *slog.Logger) ParserOption {
	return func(lp *LogParser) {
		lp.logger = logger
	}
}

func WithContext(ctx context.Context) ParserOption {
	return func(lp *LogParser) {
		lp.ctx = ctx
	}
}

func NewLogParser(opts ...ParserOption) *LogParser {
	lp := &LogParser{
		Rules:      []*Rule{},
		MaxMatches: 1,
	}
	for _, opt := range opts {
		opt(lp)
	}

	if lp.logger == nil {
		lp.logger = slog.New(slog.DiscardHandler)
	}

	if lp.ctx == nil {
		lp.ctx = context.Background()
	}

	return lp
}

// ParseFromSource parses logs from a LogSource and applies optional recursive
// parsing when the source implements RecursiveLogSource.
func (lp *LogParser) ParseFromSource(source LogSource) ([]*ParseMatch, Stats, error) {
	startTime := time.Now()
	logs, err := source.GetLogs()
	if err != nil {
		return nil, Stats{}, fmt.Errorf("failed to get logs from source: %w", err)
	}

	recursiveSource, ok := source.(RecursiveLogSource) // If the source does not support downstream error logs, we can just parse the logs once and return the results.

	if ok {
		lp.Rules = append(lp.Rules, recursiveSource.GetDownstreamErrorRule())
	}
	matches, stats, err := lp.Parse(logs)

	if !ok {
		stats.Duration = time.Since(startTime)
		return matches, stats, err
	}

	if err != nil {
		return nil, stats, fmt.Errorf("failed to parse logs: %w", err)
	}

	for range max(recursiveSource.GetMaxRecursionDepth(), 3) {
		if len(matches) == 1 && matches[0].Rule == recursiveSource.GetDownstreamErrorRule() {
			lp.logger.Info("found potential downstream failure mention, attempting to get downstream logs", slog.String("content", matches[0].MatchedLines[0].Content))
			downstreamLogs, err := recursiveSource.GetDownstreamErrorLogs(matches[0])
			if err != nil {
				return nil, stats, fmt.Errorf("failed to get downstream error logs: %w", err)
			}
			downstreamMatches, downstreamStats, err := lp.Parse(downstreamLogs)
			if err != nil {
				return nil, stats, fmt.Errorf("failed to parse downstream error logs: %w", err)
			}
			stats.PartialMatches += downstreamStats.PartialMatches
			stats.CompleteMatches += downstreamStats.CompleteMatches
			stats.LinesParsed += downstreamStats.LinesParsed
			if len(downstreamMatches) == 0 {
				lp.logger.Info("no further matches found in downstream logs")
				break
			}
			matches = append(matches, downstreamMatches...)
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

// Parse scans a log stream line by line and returns all completed matches.
func (lp *LogParser) Parse(r io.ReadCloser) ([]*ParseMatch, Stats, error) {
	ctx, cancel := context.WithCancel(lp.ctx)
	defer func() { _ = r.Close() }()
	defer cancel()

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
			lp.logger.Debug("skipping line as it's more than 4kb", slog.Int("line_number", lineNo))
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

		pendingMatchers := initialCheckLine(line, lp.Rules)
		for _, m := range pendingMatchers {
			go runMatcher(ctx, m, matchChan)
		}

		stats.PartialMatches = stats.PartialMatches + len(pendingMatchers)

		activeMatchers = append(activeMatchers, pendingMatchers...)
		newMatches := getNewParseMatches(matchChan)
		if len(newMatches) > 0 {
			matches = append(matches, newMatches...)
			if len(matches) > lp.MaxMatches {
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

	// Remove any excess matches if we exceeded the maxMatches limit.
	if len(matches) > lp.MaxMatches {
		matches = matches[:lp.MaxMatches]
	}

	return matches, stats, nil
}
