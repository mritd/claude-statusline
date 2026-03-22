package module

import (
	"path/filepath"
)

type ProjectModule struct{}

func NewProjectModule() *ProjectModule              { return &ProjectModule{} }
func (m *ProjectModule) Name() string               { return "project" }
func (m *ProjectModule) Collect(ctx *Context) error { return nil }

func (m *ProjectModule) Vars(ctx *Context) map[string]string {
	model := ""
	if ctx.Stdin != nil {
		model = ctx.Stdin.Model.DisplayName
	}
	path := ""
	if ctx.CWD != "" {
		path = filepath.Base(ctx.CWD)
	}
	if model == "" && path == "" {
		return nil
	}
	return map[string]string{
		"model": model,
		"plan":  "",
		"path":  path,
	}
}

func (m *ProjectModule) DefaultFormat() string {
	return "[{model} | {plan}] {path}"
}
