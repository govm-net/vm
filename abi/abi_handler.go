package abi

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// HandlerGenerator generates handler functions from ABI
type HandlerGenerator struct {
	abi *ABI
}

var EnableFormatAfterGenerate = true

// NewHandlerGenerator creates a new handler generator
func NewHandlerGenerator(abi *ABI) *HandlerGenerator {
	return &HandlerGenerator{
		abi: abi,
	}
}

// Format formats the generated code using gofmt
func Format(code string) (string, error) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "vm-handler-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建临时文件
	tmpFile := filepath.Join(tmpDir, "handler.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	// 运行 gofmt
	cmd := exec.Command("gofmt", "-s", "-w", tmpFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("gofmt failed: %s: %w", string(output), err)
	}

	// 读取格式化后的代码
	formattedCode, err := os.ReadFile(tmpFile)
	if err != nil {
		return "", fmt.Errorf("failed to read formatted code: %w", err)
	}

	return string(formattedCode), nil
}

// GenerateHandlers generates handler functions for all exported functions
func (g *HandlerGenerator) GenerateHandlers() string {
	var sb strings.Builder

	// 添加包名和导入
	sb.WriteString(fmt.Sprintf("package %s\n\n", g.abi.PackageName))
	sb.WriteString("import (\n")

	// 收集所有需要的导入
	imports := make(map[string]string) // path -> alias
	// 添加基础导入
	// imports["github.com/govm-net/vm/core"] = ""
	imports["fmt"] = ""
	imports["encoding/json"] = ""
	for _, fn := range g.abi.Functions {

		// 检查输入参数类型
		for _, input := range fn.Inputs {
			if imp := g.findImportForType(input.Type); imp != nil {
				if imp.Name != "" {
					imports[imp.Path] = imp.Name
				} else {
					imports[imp.Path] = ""
				}
			}
		}

		// 检查输出参数类型
		for _, output := range fn.Outputs {
			if imp := g.findImportForType(output.Type); imp != nil {
				if imp.Name != "" {
					imports[imp.Path] = imp.Name
				} else {
					imports[imp.Path] = ""
				}
			}
		}
	}

	// 添加收集到的导入
	for path, alias := range imports {
		if alias != "" {
			sb.WriteString(fmt.Sprintf("\t%s \"%s\"\n", alias, path))
		} else {
			sb.WriteString(fmt.Sprintf("\t\"%s\"\n", path))
		}
	}
	sb.WriteString(")\n\n")

	// import json/fmt, 避免在wasm中编译失败
	sb.WriteString(`
func init(){
	if false{
		fmt.Println("init")
		json.Marshal(nil)
	}
}

`)

	// 为每个导出函数生成参数结构体和 handler
	for _, fn := range g.abi.Functions {
		sb.WriteString(g.generateParamStruct(fn))
		sb.WriteString(g.generateHandler(fn))
	}

	return sb.String()
}

// findImportForType 根据类型查找对应的导入
func (g *HandlerGenerator) findImportForType(typeStr string) *Import {
	// 处理指针类型
	typeStr = strings.TrimPrefix(typeStr, "*")

	// 处理数组类型
	typeStr = strings.TrimPrefix(typeStr, "[]")

	// 处理带包名的类型
	if strings.Contains(typeStr, ".") {
		parts := strings.Split(typeStr, ".")
		pkgName := parts[0]

		// 在导入列表中查找匹配的包
		for _, imp := range g.abi.Imports {
			// 如果导入有别名，检查别名是否匹配
			if imp.Name == pkgName {
				return &imp
			}

			// 如果没有别名，检查包路径的最后一部分是否匹配
			if imp.Name == "" {
				pathParts := strings.Split(imp.Path, "/")
				if pathParts[len(pathParts)-1] == pkgName {
					return &imp
				}
			}
		}
	}

	return nil
}

// generateParamStruct generates a parameter struct for a function
func (g *HandlerGenerator) generateParamStruct(fn Function) string {
	var sb strings.Builder

	// 生成结构体定义
	sb.WriteString(fmt.Sprintf("type %sParams struct {\n", fn.Name))
	for _, input := range fn.Inputs {
		// 使用 json tag 来匹配参数名，并添加 omitempty
		sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s,omitempty\"`\n",
			cases.Title(language.English).String(input.Name), input.Type, input.Name))
	}
	sb.WriteString("}\n\n")

	return sb.String()
}

// generateHandler generates a handler function for a given function
func (g *HandlerGenerator) generateHandler(fn Function) string {
	var sb strings.Builder

	// 生成函数签名
	sb.WriteString(fmt.Sprintf("func handle%s(params []byte) (any, error) {\n", fn.Name))

	// 生成参数解析代码
	if len(fn.Inputs) > 0 {
		sb.WriteString(fmt.Sprintf("\tvar args %sParams\n", fn.Name))
		sb.WriteString("\tif len(params) > 0 {\n")
		sb.WriteString("\t\tif err := json.Unmarshal(params, &args); err != nil {\n")
		sb.WriteString("\t\t\treturn nil, fmt.Errorf(\"failed to unmarshal params: %w\", err)\n")
		sb.WriteString("\t\t}\n")
		sb.WriteString("\t}\n\n")
	}

	// 检查是否需要返回值变量
	hasReturnValue := len(fn.Outputs) > 0
	if hasReturnValue {
		// 生成返回值变量声明
		sb.WriteString("\t")
		for i, output := range fn.Outputs {
			if i > 0 {
				sb.WriteString(", ")
			}
			if output.Name != "" {
				sb.WriteString(output.Name)
			} else {
				sb.WriteString(fmt.Sprintf("result%d", i))
			}
		}
		sb.WriteString(" := ")
	}

	// 生成函数调用
	sb.WriteString(fn.Name)
	sb.WriteString("(")

	// 添加参数
	firstParam := true
	for _, input := range fn.Inputs {
		if !firstParam {
			sb.WriteString(", ")
		}
		firstParam = false

		sb.WriteString(fmt.Sprintf("args.%s", cases.Title(language.English).String(input.Name)))
	}
	sb.WriteString(")\n\n")

	// 处理返回值
	if hasReturnValue && len(fn.Outputs) > 1 {
		sb.WriteString("\t// 处理返回值\n")
		sb.WriteString("\tresults := make([]any, 0)\n")
		for i, output := range fn.Outputs {
			if output.Name != "" {
				sb.WriteString(fmt.Sprintf("\tresults = append(results, %s)\n", output.Name))
			} else {
				sb.WriteString(fmt.Sprintf("\tresults = append(results, result%d)\n", i))
			}
		}
		sb.WriteString("\treturn results, nil\n")
	} else if hasReturnValue && len(fn.Outputs) == 1 {
		sb.WriteString("\treturn ")
		if fn.Outputs[0].Name != "" {
			sb.WriteString(fmt.Sprintf("%s, nil\n", fn.Outputs[0].Name))
		} else {
			sb.WriteString("result0, nil\n")
		}
	} else {
		sb.WriteString("\treturn nil, nil\n")
	}
	sb.WriteString("}\n\n")
	return sb.String()
}

// GenerateHandlerFile generates a complete handler file
func GenerateHandlerFile(abi *ABI) (string, error) {
	generator := NewHandlerGenerator(abi)
	code := generator.GenerateHandlers()
	if !EnableFormatAfterGenerate {
		return code, nil
	}

	// 格式化代码
	formattedCode, err := Format(code)
	if err != nil {
		return "", fmt.Errorf("failed to format code: %w", err)
	}

	return formattedCode, nil
}
