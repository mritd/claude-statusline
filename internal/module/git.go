package module

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mritd/claude-statusline/internal/debug"
)

const gitTimeout = time.Second

type GitModule struct {
	branch string
	dirty  bool
	ahead  int
	behind int
}

func NewGitModule() *GitModule {
	return &GitModule{}
}

func (m *GitModule) Name() string { return "git" }

func (m *GitModule) Collect(ctx *Context) error {
	cwd := ctx.CWD
	if cwd == "" {
		return fmt.Errorf("no cwd")
	}

	var (
		wg           sync.WaitGroup
		branch       string
		porcelain    string
		revlist      string
		branchErr    error
		porcelainErr error
		revlistErr   error
	)

	wg.Add(3)
	go func() {
		defer wg.Done()
		branch, branchErr = gitCmd(cwd, "rev-parse", "--abbrev-ref", "HEAD")
	}()
	go func() {
		defer wg.Done()
		porcelain, porcelainErr = gitCmd(cwd, "--no-optional-locks", "status", "--porcelain")
	}()
	go func() {
		defer wg.Done()
		revlist, revlistErr = gitCmd(cwd, "rev-list", "--left-right", "--count", "@{upstream}...HEAD")
	}()
	wg.Wait()

	if branchErr != nil {
		return branchErr
	}

	m.branch = strings.TrimSpace(branch)

	if porcelainErr == nil {
		m.dirty = parsePorcelainDirty(porcelain)
	}

	if revlistErr == nil {
		m.ahead, m.behind = parseAheadBehind(revlist)
	} else {
		debug.Log("git", "rev-list error (no upstream?): %v", revlistErr)
	}

	return nil
}

func (m *GitModule) Vars(ctx *Context) map[string]string {
	const (
		green  = "\x1b[1;32m" // bold green - branch (prezto sorin)
		yellow = "\x1b[33m"   // yellow - dirty indicator
		reset  = "\x1b[0m"
	)

	vars := map[string]string{
		"branch": green + m.branch + reset,
		"dirty":  "",
		"ahead":  "",
		"behind": "",
	}
	if m.dirty {
		vars["dirty"] = yellow + "*" + reset
	}
	if m.ahead > 0 {
		vars["ahead"] = fmt.Sprintf("↑%d", m.ahead)
	}
	if m.behind > 0 {
		vars["behind"] = fmt.Sprintf("↓%d", m.behind)
	}
	return vars
}

func (m *GitModule) DefaultFormat() string {
	return "{branch}{dirty}"
}

func parsePorcelainDirty(output string) bool {
	return strings.TrimSpace(output) != ""
}

func parseAheadBehind(output string) (ahead, behind int) {
	parts := strings.Fields(strings.TrimSpace(output))
	if len(parts) != 2 {
		return 0, 0
	}
	// git rev-list --left-right --count @{upstream}...HEAD
	// outputs: <upstream_count>\t<head_count> = <behind>\t<ahead>
	behind, _ = strconv.Atoi(parts[0])
	ahead, _ = strconv.Atoi(parts[1])
	return
}

func gitCmd(cwd string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = cwd
	out, err := cmd.Output()
	return string(out), err
}
