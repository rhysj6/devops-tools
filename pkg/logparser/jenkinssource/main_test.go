package jenkinssource

import (
	"context"
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/rhysj6/devops-tools/pkg/logparser"
)

var _ Client = (*MockJenkinsClient)(nil)

type MockJenkinsClient struct {
	GetJobNameAndNumberFromURLFunc func(url string) (string, int, error)
	GetBuildLogsFunc               func(jobName string, buildNumber int) (io.ReadCloser, error)
}

func (m *MockJenkinsClient) GetBuildLogs(ctx context.Context, jobName string, buildNumber int) (io.ReadCloser, error) {
	if m.GetBuildLogsFunc != nil {
		return m.GetBuildLogsFunc(jobName, buildNumber)
	}
	panic("GetBuildLogs is unimplemented")

}
func (m *MockJenkinsClient) IsJobURL(string) bool {
	panic("IsJobURL is unimplemented")
}

func (m *MockJenkinsClient) GetJobNameAndNumberFromURL(url string) (string, int, error) {
	if m.GetJobNameAndNumberFromURLFunc != nil {
		return m.GetJobNameAndNumberFromURLFunc(url)
	}
	panic("GetJobNameAndNumberFromURL is unimplemented")
}

func TestNewJenkinsLogSource(t *testing.T) {
	t.Run("creates source with single URL argument", func(t *testing.T) {
		mockClient := &MockJenkinsClient{
			GetJobNameAndNumberFromURLFunc: func(url string) (string, int, error) {
				return "test-job", 42, nil
			},
		}

		source, err := NewJenkinsLogSource(mockClient, []string{"http://jenkins.example.com/job/test-job/42"}, t.Context())

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if source == nil {
			t.Fatal("Expected source, got nil")
		}
		if source.jobName != "test-job" {
			t.Errorf("Expected jobName 'test-job', got: %s", source.jobName)
		}
		if source.buildNumber != 42 {
			t.Errorf("Expected buildNumber 42, got: %d", source.buildNumber)
		}
	})

	t.Run("creates source with job name and build number arguments", func(t *testing.T) {
		mockClient := &MockJenkinsClient{}
		source, err := NewJenkinsLogSource(mockClient, []string{"my-job", "123"}, t.Context())

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if source == nil {
			t.Fatal("Expected source, got nil")
		}
		if source.jobName != "my-job" {
			t.Errorf("Expected jobName 'my-job', got: %s", source.jobName)
		}
		if source.buildNumber != 123 {
			t.Errorf("Expected buildNumber 123, got: %d", source.buildNumber)
		}
	})

	t.Run("returns error when URL parsing fails", func(t *testing.T) {
		expectedErr := errors.New("invalid URL")
		mockClient := &MockJenkinsClient{
			GetJobNameAndNumberFromURLFunc: func(url string) (string, int, error) {
				return "", 0, expectedErr
			},
		}

		source, err := NewJenkinsLogSource(mockClient, []string{"invalid-url"}, t.Context())

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if source != nil {
			t.Error("Expected nil source on error")
		}
	})

	t.Run("returns error for invalid build number", func(t *testing.T) {
		mockClient := &MockJenkinsClient{}
		source, err := NewJenkinsLogSource(mockClient, []string{"job-name", "not-a-number"}, t.Context())

		if err == nil {
			t.Error("Expected error for invalid build number, got nil")
		}
		if source != nil {
			t.Error("Expected nil source on error")
		}
	})

	t.Run("returns error for zero arguments", func(t *testing.T) {
		mockClient := &MockJenkinsClient{}
		source, err := NewJenkinsLogSource(mockClient, []string{}, t.Context())

		if err == nil {
			t.Error("Expected error for zero arguments, got nil")
		}
		if source != nil {
			t.Error("Expected nil source on error")
		}
	})

	t.Run("returns error for more than two arguments", func(t *testing.T) {
		mockClient := &MockJenkinsClient{}
		source, err := NewJenkinsLogSource(mockClient, []string{"arg1", "arg2", "arg3"}, t.Context())

		if err == nil {
			t.Error("Expected error for three arguments, got nil")
		}
		if source != nil {
			t.Error("Expected nil source on error")
		}
	})
}

