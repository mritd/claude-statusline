package module

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mritd/claude-statusline/internal/keychain"
	"github.com/mritd/claude-statusline/internal/stdin"
)

func TestUsageCacheFresh(t *testing.T) {
	dir := t.TempDir()
	cache := &usageCache{
		Data:      &usageData{FiveHour: 25, SevenDay: 35, FiveHourResetAt: time.Now().Add(time.Hour)},
		Timestamp: time.Now().UnixMilli(),
	}
	data, _ := json.Marshal(cache)
	_ = os.WriteFile(filepath.Join(dir, ".usage-cache.json"), data, 0600)

	m := &UsageModule{cacheDir: dir, cacheTTL: 5 * time.Minute}
	existing := m.readExistingCache()
	if existing == nil || existing.Data == nil {
		t.Fatal("expected fresh cache")
	}
	if existing.Data.FiveHour != 25 {
		t.Fatalf("expected 25, got %d", existing.Data.FiveHour)
	}
}

func TestUsageCacheExpired(t *testing.T) {
	dir := t.TempDir()
	cache := &usageCache{
		Data:      &usageData{FiveHour: 25},
		Timestamp: time.Now().Add(-6 * time.Minute).UnixMilli(),
	}
	data, _ := json.Marshal(cache)
	_ = os.WriteFile(filepath.Join(dir, ".usage-cache.json"), data, 0600)

	m := &UsageModule{cacheDir: dir, cacheTTL: 5 * time.Minute}
	existing := m.readExistingCache()
	age := time.Since(time.UnixMilli(existing.Timestamp))
	if age <= m.cacheTTL {
		t.Fatal("expected expired cache")
	}
}

func TestBackoffDuration(t *testing.T) {
	tests := []struct {
		count int
		want  time.Duration
	}{
		{0, 60 * time.Second},
		{1, 60 * time.Second},
		{2, 120 * time.Second},
		{3, 240 * time.Second},
		{4, 5 * time.Minute},
		{5, 5 * time.Minute},
		{10, 5 * time.Minute},
		{-1, 60 * time.Second},
	}
	for _, tt := range tests {
		got := backoffDuration(tt.count)
		if got != tt.want {
			t.Errorf("backoffDuration(%d) = %v, want %v", tt.count, got, tt.want)
		}
	}
}

