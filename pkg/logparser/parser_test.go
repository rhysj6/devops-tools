package logparser

import (
	"context"
	"testing"
	"time"
)

func TestGetNewParseMatches(t *testing.T) {
	c := make(chan *ParseMatch, 2)

	r := getNewParseMatches(c)

	if len(r) != 0 {
		t.Fatalf("Expected 0 ParseMatches got %v", len(r))
	}

	c <- &ParseMatch{}
	c <- &ParseMatch{}

	r = getNewParseMatches(c)

	if len(r) != 2 {
		t.Fatalf("Expected 2 ParseMatches got %v", len(r))
	}
}

func TestNewMatcher(t *testing.T) {
	r := Rule{
		Name: "TestRule",
		Checks: []LineMatcher{
			{Contains: "Hi"},
			{Contains: "Hello"},
		},
		MaxLines: 100,
	}
	l := LogLine{
		LineNumber: 17,
	}
	m := newMatcher(&l, &r)

	if m.Rule != &r {
		t.Fatal("Not including correct Rule pointer")
	}

	if m.FirstLine.LineNumber != 17 {
		t.Fatalf("Expected StartLine of 17 got %v", m.FirstLine.LineNumber)
	}

	if m.FinalLineNumber > 116 {
		t.Fatalf("Expected FinalLineNumber of 116 got %v", m.FinalLineNumber)
	}
}

func TestBroadcastLogLine_BroadcastsToActiveChannels(t *testing.T) {
	Rule := Rule{Checks: []LineMatcher{{Contains: "Hi"}}} // Rule with at least 1 check so line channel has a buffer
	m1 := newMatcher(&LogLine{LineNumber: 1}, &Rule)
	m2 := newMatcher(&LogLine{LineNumber: 1}, &Rule)

	matchers := []*matcher{m1, m2}

	broadcastLogLine(&LogLine{Content: "Hi"}, matchers)

	for range 2 {
		select {
		case msg := <-m1.LineChannel:
			if msg.Content != "Hi" {
				t.Fatalf("Expected message Hi got %v", msg.Content)
			}

		case msg := <-m2.LineChannel:
			if msg.Content != "Hi" {
				t.Fatalf("Expected message Hi got %v", msg.Content)
			}
		default:
			t.Fatalf("Expected 2 messages on both channels, defaulted instead.")
		}
	}

}

func TestPurgeInactiveMatchers(t *testing.T) {
	Rule := Rule{}

	expiredMatcher := newMatcher(&LogLine{LineNumber: 1}, &Rule)
	closedMatcher := newMatcher(&LogLine{LineNumber: 17}, &Rule)
	close(closedMatcher.DoneChannel)
	m := newMatcher(&LogLine{LineNumber: 17}, &Rule)

	ams := []*matcher{expiredMatcher, closedMatcher, m}

	r := purgeInactiveMatchers(5, ams)

	if len(r) != 1 {
		t.Fatalf("Expected 1 matcher got %v", len(r))
	}

	if r[0] != m {
		t.Fatalf("Matcher is not correct, should be matcher")
	}
}

func TestInitialCheckLine(t *testing.T) {
	r1 := Rule{
		Name:   "match",
		Checks: []LineMatcher{{Contains: "match"}},
	}
	r2 := Rule{
		Name:   "don't match",
		Checks: []LineMatcher{{Contains: "don't match"}},
	}

	matchers := initialCheckLine(&LogLine{LineNumber: 1, Content: "match"}, []*Rule{&r1, &r2})

	if len(matchers) != 1 {
		t.Fatalf("Expected 1 matcher got %v", len(matchers))
	}

	if matchers[0].Rule != &r1 {
		t.Fatal("Expected to get matcher with Rule of r1")
	}
}

func TestRunMatcher_ContextCancel(t *testing.T) {
	Rule := Rule{Checks: []LineMatcher{{Contains: "match"}, {Contains: "error"}}}
	firstLog := LogLine{LineNumber: 1, Content: "match"}
	m := newMatcher(&firstLog, &Rule)

	ctx, cancel := context.WithCancel(context.Background())
	matchChan := make(chan *ParseMatch)

	go runMatcher(ctx, m, matchChan) // This matcher will be expecting another line to continue the checks

	m.LineChannel <- &LogLine{LineNumber: 2, Content: "not"} // Send a log line to ensure

	cancel()

	select {
	case <-m.DoneChannel:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("RunMatcher did not exit on context cancel")
	}
}

