package module

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mritd/claude-statusline/internal/debug"
	"github.com/mritd/claude-statusline/internal/keychain"
	"github.com/mritd/claude-statusline/internal/ansi"
)

// QuotaBarFunc renders a quota progress bar for a given percentage and width.
type QuotaBarFunc func(pct, width int) string

// UsageModule fetches and displays Anthropic API usage quotas.
type UsageModule struct {
	cacheDir   string
	cacheTTL   time.Duration
	failureTTL time.Duration
	data       *usageData
	quotaBar   QuotaBarFunc
}

type usageData struct {
	FiveHour        int           `json:"fiveHour"`
	SevenDay        int           `json:"sevenDay"`
	FiveHourResetAt time.Time     `json:"fiveHourResetAt"`
	SevenDayResetAt time.Time     `json:"sevenDayResetAt"`
	PlanName        string        `json:"planName"`
	Syncing         bool          `json:"syncing"`
	APIError        int           `json:"apiError,omitempty"`
	RetryAfter      time.Duration `json:"-"`
}

type usageCache struct {
	Data             *usageData `json:"data"`
	Timestamp        int64      `json:"timestamp"`
	RateLimitedCount int        `json:"rateLimitedCount"`
	RetryAfterUntil  int64      `json:"retryAfterUntil"`
	LastGoodData     *usageData `json:"lastGoodData"`
	IsFailure        bool       `json:"isFailure,omitempty"`
}

// apiResponse mirrors the Anthropic usage API response.
type apiResponse struct {
	FiveHour struct {
		Utilization float64 `json:"utilization"`
		ResetsAt    string  `json:"resets_at"`
	} `json:"five_hour"`
	SevenDay struct {
		Utilization float64 `json:"utilization"`
		ResetsAt    string  `json:"resets_at"`
	} `json:"seven_day"`
}

const (
	usageAPIURL    = "https://api.anthropic.com/api/oauth/usage"
	maxBackoff     = 5 * time.Minute
	lockStaleSecs  = 30
	defaultBarSize = 10
)

// NewUsageModule creates a usage module with the given cache directory and TTLs.
func NewUsageModule(cacheDir string, cacheTTL, failureTTL time.Duration, quotaBar QuotaBarFunc) *UsageModule {
	if cacheTTL <= 0 {
		cacheTTL = 5 * time.Minute // matches Anthropic usage API rate limit window
	}
	if failureTTL <= 0 {
		failureTTL = 15 * time.Second
	}
	return &UsageModule{
		cacheDir:   cacheDir,
		cacheTTL:   cacheTTL,
		failureTTL: failureTTL,
		quotaBar:   quotaBar,
	}
}

func (m *UsageModule) Name() string { return "usage" }

func (m *UsageModule) Collect(ctx *Context) error {
	// Skip if custom API endpoint is configured
	if os.Getenv("ANTHROPIC_BASE_URL") != "" || os.Getenv("ANTHROPIC_API_KEY") != "" {
		debug.Log("usage", "custom API endpoint detected, skipping usage")
		return nil
	}

	// Read cache once, reuse throughout
	existing := m.readExistingCache()

	// Try fresh cache first
	if existing != nil && existing.Data != nil {
		age := time.Since(time.UnixMilli(existing.Timestamp))
		ttl := m.cacheTTL
		if existing.IsFailure {
			ttl = m.failureTTL
		}
		if age <= ttl {
			debug.Log("usage", "using cached data")
			m.data = existing.Data
			return nil
		}
	}

	// Try to acquire lock for fetching
	if !m.tryLock() {
		debug.Log("usage", "lock busy, using last good data")
		if lastGood := lastGoodFrom(existing); lastGood != nil {
			lastGood.Syncing = true
			m.data = lastGood
		}
		return nil
	}
	defer m.unlock()

	// Read credentials
	creds, err := keychain.Read()
	if err != nil {
		debug.Log("usage", "keychain read: %v", err)
		m.writeCacheFailureFrom(existing)
		return nil
	}

	// Check rate limit backoff
	if existing != nil && existing.RetryAfterUntil > 0 {
		if time.Now().UnixMilli() < existing.RetryAfterUntil {
			debug.Log("usage", "in backoff period, using last good data")
			if existing.LastGoodData != nil {
				existing.LastGoodData.Syncing = true
				m.data = existing.LastGoodData
			}
			return nil
		}
	}

	// Fetch from API
	usage, statusCode, retryAfter, err := m.fetchAPI(creds.AccessToken)
	if err != nil {
		debug.Log("usage", "API fetch: %v", err)
		if statusCode == 429 {
			m.writeCacheRateLimited(existing, creds.SubscriptionType)
		} else {
			m.writeCacheFailureFrom(existing)
		}
		if existing != nil && existing.LastGoodData != nil {
			existing.LastGoodData.Syncing = true
			existing.LastGoodData.APIError = statusCode
			existing.LastGoodData.RetryAfter = retryAfter
			m.data = existing.LastGoodData
		} else if statusCode > 0 {
			m.data = &usageData{APIError: statusCode, RetryAfter: retryAfter}
		}
		return nil
	}

	// Parse response into usageData
	data := &usageData{
		FiveHour: clampPercent(usage.FiveHour.Utilization),
		SevenDay: clampPercent(usage.SevenDay.Utilization),
		PlanName: planName(creds.SubscriptionType),
	}
	if t, err := time.Parse(time.RFC3339, usage.FiveHour.ResetsAt); err == nil {
		data.FiveHourResetAt = t
	}
	if t, err := time.Parse(time.RFC3339, usage.SevenDay.ResetsAt); err == nil {
		data.SevenDayResetAt = t
	}

	m.writeCacheSuccess(data)
	m.data = data
	return nil
}

