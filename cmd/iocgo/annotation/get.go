package annotation

import (
	"fmt"
	"github.com/iocgo/sdk/gen/annotation"
	"go/ast"
)

type GET struct {
	*annotation.Anon
	Path string `annotation:"name=path,default=/"`
}

var _ annotation.M = (*GET)(nil)

func (g GET) Match(node ast.Node) (err error) {
	if _, ok := node.(*ast.FuncDecl); !ok {
		err = fmt.Errorf(`"@GET" annotation is only allowed to be defined on the method`)
		return
	}

	if err = g.As().Match(node); err != nil {
		return
	}
	return
}

func (g GET) As() annotation.M {
	return annotation.Router{
		Method: "GET",
		Path:   g.Path,
	}
}
