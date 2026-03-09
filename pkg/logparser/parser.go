package logparser

import (
	"context"
	"sync"
)

type ParseMatch struct {
	Rule         *Rule
	MatchedLines []*LogLine
}

type matcher struct {
	Rule            *Rule
	FirstLine       *LogLine
	FinalLineNumber int
	LineChannel     chan *LogLine // Used for adding new log lines
	DoneChannel     chan struct{} // Used to signal that matcher finished
}

type LogLine struct {
	Content    string
	LineNumber int
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

func purgeInactiveMatchers(lineNumber int, matchers []*matcher) []*matcher {
	activeMatchers := []*matcher{}

	for _, m := range matchers {
		if lineNumber > m.FinalLineNumber {
			close(m.LineChannel) // Reached maximum lines
			continue
		}
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

func broadcastLogLine(line *LogLine, matchers []*matcher) {
	for _, m := range matchers {
		m.LineChannel <- line
	}
}

func initialCheckLine(line *LogLine, rules []*Rule) []*matcher {
	c := make(chan *matcher, len(rules))
	var wg sync.WaitGroup

	for _, r := range rules {
		wg.Add(1)
		go func(wg *sync.WaitGroup, c chan *matcher, r *Rule, l *LogLine) {
			defer wg.Done()
			if len(r.Checks) > 0 && r.Checks[0].CheckLine(l.Content) {
				c <- newMatcher(line, r)
			}
		}(&wg, c, r, line)
	}

	wg.Wait()
	matchers := []*matcher{}

	for {
		select {
		case msg := <-c:
			matchers = append(matchers, msg)
		default:
			return matchers
		}
	}
}

func newMatcher(firstLine *LogLine, rule *Rule) *matcher {
	return &matcher{
		LineChannel:     make(chan *LogLine, rule.getNeededLineCount()),
		DoneChannel:     make(chan struct{}),
		Rule:            rule,
		FirstLine:       firstLine,
		FinalLineNumber: firstLine.LineNumber + rule.getNeededLineCount() - 1,
	}
}

func runMatcher(ctx context.Context, m *matcher, mc chan *ParseMatch) {
	defer close(m.DoneChannel)
	matchedLines := []*LogLine{m.FirstLine}
	checkIndex := 1 // Already matched the first line

	for {
		if ctx.Err() != nil {
			return
		} else if checkIndex >= len(m.Rule.Checks) {
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
			return // Just in case the context is cancelled while waiting for messages
		}

	}
}
