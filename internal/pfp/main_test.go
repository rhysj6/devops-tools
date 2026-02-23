package pfp

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

		rules := []*Rule{}
		matches, stats, err := ParseFromSource(mockSource, rules, 10)

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

		rules := []*Rule{}
		_, _, err := ParseFromSource(mockSource, rules, 10)

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if !strings.Contains(err.Error(), "failed to get logs from source") {
			t.Fatalf("Expected error message to contain 'failed to get logs from source', got: %v", err)
		}

		t.Run("successfully parses logs when downstream builds are supported", func(t *testing.T) {
			downstreamLogRule := &Rule{
				Checks: []LineMatcher{{Contains: "downstream log line"}},
			}
			finalLogRule := &Rule{
				Checks: []LineMatcher{{Contains: "2nd level"}},
			}

			mockSource := &MockLogSource{
				logs:                          io.NopCloser(strings.NewReader("line1\nline2\nline3\ndownstream log line\n")),
				supportDownstreamFailedBuilds: true,
				GetDownstreamFailedBuildRuleFunc: func() *Rule {
					return downstreamLogRule
				},
				GetDownstreamFailedBuildLogsFunc: func(pm *ParseMatch) (io.Reader, error) {
					if pm.Rule != downstreamLogRule {
						return nil, errors.New("unexpected rule in GetDownstreamFailedBuildLogs")
					}
					return strings.NewReader("2nd level\n"), nil
				},
			}

			rules := []*Rule{finalLogRule}
			matches, stats, err := ParseFromSource(mockSource, rules, 10)

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
				t.Fatal("Expected match to be for finalLogRule")
			}
		})
	})

	t.Run("calls Parse when downstream builds not supported", func(t *testing.T) {
		mockSource := &MockLogSource{
			logs:                          io.NopCloser(strings.NewReader("test\n")),
			supportDownstreamFailedBuilds: false,
		}

		rules := []*Rule{}
		_, _, err := ParseFromSource(mockSource, rules, 10)

		if err != nil {
			t.Fatalf("ParseFromSource returned error: %v", err)
		}
	})
}

type MockLogSource struct {
	logs                             io.ReadCloser
	getLogsErr                       error
	closeCalled                      bool
	supportDownstreamFailedBuilds    bool
	GetDownstreamFailedBuildRuleFunc func() *Rule
	GetDownstreamFailedBuildLogsFunc func(*ParseMatch) (io.Reader, error)
}

func (m *MockLogSource) GetLogs() (io.ReadCloser, error) {
	if m.getLogsErr != nil {
		return nil, m.getLogsErr
	}
	return &mockReadCloser{Reader: m.logs, onClose: func() { m.closeCalled = true }}, nil
}

func (m *MockLogSource) SupportDownstreamFailedBuilds() bool {
	return m.supportDownstreamFailedBuilds
}

func (m *MockLogSource) GetDownstreamFailedBuildRule() *Rule {
	if m.GetDownstreamFailedBuildRuleFunc != nil {
		return m.GetDownstreamFailedBuildRuleFunc()
	}
	panic("unimplemented")
}

func (m *MockLogSource) GetDownstreamFailedBuildLogs(*ParseMatch) (io.Reader, error) {
	if m.GetDownstreamFailedBuildLogsFunc != nil {
		return m.GetDownstreamFailedBuildLogsFunc(nil)
	}
	panic("unimplemented")
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
		reader := strings.NewReader(input)

		_, stats, err := Parse(reader, []*Rule{}, 10)
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		if stats.LinesParsed != 4 {
			t.Fatalf("LinesParsed = %d, want 4", stats.LinesParsed)
		}
		if stats.Duration == 0 {
			t.Fatal("Duration should be non-zero")
		}
	})

	t.Run("returns no matches when rules are empty", func(t *testing.T) {
		input := "line1\nline2\n"
		reader := strings.NewReader(input)

		matches, _, err := Parse(reader, []*Rule{}, 10)
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		if len(matches) != 0 {
			t.Fatalf("Expected 0 matches, got %d", len(matches))
		}
	})

	t.Run("handles reader with no newlines", func(t *testing.T) {
		input := "single line"
		reader := strings.NewReader(input)

		_, stats, err := Parse(reader, []*Rule{}, 10)
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		if stats.LinesParsed < 1 {
			t.Fatalf("LinesParsed = %d, want at least 1", stats.LinesParsed)
		}
	})

	t.Run("handles empty reader", func(t *testing.T) {
		reader := strings.NewReader("")

		matches, stats, err := Parse(reader, []*Rule{}, 10)
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		if len(matches) != 0 {
			t.Fatalf("Expected 0 matches, got %d", len(matches))
		}
		if stats.LinesParsed != 1 {
			t.Fatalf("LinesParsed = %d, want 1", stats.LinesParsed)
		}
	})

	t.Run("measures duration", func(t *testing.T) {
		input := strings.Repeat("line\n", 10)
		reader := strings.NewReader(input)

		_, stats, err := Parse(reader, []*Rule{}, 100)
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		if stats.Duration <= 0 {
			t.Fatalf("Duration should be positive, got %v", stats.Duration)
		}
	})
}
