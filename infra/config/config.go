package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Loader struct {
	basePath string
}

func NewLoader(basePath string) *Loader {
	return &Loader{basePath: basePath}
}

func LoadInto[T any](l *Loader, fileName string) (*T, error) {
	fullPath := filepath.Join(l.basePath, fileName)
	// #nosec G304 - fileName is hardcoded at call sites, not user-controlled
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("config reading %q: %w", fullPath, err)
	}

	var out T
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("config unmarshalling %q: %w", fullPath, err)
	}

	return &out, nil
}
