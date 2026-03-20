package module

import (
	"strings"

	"github.com/mritd/claude-statusline/internal/config"
	"github.com/mritd/claude-statusline/internal/debug"
	"github.com/mritd/claude-statusline/internal/stdin"
	"github.com/mritd/claude-statusline/internal/transcript"
)

type Context struct {
	Stdin      *stdin.Data
	Transcript *transcript.Data
	Config     *config.Config
	CWD        string
}

type Module interface {
	Name() string
	Collect(ctx *Context) error
	Vars(ctx *Context) map[string]string
	DefaultFormat() string
}

type Registry struct {
	modules map[string]Module
}

func NewRegistry() *Registry {
	return &Registry{modules: make(map[string]Module)}
}

func (r *Registry) Register(m Module) {
	r.modules[m.Name()] = m
}

func (r *Registry) Enabled(names []string) []Module {
	var result []Module
	for _, name := range names {
		if m, ok := r.modules[name]; ok {
			result = append(result, m)
		}
	}
	return result
}

func ExpandFormat(format string, vars map[string]string) string {
	var b strings.Builder
	i := 0
	for i < len(format) {
		if format[i] == '}' && i+1 < len(format) && format[i+1] == '}' {
			b.WriteByte('}')
			i += 2
			continue
		}
		if format[i] == '{' {
			if i+1 < len(format) && format[i+1] == '{' {
				b.WriteByte('{')
				i += 2
				continue
			}
			end := strings.IndexByte(format[i+1:], '}')
			if end < 0 {
				b.WriteByte(format[i])
				i++
				continue
			}
			key := format[i+1 : i+1+end]
			if val, ok := vars[key]; ok {
				b.WriteString(val)
			} else {
				debug.Log("format", "unknown var {%s}", key)
			}
			i = i + 1 + end + 1
		} else {
			b.WriteByte(format[i])
			i++
		}
	}
	return b.String()
}
