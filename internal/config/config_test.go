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
		ConfigVersion:  CurrentConfigVersion,
		Provider:       "ollama",
		AutoStage:      true,
		CommitTemplate: "/repo/.gitmessage",
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
	if got.AutoStage != want.AutoStage {
		t.Errorf("AutoStage: got %v, want %v", got.AutoStage, want.AutoStage)
	}
	if got.Ollama.BaseURL != want.Ollama.BaseURL {
		t.Errorf("Ollama.BaseURL: got %q, want %q", got.Ollama.BaseURL, want.Ollama.BaseURL)
	}
	if got.Ollama.Model != want.Ollama.Model {
		t.Errorf("Ollama.Model: got %q, want %q", got.Ollama.Model, want.Ollama.Model)
	}
	if got.CommitTemplate != want.CommitTemplate {
		t.Errorf("CommitTemplate: got %q, want %q", got.CommitTemplate, want.CommitTemplate)
	}
	if got.ConfigVersion != want.ConfigVersion {
		t.Errorf("ConfigVersion: got %d, want %d", got.ConfigVersion, want.ConfigVersion)
	}
}

func TestWriteDefaultTemplate(t *testing.T) {
	dir := t.TempDir()

	path, err := WriteDefaultTemplate(dir)
	if err != nil {
		t.Fatalf("WriteDefaultTemplate: %v", err)
	}
	if path == "" {
		t.Fatal("expected non-empty path")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading template: %v", err)
	}
	if len(content) == 0 {
		t.Error("default template file should not be empty")
	}

	// Second call must not overwrite the file.
	if err := os.WriteFile(path, []byte("custom"), 0o644); err != nil {
		t.Fatalf("writing custom content: %v", err)
	}
	if _, err := WriteDefaultTemplate(dir); err != nil {
		t.Fatalf("second WriteDefaultTemplate: %v", err)
	}
	preserved, _ := os.ReadFile(path)
	if string(preserved) != "custom" {
		t.Error("WriteDefaultTemplate should not overwrite an existing template file")
	}
}

func TestLoadOutdatedConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	old := &Config{
		ConfigVersion: 0,
		Provider:      "ollama",
		Ollama:        OllamaConfig{BaseURL: "http://localhost:11434", Model: "llama3"},
	}
	if err := SaveTo(path, old); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	got, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if got.ConfigVersion != 0 {
		t.Errorf("expected ConfigVersion 0 for outdated config, got %d", got.ConfigVersion)
	}
	if got.ConfigVersion >= CurrentConfigVersion {
		t.Errorf("outdated config should have ConfigVersion < %d, got %d", CurrentConfigVersion, got.ConfigVersion)
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
