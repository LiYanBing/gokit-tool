package main

import (
	"fmt"
	"log"
	"path/filepath"

	"ptapp.cn/gokit-tool/tools"
)

var (
	ServiceName = "Service"
)

func main() {
	projectPath := "/Users/Leo/go/src/ptapp.cn/myapp1"
	//projectPath := "/Users/Leo/Desktop/myapp"

	err := tools.CreateProtoAndCompile(filepath.Join(projectPath, "grpc"), ServiceName, filepath.Base(projectPath))
	if err != nil {
		log.Fatal(err)
	}

	err = genGRPCServerAndClient(projectPath)
	if err != nil {
		log.Fatal(err)
	}
}

func genGRPCServerAndClient(projectPath string) error {
	pkgName := filepath.Base(projectPath)
	importPath := filepath.Join(tools.ParseProjectImportPath(projectPath), "grpc")
	grpcPath := filepath.Join(projectPath, "grpc")
	protoFilePath := filepath.Join(grpcPath, fmt.Sprintf("%v.pb.go", pkgName))

	// parse project/grpc/prject.pb.go file
	methodList, err := tools.ParseProtoPBFile(protoFilePath, ServiceName)
	if err != nil {
		return err
	}

	// create /project/api/api.go file
	err = tools.GenAPI(filepath.Join(projectPath, "api", "api.go"), importPath, pkgName, ServiceName, methodList)
	if err != nil {
		return err
	}

	// create project/grpc/endpoints/endpoints.go
	err = tools.GenEndpoints(filepath.Join(grpcPath, "endpoints", "endpoints.go"), pkgName, ServiceName, importPath, methodList)
	if err != nil {
		return err
	}

	// create project/grpc/transport/transport.go
	err = tools.GenTransport(filepath.Join(grpcPath, "transport", "transport.go"), pkgName, ServiceName, importPath, methodList)
	if err != nil {
		return err
	}

	// create project/grpc/client/client.go
	err = tools.GenClient(filepath.Join(grpcPath, "client", "client.go"), pkgName, ServiceName, importPath, methodList)
	if err != nil {
		return err
	}

	// create project/internal/service/service.go
	err = tools.GenInternal(projectPath, pkgName, ServiceName, methodList)
	if err != nil {
		return err
	}

	// create project/cmd/main.go
	err = tools.GenCMD(projectPath, pkgName, importPath)
	if err != nil {
		return err
	}

	return nil
}
