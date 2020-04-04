package tools

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// 根据传递的文件路径创建路径中不存在的文件夹跟对应文件，并将文件内存写入文件中
// 例如 /Home/lise/Desktop/app/test.txt    1235
// 如果 /Home/lise/Desktop/app 中有不存在的文件夹则会创建整个路径的文件夹
// 然后创建test.txt文件并蒋 1235 写入test.txt中
func createFile(filePath, content string, perm os.FileMode) error {
	dir := filepath.Dir(filePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0777)
		if err != nil {
			return err
		}
	}

	_, err := os.Lstat(filePath)
	if !os.IsNotExist(err) {
		return nil
	}

	newFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer func() {
		_ = newFile.Sync()
		_ = newFile.Close()
	}()

	_, err = newFile.WriteString(content)
	if err != nil {
		return err
	}
	return nil
}

// 根据传递的项目路径生成当前项目的导入路径
// 1、如果当前的 projectPath 已经存在于go mod 的项目下，则会以go mod 的路径+/projectName 为导入路径
// 例如 /Home/lise/Desktop/app/login；login为新创建的登录分布式模块，而/Home/lise/Desktop/app已经是一个存在go.mod(github.com/app)文件的项目
// 则返回 github.com/app/login
// 2、如果当前项目在GOPATH下面，则返回以GOPATH为准的导入路径
// 例如给定路径为 /Users/zhangsan/go/src/app/login；设置的GOPATH路径为 /Users/zhangsan/go 则返回的路径为 app/login
// 3、如果项目处在GOPATH下面，且项目使用go.mod进行管理，则优先会使用 go.mod 的项目路径，跟情况1相同
func ParseProjectImportPath(projectPath string) string {
	dir := filepath.Dir(projectPath)

	// from go mod
	modPath := parseGoMode(dir)
	if modPath != "" {
		return filepath.Join(modPath, filepath.Base(projectPath))
	}

	// from gopath
	goPath := filepath.Join(os.Getenv("GOPATH"), "src")
	if strings.HasPrefix(projectPath, goPath) {
		importPath := strings.TrimPrefix(projectPath, goPath)
		return strings.TrimPrefix(importPath, string(filepath.Separator))
	}

	newGoModPath := filepath.Join("github.com", filepath.Base(projectPath))
	execCommand("/bin/sh", "-c", fmt.Sprintf("cd %s && go mod init %s", projectPath, newGoModPath))
	return newGoModPath
}

// 从给定的path中向上找go.mod路径，如果所有路径的文件夹中没有找到go.mod文件则直接返回空字符串
func parseGoMode(path string) string {
	suffix := ""

	for {
		if path == "" || path == string(filepath.Separator) {
			break
		}

		modPath := filepath.Join(path, "go.mod")
		_, err := os.Lstat(modPath)
		if os.IsNotExist(err) {
			suffix = filepath.Join(filepath.Base(path), suffix)
			path = filepath.Dir(path)
			continue
		}

		modFile, err := os.Open(modPath)
		if err != nil {
			panic(err)
		}

		reader := bufio.NewScanner(modFile)
		for reader.Scan() {
			line := reader.Text()
			if strings.HasPrefix(line, "module") {
				return filepath.Join(strings.TrimSpace(strings.TrimPrefix(line, "module")), suffix)
			}
		}
	}
	return ""
}

func execCommand(name string, args ...string) {
	if err := exec.Command(name, args...).Run(); err != nil {
		fmt.Println("execCommand Error", err)
	}
}

func FirstLower(input string) string {
	if len(input) <= 0 {
		return input
	}

	return strings.ToLower(input[:1]) + input[1:]
}
