//go:build !darwin

package keychain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Read retrieves OAuth credentials from the .credentials.json file.
func Read() (*Credentials, error) {
	configDir := os.Getenv("CLAUDE_CONFIG_DIR")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		configDir = filepath.Join(home, ".claude")
	}
	path := filepath.Join(configDir, ".credentials.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read credentials: %w", err)
	}

	var wrapper struct {
		ClaudeAiOauth Credentials `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}
	if wrapper.ClaudeAiOauth.AccessToken == "" {
		return nil, fmt.Errorf("no access token in credentials file")
	}
	return &wrapper.ClaudeAiOauth, nil
}
