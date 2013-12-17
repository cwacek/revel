package template_engine

import "html/template"
import "path"
import "fmt"
import "log"
import "io/ioutil"
import "os"
import "strings"

type TemplateLoader func(tmplName, tmplStr string, delims []string) (*template.Template, error)

type TemplateEngine struct {
	// template paths already seen
	seen_paths map[string]string
	// TemplateLoaders for different file extensions
	handlers map[string]TemplateLoader
	// The template merged
	TemplateSet *template.Template
	// Delimiters
	delims []string
}

type Error struct {
	Title       string
	Path        string
	Description string
	Line        int
	SourceLines []string
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Title, e.Description)
}

type TemplateInfo struct {
	Name string
	Path string
}

var (
	engine *TemplateEngine
)

func init() {
	engine = new(TemplateEngine)
	engine.seen_paths = make(map[string]string)
	engine.handlers = make(map[string]TemplateLoader)
	engine.delims = []string{"", ""}
}

func SetDelims(delims []string) {
	engine.delims = delims
	if len(engine.delims) != 2 {
		log.Fatalln("app.conf: Incorrect format for template.delimiters")
	}
}

func CompiledTemplates() *template.Template {
	return engine.TemplateSet
}

func Clear() {
	engine.TemplateSet = nil
	engine.seen_paths = make(map[string]string)
}

func RegisterTemplater(extension string, loader TemplateLoader) {
	engine.handlers[extension] = loader
}

func AddTemplate(info *TemplateInfo) (err error) {

	var (
		templateName, fileStr string
	)

	// Convert template names to use forward slashes, even on Windows.
	if os.PathSeparator == '\\' {
		templateName = strings.Replace(info.Name, `\`, `/`, -1) // `
	}

	// If we already loaded a template of this name, skip it.
	if _, ok := engine.seen_paths[templateName]; ok {
		return nil
	}
	engine.seen_paths[templateName] = info.Path

	// Load the file if we haven't already
	if fileStr == "" {
		fileBytes, err := ioutil.ReadFile(info.Path)
		if err != nil {
			log.Printf("Failed reading file:", info.Path)
			return nil
		}

		fileStr = string(fileBytes)
	}

	// html is equivalent to no extension - the default
	ext := path.Ext(info.Path)
	if ext == "html" {
		ext = ""
	}

	var loader TemplateLoader
	var ok bool
	if loader, ok = engine.handlers[ext]; !ok {
		return &Error{
			Title:       "Template Load Error",
			Path:        info.Path,
			Description: fmt.Sprintf("No known handler for extension '%s'", ext),
			Line:        -1,
			SourceLines: strings.Split(fileStr, "\n"),
		}
	}

	template, err := loader(info.Name, fileStr, engine.delims)
	if err != nil {
		return err
	}

	if engine.TemplateSet == nil {
		log.Printf("Adding template to template set: %v", template)
		engine.TemplateSet = template
	} else {
		_, err := engine.TemplateSet.AddParseTree(info.Name, template.Tree)
		if err != nil {
			return err
		}
	}

	return nil
}
