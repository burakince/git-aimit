package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// CurrentConfigVersion is incremented whenever new required fields are added to Config.
// Loaded configs with a lower version are considered outdated.
const CurrentConfigVersion = 1

type OllamaConfig struct {
	BaseURL string `json:"base_url" mapstructure:"base_url"`
	Model   string `json:"model"    mapstructure:"model"`
}

type Config struct {
	ConfigVersion  int          `json:"config_version"   mapstructure:"config_version"`
	Provider       string       `json:"provider"         mapstructure:"provider"`
	AutoStage      bool         `json:"auto_stage"       mapstructure:"auto_stage"`
	CommitTemplate string       `json:"commit_template"  mapstructure:"commit_template"`
	Ollama         OllamaConfig `json:"ollama"           mapstructure:"ollama"`
}

// ConfigPath returns the absolute path to the config file.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "git-aimit", "config.json"), nil
}

// LoadFrom reads and parses the config file at the given path.
func LoadFrom(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("json")

	if err := v.ReadInConfig(); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config not found -- run `git aimit init` first")
		}
		// viper wraps the underlying error; check the message for not-found cases
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			return nil, fmt.Errorf("config not found -- run `git aimit init` first")
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// Load reads the config from the default location.
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	return LoadFrom(path)
}

// SaveTo writes the config as JSON to the given path, creating directories as needed.
func SaveTo(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// Save writes the config to the default location.
func Save(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	return SaveTo(path, cfg)
}