func TestGetDownstreamFailedBuildRule(t *testing.T) {
	source := &JenkinsLogSource{}
	t.Run("returns Rule", func(t *testing.T) {
		rule := source.GetDownstreamErrorRule()
		if rule == nil {
			t.Fatal("Expected non-nil Rule, got nil")
		}
	})

	t.Run("rule matches log line", func(t *testing.T) {
		rule := source.GetDownstreamErrorRule()
		logLines := []string{
			"Build Example_Build #5 completed: FAILURE",
			"Build Example_Build #5: build_name completed: FAILURE",
		}

		if len(rule.Checks) != 1 {
			t.Fatalf("Expected 1 check in rule, got %d", len(rule.Checks))
		}

		matches := 0
		for _, line := range logLines {
			if rule.Checks[0].CheckLine(line) {
				matches++
			}
		}
		if matches != len(logLines) {
			t.Errorf("Expected %d matches, got %d", len(logLines), matches)
		}
	})
}

func TestGetJobNameAndBuildNumberFromMatch(t *testing.T) {
	source := &JenkinsLogSource{}

	t.Run("successfully extracts job name and build number", func(t *testing.T) {
		rule := source.GetDownstreamErrorRule()
		match := &logparser.ParseMatch{
			Rule: rule,
			MatchedLines: []*logparser.LogLine{
				{Content: "Build Example Build #42 completed: FAILURE"},
			},
		}

		jobName, buildNumber, err := source.getJobNameAndBuildNumberFromMatch(match)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if jobName != "Example Build" {
			t.Errorf("Expected jobName 'Example Build', got: %s", jobName)
		}
		if buildNumber != 42 {
			t.Errorf("Expected buildNumber 42, got: %d", buildNumber)
		}
	})

	t.Run("extracts from log line with optional build name", func(t *testing.T) {
		rule := source.GetDownstreamErrorRule()
		match := &logparser.ParseMatch{
			Rule: rule,
			MatchedLines: []*logparser.LogLine{
				{Content: "Build My-Job-Name #123: some_build_name completed: FAILURE"},
			},
		}

		jobName, buildNumber, err := source.getJobNameAndBuildNumberFromMatch(match)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if jobName != "My-Job-Name" {
			t.Errorf("Expected jobName 'My-Job-Name', got: %s", jobName)
		}
		if buildNumber != 123 {
			t.Errorf("Expected buildNumber 123, got: %d", buildNumber)
		}
	})

	t.Run("returns error when regex is nil", func(t *testing.T) {
		match := &logparser.ParseMatch{
			Rule: &logparser.Rule{
				Checks: []logparser.LineMatcher{
					{Contains: "test", Regex: nil},
				},
			},
			MatchedLines: []*logparser.LogLine{
				{Content: "Build Example_Build #42 completed: FAILURE"},
			},
		}

		_, _, err := source.getJobNameAndBuildNumberFromMatch(match)

		if err == nil {
			t.Error("Expected error for nil regex, got nil")
		}
	})

	t.Run("returns error when regex does not match", func(t *testing.T) {
		rule := source.GetDownstreamErrorRule()
		match := &logparser.ParseMatch{
			Rule: rule,
			MatchedLines: []*logparser.LogLine{
				{Content: "This does not match the expected pattern"},
			},
		}

		_, _, err := source.getJobNameAndBuildNumberFromMatch(match)

		if err == nil {
			t.Error("Expected error for non-matching regex, got nil")
		}
	})

	t.Run("returns error when named groups are missing", func(t *testing.T) {
		match := &logparser.ParseMatch{
			Rule: &logparser.Rule{
				Checks: []logparser.LineMatcher{
					{Contains: "test", Regex: regexp.MustCompile(`(?m)^Build\s+(.+?)\s+#(\d+)\s+completed:\s+FAILURE\s*$`)},
				},
			},
			MatchedLines: []*logparser.LogLine{
				{Content: "Build Example_Build #42 completed: FAILURE"},
			},
		}

		_, _, err := source.getJobNameAndBuildNumberFromMatch(match)

		if err == nil {
			t.Error("Expected error for missing named groups, got nil")
		}
	})

	t.Run("returns error when build number is not a valid integer", func(t *testing.T) {
		match := &logparser.ParseMatch{
			Rule: &logparser.Rule{
				Checks: []logparser.LineMatcher{
					{Contains: "test", Regex: regexp.MustCompile(`(?m)^Build\s+(?P<job>.+?)\s+#(?P<number>\D+)\s+completed:\s+FAILURE\s*$`)},
				},
			},
			MatchedLines: []*logparser.LogLine{
				{Content: "Build Example_Build #abc completed: FAILURE"},
			},
		}

		_, _, err := source.getJobNameAndBuildNumberFromMatch(match)

		if err == nil {
			t.Error("Expected error for invalid build number, got nil")
		}
	})
}

