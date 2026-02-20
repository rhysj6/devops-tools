package jenkins

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsJobURL(t *testing.T) {
	client := JenkinsClient{URL: "https://jenkins.example.com"}

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "matches job url",
			input: "https://jenkins.example.com/job/my-job/123",
			want:  true,
		},
		{
			name:  "does not match job name only",
			input: "my-job",
			want:  false,
		},
		{
			name:  "does not match build number only",
			input: "123",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.IsJobURL(tt.input)
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
			name:            "parses valid job url",
			input:           "https://jenkins.example.com/job/my-job/123",
			wantName:        "my-job",
			wantBuildNumber: 123,
			wantErr:         false,
		},
		{
			name:            "parses url encoded job name",
			input:           "https://jenkins.example.com/job/My+Job/456",
			wantName:        "My Job",
			wantBuildNumber: 456,
			wantErr:         false,
		},
		{
			name:    "returns error for non job url",
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
	t.Run("returns console text for valid credentials", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/job/my-job/123/consoleText" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}

			username, password, ok := r.BasicAuth()
			if !ok || username != "user" || password != "pass" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("line1\nline2\n"))
		}))
		defer server.Close()

		client := JenkinsClient{
			URL:      server.URL,
			Username: "user",
			Password: "pass",
		}

		reader, err := client.GetBuildLogs("my-job", 123)
		if err != nil {
			t.Fatalf("GetBuildLogs returned error: %v", err)
		}
		defer reader.Close()

		body, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("failed to read logs body: %v", err)
		}
		logs := string(body)

		if logs != "line1\nline2\n" {
			t.Fatalf("GetBuildLogs returned logs %q, want %q", logs, "line1\nline2\n")
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
}
