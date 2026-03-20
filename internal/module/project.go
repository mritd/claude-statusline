package module

import (
	"path/filepath"
)

type ProjectModule struct{}

func NewProjectModule() *ProjectModule              { return &ProjectModule{} }
func (m *ProjectModule) Name() string               { return "project" }
func (m *ProjectModule) Collect(ctx *Context) error { return nil }

func (m *ProjectModule) Vars(ctx *Context) map[string]string {
	vars := map[string]string{
		"model": "",
		"plan":  "",
		"path":  "",
	}
	if ctx.Stdin != nil {
		vars["model"] = ctx.Stdin.Model.DisplayName
	}
	if ctx.CWD != "" {
		vars["path"] = filepath.Base(ctx.CWD)
	}
	return vars
}

func (m *ProjectModule) DefaultFormat() string {
	return "[{model} | {plan}] {path}"
}
