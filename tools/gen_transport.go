package tools

import (
	"bytes"
	"fmt"
	"go/ast"
	"path/filepath"
)

const (
	transportTemplate = `
package transport

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/transport/grpc"
	"github.com/opentracing/opentracing-go"
	"github.com/liyanbing/golang-pack/grpc-tool"

	"%s"

	%s "%s"
)
`

	grpcServerStructTemplate = `
type gRPCServer struct {
`
	grpcServerFieldTemplate = `%sHandler grpc.Handler
`

	grpcServerMethodTemplate = `
func (g *gRPCServer) %s(ctx context.Context, req *%s.%s) (*%s.%s, error) {
	_, ret, err := g.%sHandler.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}

	return ret.(*%s.%s), nil
}
`

	encodeDecodeTemplate = `
func DecodeRequestFunc(_ context.Context, req interface{}) (request interface{}, err error) {
	request = req
	return
}

func EncodeResponseFunc(_ context.Context, resp interface{}) (response interface{}, err error) {
	response = resp
	return
}
`
	newGRPCServerFuncTemplate = `
func NewGRPCServer(service %s.%sServer, tracer opentracing.Tracer, logger log.Logger) %s.%sServer {
	options := &grpc_tool.Options{
		ServiceName: %s.ServiceName,
		Tracer:      tracer,
		Log:         logger,
	}
	eps := endpoints.WrapEndpoints(%s.ServiceName, service, tracer)

	return &gRPCServer{`

	grpcServerFieldGenTemplate = `
%sHandler: grpc.NewServer(
			eps.%sEndpoint,
			DecodeRequestFunc,
			EncodeResponseFunc,
			grpc.ServerBefore(grpc_tool.GRPCToContext(options, "%s")...)),
`
)

func GenTransport(path, pkgName, serviceName, importPath string, methodList []*ast.Field) error {
	contentBuf := bytes.NewBufferString(fmt.Sprintf(transportTemplate, filepath.Join(importPath, "endpoints"), pkgName, importPath))
	grpcServerStructBuf := bytes.NewBufferString(grpcServerStructTemplate)
	grpcServerMethodBuf := bytes.NewBufferString("")
	grpcServerNewFuncBuf := bytes.NewBufferString(fmt.Sprintf(newGRPCServerFuncTemplate, pkgName, serviceName, pkgName, serviceName, pkgName, pkgName))

	for _, m := range methodList {
		curF := m.Type.(*ast.FuncType)

		methodName := m.Names[0].Name
		secondArgs := curF.Params.List[1].Type.(*ast.StarExpr).X.(*ast.Ident).Name
		firstResp := curF.Results.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name

		grpcServerStructBuf.WriteString(fmt.Sprintf(grpcServerFieldTemplate, FirstLower(methodName)))
		grpcServerMethodBuf.WriteString(fmt.Sprintf(grpcServerMethodTemplate, methodName, pkgName, secondArgs, pkgName, firstResp, FirstLower(methodName), pkgName, firstResp))
		grpcServerNewFuncBuf.WriteString(fmt.Sprintf(grpcServerFieldGenTemplate, FirstLower(methodName), methodName, methodName))
	}
	grpcServerStructBuf.WriteString(fmt.Sprintln("}"))
	grpcServerNewFuncBuf.WriteString(fmt.Sprintln("}}"))

	contentBuf.WriteString(grpcServerStructBuf.String())
	contentBuf.WriteString(grpcServerMethodBuf.String())
	contentBuf.WriteString(grpcServerNewFuncBuf.String())
	contentBuf.WriteString(encodeDecodeTemplate)

	err := createFile(path, contentBuf.String(), 0666)
	if err != nil {
		return err
	}

	execCommand("gofmt", "-w", path)
	return nil
}