func (m *UsageModule) Vars(ctx *Context) map[string]string {
	if m.data == nil {
		return nil
	}

	d := m.data
	vars := map[string]string{
		"plan":     d.PlanName,
		"5h_pct":   fmt.Sprintf("%d%%", d.FiveHour),
		"7d_pct":   fmt.Sprintf("%d%%", d.SevenDay),
		"5h_bar":   "",
		"7d_bar":   "",
		"5h_reset": "",
		"7d_reset": "",
		"syncing":  "",
	}

	if m.quotaBar != nil {
		vars["5h_bar"] = m.quotaBar(d.FiveHour, defaultBarSize)
		vars["7d_bar"] = m.quotaBar(d.SevenDay, defaultBarSize)
	}

	if !d.FiveHourResetAt.IsZero() {
		remaining := time.Until(d.FiveHourResetAt)
		if remaining > 0 {
			vars["5h_reset"] = "(" + formatResetTime(remaining) + ")"
		}
	}
	if !d.SevenDayResetAt.IsZero() {
		remaining := time.Until(d.SevenDayResetAt)
		if remaining > 0 {
			vars["7d_reset"] = "(" + formatResetTime(remaining) + ")"
		}
	}

	if d.Syncing {
		vars["syncing"] = " " + ansi.Dim("⟳ syncing...")
	}

	if d.APIError > 0 {
		errText := fmt.Sprintf("API %d*", d.APIError)
		if d.RetryAfter > 0 {
			errText = fmt.Sprintf("API %d* (%s)", d.APIError, formatResetTime(d.RetryAfter))
		}
		vars["api_error"] = ansi.Yellow(errText)
	} else {
		vars["api_error"] = ""
	}

	return vars
}

func (m *UsageModule) DefaultFormat() string {
	if m.data != nil && m.data.APIError > 0 && m.data.FiveHour == 0 && m.data.SevenDay == 0 {
		return "{api_error}"
	}
	return "Usage {5h_bar} {5h_pct} {5h_reset} | {7d_bar} {7d_pct} {7d_reset}{syncing} {api_error}"
}

// Cache file paths

func (m *UsageModule) cachePath() string {
	return filepath.Join(m.cacheDir, ".usage-cache.json")
}

func (m *UsageModule) lockPath() string {
	return filepath.Join(m.cacheDir, ".usage-cache.lock")
}

// Cache read/write operations

func lastGoodFrom(cache *usageCache) *usageData {
	if cache == nil {
		return nil
	}
	if cache.LastGoodData != nil {
		return cache.LastGoodData
	}
	return cache.Data
}

func (m *UsageModule) readExistingCache() *usageCache {
	data, err := os.ReadFile(m.cachePath())
	if err != nil {
		return nil
	}
	var cache usageCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil
	}
	return &cache
}

