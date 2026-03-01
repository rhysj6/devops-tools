package jenkins

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsJobURL(t *testing.T) {
	jenkinsURL := "https://jenkins.example.com"

	tests := []struct {
		name   string
		input  string
		want   bool
		client JenkinsClient
	}{
		{
			name:   "matches job URL",
			input:  "https://jenkins.example.com/job/my-job/123",
			want:   true,
			client: JenkinsClient{URL: jenkinsURL},
		},
		{
			name:   "matches job URL with path prefix",
			input:  "https://jenkins.example.com/path/job/my-job/123",
			want:   true,
			client: JenkinsClient{URL: jenkinsURL + "/path"},
		},
		{
			name:   "matches job URL with path prefix ending in slash",
			input:  "https://jenkins.example.com/path/job/my-job/123",
			want:   true,
			client: JenkinsClient{URL: jenkinsURL + "/path/"},
		},
		{
			name:   "does not match job name only",
			input:  "my-job",
			want:   false,
			client: JenkinsClient{URL: jenkinsURL},
		},
		{
			name:   "does not match build number only",
			input:  "123",
			want:   false,
			client: JenkinsClient{URL: jenkinsURL},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.client.IsJobURL(tt.input)
			if got != tt.want {
				t.Fatalf("IsJobURL(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetJobNameAndNumberFromURL(t *testing.T) {
	client := JenkinsClient{URL: "https://jenkins.example.com"}

	tests := []struct {
		name            string
		input           string
		wantName        string
		wantBuildNumber int
		wantErr         bool
	}{
		{
			name:            "parses valid job URL",
			input:           "https://jenkins.example.com/job/my-job/123",
			wantName:        "my-job",
			wantBuildNumber: 123,
			wantErr:         false,
		},
		{
			name:            "parses URL encoded job name",
			input:           "https://jenkins.example.com/job/My+Job/456",
			wantName:        "My Job",
			wantBuildNumber: 456,
			wantErr:         false,
		},
		{
			name:            "parses URL with job within multiple folders",
			input:           "https://jenkins.example.com/job/old%20ansible/job/linux/job/setup/456",
			wantName:        "old ansible/linux/setup",
			wantBuildNumber: 456,
			wantErr:         false,
		},
		{
			name:    "returns error for non job URL",
			input:   "https://jenkins.example.com/view/all",
			wantErr: true,
		},
		{
			name:    "returns error for invalid escaped name",
			input:   "https://jenkins.example.com/job/%ZZ/123",
			wantErr: true,
		},
		{
			name:    "returns error for invalid build number",
			input:   "https://jenkins.example.com/job/my-job/not-a-number",
			wantErr: true,
		},
		{
			name:    "returns error for missing build segment",
			input:   "https://jenkins.example.com/job/my-job/",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotBuildNumber, err := client.GetJobNameAndNumberFromURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetJobNameAndNumberFromURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if gotName != tt.wantName {
				t.Fatalf("GetJobNameAndNumberFromURL(%q) name = %q, want %q", tt.input, gotName, tt.wantName)
			}

			if gotBuildNumber != tt.wantBuildNumber {
				t.Fatalf("GetJobNameAndNumberFromURL(%q) buildNumber = %d, want %d", tt.input, gotBuildNumber, tt.wantBuildNumber)
			}
		})
	}
}

func TestGetBuildLogs(t *testing.T) {
	t.Run("returns console text", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/job/my-job/123/consoleText" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("line1\nline2\n"))
		}))
		defer server.Close()

		client := JenkinsClient{URL: server.URL, Username: "u", Password: "p"}

		reader, err := client.GetBuildLogs("my-job", 123)
		if err != nil {
			t.Fatalf("GetBuildLogs returned error: %v", err)
		}
		defer func() { _ = reader.Close() }()

		body, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("failed to read logs body: %v", err)
		}
		logs := string(body)

		if logs != "line1\nline2\n" {
			t.Fatalf("GetBuildLogs returned logs %q, want %q", logs, "line1\nline2\n")
		}
	})

	t.Run("handles jenkins folder paths correctly", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/job/old ansible/job/linux/job/my-job/123/consoleText" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("line1\nline2\n"))
		}))
		defer server.Close()

		client := JenkinsClient{URL: server.URL, Username: "u", Password: "p"}

		reader, err := client.GetBuildLogs("old ansible/linux/my-job", 123)
		if err != nil {
			t.Fatalf("GetBuildLogs returned error: %v", err)
		}
		_ = reader.Close()
	})

	t.Run("returns no error for correct credentials", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok || username != "u" || password != "p" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := JenkinsClient{URL: server.URL, Username: "u", Password: "p"}

		_, err := client.GetBuildLogs("my-job", 123)
		if err != nil {
			t.Fatalf("GetBuildLogs returned error: %v", err)
		}
	})

	t.Run("returns unauthorized error for incorrect credentials", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok || username != "user" || password != "pass" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := JenkinsClient{
			URL:      server.URL,
			Username: "user",
			Password: "wrong-pass",
		}

		_, err := client.GetBuildLogs("my-job", 123)
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("GetBuildLogs error = %v, want ErrUnauthorized", err)
		}
	})

	t.Run("returns error for invalid parameters", func(t *testing.T) {
		client := JenkinsClient{
			URL:      "https://jenkins.example.com",
			Username: "user",
			Password: "pass",
		}

		_, err := client.GetBuildLogs("", 123)
		if err == nil {
			t.Fatal("GetBuildLogs should return error for empty job name")
		}

		_, err = client.GetBuildLogs("my-job", 0)
		if err == nil {
			t.Fatal("GetBuildLogs should return error for invalid build number")
		}
	})

	t.Run("returns error for 404 status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := JenkinsClient{
			URL:      server.URL,
			Username: "user",
			Password: "wrong-pass",
		}

		_, err := client.GetBuildLogs("my-job", 123)
		if err == nil {
			t.Fatal("Expected error got nil")
		}
		if err.Error() != "failed to fetch build logs: status 404" {
			t.Fatalf("Expected 'failed to fetch build logs: status 404' got %v", err.Error())
		}
	})
}
