package logparser

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestParseFromSource(t *testing.T) {
	t.Run("successfully parses logs from source", func(t *testing.T) {
		mockSource := &MockLogSource{
			logs: io.NopCloser(strings.NewReader("line1\nline2\nline3\n")),
		}

		parser := NewLogParser()

		matches, stats, err := parser.ParseFromSource(mockSource)

		if err != nil {
			t.Fatalf("ParseFromSource returned error: %v", err)
		}

		if stats.LinesParsed == 0 {
			t.Fatal("Expected some lines to be parsed")
		}

		if matches == nil {
			t.Fatal("Expected matches to not be nil")
		}

		if !mockSource.closeCalled {
			t.Fatal("Expected Close to be called on logs")
		}
	})

	t.Run("returns error when GetLogs fails", func(t *testing.T) {
		expectedErr := errors.New("failed to get logs")
		mockSource := &MockLogSource{
			getLogsErr: expectedErr,
		}

		parser := NewLogParser()
		_, _, err := parser.ParseFromSource(mockSource)

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if !strings.Contains(err.Error(), "failed to get logs from source") {
			t.Fatalf("Expected error message to contain 'failed to get logs from source', got: %v", err)
		}
	})

	t.Run("successfully parses logs when downstream logs are supported and only returns final log matches if there are any", func(t *testing.T) {
		downstreamLogRule := &Rule{
			Checks: []LineMatcher{{Contains: "downstream log line"}},
		}
		finalLogRule := &Rule{
			Checks: []LineMatcher{{Contains: "2nd level"}},
		}

		mockSource := &MockLogSource{
			logs: io.NopCloser(strings.NewReader("line1\nline2\nline3\ndownstream log line\n")),
			GetDownstreamErrorRuleFunc: func() *Rule {
				return downstreamLogRule
			},
			GetDownstreamErrorLogsFunc: func(pm *ParseMatch) (io.ReadCloser, error) {
				if pm.Rule != downstreamLogRule {
					return nil, errors.New("unexpected rule in GetDownstreamErrorLogs")
				}
				return io.NopCloser(strings.NewReader("2nd level\n")), nil
			},
		}

		parser := NewLogParser(
			WithRules([]*Rule{finalLogRule}),
		)
		matches, stats, err := parser.ParseFromSource(mockSource)

		if err != nil {
			t.Fatalf("ParseFromSource returned error: %v", err)
		}

		if stats.LinesParsed == 0 {
			t.Fatal("Expected some lines to be parsed")
		}

		if len(matches) != 1 {
			t.Fatalf("Expected 1 match, got %d", len(matches))
		}

		if matches[0].Rule != finalLogRule {
			t.Fatal("Expected first match to be for finalLogRule")
		}
	})

	t.Run("successfully parses logs when downstream logs are supported and returns downstream matches if there aren't any final log matches", func(t *testing.T) {
		downstreamLogRule := &Rule{
			Checks: []LineMatcher{{Contains: "downstream log line"}},
		}
		finalLogRule := &Rule{
			Checks: []LineMatcher{{Contains: "2nd level"}},
		}

		hasGetDownstreamErrorLogsFuncBeenCalled := false

		mockSource := &MockLogSource{
			logs: io.NopCloser(strings.NewReader("line1\nline2\nline3\ndownstream log line\n")),
			GetDownstreamErrorRuleFunc: func() *Rule {
				return downstreamLogRule
			},
			GetDownstreamErrorLogsFunc: func(pm *ParseMatch) (io.ReadCloser, error) {
				if pm.Rule != downstreamLogRule {
					return nil, errors.New("unexpected rule in GetDownstreamErrorLogs")
				}
				if hasGetDownstreamErrorLogsFuncBeenCalled {
					return nil, errors.New("GetDownstreamErrorLogs called more than once, unexpected")
				}
				hasGetDownstreamErrorLogsFuncBeenCalled = true
				return io.NopCloser(strings.NewReader("something else\n")), nil
			},
		}

		parser := NewLogParser(
			WithRules([]*Rule{finalLogRule}),
		)
		matches, stats, err := parser.ParseFromSource(mockSource)

		if err != nil {
			t.Fatalf("ParseFromSource returned error: %v", err)
		}

		if stats.LinesParsed == 0 {
			t.Fatal("Expected some lines to be parsed")
		}

		if len(matches) != 1 {
			t.Fatalf("Expected 1 match, got %d", len(matches))
		}

		if matches[0].Rule != downstreamLogRule {
			t.Fatal("Expected first match to be for downstreamLogRule")
		}
	})

	t.Run("calls Parse when downstream logs not supported", func(t *testing.T) {
		mockSource := &MockLogSource{
			logs: io.NopCloser(strings.NewReader("test\n")),
		}

		parser := NewLogParser()
		_, _, err := parser.ParseFromSource(mockSource)

		if err != nil {
			t.Fatalf("ParseFromSource returned error: %v", err)
		}
	})
}

