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
