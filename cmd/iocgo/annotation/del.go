package annotation

import (
	"fmt"
	"github.com/iocgo/sdk/gen/annotation"
	"go/ast"
)

type DEL struct {
	*annotation.Anon
	Path string `annotation:"name=path,default=/"`
}

var _ annotation.M = (*DEL)(nil)

func (g DEL) Match(node ast.Node) (err error) {
	if _, ok := node.(*ast.FuncDecl); !ok {
		err = fmt.Errorf(`"@DEL" annotation is only allowed to be defined on the method`)
		return
	}

	if err = g.As().Match(node); err != nil {
		return
	}
	return
}

func (g DEL) As() annotation.M {
	return annotation.Router{
		Method: "DELETE",
		Path:   g.Path,
	}
}