func TestGetMaxRecursionDepth(t *testing.T) {
	source := &JenkinsLogSource{}
	expectedDepth := 3
	if source.GetMaxRecursionDepth() != expectedDepth {
		t.Errorf("Expected max recursion depth %d, got %d", expectedDepth, source.GetMaxRecursionDepth())
	}
}
func TestGetDownstreamFailedBuildLogs(t *testing.T) {
	t.Run("returns an error for match with irrelevant rule", func(t *testing.T) {
		source := &JenkinsLogSource{
			client: &MockJenkinsClient{},
		}
		match := &logparser.ParseMatch{
			Rule: &logparser.Rule{},
			MatchedLines: []*logparser.LogLine{
				{Content: "Build Example_Build #5 completed: FAILURE"},
			},
		}

		logs, err := source.GetDownstreamErrorLogs(match)
		if err == nil {
			t.Error("Expected error for match with irrelevant rule, got nil")
		}
		if logs != nil {
			t.Error("Expected nil logs on error")
		}
	})
	t.Run("returns error when no matched lines", func(t *testing.T) {
		source := &JenkinsLogSource{
			client: &MockJenkinsClient{},
		}
		rule := source.GetDownstreamErrorRule()
		match := &logparser.ParseMatch{
			Rule:         rule,
			MatchedLines: []*logparser.LogLine{},
		}

		logs, err := source.GetDownstreamErrorLogs(match)
		if err == nil {
			t.Error("Expected error for empty matched lines, got nil")
		}
		if logs != nil {
			t.Error("Expected nil logs on error")
		}
	})

	t.Run("returns error when extraction fails", func(t *testing.T) {
		source := &JenkinsLogSource{
			client: &MockJenkinsClient{},
		}
		rule := source.GetDownstreamErrorRule()
		match := &logparser.ParseMatch{
			Rule: rule,
			MatchedLines: []*logparser.LogLine{
				{Content: "Invalid log line format"},
			},
		}

		logs, err := source.GetDownstreamErrorLogs(match)
		if err == nil {
			t.Error("Expected error when extraction fails, got nil")
		}
		if logs != nil {
			t.Error("Expected nil logs on error")
		}
	})

	t.Run("successfully retrieves build logs for matched rule", func(t *testing.T) {
		expectedReadCloser := io.NopCloser(strings.NewReader("Example log content"))
		mockClient := &MockJenkinsClient{
			GetBuildLogsFunc: func(jobName string, buildNumber int) (io.ReadCloser, error) {
				if jobName == "Example_Build" && buildNumber == 5 {
					return expectedReadCloser, nil
				}
				return nil, errors.New("unexpected job name or build number")
			},
		}
		source := &JenkinsLogSource{client: mockClient}
		rule := source.GetDownstreamErrorRule()
		match := &logparser.ParseMatch{
			Rule: rule,
			MatchedLines: []*logparser.LogLine{
				{Content: "Build Example_Build #5 completed: FAILURE"},
			},
		}

		logs, err := source.GetDownstreamErrorLogs(match)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		logContent, err := io.ReadAll(logs)
		if err != nil {
			t.Errorf("Expected no error reading logs, got: %v", err)
		}
		if string(logContent) != "Example log content" {
			t.Errorf("Expected 'Example log content', got: %s", string(logContent))
		}
	})
}
