//go:build darwin

package keychain

import (
	"fmt"
	"testing"
	"time"
)

func TestServiceName(t *testing.T) {
	home := "/Users/testuser"
	tests := []struct {
		name      string
		configDir string
		want      string
	}{
		{"default dir", "/Users/testuser/.claude", "Claude Code-credentials"},
		{"default dir trailing slash", "/Users/testuser/.claude/", "Claude Code-credentials"},
		{"custom dir", "/tmp/custom-claude", "Claude Code-credentials-"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serviceName(tt.configDir, home)
			if tt.configDir == "/tmp/custom-claude" {
				// Custom dirs get a hash suffix -- just check prefix
				if len(got) <= len("Claude Code-credentials-") {
					t.Errorf("serviceName(%q) = %q, want hash suffix", tt.configDir, got)
				}
			} else if got != tt.want {
				t.Errorf("serviceName(%q) = %q, want %q", tt.configDir, got, tt.want)
			}
		})
	}
}

func TestServiceNames(t *testing.T) {
	home := "/Users/testuser"

	t.Run("default config no env", func(t *testing.T) {
		names := serviceNames(home, "")
		if len(names) != 1 || names[0] != "Claude Code-credentials" {
			t.Errorf("got %v, want [Claude Code-credentials]", names)
		}
	})

	t.Run("deduplicates", func(t *testing.T) {
		names := serviceNames(home, "/Users/testuser/.claude")
		if len(names) != 1 {
			t.Errorf("got %v, want single entry (deduplicated)", names)
		}
	})

	t.Run("custom env adds entry", func(t *testing.T) {
		names := serviceNames(home, "/tmp/custom")
		if len(names) < 2 {
			t.Errorf("got %v, want at least 2 entries", names)
		}
		// Last entry should be the legacy fallback
		if names[len(names)-1] != "Claude Code-credentials" {
			t.Errorf("last entry = %q, want legacy fallback", names[len(names)-1])
		}
	})

	t.Run("tilde resolves to default", func(t *testing.T) {
		names := serviceNames(home, "~/.claude")
		if len(names) != 1 || names[0] != "Claude Code-credentials" {
			t.Errorf("got %v, want single [Claude Code-credentials] (tilde resolved)", names)
		}
	})
}

func TestParseKeychainData(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantTok string
		wantSub string
	}{
		{
			name:    "valid credentials",
			input:   `{"claudeAiOauth":{"accessToken":"tok123","subscriptionType":"claude_pro_2024"}}`,
			wantTok: "tok123",
			wantSub: "claude_pro_2024",
		},
		{
			name:    "empty token",
			input:   `{"claudeAiOauth":{"accessToken":"","subscriptionType":"pro"}}`,
			wantErr: true,
		},
		{
			name:    "missing oauth field",
			input:   `{"other":"data"}`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			input:   `not json`,
			wantErr: true,
		},
		{
			name:    "expired token",
			input:   `{"claudeAiOauth":{"accessToken":"tok","subscriptionType":"pro","expiresAt":1000}}`,
			wantErr: true,
		},
		{
			name:    "future expiration",
			input:   fmt.Sprintf(`{"claudeAiOauth":{"accessToken":"tok","subscriptionType":"pro","expiresAt":%d}}`, time.Now().UnixMilli()+60000),
			wantTok: "tok",
			wantSub: "pro",
		},
		{
			name:    "no expiration field",
			input:   `{"claudeAiOauth":{"accessToken":"tok","subscriptionType":"pro"}}`,
			wantTok: "tok",
			wantSub: "pro",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds, err := parseKeychainData([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if creds.AccessToken != tt.wantTok {
				t.Errorf("token = %q, want %q", creds.AccessToken, tt.wantTok)
			}
			if creds.SubscriptionType != tt.wantSub {
				t.Errorf("subscriptionType = %q, want %q", creds.SubscriptionType, tt.wantSub)
			}
		})
	}
}
