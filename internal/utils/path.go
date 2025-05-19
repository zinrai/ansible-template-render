package utils

import (
	"os"
	"path/filepath"
)

// Helps manage paths consistently
type PathResolver struct {
	BaseDir string
}

func NewPathResolver(baseDir string) (*PathResolver, error) {
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, err
	}

	return &PathResolver{
		BaseDir: abs,
	}, nil
}

// Converts a relative path to an absolute path
func (p *PathResolver) ResolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(p.BaseDir, path)
}

// Converts an absolute path to a relative path from the base directory
func (p *PathResolver) RelativePath(path string) (string, error) {
	return filepath.Rel(p.BaseDir, path)
}

// Ensures a directory exists, creating it if necessary
func (p *PathResolver) EnsureDirectoryExists(path string) error {
	absPath := p.ResolvePath(path)
	return os.MkdirAll(absPath, 0755)
}
