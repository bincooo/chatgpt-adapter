package annotation

import (
	"encoding/json"
	"fmt"
	"github.com/iocgo/sdk/gen/annotation"
	"github.com/iocgo/sdk/stream"
	"go/ast"
)

type Cobra struct {
	*annotation.Anon

	N         string `annotation:"name=name,default=" json:"-"`
	Qualifier string `annotation:"name=qualifier,default=" json:"-"`

	Use     string `annotation:"name=use,default="`
	Short   string `annotation:"name=short,default="`
	Long    string `annotation:"name=long,default="`
	Version string `annotation:"name=version,default="`
	Example string `annotation:"name=example,default="`

	Run string `annotation:"name=run,default="`
}

var _ annotation.M = (*Cobra)(nil)

func (g Cobra) As() annotation.M {
	config, _ := json.Marshal(g)
	return annotation.Inject{
		N:         g.N,
		IsLazy:    true,
		Singleton: true,
		Qualifier: g.Qualifier,
		Config:    string(config),
	}
}

func (g Cobra) Match(node ast.Node) (err error) {
	if err = g.As().Match(node); err != nil {
		return
	}

	fd := node.(*ast.FuncDecl)
	if stream.OfSlice(fd.Type.Params.List).Filter(isStringField).One() == nil {
		err = fmt.Errorf(`'@Cobra' annotation requires a receive parameter of type 'string'`)
		return
	}

	if stream.OfSlice(fd.Type.Results.List).Filter(isCobraField).One() == nil {
		err = fmt.Errorf(`'@Cobra' annotation requires a receive returns of type 'cobra.ICobra'`)
		return
	}
	return
}

func isCobraField(field *ast.Field) bool {
	switch expr := field.Type.(type) {
	case *ast.Ident:
		if expr.Name == "ICobra" {
			return true
		}
	case *ast.StarExpr:
		selectorExpr := expr.X.(*ast.SelectorExpr)
		if selectorExpr.Sel.Name == "ICobra" {
			return true
		}
	case *ast.SelectorExpr:
		if expr.Sel.Name == "ICobra" {
			return true
		}
	}
	return false
}

func isStringField(field *ast.Field) bool {
	switch expr := field.Type.(type) {
	case *ast.Ident:
		if expr.Name == "string" {
			return true
		}
	}
	return false
}
