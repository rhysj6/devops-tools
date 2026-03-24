package logparser

import (
	"context"
	"sync"
)

// ParseMatch represents a successful rule match and the lines that matched.
type ParseMatch struct {
	Rule         *MatchRule `json:"rule"`
	MatchedLines []*LogLine `json:"matchedLines"`
}

type parseMatchCandidate struct {
	Rule            *MatchRule
	FirstLine       *LogLine
	FinalLineNumber int
	LineChannel     chan *LogLine // Used for adding new log lines
	DoneChannel     chan struct{} // Used to signal that matcher finished
}

// LogLine is a single parsed line and its 1-based position in the source.
type LogLine struct {
	Content    string `json:"content"`
	LineNumber int    `json:"lineNumber"`
}

func getNewParseMatches(c <-chan *ParseMatch) []*ParseMatch {
	matches := []*ParseMatch{}
	for {
		select {
		case msg := <-c:
			matches = append(matches, msg)
		default:
			return matches
		}
	}
}

func purgeInactiveMatchCandidates(lineNumber int, matcherCandidates []*parseMatchCandidate) []*parseMatchCandidate {
	activeMatchers := []*parseMatchCandidate{}

	for _, m := range matcherCandidates {
		if lineNumber > m.FinalLineNumber {
			close(m.LineChannel) // Reached maximum lines
			continue
		}
		// If the matcher is done, don't add it to the active matchers list and close its channel.
		select {
		case <-m.DoneChannel:
			close(m.LineChannel) // Matcher is done
			continue
		default:
			activeMatchers = append(activeMatchers, m)
		}
	}

	return activeMatchers
}

func broadcastLogLine(line *LogLine, matchers []*parseMatchCandidate) {
	for _, m := range matchers {
		m.LineChannel <- line
	}
}

func matchLineAgainstFirstChecks(line *LogLine, rules []*MatchRule) []*parseMatchCandidate {
	c := make(chan *parseMatchCandidate, len(rules))
	var wg sync.WaitGroup

	for _, r := range rules {
		wg.Add(1)
		go func(wg *sync.WaitGroup, c chan *parseMatchCandidate, r *MatchRule, l *LogLine) {
			defer wg.Done()
			if len(r.Checks) > 0 && r.Checks[0].CheckLine(l.Content) {
				c <- createMatchCandidate(line, r)
			}
		}(&wg, c, r, line)
	}

	// Wait for all checks to finish
	wg.Wait()
	matchers := []*parseMatchCandidate{}

	// Collect all the matchers from the channel until it's empty
	for {
		select {
		case msg := <-c:
			matchers = append(matchers, msg)
		default:
			return matchers
		}
	}
}

func createMatchCandidate(firstLine *LogLine, rule *MatchRule) *parseMatchCandidate {
	return &parseMatchCandidate{
		LineChannel:     make(chan *LogLine, rule.getNeededLineCount()),
		DoneChannel:     make(chan struct{}),
		Rule:            rule,
		FirstLine:       firstLine,
		FinalLineNumber: firstLine.LineNumber + rule.getNeededLineCount() - 1,
	}
}

func runMatchCandidate(ctx context.Context, m *parseMatchCandidate, mc chan *ParseMatch) {
	defer close(m.DoneChannel)
	matchedLines := []*LogLine{m.FirstLine}
	checkIndex := 1 // Already matched the first line

	for {
		if checkIndex >= len(m.Rule.Checks) {
			mc <- &ParseMatch{
				Rule:         m.Rule,
				MatchedLines: matchedLines,
			}
			return
		} else if matchedLines[len(matchedLines)-1].LineNumber >= m.FinalLineNumber {
			return // The last line checked was the final line to check, no matches found
		}
		select {
		case line, ok := <-m.LineChannel:
			if !ok {
				return // Channel has closed
			}
			matchedLines = append(matchedLines, line)

			if m.Rule.Checks[checkIndex].CheckLine(line.Content) {
				checkIndex++
			}
		case <-ctx.Done():
			return
		}

	}
}
