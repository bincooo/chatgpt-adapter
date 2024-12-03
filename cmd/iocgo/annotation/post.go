package annotation

import (
	"fmt"
	"github.com/iocgo/sdk/gen/annotation"
	"go/ast"
)

type POST struct {
	*annotation.Anon
	Path string `annotation:"name=path,default=/"`
}

var _ annotation.M = (*POST)(nil)

func (g POST) Match(node ast.Node) (err error) {
	if _, ok := node.(*ast.FuncDecl); !ok {
		err = fmt.Errorf(`"@POST" annotation is only allowed to be defined on the method`)
		return
	}

	if err = g.As().Match(node); err != nil {
		return
	}
	return
}

func (g POST) As() annotation.M {
	return annotation.Router{
		Method: "POST",
		Path:   g.Path,
	}
}
