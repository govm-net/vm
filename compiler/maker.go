// Package runtime provides the execution environment for smart contracts.
package compiler

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/govm-net/vm/abi"
	"github.com/govm-net/vm/api"
)

// Maker handles the compilation and validation of smart contracts.
type Maker struct {
	config api.ContractConfig
}

var VM_IMPORT_PATH = "./"

var BuildParams = []string{"build", "-o", "contract.wasm", "-target", "wasi", "-opt", "z", "-no-debug", "./"}

// var BuildParams = []string{"build", "-o", "contract.wasm", "-target", "wasi", "./"}

// unique removes duplicates from a string slice
func unique(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

var RestrictedCommentPrefixes []string = []string{
	"go ",
	"+build",
	"-build",
	"go:",
	"// +build",
	"//line",
	"//export",
	"//extern",
	"//cgo",
	"//syscall",
	"//unsafe",
	"//runtime",
	"//internal",
	"//vendor",
	// exist "go:"", ignore below items
	// "//go:build",
	// "//go:generate",
	// "//go:linkname",
	// "//go:nosplit",
	// "//go:noescape",
	// "//go:noinline",
	// "//go:systemstack",
	// "//go:nowritebarrier",
	// "//go:yeswritebarrier",
	// "//go:nointerface",
	// "//go:norace",
	// "//go:nocheckptr",
	// "//go:embed",
	// "//go:cgo_",
	// "//go:linkname",
}

func init() {
	VM_IMPORT_PATH, _ = os.Getwd()
	VM_IMPORT_PATH += "/../"

	// 预处理限制的注释前缀
	rawPrefixes := RestrictedCommentPrefixes

	RestrictedCommentPrefixes = make([]string, len(rawPrefixes))
	for i, prefix := range rawPrefixes {
		// 移除前缀中的 "//" 或 "// " 进行比较
		prefixToCheck := strings.TrimPrefix(prefix, "//")
		prefixToCheck = strings.TrimPrefix(prefixToCheck, " ")
		RestrictedCommentPrefixes[i] = prefixToCheck
	}
	//去重
	RestrictedCommentPrefixes = unique(RestrictedCommentPrefixes)
}

// NewMaker creates a new contract maker with the given configuration.
func NewMaker(config api.ContractConfig) *Maker {
	return &Maker{
		config: config,
	}
}

// ValidateContract checks if the smart contract code adheres to the
// restrictions and rules defined for the VM.
func (m *Maker) ValidateContract(code []byte) error {
	// Parse the contract source code
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", code, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("failed to parse contract: %w", err)
	}

	// Validate imports
	if err := m.validateImports(file); err != nil {
		return err
	}

	// Validate no restricted keywords are used
	if err := m.validateNoRestrictedKeywords(file); err != nil {
		return err
	}

	// Validate no malicious commands in comments
	if err := m.validateNoMaliciousCommands(file); err != nil {
		return err
	}

	// Validate contract size
	if len(code) > int(m.config.MaxCodeSize) {
		return fmt.Errorf("contract size exceeds maximum allowed size of %d bytes", m.config.MaxCodeSize)
	}

	// Check if there's at least one exported function
	hasExportedFunctions := false
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Name.IsExported() {
			hasExportedFunctions = true
			break
		}
	}

	if !hasExportedFunctions {
		return errors.New("contract must have at least one exported (public) function")
	}

	// 创建临时目录进行编译验证
	tmpDir, err := os.MkdirTemp("", "vm-contract-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建源代码目录
	srcDir := tmpDir

	// 写入合约代码
	contractFile := filepath.Join(srcDir, "contract.go")
	if err := os.WriteFile(contractFile, code, 0644); err != nil {
		return fmt.Errorf("failed to write contract code: %w", err)
	}

	// 创建 go.mod 文件
	goModContent := fmt.Sprintf(`module %s

go 1.23

require github.com/govm-net/vm v0.0.0

replace github.com/govm-net/vm => %s
`, file.Name.Name, VM_IMPORT_PATH)
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		return fmt.Errorf("failed to write go.mod: %w", err)
	}
	// 尝试编译代码
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tmpDir
	// cmd.Env = append(os.Environ(), "GOPATH="+tmpDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go mod tidy failed: %s\nOutput: %s", err, string(output))
	}

	// 尝试编译代码
	cmd = exec.Command("go", "build", "-v", "./")
	cmd.Dir = tmpDir
	// cmd.Env = append(os.Environ(), "GOPATH="+tmpDir)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("contract compilation failed: %s\nOutput: %s", err, string(output))
	}

	return nil
}

