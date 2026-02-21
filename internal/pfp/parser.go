package pfp

import (
	"context"
	"sync"
)

type ParseMatch struct {
	Rule         *Rule
	MatchedLines []*LogLine
}

type Matcher struct {
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

func GetNewParseMatches(c <-chan *ParseMatch) []*ParseMatch {
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

func PurgeInactiveMatchers(lineNumber int, matchers []*Matcher) []*Matcher {
	activeMatchers := []*Matcher{}

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

func BroadcastLogLine(line *LogLine, matchers []*Matcher) {
	for _, m := range matchers {
		m.LineChannel <- line
	}
}

func InitialCheckLine(line *LogLine, rules []*Rule) []*Matcher {
	c := make(chan *Matcher, len(rules))
	var wg sync.WaitGroup

	for _, r := range rules {
		wg.Add(1)
		go func(wg *sync.WaitGroup, c chan *Matcher, r *Rule, l *LogLine) {
			defer wg.Done()
			if len(r.Checks) > 0 && r.Checks[0].CheckLine(l.Content) {
				c <- newMatcher(line, r)
			}
		}(&wg, c, r, line)
	}

	wg.Wait()
	matchers := []*Matcher{}

	for {
		select {
		case msg := <-c:
			matchers = append(matchers, msg)
		default:
			return matchers
		}
	}
}

func newMatcher(firstLine *LogLine, rule *Rule) *Matcher {
	return &Matcher{
		LineChannel:     make(chan *LogLine, rule.GetNeededLineCount()),
		DoneChannel:     make(chan struct{}),
		Rule:            rule,
		FirstLine:       firstLine,
		FinalLineNumber: firstLine.LineNumber + rule.GetNeededLineCount() - 1,
	}
}

func RunMatcher(m *Matcher, mc chan *ParseMatch, ctx context.Context) {
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