func (m *UsageModule) writeCache(cache *usageCache) {
	data, err := json.Marshal(cache)
	if err != nil {
		debug.Log("usage", "marshal cache: %v", err)
		return
	}
	if err := os.MkdirAll(m.cacheDir, 0700); err != nil {
		debug.Log("usage", "mkdir cache dir: %v", err)
		return
	}
	_ = os.WriteFile(m.cachePath(), data, 0600)
}

func (m *UsageModule) writeCacheSuccess(data *usageData) {
	cache := &usageCache{
		Data:         data,
		Timestamp:    time.Now().UnixMilli(),
		LastGoodData: data,
	}
	m.writeCache(cache)
}

func (m *UsageModule) writeCacheFailureFrom(existing *usageCache) {
	cache := &usageCache{
		Timestamp: time.Now().UnixMilli(),
		IsFailure: true,
	}
	if existing != nil {
		cache.LastGoodData = existing.LastGoodData
		if cache.LastGoodData == nil {
			cache.LastGoodData = existing.Data
		}
	}
	if cache.LastGoodData != nil {
		syncing := *cache.LastGoodData
		syncing.Syncing = true
		cache.Data = &syncing
	}
	m.writeCache(cache)
}

func (m *UsageModule) writeCacheRateLimited(existing *usageCache, subscriptionType string) {
	count := 1
	if existing != nil {
		count = existing.RateLimitedCount + 1
	}

	backoff := backoffDuration(count)
	retryUntil := time.Now().Add(backoff).UnixMilli()

	cache := &usageCache{
		Timestamp:        time.Now().UnixMilli(),
		RateLimitedCount: count,
		RetryAfterUntil:  retryUntil,
	}
	if existing != nil {
		cache.LastGoodData = existing.LastGoodData
		if cache.LastGoodData == nil {
			cache.LastGoodData = existing.Data
		}
	}
	if cache.LastGoodData != nil {
		syncing := *cache.LastGoodData
		syncing.Syncing = true
		cache.Data = &syncing
	}
	m.writeCache(cache)
}

// File lock for coordinating between short-lived processes

func (m *UsageModule) tryLock() bool {
	_ = os.MkdirAll(m.cacheDir, 0700)
	lockFile := m.lockPath()

	// Check for stale lock
	if info, err := os.Stat(lockFile); err == nil {
		if time.Since(info.ModTime()) > time.Duration(lockStaleSecs)*time.Second {
			debug.Log("usage", "removing stale lock")
			_ = os.Remove(lockFile)
		}
	}

	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

func (m *UsageModule) unlock() {
	_ = os.Remove(m.lockPath())
}

// API fetch

func (m *UsageModule) fetchAPI(token string) (*apiResponse, int, time.Duration, error) {
	req, err := http.NewRequest("GET", usageAPIURL, nil)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("User-Agent", "claude-code/2.1")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("http request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, 0, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var retryAfter time.Duration
		if secs := parseRetryAfter(resp.Header.Get("Retry-After")); secs > 0 {
			retryAfter = time.Duration(secs) * time.Second
		}
		return nil, resp.StatusCode, retryAfter, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	var result apiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, resp.StatusCode, 0, fmt.Errorf("parse response: %w", err)
	}

	return &result, resp.StatusCode, 0, nil
}

// Helper functions

func parseRetryAfter(val string) int {
	if val == "" {
		return 0
	}
	secs, err := strconv.Atoi(val)
	if err != nil || secs < 0 {
		return 0
	}
	return secs
}

func backoffDuration(count int) time.Duration {
	if count <= 0 {
		count = 1
	}
	d := 60 * time.Second
	for i := 1; i < count; i++ {
		d *= 2
	}
	if d > maxBackoff {
		d = maxBackoff
	}
	return d
}

func planName(sub string) string {
	s := strings.ToLower(sub)
	switch {
	case strings.Contains(s, "max"):
		return "Max"
	case strings.Contains(s, "pro"):
		return "Pro"
	case strings.Contains(s, "team"):
		return "Team"
	default:
		return ""
	}
}

func formatResetTime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, minutes)
	default:
		return fmt.Sprintf("%dm", minutes)
	}
}

// clampPercent rounds a utilization value (0-100 from API) to int, clamped to [0, 100].
func clampPercent(v float64) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return int(math.Round(v))
}
