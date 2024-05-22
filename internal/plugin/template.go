package plugin

import (
	"bytes"
	"text/template"
)

type TempWrapper struct {
	t       *template.Template
	context map[string]interface{}
	funcM   template.FuncMap
}

func templateBuilder() *TempWrapper {
	t := template.New("root")
	context := make(map[string]interface{})
	funcMap := template.FuncMap{}
	return &TempWrapper{t, context, funcMap}
}

func (tpl *TempWrapper) Vars(key string, value interface{}) *TempWrapper {
	tpl.context[key] = value
	return tpl
}

func (tpl *TempWrapper) Func(key string, fun interface{}) *TempWrapper {
	tpl.funcM[key] = fun
	return tpl
}

func (tpl *TempWrapper) Do() func(templateVar string) (string, error) {
	tpl.t.Funcs(tpl.funcM)
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
