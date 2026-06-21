package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	want := &Config{
		Provider: "ollama",
		Ollama: OllamaConfig{
			BaseURL: "http://localhost:11434",
			Model:   "llama3",
		},
	}

	if err := SaveTo(path, want); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	got, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	if got.Provider != want.Provider {
		t.Errorf("Provider: got %q, want %q", got.Provider, want.Provider)
	}
	if got.Ollama.BaseURL != want.Ollama.BaseURL {
		t.Errorf("Ollama.BaseURL: got %q, want %q", got.Ollama.BaseURL, want.Ollama.BaseURL)
	}
	if got.Ollama.Model != want.Ollama.Model {
		t.Errorf("Ollama.Model: got %q, want %q", got.Ollama.Model, want.Ollama.Model)
	}
}

func TestSaveCreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "dir", "config.json")

	cfg := &Config{Provider: "ollama", Ollama: OllamaConfig{BaseURL: "http://x", Model: "m"}}
	if err := SaveTo(path, cfg); err != nil {
		t.Fatalf("SaveTo with nested path: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestSaveFilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := &Config{Provider: "ollama", Ollama: OllamaConfig{BaseURL: "http://x", Model: "m"}}
	if err := SaveTo(path, cfg); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file permissions: got %o, want 600", perm)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := LoadFrom("/nonexistent/path/config.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "git aimit init") {
		t.Errorf("error message should mention 'git aimit init', got: %v", err)
	}
}
