package annotation

import (
	"fmt"
	"github.com/iocgo/sdk/gen/annotation"
	"go/ast"
)

type PUT struct {
	*annotation.Anon
	Path string `annotation:"name=path,default=/"`
}

var _ annotation.M = (*PUT)(nil)

func (g PUT) Match(node ast.Node) (err error) {
	if _, ok := node.(*ast.FuncDecl); !ok {
		err = fmt.Errorf(`"@PUT" annotation is only allowed to be defined on the method`)
		return
	}

	if err = g.As().Match(node); err != nil {
		return
	}
	return
}

func (g PUT) As() annotation.M {
	return annotation.Router{
		Method: "PUT",
		Path:   g.Path,
	}
}
