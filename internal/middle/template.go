package middle

import (
	"bytes"
	"text/template"
)

type TemplateWrapper struct {
	t       *template.Template
	context map[string]interface{}
	funcMap template.FuncMap
}

func NewTemplateWrapper() *TemplateWrapper {
	t := template.New("root")
	context := make(map[string]interface{})
	funcMap := template.FuncMap{}
	return &TemplateWrapper{t, context, funcMap}
}

func (tpl *TemplateWrapper) Variables(key string, value interface{}) *TemplateWrapper {
	tpl.context[key] = value
	return tpl
}

func (tpl *TemplateWrapper) Func(key string, fun interface{}) *TemplateWrapper {
	tpl.funcMap[key] = fun
	return tpl
}

func (tpl *TemplateWrapper) Build() func(templateVar string) (string, error) {
	tpl.t.Funcs(tpl.funcMap)

	return func(templateVar string) (string, error) {
		t, err := tpl.t.Parse(templateVar)
		if err != nil {
			return "", err
		}

		var buffer bytes.Buffer
		if err = t.Execute(&buffer, tpl.context); err != nil {
			return "", err
		}

		return buffer.String(), nil
	}
}
