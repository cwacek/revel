package template_engines

import "log"
import "time"
import "html/template"
import "errors"
import "io/ioutil"

func InitializeEngine(tmplBasePath string) TemplateEngine {
	engine := new(GoTemplateEngine)
	engine.seenPaths = make(map[string]string)

	return engine
}

type GoTemplateEngine struct {
	basePath string
	// template paths already seen
	seenPaths map[string]string
	// The template merged
	templateSet *template.Template
	//Delims
	delims []string
}

func (engine *GoTemplateEngine) Clear() {
	engine.templateSet = nil
	engine.seenPaths = make(map[string]string)
}

func (engine *GoTemplateEngine) CompiledTemplates() *template.Template {
	return engine.templateSet
}

func (engine *GoTemplateEngine) Handles(extension string) bool {
	return true
}

func (engine *GoTemplateEngine) SetDelims(delims []string) {
	engine.delims = delims
}

func (engine *GoTemplateEngine) AddTemplate(info *TemplateInfo) (err error, unrecoverable bool) {
	var fileStr string

	if info == nil {
		log.Printf("Invalid template info supplied")
		return
	}

	// Load the file if we haven't already
	if fileStr == "" {
		fileBytes, err := ioutil.ReadFile(info.Path)
		if err != nil {
			log.Printf("Failed reading file:", info.Path)
			return errors.New("Failed reading file: " + info.Path), false
		}

		fileStr = string(fileBytes)
	}

	template, err := goTemplater(info.Name, fileStr, engine.delims)
	if err != nil {
		log.Printf("Failed to load template: %v", err)
		return err
	}
	log.Printf("Successfully loaded %s using %s", info.Name, "GoTemplateEngine")

	if engine.templateSet == nil {
		engine.templateSet = template
	} else {
		_, err := engine.templateSet.AddParseTree(info.Name, template.Tree)
		if err != nil {
			log.Printf("Failed to successfully add parse tree")
			return err
		}
	}

	return nil
}

var (
	// The functions available for use in the templates.
	TemplateFuncs = map[string]interface{}{
		"url": ReverseUrl,
		"eq":  Equal,
		"set": func(renderArgs map[string]interface{}, key string, value interface{}) template.HTML {
			renderArgs[key] = value
			return template.HTML("")
		},
		"append": func(renderArgs map[string]interface{}, key string, value interface{}) template.HTML {
			if renderArgs[key] == nil {
				renderArgs[key] = []interface{}{value}
			} else {
				renderArgs[key] = append(renderArgs[key].([]interface{}), value)
			}
			return template.HTML("")
		},
		"field": NewField,
		"option": func(f *Field, val, label string) template.HTML {
			selected := ""
			if f.Flash() == val {
				selected = " selected"
			}
			return template.HTML(fmt.Sprintf(`<option value="%s"%s>%s</option>`,
				html.EscapeString(val), selected, html.EscapeString(label)))
		},
		"radio": func(f *Field, val string) template.HTML {
			checked := ""
			if f.Flash() == val {
				checked = " checked"
			}
			return template.HTML(fmt.Sprintf(`<input type="radio" name="%s" value="%s"%s>`,
				html.EscapeString(f.Name), html.EscapeString(val), checked))
		},
		"checkbox": func(f *Field, val string) template.HTML {
			checked := ""
			if f.Flash() == val {
				checked = " checked"
			}
			return template.HTML(fmt.Sprintf(`<input type="checkbox" name="%s" value="%s"%s>`,
				html.EscapeString(f.Name), html.EscapeString(val), checked))
		},
		// Pads the given string with &nbsp;'s up to the given width.
		"pad": func(str string, width int) template.HTML {
			if len(str) >= width {
				return template.HTML(html.EscapeString(str))
			}
			return template.HTML(html.EscapeString(str) + strings.Repeat("&nbsp;", width-len(str)))
		},

		"errorClass": func(name string, renderArgs map[string]interface{}) template.HTML {
			errorMap, ok := renderArgs["errors"].(map[string]*ValidationError)
			if !ok || errorMap == nil {
				WARN.Println("Called 'errorClass' without 'errors' in the render args.")
				return template.HTML("")
			}
			valError, ok := errorMap[name]
			if !ok || valError == nil {
				return template.HTML("")
			}
			return template.HTML(ERROR_CLASS)
		},

		"msg": func(renderArgs map[string]interface{}, message string, args ...interface{}) template.HTML {
			return template.HTML(Message(renderArgs[CurrentLocaleRenderArg].(string), message, args...))
		},

		// Replaces newlines with <br>
		"nl2br": func(text string) template.HTML {
			return template.HTML(strings.Replace(template.HTMLEscapeString(text), "\n", "<br>", -1))
		},

		// Skips sanitation on the parameter.  Do not use with dynamic data.
		"raw": func(text string) template.HTML {
			return template.HTML(text)
		},

		// Pluralize, a helper for pluralizing words to correspond to data of dynamic length.
		// items - a slice of items, or an integer indicating how many items there are.
		// pluralOverrides - optional arguments specifying the output in the
		//     singular and plural cases.  by default "" and "s"
		"pluralize": func(items interface{}, pluralOverrides ...string) string {
			singular, plural := "", "s"
			if len(pluralOverrides) >= 1 {
				singular = pluralOverrides[0]
				if len(pluralOverrides) == 2 {
					plural = pluralOverrides[1]
				}
			}

			switch v := reflect.ValueOf(items); v.Kind() {
			case reflect.Int:
				if items.(int) != 1 {
					return plural
				}
			case reflect.Slice:
				if v.Len() != 1 {
					return plural
				}
			default:
				ERROR.Println("pluralize: unexpected type: ", v)
			}
			return singular
		},

		// Format a date according to the application's default date(time) format.
		"date": func(date time.Time) string {
			return date.Format(DateFormat)
		},
		"datetime": func(date time.Time) string {
			return date.Format(DateTimeFormat)
		},
		"slug": Slug,
	}
)

