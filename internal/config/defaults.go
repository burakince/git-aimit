package config

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed assets/commit-template.txt
var defaultCommitTemplate string

const DefaultTemplateFilename = "commit-template.txt"

// WriteDefaultTemplate writes the built-in commit template to configDir if the
// file does not already exist, preserving any user edits on re-runs. Returns
// the full path to the template file.
func WriteDefaultTemplate(configDir string) (string, error) {
	path := filepath.Join(configDir, DefaultTemplateFilename)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(defaultCommitTemplate), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
