package tfstate

import (
	"context"
	"fmt"
	"os"
)

// LocalSource reads Terraform state from a file on disk.
type LocalSource struct {
	Path string
}

// NewLocalSource creates a StateSource backed by a local .tfstate file.
func NewLocalSource(path string) *LocalSource {
	return &LocalSource{Path: path}
}

// Read implements StateSource.
func (s *LocalSource) Read(_ context.Context) ([]byte, string, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		return nil, "", fmt.Errorf("read local state %q: %w", s.Path, err)
	}
	return data, "file://" + s.Path, nil
}
