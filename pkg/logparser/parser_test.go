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
	r := MatchRule{
		Name: "TestRule",
		Checks: []LineCheck{
			{Contains: "Hi"},
			{Contains: "Hello"},
		},
		MaxLines: 100,
	}
	l := LogLine{
		LineNumber: 17,
	}
	m := createMatchCandidate(&l, &r)

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
	r1 := MatchRule{Checks: []LineCheck{{Contains: "Hi"}, {Contains: "Hello"}}}
	m1 := createMatchCandidate(&LogLine{LineNumber: 1}, &r1)
	m2 := createMatchCandidate(&LogLine{LineNumber: 1}, &r1)

	// These matchers will be testing the final line number logic.
	r2 := MatchRule{Checks: []LineCheck{{Contains: "Hi"}}, MaxLines: 1}
	m3 := createMatchCandidate(&LogLine{LineNumber: 1}, &r2)
	m4 := createMatchCandidate(&LogLine{LineNumber: 1}, &r2)

	// Close the line channel for m3 to simulate a matcher that has received all its lines. This will panic if broadcastLogLine tries to send to it, which is what we want to test against.
	close(m3.LineChannel)
	m3.AcceptingLines = false

	matchers := []*parseMatchCandidate{m1, m2, m3, m4}

	broadcastLogLine(&LogLine{Content: "Hi", LineNumber: 2}, matchers)

	if m4.AcceptingLines != false {
		t.Fatal("Matcher should have AcceptingLines set to false after reaching FinalLineNumber")
	}

	for range 3 {
		select {
		case msg, ok := <-m1.LineChannel:
			if !ok {
				t.Fatal("Channel was closed when it should be open")
			} else if msg.Content != "Hi" {
				t.Fatalf("Expected message Hi got %v", msg.Content)
			}

		case msg, ok := <-m2.LineChannel:
			if !ok {
				t.Fatal("Channel was closed when it should be open")
			} else if msg.Content != "Hi" {
				t.Fatalf("Expected message Hi got %v", msg.Content)
			}
		case _, ok := <-m4.LineChannel:
			if ok {
				t.Fatal("Channel was open when it should be closed after reaching FinalLineNumber")
			}
		default:
			t.Fatalf("Expected 2 messages on both channels, defaulted instead.")
		}
	}

}

func TestPurgeInactiveMatchers(t *testing.T) {
	Rule := MatchRule{
		MaxLines: 2,
		Checks: []LineCheck{
			{Contains: "Hi"},
			{Contains: "Hello"},
		},
	} // Rule with MaxLines so that we can test purging based on line number

	// Expired because first line number is 1 and current line number is 5
	expiredMatcher := createMatchCandidate(&LogLine{LineNumber: 1}, &Rule)

	closedMatcher := createMatchCandidate(&LogLine{LineNumber: 17}, &Rule)
	close(closedMatcher.DoneChannel)

	m := createMatchCandidate(&LogLine{LineNumber: 3}, &Rule)

	ams := []*parseMatchCandidate{expiredMatcher, closedMatcher, m}

	r := purgeInactiveMatchCandidates(4, ams)

	if len(r) != 2 {
		t.Fatalf("Expected 2 matchers got %v", len(r))
	}
	if r[0] != expiredMatcher {
		t.Fatalf("Matcher is not correct, should be expiredMatcher")
	} else if r[0].AllLinesReceived == false {
		t.Fatal("Expired matcher should have AllLinesReceived set to true")
	}
	if r[1] != m {
		t.Fatalf("Matcher is not correct, should be matcher")
	}
}

func TestInitialCheckLine(t *testing.T) {
	r1 := MatchRule{
		Name:   "match",
		Checks: []LineCheck{{Contains: "match"}},
	}
	r2 := MatchRule{
		Name:   "don't match",
		Checks: []LineCheck{{Contains: "don't match"}},
	}

	matchers := matchLineAgainstFirstChecks(&LogLine{LineNumber: 1, Content: "match"}, []*MatchRule{&r1, &r2})

	if len(matchers) != 1 {
		t.Fatalf("Expected 1 matcher got %v", len(matchers))
	}

	if matchers[0].Rule != &r1 {
		t.Fatal("Expected to get matcher with Rule of r1")
	}
}

func TestRunMatcher_ContextCancel(t *testing.T) {
	Rule := MatchRule{Checks: []LineCheck{{Contains: "match"}, {Contains: "error"}}}
	firstLog := LogLine{LineNumber: 1, Content: "match"}
	m := createMatchCandidate(&firstLog, &Rule)

	ctx, cancel := context.WithCancel(context.Background())
	matchChan := make(chan *ParseMatch)

	go runMatchCandidate(ctx, m, matchChan) // This matcher will be expecting another line to continue the checks

	m.LineChannel <- &LogLine{LineNumber: 2, Content: "not"} // Send a log line to ensure

	cancel()

	select {
	case <-m.DoneChannel:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("RunMatcher did not exit on context cancel")
	}
}

func TestRunMatcher_HandlesSingleCheck(t *testing.T) {
	Rule := MatchRule{Checks: []LineCheck{{Contains: "match"}}}
	firstLog := LogLine{LineNumber: 1, Content: "match"}
	m := createMatchCandidate(&firstLog, &Rule)

	ctx := t.Context()
	matchChan := make(chan *ParseMatch)

	go runMatchCandidate(ctx, m, matchChan) // This matcher will be expecting another line to continue the checks

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
	Rule := MatchRule{Checks: []LineCheck{{Contains: "match"}, {Contains: "definitely an error"}}, MaxLines: 100}
	firstLog := LogLine{LineNumber: 1, Content: "match"}
	m := createMatchCandidate(&firstLog, &Rule)

	ctx := t.Context()
	matchChan := make(chan *ParseMatch)

	go runMatchCandidate(ctx, m, matchChan)

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
			t.Fatalf("Expected log: %v instead got %v", finalLog.Content, msg.MatchedLines[len(msg.MatchedLines)-1].Content)
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
	t.Run("handle maxlines for rule", func(t *testing.T) {
		Rule := MatchRule{Checks: []LineCheck{{Contains: "match"}, {Contains: "definitely an error"}}, MaxLines: 10}
		firstLog := LogLine{LineNumber: 1, Content: "match"}
		m := createMatchCandidate(&firstLog, &Rule)

		ctx := t.Context()
		matchChan := make(chan *ParseMatch)

		go runMatchCandidate(ctx, m, matchChan)

		// Spam some non matching lines
		for i := range 9 {
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
	})

	t.Run("handle line check maxlines", func(t *testing.T) {
		Rule := MatchRule{Checks: []LineCheck{{Contains: "match"}, {Contains: "definitely an error", MaxLines: 1}}, MaxLines: 15}
		firstLog := LogLine{LineNumber: 1, Content: "match"}
		m := createMatchCandidate(&firstLog, &Rule)

		ctx := t.Context()
		matchChan := make(chan *ParseMatch, 1) // Buffer of 1 to ensure that if an unexpected match is sent, it doesn't block the test from finishing

		go runMatchCandidate(ctx, m, matchChan)

		// Spam some non matching lines
		for i := range 8 {
			m.LineChannel <- &LogLine{
				LineNumber: i + 2,
				Content:    "This won't match the Rules and is there for fun",
			}
		}

		// This line matches the second check but is outside of the maxlines for that check
		m.LineChannel <- &LogLine{
			LineNumber: 10,
			Content:    "definitely an error",
		}

		for i := range 5 {
			m.LineChannel <- &LogLine{
				LineNumber: i + 11,
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
	})
}
