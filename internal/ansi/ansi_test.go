package ansi

import "testing"

func TestContextColor(t *testing.T) {
	tests := []struct {
		pct  int
		want string
	}{
		{50, GREEN},
		{70, YELLOW},
		{85, RED},
		{95, RED},
	}
	for _, tt := range tests {
		got := ContextColor(tt.pct)
		if got != tt.want {
			t.Errorf("ContextColor(%d) = %q, want %q", tt.pct, got, tt.want)
		}
	}
}

func TestQuotaColor(t *testing.T) {
	tests := []struct {
		pct  int
		want string
	}{
		{50, BRIGHT_BLUE},
		{75, BRIGHT_MAGENTA},
		{90, RED},
	}
	for _, tt := range tests {
		got := QuotaColor(tt.pct)
		if got != tt.want {
			t.Errorf("QuotaColor(%d) = %q, want %q", tt.pct, got, tt.want)
		}
	}
}