func TestPlanName(t *testing.T) {
	tests := []struct {
		sub  string
		want string
	}{
		{"claude_max_monthly", "Max"},
		{"pro_annual", "Pro"},
		{"team_enterprise", "Team"},
		{"api", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := planName(tt.sub)
		if got != tt.want {
			t.Errorf("planName(%q) = %q, want %q", tt.sub, got, tt.want)
		}
	}
}

func TestFormatResetTime(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Minute, "30m"},
		{90 * time.Minute, "1h 30m"},
		{25 * time.Hour, "1d 1h"},
	}
	for _, tt := range tests {
		got := formatResetTime(tt.d)
		if got != tt.want {
			t.Errorf("formatResetTime(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestUsageCacheFailureUsesFailureTTL(t *testing.T) {
	dir := t.TempDir()
	cache := &usageCache{
		Data:      &usageData{FiveHour: 25, Syncing: true},
		Timestamp: time.Now().Add(-20 * time.Second).UnixMilli(),
		IsFailure: true,
	}
	data, _ := json.Marshal(cache)
	_ = os.WriteFile(filepath.Join(dir, ".usage-cache.json"), data, 0600)

	m := &UsageModule{cacheDir: dir, cacheTTL: 5 * time.Minute, failureTTL: 15 * time.Second}
	existing := m.readExistingCache()
	age := time.Since(time.UnixMilli(existing.Timestamp))
	ttl := m.failureTTL
	if age <= ttl {
		t.Fatal("expected failure cache to expire at failureTTL (15s)")
	}
}

func TestUsageCacheFailureFreshWithinTTL(t *testing.T) {
	dir := t.TempDir()
	cache := &usageCache{
		Data:      &usageData{FiveHour: 25, Syncing: true},
		Timestamp: time.Now().Add(-10 * time.Second).UnixMilli(),
		IsFailure: true,
	}
	data, _ := json.Marshal(cache)
	_ = os.WriteFile(filepath.Join(dir, ".usage-cache.json"), data, 0600)

	m := &UsageModule{cacheDir: dir, cacheTTL: 5 * time.Minute, failureTTL: 15 * time.Second}
	existing := m.readExistingCache()
	age := time.Since(time.UnixMilli(existing.Timestamp))
	ttl := m.failureTTL
	if age > ttl {
		t.Fatal("expected failure cache to be fresh within failureTTL")
	}
	if existing.Data.FiveHour != 25 {
		t.Fatalf("expected 25, got %d", existing.Data.FiveHour)
	}
}

func TestWriteCacheFailureSetsIsFailure(t *testing.T) {
	dir := t.TempDir()
	m := &UsageModule{cacheDir: dir, cacheTTL: 5 * time.Minute, failureTTL: 15 * time.Second}

	m.writeCacheSuccess(&usageData{FiveHour: 50, SevenDay: 20, PlanName: "Pro"})
	m.writeCacheFailureFrom(m.readExistingCache())

	existing := m.readExistingCache()
	if !existing.IsFailure {
		t.Fatal("expected IsFailure to be true")
	}
	if existing.LastGoodData == nil || existing.LastGoodData.FiveHour != 50 {
		t.Fatal("expected LastGoodData to be preserved")
	}
	if existing.Data == nil || !existing.Data.Syncing {
		t.Fatal("expected Data to have Syncing=true")
	}
}

func TestWriteCacheSuccess(t *testing.T) {
	dir := t.TempDir()
	m := &UsageModule{cacheDir: dir, cacheTTL: 5 * time.Minute, failureTTL: 15 * time.Second}

	data := &usageData{FiveHour: 42, SevenDay: 18, PlanName: "Pro"}
	m.writeCacheSuccess(data)

	existing := m.readExistingCache()
	if existing == nil || existing.Data == nil {
		t.Fatal("expected cache to be readable after writeCacheSuccess")
	}
	if existing.Data.FiveHour != 42 || existing.Data.SevenDay != 18 {
		t.Fatalf("unexpected data: 5h=%d, 7d=%d", existing.Data.FiveHour, existing.Data.SevenDay)
	}
	if existing.LastGoodData == nil || existing.LastGoodData.FiveHour != 42 {
		t.Fatal("expected lastGoodData to be set")
	}
	if existing.IsFailure {
		t.Fatal("success cache should not have IsFailure set")
	}
}

func TestWriteCacheRateLimited(t *testing.T) {
	dir := t.TempDir()
	m := &UsageModule{cacheDir: dir, cacheTTL: 5 * time.Minute, failureTTL: 15 * time.Second}

	m.writeCacheRateLimited(nil)
	existing := m.readExistingCache()
	if existing.RateLimitedCount != 1 {
		t.Fatalf("expected count=1, got %d", existing.RateLimitedCount)
	}
	if existing.RetryAfterUntil <= 0 {
		t.Fatal("expected RetryAfterUntil to be set")
	}

	m.writeCacheRateLimited(existing)
	existing2 := m.readExistingCache()
	if existing2.RateLimitedCount != 2 {
		t.Fatalf("expected count=2, got %d", existing2.RateLimitedCount)
	}
	if existing2.RetryAfterUntil <= existing.RetryAfterUntil {
		t.Fatal("expected longer backoff on second rate limit")
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		val  string
		want int
	}{
		{"120", 120},
		{"0", 0},
		{"", 0},
		{"-5", 0},
		{"abc", 0},
		{"3600", 3600},
	}
	for _, tt := range tests {
		got := parseRetryAfter(tt.val)
		if got != tt.want {
			t.Errorf("parseRetryAfter(%q) = %d, want %d", tt.val, got, tt.want)
		}
	}
}

func TestUsageFromStdin(t *testing.T) {
	reset := float64(time.Now().Add(2 * time.Hour).Unix())
	tests := []struct {
		name     string
		in       *stdin.Data
		wantNil  bool
		want5h   int
		want7d   int
		want5hAt bool
	}{
		{name: "nil stdin", in: nil, wantNil: true},
		{name: "no rate_limits", in: &stdin.Data{}, wantNil: true},
		{name: "empty rate_limits", in: &stdin.Data{RateLimits: &stdin.RateLimits{}}, wantNil: true},
		{
			name: "both windows",
			in: &stdin.Data{RateLimits: &stdin.RateLimits{
				FiveHour: &stdin.RateWindow{UsedPercentage: 12.6, ResetsAt: reset},
				SevenDay: &stdin.RateWindow{UsedPercentage: 40},
			}},
			want5h: 13, want7d: 40, want5hAt: true,
		},
		{
			name: "five hour window reset",
			in: &stdin.Data{RateLimits: &stdin.RateLimits{
				SevenDay: &stdin.RateWindow{UsedPercentage: 150},
			}},
			want5h: 0, want7d: 100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := usageFromStdin(tt.in)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil || got.FiveHour != tt.want5h || got.SevenDay != tt.want7d {
				t.Fatalf("got %+v, want 5h=%d 7d=%d", got, tt.want5h, tt.want7d)
			}
			if got.FiveHourResetAt.IsZero() == tt.want5hAt {
				t.Fatalf("unexpected 5h reset time: %v", got.FiveHourResetAt)
			}
		})
	}
}

func TestUsageCollectPrefersStdin(t *testing.T) {
	dir := t.TempDir()
	m := NewUsageModule(dir, 0, 0, 0, nil)
	ctx := &Context{Stdin: &stdin.Data{RateLimits: &stdin.RateLimits{
		FiveHour: &stdin.RateWindow{UsedPercentage: 7},
	}}}
	if err := m.Collect(ctx); err != nil {
		t.Fatal(err)
	}
	if m.data == nil || m.data.FiveHour != 7 {
		t.Fatalf("expected stdin data, got %+v", m.data)
	}
	if _, err := os.Stat(filepath.Join(dir, ".usage-cache.json")); !os.IsNotExist(err) {
		t.Fatal("stdin path should not touch the cache")
	}
}

// newTestUsage returns a usage module whose credential read and API fetch
// fail the test unless a case overrides them, so no test can reach the real
// keychain or network.
func newTestUsage(t *testing.T, dir string) *UsageModule {
	t.Helper()
	t.Setenv("ANTHROPIC_BASE_URL", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	m := NewUsageModule(dir, 0, 0, 0, nil)
	m.readCreds = func() (*keychain.Credentials, error) {
		t.Fatal("unexpected credential read")
		return nil, nil
	}
	m.fetch = func(string) (*apiResponse, int, time.Duration, error) {
		t.Fatal("unexpected API fetch")
		return nil, 0, 0, nil
	}
	return m
}

func writeTestCache(t *testing.T, dir string, cache *usageCache) {
	t.Helper()
	raw, _ := json.Marshal(cache)
	if err := os.WriteFile(filepath.Join(dir, ".usage-cache.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
}

func okCreds() (*keychain.Credentials, error) {
	return &keychain.Credentials{AccessToken: "tok", SubscriptionType: "claude_max"}, nil
}

func TestUsageCollectBackoffUsesLastGood(t *testing.T) {
	dir := t.TempDir()
	writeTestCache(t, dir, &usageCache{
		Timestamp:        time.Now().Add(-10 * time.Minute).UnixMilli(),
		RateLimitedCount: 1,
		RetryAfterUntil:  time.Now().Add(time.Minute).UnixMilli(),
		LastGoodData:     &usageData{FiveHour: 33},
	})
	before, _ := os.ReadFile(filepath.Join(dir, ".usage-cache.json"))

	m := newTestUsage(t, dir)
	if err := m.Collect(&Context{}); err != nil {
		t.Fatal(err)
	}
	if m.data == nil || m.data.FiveHour != 33 || !m.data.Syncing {
		t.Fatalf("expected syncing last good data, got %+v", m.data)
	}
	after, _ := os.ReadFile(filepath.Join(dir, ".usage-cache.json"))
	if string(after) != string(before) {
		t.Fatal("backoff path should leave the cache untouched")
	}
	if _, err := os.Stat(filepath.Join(dir, ".usage-cache.lock")); !os.IsNotExist(err) {
		t.Fatal("backoff path should not take the fetch lock")
	}
}

func TestUsageCollectFallbackStates(t *testing.T) {
	stale := func(lastGood *usageData) *usageCache {
		return &usageCache{
			Timestamp:    time.Now().Add(-10 * time.Minute).UnixMilli(),
			Data:         lastGood,
			LastGoodData: lastGood,
		}
	}
	tests := []struct {
		name      string
		cache     *usageCache
		lockBusy  bool
		creds     func() (*keychain.Credentials, error)
		fetch     func(string) (*apiResponse, int, time.Duration, error)
		want5h    int
		wantSync  bool
		wantErr   int
		wantNil   bool
		checkFile func(t *testing.T, c *usageCache)
	}{
		{
			name:   "fresh cache",
			cache:  &usageCache{Timestamp: time.Now().UnixMilli(), Data: &usageData{FiveHour: 11}},
			want5h: 11,
		},
		{
			name:     "lock busy",
			cache:    stale(&usageData{FiveHour: 22}),
			lockBusy: true,
			want5h:   22, wantSync: true,
		},
		{
			name:  "keychain failure",
			cache: stale(&usageData{FiveHour: 44}),
			creds: func() (*keychain.Credentials, error) { return nil, errors.New("locked") },
			// Display keeps nothing new; the failure cache carries last good data.
			wantNil: true,
			checkFile: func(t *testing.T, c *usageCache) {
				if !c.IsFailure || c.LastGoodData == nil || c.LastGoodData.FiveHour != 44 {
					t.Fatalf("expected failure cache with last good data, got %+v", c)
				}
			},
		},
		{
			name:  "api server error",
			cache: stale(&usageData{FiveHour: 55}),
			creds: okCreds,
			fetch: func(string) (*apiResponse, int, time.Duration, error) {
				return nil, 500, 0, errors.New("boom")
			},
			want5h: 55, wantSync: true, wantErr: 500,
			checkFile: func(t *testing.T, c *usageCache) {
				if !c.IsFailure || c.RetryAfterUntil != 0 {
					t.Fatalf("expected plain failure cache, got %+v", c)
				}
			},
		},
		{
			name:  "api rate limited",
			cache: stale(&usageData{FiveHour: 66}),
			creds: okCreds,
			fetch: func(string) (*apiResponse, int, time.Duration, error) {
				return nil, 429, 30 * time.Second, errors.New("slow down")
			},
			want5h: 66, wantSync: true, wantErr: 429,
			checkFile: func(t *testing.T, c *usageCache) {
				if c.RateLimitedCount != 1 || c.RetryAfterUntil <= time.Now().UnixMilli() {
					t.Fatalf("expected backoff cache, got %+v", c)
				}
			},
		},
		{
			name:  "api success",
			cache: stale(&usageData{FiveHour: 1}),
			creds: okCreds,
			fetch: func(string) (*apiResponse, int, time.Duration, error) {
				var r apiResponse
				r.FiveHour.Utilization = 77
				r.SevenDay.Utilization = 12
				return &r, 200, 0, nil
			},
			want5h: 77,
			checkFile: func(t *testing.T, c *usageCache) {
				if c.IsFailure || c.LastGoodData == nil || c.LastGoodData.FiveHour != 77 || c.LastGoodData.PlanName != "Max" {
					t.Fatalf("expected success cache, got %+v", c)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeTestCache(t, dir, tt.cache)
			m := newTestUsage(t, dir)
			if tt.creds != nil {
				m.readCreds = tt.creds
			}
			if tt.fetch != nil {
				m.fetch = tt.fetch
			}
			if tt.lockBusy {
				_ = os.WriteFile(filepath.Join(dir, ".usage-cache.lock"), nil, 0600)
			}

			if err := m.Collect(&Context{}); err != nil {
				t.Fatal(err)
			}
			if tt.wantNil {
				if m.data != nil {
					t.Fatalf("expected no data, got %+v", m.data)
				}
			} else if m.data == nil || m.data.FiveHour != tt.want5h || m.data.Syncing != tt.wantSync || m.data.APIError != tt.wantErr {
				t.Fatalf("got %+v, want 5h=%d syncing=%v apiError=%d", m.data, tt.want5h, tt.wantSync, tt.wantErr)
			}
			if tt.checkFile != nil {
				tt.checkFile(t, m.readExistingCache())
			}
			if !tt.lockBusy {
				if _, err := os.Stat(filepath.Join(dir, ".usage-cache.lock")); !os.IsNotExist(err) {
					t.Fatal("lock should be released")
				}
			}
		})
	}
}

func TestUsageBarWidth(t *testing.T) {
	var gotWidth int
	bar := func(pct, width int) string { gotWidth = width; return strings.Repeat("#", width) }
	m := NewUsageModule(t.TempDir(), 0, 0, 4, bar)
	m.data = &usageData{FiveHour: 50, SevenDay: 50}
	vars := m.Vars(&Context{})
	if gotWidth != 4 || vars["5h_bar"] != "####" {
		t.Fatalf("expected bar width 4, got %d (%q)", gotWidth, vars["5h_bar"])
	}
}
