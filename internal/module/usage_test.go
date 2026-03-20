package module

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
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

	m.writeCacheRateLimited(nil, "pro_monthly")
	existing := m.readExistingCache()
	if existing.RateLimitedCount != 1 {
		t.Fatalf("expected count=1, got %d", existing.RateLimitedCount)
	}
	if existing.RetryAfterUntil <= 0 {
		t.Fatal("expected RetryAfterUntil to be set")
	}

	m.writeCacheRateLimited(existing, "pro_monthly")
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
