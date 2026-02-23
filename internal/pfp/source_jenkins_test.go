package pfp

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/rhysj6/devops-tools/internal/jenkins"
)

var _ jenkins.Client = (*MockJenkinsClient)(nil)

type MockJenkinsClient struct {
	GetJobNameAndNumberFromURLFunc func(url string) (string, int, error)
}

func (m *MockJenkinsClient) GetBuildLogsWithContext(ctx context.Context, jobName string, buildNumber int) (io.ReadCloser, error) {
	panic("GetBuildLogsWithContext is unimplemented")
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

func (m *MockJenkinsClient) GetBuildLogs(jobName string, buildNumber int) (io.ReadCloser, error) {
	panic("GetBuildLogs is unimplemented")
}

func TestNewJenkinsLogSource(t *testing.T) {
	t.Run("creates source with single URL argument", func(t *testing.T) {
		mockClient := &MockJenkinsClient{
			GetJobNameAndNumberFromURLFunc: func(url string) (string, int, error) {
				return "test-job", 42, nil
			},
		}

		source, err := NewJenkinsLogSource(mockClient, []string{"http://jenkins.example.com/job/test-job/42"})

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
		source, err := NewJenkinsLogSource(mockClient, []string{"my-job", "123"})

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

		source, err := NewJenkinsLogSource(mockClient, []string{"invalid-url"})

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if source != nil {
			t.Error("Expected nil source on error")
		}
	})

	t.Run("returns error for invalid build number", func(t *testing.T) {
		mockClient := &MockJenkinsClient{}
		source, err := NewJenkinsLogSource(mockClient, []string{"job-name", "not-a-number"})

		if err == nil {
			t.Error("Expected error for invalid build number, got nil")
		}
		if source != nil {
			t.Error("Expected nil source on error")
		}
	})

	t.Run("returns error for zero arguments", func(t *testing.T) {
		mockClient := &MockJenkinsClient{}
		source, err := NewJenkinsLogSource(mockClient, []string{})

		if err == nil {
			t.Error("Expected error for zero arguments, got nil")
		}
		if source != nil {
			t.Error("Expected nil source on error")
		}
	})

	t.Run("returns error for more than two arguments", func(t *testing.T) {
		mockClient := &MockJenkinsClient{}
		source, err := NewJenkinsLogSource(mockClient, []string{"arg1", "arg2", "arg3"})

		if err == nil {
			t.Error("Expected error for three arguments, got nil")
		}
		if source != nil {
			t.Error("Expected nil source on error")
		}
	})
}