func TestRunMatcher_HandlesSingleCheck(t *testing.T) {
	Rule := Rule{Checks: []LineMatcher{{Contains: "match"}}}
	firstLog := LogLine{LineNumber: 1, Content: "match"}
	m := newMatcher(&firstLog, &Rule)

	ctx := t.Context()
	matchChan := make(chan *ParseMatch)

	go runMatcher(ctx, m, matchChan) // This matcher will be expecting another line to continue the checks

	select {
	case msg := <-matchChan:
		if len(msg.MatchedLines) != 1 {
			t.Fatalf("Expected 1 log line got %v", len(msg.MatchedLines))
		} else if msg.MatchedLines[0] != &firstLog {
			t.Fatalf("Expected log: %v instead got %v", firstLog.Content, msg.MatchedLines[0].Content)
		} else if msg.Rule != &Rule {
			t.Fatal("ParseMatch Rule doesn't match Rule")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("RunMatcher did not send parseMatch")
	}

	select {
	case _, ok := <-m.DoneChannel:
		if ok {
			t.Fatal("Message sent to DoneChannel when it should be closed.")
		}
	case <-time.After(10 * time.Millisecond):
		t.Fatal("RunMatcher did not exit in time")
	}

}

func TestRunMatcher_HandlesMatchAndExit(t *testing.T) {
	Rule := Rule{Checks: []LineMatcher{{Contains: "match"}, {Contains: "definitely an error"}}, MaxLines: 100}
	firstLog := LogLine{LineNumber: 1, Content: "match"}
	m := newMatcher(&firstLog, &Rule)

	ctx := t.Context()
	matchChan := make(chan *ParseMatch)

	go runMatcher(ctx, m, matchChan)

	// Spam some non matching lines
	for i := range 97 {
		m.LineChannel <- &LogLine{
			LineNumber: i + 1,
			Content:    "This won't match the Rules and is there for fun",
		}
	}

	finalLog := LogLine{
		LineNumber: 99,
		Content:    "definitely an error",
	}
	m.LineChannel <- &finalLog

	select {
	case msg := <-matchChan:
		if len(msg.MatchedLines) != 99 {
			t.Fatalf("Expected 99 log lines got %v", len(msg.MatchedLines))
		} else if msg.MatchedLines[0] != &firstLog {
			t.Fatalf("Expected log: %v instead got %v", firstLog.Content, msg.MatchedLines[0].Content)
		} else if msg.MatchedLines[len(msg.MatchedLines)-1] != &finalLog {
			t.Fatalf("Expected log: %v instead got %v", firstLog.Content, msg.MatchedLines[len(msg.MatchedLines)-1].Content)
		} else if msg.Rule != &Rule {
			t.Fatal("ParseMatch Rule doesn't match Rule")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("RunMatcher did not send parseMatch")
	}

	select {
	case <-m.DoneChannel:
	case <-time.After(time.Millisecond):
		t.Fatal("RunMatcher did not exit in time")
	}

}

func TestRunMatcher_HandlesNoMatch(t *testing.T) {
	Rule := Rule{Checks: []LineMatcher{{Contains: "match"}, {Contains: "definitely an error"}}, MaxLines: 100}
	firstLog := LogLine{LineNumber: 1, Content: "match"}
	m := newMatcher(&firstLog, &Rule)

	ctx := t.Context()
	matchChan := make(chan *ParseMatch)

	go runMatcher(ctx, m, matchChan)

	// Spam some non matching lines
	for i := range 99 {
		m.LineChannel <- &LogLine{
			LineNumber: i + 2,
			Content:    "This won't match the Rules and is there for fun",
		}
	}

	select {
	case <-m.DoneChannel:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("RunMatcher did not exit in time")
	}

	select {
	case msg := <-matchChan:
		t.Fatalf("Unexpected parse match: %+v", msg)
	default:
		// No message as expected
	}
}
