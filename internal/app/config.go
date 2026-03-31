package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const DefaultBaseURL = "https://api.quickpod.org"

type Config struct {
	BaseURL string `json:"base_url"`
	Token   string `json:"token"`
	Output  string `json:"output"`
}

func NormalizeBaseURL(raw string) (string, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(raw), "/")
	if trimmed == "" {
		return DefaultBaseURL, nil
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported base URL scheme %q", parsed.Scheme)
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", fmt.Errorf("base URL must include a host")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
	}

func DefaultConfig() Config {
	return Config{
		BaseURL: DefaultBaseURL,
		Output:  "table",
	}
}

func DefaultConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "quickpod-cli", "config.json"), nil
}

func LoadConfig(path string) (Config, error) {
	if strings.TrimSpace(path) == "" {
		return Config{}, errors.New("config path is empty")
	}

	config := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config, nil
		}
		return Config{}, err
	}

	if len(data) == 0 {
		return config, nil
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, err
	}

	normalizedBaseURL, err := NormalizeBaseURL(config.BaseURL)
	if err != nil {
		return Config{}, err
	}
	config.BaseURL = normalizedBaseURL
	if strings.TrimSpace(config.Output) == "" {
		config.Output = "table"
	}

	return config, nil
}

func SaveConfig(path string, config Config) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("config path is empty")
	}
	normalizedBaseURL, err := NormalizeBaseURL(config.BaseURL)
	if err != nil {
		return err
	}
	config.BaseURL = normalizedBaseURL
	if strings.TrimSpace(config.Output) == "" {
		config.Output = "table"
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, append(data, '\n'), 0o600)
}