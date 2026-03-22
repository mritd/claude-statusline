package main

import (
	"os"
	"time"

	"github.com/mritd/claude-statusline/internal/config"
	"github.com/mritd/claude-statusline/internal/module"
	"github.com/mritd/claude-statusline/internal/render"
	"github.com/mritd/claude-statusline/internal/stdin"
	"github.com/mritd/claude-statusline/internal/transcript"
)

func main() {
	run()
}

func run() {
	cfg := config.Load(config.DefaultPath())

	data, err := stdin.Parse(os.Stdin)
	if err != nil {
		return
	}

	var td *transcript.Data
	if cfg.IsEnabled("tools") || cfg.IsEnabled("agents") || cfg.IsEnabled("todos") {
		if data.TranscriptPath != "" {
			td, _ = transcript.Parse(data.TranscriptPath)
		}
	}

	ctx := &module.Context{
		Stdin:      data,
		Transcript: td,
		Config:     cfg,
		CWD:        data.CWD,
	}

	filled, empty := cfg.BarChars()
	contextBar := render.NewContextBar(filled, empty)
	quotaBar := render.NewQuotaBar(filled, empty)

	registry := module.NewRegistry()

	// Context module - BarFunc injected to avoid import cycle
	mc := cfg.ModuleConfig("context")
	contextLimit := module.DefaultContextLimit
	if mc.ContextLimit != nil {
		contextLimit = *mc.ContextLimit
	}
	registry.Register(module.NewContextModule(mc.BarWidth, contextLimit, contextBar))

	// Usage module - QuotaBarFunc injected
	uc := cfg.ModuleConfig("usage")
	cacheTTL := 300 // 5 min, matches Anthropic usage API rate limit window
	failureTTL := 15
	if uc.CacheTTLSeconds > 0 {
		cacheTTL = uc.CacheTTLSeconds
	}
	if uc.FailureCacheTTLSeconds > 0 {
		failureTTL = uc.FailureCacheTTLSeconds
	}
	home, _ := os.UserHomeDir()
	cacheDir := home + "/.claude/plugins/claude-statusline"
	registry.Register(module.NewUsageModule(cacheDir,
		time.Duration(cacheTTL)*time.Second,
		time.Duration(failureTTL)*time.Second,
		quotaBar))

	// Git module
	registry.Register(module.NewGitModule())

	// Transcript modules
	registry.Register(module.NewToolsModule())
	registry.Register(module.NewAgentsModule())
	registry.Register(module.NewTodosModule())

	// Optional modules
	registry.Register(module.NewProjectModule())
	registry.Register(module.NewEnvironmentModule())

	output := render.Render(ctx, registry, cfg)
	if output != "" {
		render.Print(output)
	}
}
