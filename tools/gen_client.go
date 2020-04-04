package tools

import (
	"bytes"
	"fmt"
	"go/ast"
	"path/filepath"
)

var (
	clientTemplate = `package client

import (
	"context"
	"io"
	"time"

	"google.golang.org/grpc"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/consul"
	"github.com/go-kit/kit/sd/lb"
	"github.com/hashicorp/consul/api"
	"github.com/opentracing/opentracing-go"
	"github.com/liyanbing/golang-pack/grpc-tool"

	"%s"

	gokitopentracing "github.com/go-kit/kit/tracing/opentracing"
	grpctr "github.com/go-kit/kit/transport/grpc"
	%s "%s"
)
`

	newFuncTemplate = `
func NewClient(grpcAddr string, tracer opentracing.Tracer, logger log.Logger) (%s.%sServer, error) {
	conn, err := grpc.Dial(grpcAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	serviceName := %s.ServiceName
	options := &grpc_tool.Options{
		ServiceName: serviceName,
		Tracer:      tracer,
		Log:         logger,
	}

	return &endpoints.Endpoints{`

	clientEndpointsGenTemplate = `
%sEndpoint: gokitopentracing.TraceClient(tracer, "%s")(
			grpctr.NewClient(
				conn,
				serviceName,
				"%s",
				encodeGRPCRequest,
				decodeGRPCResponse,
				&%s.%s{},
				grpctr.ClientBefore(grpc_tool.ContextToGRPC(options, "%s")...)).Endpoint()),
`

	newFuncWithConsulTemplate = `
func NewClientWithConsul(consulAddr, dataCenter string, tags []string, tracer opentracing.Tracer, logger log.Logger) (%s.%sServer, error) {
	apiClient, err := api.NewClient(&api.Config{
		Address:    consulAddr,
		Datacenter: dataCenter,
	})
	if err != nil {
		return nil, err
	}

	maxRetry := 3
	timeout := time.Second * 5
	serviceName := %s.ServiceName
	options := &grpc_tool.Options{
		ServiceName: serviceName,
		Tracer:      tracer,
		Log:         logger,
	}
	instanter := consul.NewInstancer(consul.NewClient(apiClient), logger, serviceName, tags, true)

	return &endpoints.Endpoints{`

	clientWithConsulEndpointsGenTemplate = `
%sEndpoint: grpc_tool.Retry(
			maxRetry,
			timeout,
			lb.NewRoundRobin(
				sd.NewEndpointer(
					instanter,
					endpointFactory(options, "%s", &%s.%s{}),
					logger))),
`

	clientEncodeAndDecodeTemplate = `
func endpointFactory(option *grpc_tool.Options, method string, reply interface{}) sd.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		conn, closer, err := grpc_tool.Get(instance)
		if err != nil {
			return nil, nil, err
		}

		ep := grpctr.NewClient(
			conn,
			option.ServiceName,
			method,
			encodeGRPCRequest,
			decodeGRPCResponse,
			reply,
			grpctr.ClientBefore(grpc_tool.ContextToGRPC(option, method)...)).Endpoint()
		return gokitopentracing.TraceClient(option.Tracer, method)(ep), closer, nil
	}
}

func encodeGRPCRequest(_ context.Context, request interface{}) (interface{}, error) {
	return request, nil
}

func decodeGRPCResponse(_ context.Context, reply interface{}) (interface{}, error) {
	return reply, nil
}
`
)

func GenClient(path, pkgName, serviceName, importPath string, methodList []*ast.Field) error {
	contentBuf := bytes.NewBufferString(fmt.Sprintf(clientTemplate, filepath.Join(importPath, "endpoints"), pkgName, importPath))
	newFuncWithConsulBuf := bytes.NewBufferString(fmt.Sprintf(newFuncWithConsulTemplate, pkgName, serviceName, pkgName))
	newFuncBuf := bytes.NewBufferString(fmt.Sprintf(newFuncTemplate, pkgName, serviceName, pkgName))

	for _, m := range methodList {
		curF := m.Type.(*ast.FuncType)
		firstResp := curF.Results.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
		methodName := m.Names[0].Name

		newFuncWithConsulBuf.WriteString(fmt.Sprintf(clientWithConsulEndpointsGenTemplate, methodName, methodName, pkgName, firstResp))
		newFuncBuf.WriteString(fmt.Sprintf(clientEndpointsGenTemplate, methodName, methodName, methodName, pkgName, firstResp, methodName))
	}
	newFuncBuf.WriteString(fmt.Sprintln(`
}, nil}`))
	newFuncWithConsulBuf.WriteString(fmt.Sprintln(`
}, nil}`))

	contentBuf.WriteString(newFuncBuf.String())
	contentBuf.WriteString(newFuncWithConsulBuf.String())
	contentBuf.WriteString(clientEncodeAndDecodeTemplate)

	err := createFile(path, contentBuf.String(), 0666)
	if err != nil {
		return err
	}

	execCommand("gofmt", "-w", path)
	return nil
}
