//go:build darwin

package keychain

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

const (
	legacyServiceName = "Claude Code-credentials"
	keychainTimeout   = 3 * time.Second
)

var (
	errNoToken      = errors.New("no access token in keychain data")
	errTokenExpired = errors.New("OAuth token expired")
)

// Read retrieves OAuth credentials from the macOS Keychain via /usr/bin/security.
func Read() (*Credentials, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	envConfigDir := os.Getenv("CLAUDE_CONFIG_DIR")
	names := serviceNames(home, envConfigDir)
	account := accountName()

	for _, svc := range names {
		// Try account-scoped first
		if account != "" {
			if creds, err := tryKeychain(svc, account); err == nil {
				return creds, nil
			}
		}
		// Fall back to generic (no account)
		if creds, err := tryKeychain(svc, ""); err == nil {
			return creds, nil
		}
	}

	return nil, fmt.Errorf("no credentials found in keychain (tried %d service names)", len(names))
}

// tryKeychain calls /usr/bin/security to read a keychain item.
func tryKeychain(service, account string) (*Credentials, error) {
	ctx, cancel := context.WithTimeout(context.Background(), keychainTimeout)
	defer cancel()

	args := []string{"find-generic-password", "-s", service}
	if account != "" {
		args = append(args, "-a", account)
	}
	args = append(args, "-w")

	out, err := exec.CommandContext(ctx, "/usr/bin/security", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("security command: %w", err)
	}

	return parseKeychainData([]byte(strings.TrimSpace(string(out))))
}

func accountName() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}
	return u.Username
}

// parseKeychainData parses the JSON blob from the keychain and validates the token.
func parseKeychainData(data []byte) (*Credentials, error) {
	var wrapper struct {
		ClaudeAiOauth struct {
			AccessToken      string `json:"accessToken"`
			SubscriptionType string `json:"subscriptionType"`
			ExpiresAt        *int64 `json:"expiresAt"`
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("parse keychain data: %w", err)
	}
	if wrapper.ClaudeAiOauth.AccessToken == "" {
		return nil, errNoToken
	}
	if wrapper.ClaudeAiOauth.ExpiresAt != nil && *wrapper.ClaudeAiOauth.ExpiresAt <= time.Now().UnixMilli() {
		return nil, errTokenExpired
	}
	return &Credentials{
		AccessToken:      wrapper.ClaudeAiOauth.AccessToken,
		SubscriptionType: wrapper.ClaudeAiOauth.SubscriptionType,
	}, nil
}

// serviceName returns the keychain service name for a given config directory.
func serviceName(configDir, homeDir string) string {
	absConfig, _ := filepath.Abs(configDir)
	absDefault, _ := filepath.Abs(filepath.Join(homeDir, ".claude"))
	if filepath.Clean(absConfig) == filepath.Clean(absDefault) {
		return legacyServiceName
	}
	h := sha256.Sum256([]byte(absConfig))
	return fmt.Sprintf("%s-%x", legacyServiceName, h[:4])
}

// serviceNames returns deduplicated keychain service names to try, in order.
func serviceNames(homeDir, envConfigDir string) []string {
	defaultDir := filepath.Join(homeDir, ".claude")
	seen := make(map[string]bool)
	var names []string

	add := func(name string) {
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}

	// 1. Resolved config dir
	configDir := envConfigDir
	if configDir == "" {
		configDir = defaultDir
	}
	if strings.HasPrefix(configDir, "~/") {
		configDir = filepath.Join(homeDir, configDir[2:])
	} else if configDir == "~" {
		configDir = homeDir
	}
	add(serviceName(configDir, homeDir))

	// 2. Raw env var (may differ from resolved, but with tilde expanded)
	if envConfigDir != "" {
		rawDir := envConfigDir
		if strings.HasPrefix(rawDir, "~/") {
			rawDir = filepath.Join(homeDir, rawDir[2:])
		} else if rawDir == "~" {
			rawDir = homeDir
		}
		add(serviceName(rawDir, homeDir))
	}

	// 3. Legacy fallback
	add(legacyServiceName)

	return names
}
