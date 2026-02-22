package pfp

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestFileLogSource_GetLogs(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.log")
	testContent := "test log content"

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	t.Run("successfully opens existing file", func(t *testing.T) {
		source := &FileLogSource{FilePath: testFile}
		reader, err := source.GetLogs()

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if reader == nil {
			t.Error("Expected reader, got nil")
		}

		// Verify content
		content, _ := io.ReadAll(reader)
		if string(content) != testContent {
			t.Errorf("Expected content %q, got %q", testContent, string(content))
		}
		reader.Close()
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		source := &FileLogSource{FilePath: "/non/existent/file.log"}
		reader, err := source.GetLogs()
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}

		if reader != nil {
			t.Errorf("Expected nil reader for non-existent file, got non-nil")
			defer reader.Close()
		}
	})

	t.Run("closes reader properly", func(t *testing.T) {
		source := &FileLogSource{FilePath: testFile}
		reader, err := source.GetLogs()

		if err != nil {
			t.Fatalf("Failed to get logs: %v", err)
		}

		if err := reader.Close(); err != nil {
			t.Errorf("Failed to close reader: %v", err)
		}
	})

	t.Run("returns error for empty file path", func(t *testing.T) {
		source := &FileLogSource{FilePath: ""}
		reader, err := source.GetLogs()

		if err == nil {
			t.Error("Expected error for empty file path, got nil")
		}
		if reader != nil {
			t.Error("Expected nil reader for empty file path")
			reader.Close()
		}
	})
}

func TestFileLogSource_SupportDownstreamFailedBuilds(t *testing.T) {
	source := &FileLogSource{}

	if source.SupportDownstreamFailedBuilds() {
		t.Error("Expected SupportDownstreamFailedBuilds to return false")
	}
}

func TestFileLogSource_GetDownstreamFailedBuildRule(t *testing.T) {
	source := &FileLogSource{}

	if rule := source.GetDownstreamFailedBuildRule(); rule != nil {
		t.Errorf("Expected nil rule, got: %v", rule)
	}
}

func TestFileLogSource_GetDownstreamFailedBuildLogs(t *testing.T) {
	source := &FileLogSource{}
	match := &ParseMatch{}

	reader, err := source.GetDownstreamFailedBuildLogs(match)

	if err != nil {
		t.Errorf("Expected nil error, got: %v", err)
	}
	if reader != nil {
		t.Errorf("Expected nil reader, got: %v", reader)
	}
}

func TestFileLogSource_ImplementsLogSource(t *testing.T) {
	var _ LogSource = (*FileLogSource)(nil)
}
