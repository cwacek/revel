package template_engines

import "html/template"
import "errors"

type TemplateEngine interface {
	Handles(extension string) bool
	Clear()
	// Add info to the template set, returning errors
	// that occur and whether or not they're recoverable.
	AddTemplate(info *TemplateInfo) (err error, unrecoverable bool)
	CompiledTemplates() *template.Template
	SetDelims([]string)
}

type TemplateEngineNew func(tmplBasePath string) TemplateEngine

type TemplateInfo struct {
	Name string
	Path string
}

type FuncLoadError struct {
	Title       string
	Description string
}

func (t *FuncLoadError) Error() string {
	return t.Title + ": " + t.Description
}

func CheckTemplateModule(importPath string) error {

	pkg, err := build.Default.Import(importPath, "", build.AllowBinary)
	if err == nil {
		return errors.New("Couldn't import template engine: %s", err)
	}

}
