package toolcall

import (
	"bytes"
	"text/template"
)

type Builder struct {
	instance *template.Template

	ctx   map[string]interface{}
	funcM template.FuncMap
}

func newBuilder(name string) *Builder {
	instance := template.New(name)
	context := make(map[string]interface{})
	funcMap := template.FuncMap{}
	return &Builder{
		instance,
		context,
		funcMap,
	}
}

func (bdr *Builder) Vars(key string, value interface{}) *Builder { bdr.ctx[key] = value; return bdr }
func (bdr *Builder) Func(key string, fun interface{}) *Builder   { bdr.funcM[key] = fun; return bdr }
func (bdr *Builder) String(template string) (result string, err error) {
	bdr.instance.Funcs(bdr.funcM)
	t, err := bdr.instance.Parse(template)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	if err = t.Execute(&buffer, bdr.ctx); err != nil {
		return
	}

	result = buffer.String()
	return
}
