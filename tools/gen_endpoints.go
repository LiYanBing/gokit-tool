package tools

import (
	"bytes"
	"fmt"
	"go/ast"
)

const (
	endpointsTemplate = `package endpoints

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/opentracing/opentracing-go"
	"github.com/liyanbing/golang-pack/grpc-tool"

	%s "%s"
)
`
	endpointsStructTemplate = `
type Endpoints struct {
`
	endpointsFieldTemplate = `%sEndpoint endpoint.Endpoint
`

	endpointsWrapTemplate = `
func wrap%s(service %s.%sServer) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(*%s.%s)
		return service.%s(ctx, req)
	}
}
`

	endpointsMethodTemplate = `
func (e *Endpoints) %s(ctx context.Context, req *%s.%s) (*%s.%s, error) {
	ret, err := e.%sEndpoint(ctx, req)
	if err != nil {
		return nil, err
	}

	return ret.(*%s.%s), nil
}
`

	wrapFuncTemplate = `
func wrapMethod(options *grpc_tool.Options, method string, endpoint endpoint.Endpoint) endpoint.Endpoint {
	wrappers := grpc_tool.EndpointWrapperChain(options, method)
	for _, wrap := range wrappers {
		endpoint = wrap(endpoint)
	}
	return endpoint
}

func WrapEndpoints(serviceName string, service %s.%sServer, tracer opentracing.Tracer) *Endpoints {
	options := &grpc_tool.Options{
		ServiceName: serviceName,
		Tracer:      tracer,
	}

	return &Endpoints{
`

	endpointsFieldGenTemplate = `%sEndpoint: wrapMethod(options, "%s", wrap%s(service)),
`
)

func GenEndpoints(filePath, pkgName, serviceName, importPath string, methodList []*ast.Field) error {
	contentBuf := bytes.NewBufferString(fmt.Sprintf(endpointsTemplate, pkgName, importPath))

	endpointsStruct := bytes.NewBufferString(endpointsStructTemplate)
	endpointsMethod := bytes.NewBufferString("")
	endpointsWrap := bytes.NewBufferString("")
	wrapFunc := bytes.NewBufferString(fmt.Sprintf(wrapFuncTemplate, pkgName, serviceName))

	for _, m := range methodList {
		curF := m.Type.(*ast.FuncType)
		secondArgs := curF.Params.List[1].Type.(*ast.StarExpr).X.(*ast.Ident).Name
		firstResp := curF.Results.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
		methodName := m.Names[0].Name

		endpointsStruct.WriteString(fmt.Sprintf(endpointsFieldTemplate, methodName))
		endpointsMethod.WriteString(fmt.Sprintf(endpointsMethodTemplate, methodName, pkgName, secondArgs, pkgName, firstResp, methodName, pkgName, firstResp))
		endpointsWrap.WriteString(fmt.Sprintf(endpointsWrapTemplate, methodName, pkgName, serviceName, pkgName, secondArgs, methodName))
		wrapFunc.WriteString(fmt.Sprintf(endpointsFieldGenTemplate, methodName, methodName, methodName))
	}
	endpointsStruct.WriteString(fmt.Sprintln("}"))
	wrapFunc.WriteString(fmt.Sprintln("}}"))

	contentBuf.WriteString(endpointsStruct.String())
	contentBuf.WriteString(endpointsMethod.String())
	contentBuf.WriteString(endpointsWrap.String())
	contentBuf.WriteString(wrapFunc.String())

	err := createFile(filePath, contentBuf.String(), 0666)
	if err != nil {
		return err
	}

	execCommand("gofmt", "-w", filePath)
	return nil
}
