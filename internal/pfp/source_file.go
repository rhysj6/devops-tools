package pfp

import (
	"errors"
	"io"
	"os"

	"github.com/rhysj6/devops-tools/pkg/logparser"
)

var _ logparser.LogSource = (*FileLogSource)(nil)

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
