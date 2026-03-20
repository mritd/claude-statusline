package module

import (
	"testing"
)

func TestParsePorcelainDirty(t *testing.T) {
	output := " M file1.go\n?? file2.go\nA  file3.go\n D file4.go\n"
	if !parsePorcelainDirty(output) {
		t.Fatal("expected dirty")
	}
}

func TestParsePorcelainClean(t *testing.T) {
	if parsePorcelainDirty("") {
		t.Fatal("expected clean")
	}
}

func TestParseAheadBehind(t *testing.T) {
	// git rev-list --left-right --count @{upstream}...HEAD
	// outputs: <behind>\t<ahead>
	ahead, behind := parseAheadBehind("3\t1\n")
	if ahead != 1 || behind != 3 {
		t.Fatalf("expected ahead=1/behind=3, got %d/%d", ahead, behind)
	}
}

func TestParseAheadBehindEmpty(t *testing.T) {
	ahead, behind := parseAheadBehind("")
	if ahead != 0 || behind != 0 {
		t.Fatalf("expected 0/0, got %d/%d", ahead, behind)
	}
}