func goTemplater(
	templateName, templateStr string, delims []string) (
	tmpl *template.Template, err error, unrecoverable bool) {

	var funcError error

	// Create the template.  This panics if any of the funcs do not
	// conform to expectations, so we wrap it in a func and handle those
	// panics by serving an error page.
	func() {
		defer func() {
			if err := recover(); err != nil {
				funcError = err.(error)
			}
		}()

		tmpl = template.New(templateName).Funcs(TemplateFuncs)
		// If alternate delimiters set for the project, change them for this set
		if delims != nil {
			tmpl.Delims(delims[0], delims[1])
		} else {
			// Reset to default otherwise
			tmpl.Delims("", "")
		}
		_, err = tmpl.Parse(templateStr)
	}()

	if funcError != nil {
		unrecoverable = true
		return
	}

	return
}

/////////////////////
// Template functions
/////////////////////

// Return a url capable of invoking a given controller method:
// "Application.ShowApp 123" => "/app/123"
func ReverseUrl(args ...interface{}) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("no arguments provided to reverse route")
	}

	action := args[0].(string)
	actionSplit := strings.Split(action, ".")
	if len(actionSplit) != 2 {
		return "", fmt.Errorf("reversing '%s', expected 'Controller.Action'", action)
	}

	// Look up the types.
	var c Controller
	if err := c.SetAction(actionSplit[0], actionSplit[1]); err != nil {
		return "", fmt.Errorf("reversing %s: %s", action, err)
	}

	// Unbind the arguments.
	argsByName := make(map[string]string)
	for i, argValue := range args[1:] {
		Unbind(argsByName, c.MethodType.Args[i].Name, argValue)
	}

	return MainRouter.Reverse(args[0].(string), argsByName).Url, nil
}

func Slug(text string) string {
	separator := "-"
	text = strings.ToLower(text)
	text = invalidSlugPattern.ReplaceAllString(text, "")
	text = whiteSpacePattern.ReplaceAllString(text, separator)
	text = strings.Trim(text, separator)
	return text
}