var _ LogSource = (RecursiveLogSource)(nil)

type MockLogSource struct {
	logs                       io.ReadCloser
	getLogsErr                 error
	closeCalled                bool
	GetDownstreamErrorRuleFunc func() *Rule
	GetDownstreamErrorLogsFunc func(*ParseMatch) (io.ReadCloser, error)
}

func (m *MockLogSource) GetLogs() (io.ReadCloser, error) {
	if m.getLogsErr != nil {
		return nil, m.getLogsErr
	}
	return &mockReadCloser{Reader: m.logs, onClose: func() { m.closeCalled = true }}, nil
}

func (m *MockLogSource) GetDownstreamErrorRule() *Rule {
	if m.GetDownstreamErrorRuleFunc != nil {
		return m.GetDownstreamErrorRuleFunc()
	}
	return &Rule{Checks: []LineMatcher{{Contains: "NEVEREVERMATCH"}}}
}

func (m *MockLogSource) GetDownstreamErrorLogs(pm *ParseMatch) (io.ReadCloser, error) {
	if m.GetDownstreamErrorLogsFunc != nil {
		return m.GetDownstreamErrorLogsFunc(pm)
	}
	panic("GetDownstreamErrorLogs is unimplemented")
}

func (m *MockLogSource) GetMaxRecursionDepth() int {
	return 3
}

type mockReadCloser struct {
	io.Reader
	onClose func()
}

func (m *mockReadCloser) Close() error {
	if m.onClose != nil {
		m.onClose()
	}
	return nil
}

func TestParse(t *testing.T) {
	t.Run("calculates stats correctly", func(t *testing.T) {
		input := "line1\nline2\nline3\n"
		reader := io.NopCloser(strings.NewReader(input))
		parser := NewLogParser()
		_, stats, err := parser.Parse(reader)

		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		if stats.LinesParsed != 3 {
			t.Fatalf("LinesParsed = %d, want 3", stats.LinesParsed)
		}
	})

	t.Run("returns no matches when rules are empty", func(t *testing.T) {
		input := "line1\nline2\n"
		reader := io.NopCloser(strings.NewReader(input))

		parser := NewLogParser()

		matches, _, err := parser.Parse(reader)
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		if len(matches) != 0 {
			t.Fatalf("Expected 0 matches, got %d", len(matches))
		}
	})

	t.Run("handles reader with no newlines", func(t *testing.T) {
		input := "single line"
		reader := io.NopCloser(strings.NewReader(input))

		parser := NewLogParser()
		_, stats, err := parser.Parse(reader)
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		if stats.LinesParsed != 1 {
			t.Fatalf("LinesParsed = %d, want 1", stats.LinesParsed)
		}
	})

	t.Run("handles empty reader", func(t *testing.T) {
		reader := io.NopCloser(strings.NewReader(""))

		parser := NewLogParser()

		matches, stats, err := parser.Parse(reader)
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		if len(matches) != 0 {
			t.Fatalf("Expected 0 matches, got %d", len(matches))
		}
		if stats.LinesParsed != 0 {
			t.Fatalf("LinesParsed = %d, want 0", stats.LinesParsed)
		}
	})

	t.Run("handles line that's way too long", func(t *testing.T) {
		sb := strings.Builder{}
		sb.WriteString("Starting line\n")
		sb.WriteString(strings.Repeat("x", 4097))
		sb.WriteString("\nEnding line\n")

		reader := io.NopCloser(strings.NewReader(sb.String()))

		parser := NewLogParser()
		_, stats, err := parser.Parse(reader)
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		if stats.LinesParsed != 3 {
			t.Fatalf("LinesParsed = %d, want 3", stats.LinesParsed)
		}
	})

	t.Run("handles custom max line size", func(t *testing.T) {
		sb := strings.Builder{}
		sb.WriteString("Starting line\n")
		sb.WriteString(strings.Repeat("x", 8193))
		sb.WriteString("\nEnding line\n")

		reader := io.NopCloser(strings.NewReader(sb.String()))

		parser := NewLogParser(WithMaxLineSizeKB(8))
		_, stats, err := parser.Parse(reader)
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		if stats.LinesParsed != 3 {
			t.Fatalf("LinesParsed = %d, want 3", stats.LinesParsed)
		}
	})
}
