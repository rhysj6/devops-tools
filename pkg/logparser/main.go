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
	Rules         []*MatchRule
	MaxMatches    int
	MaxLineSizeKB int
	logger        *slog.Logger
	ctx           context.Context
}

type ParserOption func(*LogParser)

// WithRules sets the rules that the LogParser will use to parse logs. If not set, then no matches will be found by default.
func WithRules(rules []*MatchRule) ParserOption {
	return func(lp *LogParser) {
		lp.Rules = rules
	}
}

// WithMaxMatches sets the maximum number of matches that the parser will return. If the parser finds more matches than this, it will stop parsing and return the matches found so far. Default is 1.
func WithMaxMatches(max int) ParserOption {
	return func(lp *LogParser) {
		lp.MaxMatches = max
	}
}

// WithLogger sets the logger for the LogParser. If not set, then no logs will be emitted by default.
func WithLogger(logger *slog.Logger) ParserOption {
	return func(lp *LogParser) {
		lp.logger = logger
	}
}

// WithContext sets the context for the LogParser. This context will be used to manage cancellation and timeouts for parsing operations. If not set, context.Background() will be used by default.
func WithContext(ctx context.Context) ParserOption {
	return func(lp *LogParser) {
		lp.ctx = ctx
	}
}

// WithMaxLineSizeKB sets the maximum line size in KB that the parser will read. Lines longer than this will be skipped. Default is 4KB.
func WithMaxLineSizeKB(size int) ParserOption {
	return func(lp *LogParser) {
		lp.MaxLineSizeKB = size
	}
}

// NewLogParser creates a new LogParser with the provided options. If no logger is provided, a default logger that discards output will be used. If no context is provided, context.Background() will be used.
func NewLogParser(opts ...ParserOption) *LogParser {
	lp := &LogParser{
		Rules:         []*MatchRule{},
		MaxMatches:    1,
		MaxLineSizeKB: 4,
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
// Receives the value of the LogParser as a non-pointer to ensure that the original LogParser's rules are not modified when appending the downstream error rule for recursive parsing.
func (lp LogParser) ParseFromSource(source LogSource) ([]*ParseMatch, Stats, error) {
	startTime := time.Now()
	logs, err := source.GetLogs()
	if err != nil {
		return nil, Stats{}, fmt.Errorf("failed to get logs from source: %w", err)
	}

	recursiveSource, isRecursiveLogSource := source.(RecursiveLogSource) // If the source does not support downstream error logs, we can just parse the logs once and return the results.

	if isRecursiveLogSource {
		lp.Rules = append(lp.Rules, recursiveSource.GetDownstreamErrorRule())
	}
	// Initial parse of the logs from the source
	matches, stats, err := lp.Parse(logs)

	// If the source isn't a RecursiveLogSource, then we can return the results early.
	if !isRecursiveLogSource {
		stats.Duration = time.Since(startTime)
		return matches, stats, err
	}

	if err != nil {
		return nil, stats, fmt.Errorf("failed to parse logs: %w", err)
	}

	// Recursively parse downstream logs if we find a potential downstream failure mention, up to the maximum recursion depth specified by the RecursiveLogSource implementation or 3.
	for range max(recursiveSource.GetMaxRecursionDepth(), 3) {
		// Only checks for downstream errors if no other matches were found.
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

	// Remove any matches that are for the downstream error rule, as these are not relevant if there are other matches found in the logs.
	customMatches := []*ParseMatch{}
	for _, m := range matches {
		if m.Rule != recursiveSource.GetDownstreamErrorRule() {
			customMatches = append(customMatches, m)
		}
	}

	stats.Duration = time.Since(startTime)

	if len(customMatches) > 0 {
		stats.CompleteMatches = len(customMatches)
		return customMatches, stats, nil
	}

	return matches, stats, nil
}

// Parse scans a log stream line by line and returns all completed matches.
func (lp LogParser) Parse(r io.ReadCloser) ([]*ParseMatch, Stats, error) {
	ctx, cancel := context.WithCancel(lp.ctx)
	// Close the reader when we're done parsing, and cancel the context to stop any ongoing matcher goroutines.
	defer func() { _ = r.Close() }()
	defer cancel()

	stats := Stats{}
	reader := bufio.NewReaderSize(r, lp.MaxLineSizeKB*1024)

	// activeMatchers holds all matchers that are currently running and have not yet completed.
	activeMatchers := []*parseMatchCandidate{}
	matches := []*ParseMatch{}

	// matchChan is used to receive completed matches from the matcher goroutines.
	matchChan := make(chan *ParseMatch, 100)
	defer close(matchChan)

	var rtnErr error = nil

	lineNo := 0
	for {
		lineNo++
		// Read the line, if the line is more than MaxLineSizeKB then discard it.
		lineBytes, err := reader.ReadSlice('\n')

		if err == io.EOF && len(lineBytes) == 0 {
			lineNo-- // Don't count the EOF as a line
			break
		} else if err == bufio.ErrBufferFull {
			lp.logger.Debug("skipping line as it's more than MaxLineSizeKB", slog.Int("line_number", lineNo))
			// Discard the rest of the line
			_, err := reader.ReadString('\n')
			if err != nil && err != io.EOF && err != bufio.ErrBufferFull {
				rtnErr = fmt.Errorf("reader error: %w \n Log line number: %v", err, lineNo)
				cancel()
				break
			}
			continue

		} else if err != nil && err != io.EOF {
			rtnErr = fmt.Errorf("reader error: %w \n Log line number: %v", err, lineNo)
			cancel()
			break
		}

		line := &LogLine{
			Content:    strings.TrimRight(string(lineBytes), "\n"),
			LineNumber: lineNo,
		}

		activeMatchers = purgeInactiveMatchCandidates(lineNo, activeMatchers)
		broadcastLogLine(line, activeMatchers)

		pendingMatchers := matchLineAgainstFirstChecks(line, lp.Rules)
		for _, m := range pendingMatchers {
			go runMatchCandidate(ctx, m, matchChan)
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

	return matches, stats, rtnErr
}
