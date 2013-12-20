package revel

import "html/template"

type TemplateEngine interface {
  Handles(extension string) bool
  Clear()
  AddTemplate(info *TemplateInfo) (*template.Template, error)
  CompiledTemplates() *template.Template
  SetDelims([]string)
}

type TemplateEngineNew func(tmplBasePath string) TemplateEngine

type TemplateInfo struct {
	Name string
	Path string
}

func CheckTemplateModule(importPath string) error {

  pkg, err := build.Default.Import(importPath, )

}
