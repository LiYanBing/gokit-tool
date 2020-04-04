package tools

import (
	"bytes"
	"fmt"
	"go/ast"
)

const (
	apiGoTemplate = `
package %v

import (
	"context"

	%s "%s"
)

type %s interface {
`
	methodTemplate = `%v(context.Context, *%s.%v) (*%s.%v, error)
`
)

func GenAPI(filePath, importPath, pkgName, serviceName string, methodList []*ast.Field) error {
	contentBuf := bytes.NewBufferString(fmt.Sprintf(apiGoTemplate, pkgName, pkgName, importPath, serviceName))

	for _, m := range methodList {
		curF := m.Type.(*ast.FuncType)
		if m.Doc != nil {
			for _, v := range m.Doc.List {
				contentBuf.WriteString(v.Text)
			}
			contentBuf.WriteString("\n")
		}

		secondArgs := curF.Params.List[1].Type.(*ast.StarExpr).X.(*ast.Ident).Name
		firstResp := curF.Results.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
		methodName := m.Names[0].Name
		contentBuf.WriteString(fmt.Sprintf(methodTemplate, methodName, pkgName, secondArgs, pkgName, firstResp))
	}
	contentBuf.WriteString("}")

	err := createFile(filePath, contentBuf.String(), 0666)
	if err != nil {
		return err
	}

	execCommand("gofmt", "-w", filePath)
	return nil
}