// validateImports checks that the contract only imports allowed packages.
func (m *Maker) validateImports(file *ast.File) error {
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		allowed := false

		for _, allowedImport := range m.config.AllowedImports {
			if importPath == allowedImport || strings.HasPrefix(importPath, allowedImport+"/") {
				allowed = true
				break
			}
		}

		if !allowed {
			return fmt.Errorf("import %s is not allowed", importPath)
		}
	}

	return nil
}

// validateNoRestrictedKeywords ensures the contract doesn't use restricted keywords.
func (m *Maker) validateNoRestrictedKeywords(file *ast.File) error {
	restrictedKeywordVisitor := &restrictedKeywordVisitor{
		restrictedKeywords: api.RestrictedKeywords,
	}

	ast.Walk(restrictedKeywordVisitor, file)

	if restrictedKeywordVisitor.foundKeyword != "" {
		return fmt.Errorf("restricted keyword '%s' found in contract", restrictedKeywordVisitor.foundKeyword)
	}

	return nil
}

// restrictedKeywordVisitor is an AST visitor that detects restricted keywords.
type restrictedKeywordVisitor struct {
	restrictedKeywords []string
	foundKeyword       string
}

// Visit implements the ast.Visitor interface.
func (v *restrictedKeywordVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}

	// Check for go statements (goroutines)
	if _, ok := node.(*ast.GoStmt); ok {
		v.foundKeyword = "go"
		return nil
	}

	// Check for select statements
	if _, ok := node.(*ast.SelectStmt); ok {
		v.foundKeyword = "select"
		return nil
	}

	// Check for range expressions
	if _, ok := node.(*ast.RangeStmt); ok {
		v.foundKeyword = "range"
		return nil
	}

	// Check for recover calls
	if callExpr, ok := node.(*ast.CallExpr); ok {
		if ident, ok := callExpr.Fun.(*ast.Ident); ok && ident.Name == "recover" {
			v.foundKeyword = "recover"
			return nil
		}
	}

	return v
}

// validateNoMaliciousCommands ensures the contract doesn't contain malicious commands in comments.
func (m *Maker) validateNoMaliciousCommands(file *ast.File) error {
	fset := token.NewFileSet()
	// 检查所有注释
	for _, commentGroup := range file.Comments {
		for _, comment := range commentGroup.List {
			// 移除注释标记
			commentText := strings.TrimPrefix(comment.Text, "//")
			commentText = strings.TrimPrefix(commentText, "/*")
			commentText = strings.TrimSuffix(commentText, "*/")
			commentText = strings.TrimSpace(commentText)

			// 检查每一行
			for _, line := range strings.Split(commentText, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				for _, prefix := range RestrictedCommentPrefixes {
					if strings.HasPrefix(strings.ToLower(line), strings.ToLower(prefix)) {
						return fmt.Errorf("restricted comment prefix '%s' found at line %d", prefix, fset.Position(comment.Pos()).Line)
					}
				}
			}
		}
	}
	return nil
}

