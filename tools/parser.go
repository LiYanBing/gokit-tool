package tools

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

// 解析指定的 ***.pb.go 文件，只解析 serviceName+Server 服务端接口部分
func ParseProtoPBFile(fileName, serviceName string) ([]*ast.Field, error) {
	var (
		methodList []*ast.Field
	)

	fileSet := token.NewFileSet()
	f, err := parser.ParseFile(fileSet, fileName, nil, parser.ParseComments)
	if err != nil {
		return nil, nil
	}

	serviceServer := false
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.TypeSpec:
			if x.Name.Name == fmt.Sprintf("%sServer", serviceName) {
				serviceServer = true
			}
		case *ast.InterfaceType:
			methodList = x.Methods.List
		}
		return !serviceServer
	})

	return methodList, nil
}
