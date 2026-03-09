package filesource

import (
	"errors"
	"io"
	"os"
)

// var _ LogSource = (*FileLogSource)(nil) //TODO: Add this back once the pfp package is moved to a public logparser package and the LogSource interface is defined there.

type FileLogSource struct {
	FilePath string
}

// NewFileLogSource creates a new FileLogSource with the given file path.
// Basic usage example:
//
//	logSource := NewFileLogSource("/path/to/logfile.log")
func NewFileLogSource(filePath string) *FileLogSource {
	return &FileLogSource{FilePath: filePath}
}

// GetLogs reads the logs from the specified file and returns an io.ReadCloser.
//
//	Basic usage example:
//
//	logSource := NewFileLogSource("/path/to/logfile.log")
//
//	logs, err := logSource.GetLogs()
//	if err != nil {
//		// handle error
//	}
//	defer logs.Close()
//	// process logs
func (f *FileLogSource) GetLogs() (io.ReadCloser, error) {
	if f.FilePath == "" {
		return nil, errors.New("file path cannot be empty")
	}

	file, err := os.Open(f.FilePath)
	if err != nil {
		return nil, err
	}

	return file, nil
}