// CompileContract compiles the given contract source code.
func (m *Maker) CompileContract(code []byte) ([]byte, error) {
	// First validate the contract
	if err := m.ValidateContract(code); err != nil {
		return nil, err
	}

	// 1. 提取代码的abi
	abiInfo, err := abi.ExtractABI(code)
	if err != nil {
		return nil, fmt.Errorf("failed to extract ABI: %w", err)
	}
	//如果函数个数为0，则返回错误
	if len(abiInfo.Functions) == 0 {
		return nil, errors.New("contract must have at least one exported (public) function")
	}

	// 2. 创建临时目录
	tmpDir, err := os.MkdirTemp("", "vm-contract-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	// tmpDir := "./tmp"
	// os.MkdirAll(tmpDir, 0755)

	// 3. 将代码放到临时文件夹，并修改包名为main
	contractCode := string(code)
	contractCode = strings.Replace(contractCode, "package "+abiInfo.PackageName, "package main", 1)
	contractFile := filepath.Join(tmpDir, "source.go")
	if err := os.WriteFile(contractFile, []byte(contractCode), 0644); err != nil {
		return nil, fmt.Errorf("failed to write contract code: %w", err)
	}

	// 4. 生成handle函数
	handlerGenerator := abi.NewHandlerGenerator(abiInfo)
	handlerCode := handlerGenerator.GenerateHandlers()

	// 修改生成的代码，使用 main 包
	handlerCode = strings.Replace(handlerCode, "package "+abiInfo.PackageName, "package main", 1)

	// 修改合约代码，使用 main 包
	contractCode = string(code)
	contractCode = strings.Replace(contractCode, "package "+abiInfo.PackageName, "package main", 1)

	// 写入修改后的合约代码
	if err := os.WriteFile(contractFile, []byte(contractCode), 0644); err != nil {
		return nil, fmt.Errorf("failed to write contract code: %w", err)
	}

	handlerFile := filepath.Join(tmpDir, "handlers.go")
	if err := os.WriteFile(handlerFile, []byte(handlerCode), 0644); err != nil {
		return nil, fmt.Errorf("failed to write handler code: %w", err)
	}

	// 5. 复制wasm/contract.go到临时文件夹
	wasmContractFile := filepath.Join(VM_IMPORT_PATH, "wasm/contract.go")
	contractGoContent, err := os.ReadFile(wasmContractFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read contract.go: %w", err)
	}

	// 修改contract.go，添加handler函数的注册
	registerCode := "\nfunc init() {\n"
	for _, fn := range abiInfo.Functions {
		if fn.IsExported {
			registerCode += fmt.Sprintf("\tregisterContractFunction(\"%s\", handle%s)\n", fn.Name, fn.Name)
		}
	}
	registerCode += "}\n"

	modifiedContractGo := string(contractGoContent) + registerCode
	if err := os.WriteFile(filepath.Join(tmpDir, "contract.go"), []byte(modifiedContractGo), 0644); err != nil {
		return nil, fmt.Errorf("failed to write modified contract.go: %w", err)
	}

	// 创建 go.mod 文件
	goModContent := fmt.Sprintf(`module %s

go 1.23

require github.com/govm-net/vm v0.0.0

replace github.com/govm-net/vm => %s
`, "main", VM_IMPORT_PATH)
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write go.mod: %w", err)
	}

	// 运行 go mod tidy
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go mod tidy failed: %s\nOutput: %s", err, string(output))
	}

	// 6. 使用tinygo编译
	cmd = exec.Command("tinygo", BuildParams...)
	cmd.Dir = tmpDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tinygo build failed: %s\nOutput: %s", err, string(output))
	}

	// 读取编译后的wasm文件
	wasmFile := filepath.Join(tmpDir, "contract.wasm")
	wasmCode, err := os.ReadFile(wasmFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read compiled wasm: %w", err)
	}

	return wasmCode, nil
}

// FunctionInfo 函数信息
type FunctionInfo struct {
	Name string
	Args []struct {
		Name string
		Type string
	}
}

// ParseABI 解析合约代码获取函数信息
func (m *Maker) ParseABI(code []byte) (map[string]FunctionInfo, error) {
	// 创建文件集
	fset := token.NewFileSet()

	// 解析Go代码
	file, err := parser.ParseFile(fset, "", code, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract code: %w", err)
	}

	// 存储函数信息
	abi := make(map[string]FunctionInfo)

	// 遍历所有声明
	for _, decl := range file.Decls {
		// 只处理函数声明
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		// 跳过未导出的函数
		if !funcDecl.Name.IsExported() {
			continue
		}

		// 创建函数信息
		info := FunctionInfo{
			Name: funcDecl.Name.Name,
		}

		// 处理函数参数
		if funcDecl.Type.Params != nil {
			for _, field := range funcDecl.Type.Params.List {
				// 获取参数类型
				typeExpr := field.Type
				typeStr := ""
				switch t := typeExpr.(type) {
				case *ast.Ident:
					typeStr = t.Name
				case *ast.ArrayType:
					if ident, ok := t.Elt.(*ast.Ident); ok {
						typeStr = "[]" + ident.Name
					}
				}

				// 处理每个参数名
				for _, name := range field.Names {
					info.Args = append(info.Args, struct {
						Name string
						Type string
					}{
						Name: name.Name,
						Type: typeStr,
					})
				}
			}
		}

		abi[info.Name] = info
	}

	return abi, nil
}
