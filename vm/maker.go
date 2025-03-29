// Package runtime provides the execution environment for smart contracts.
package vm

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"strings"

	"github.com/govm-net/vm/api"
)

// Maker handles the compilation and validation of smart contracts.
type Maker struct {
	config api.ContractConfig
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

// CompileContract compiles the given contract source code.
func (m *Maker) CompileContract(code []byte) ([]byte, error) {
	// First validate the contract
	if err := m.ValidateContract(code); err != nil {
		return nil, err
	}

	// For now, we just return the source code as is
	// In a real implementation, this would compile the contract to bytecode
	// or produce an executable representation of the contract
	return code, nil
}

// InstantiateContract creates a new instance of the compiled contract.
// This method dynamically loads and instantiates the contract based on compiled code.
func (m *Maker) InstantiateContract(compiledCode []byte) (interface{}, error) {
	// 首先解析代码中的包名和结构体名称
	packageName, contractName, err := m.extractContractInfo(compiledCode)
	if err != nil {
		return nil, fmt.Errorf("failed to extract contract info: %w", err)
	}

	// 尝试使用动态加载方法
	// 设置一个环境标志以控制是否允许动态加载
	useDynamicLoading := true

	// 可以根据运行时的操作系统、环境变量等决定是否启用动态加载
	if useDynamicLoading {
		// 尝试动态编译和加载合约
		instance, err := m.dynamicallyLoadContract(packageName, contractName, compiledCode)
		if err == nil {
			return instance, nil
		}
		// 如果动态加载失败，记录错误并尝试回退方法
		fmt.Printf("Dynamic loading failed: %v. Falling back to template method.\n", err)
	}

	// 如果动态加载不可用或者失败，使用预编译模板方法
	// 首先尝试使用反射创建实例
	instance, err := m.createContractByReflection(packageName, contractName)
	if err == nil {
		return instance, nil
	}

	// 如果反射方法也失败，则使用简单的匹配方法
	return m.createContractInstance(packageName, contractName)
}

// extractContractInfo 从代码中提取包名和合约结构体名称
func (m *Maker) extractContractInfo(code []byte) (packageName, contractName string, err error) {
	// 解析Go代码
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", code, parser.AllErrors)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse contract: %w", err)
	}

	// 获取包名
	packageName = file.Name.Name
	if packageName == "" {
		return "", "", errors.New("could not determine package name")
	}

	// 查找第一个导出的结构体定义作为合约类型
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || !typeSpec.Name.IsExported() {
				continue
			}

			// 检查是否为结构体
			if _, ok := typeSpec.Type.(*ast.StructType); ok {
				return packageName, typeSpec.Name.Name, nil
			}
		}
	}

	// 如果没有找到导出的结构体，查找可能包含receiver参数的函数
	// 分析函数声明，找出可能的合约类型
	typeMap := make(map[string]int)
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || !funcDecl.Name.IsExported() || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
			continue
		}

		// 获取receiver类型
		var typeName string
		switch t := funcDecl.Recv.List[0].Type.(type) {
		case *ast.StarExpr: // 指针类型 (*Type)
			if ident, ok := t.X.(*ast.Ident); ok {
				typeName = ident.Name
			}
		case *ast.Ident: // 非指针类型 (Type)
			typeName = t.Name
		}

		if typeName != "" {
			typeMap[typeName]++
		}
	}

	// 找出出现次数最多的类型作为合约类型
	var maxCount int
	for name, count := range typeMap {
		if count > maxCount {
			maxCount = count
			contractName = name
		}
	}

	if contractName == "" {
		return "", "", errors.New("could not determine contract type")
	}

	return packageName, contractName, nil
}

// createContractInstance 创建合约实例
func (m *Maker) createContractInstance(packageName, contractName string) (interface{}, error) {
	// 在实际实现中，这里应该动态加载已编译的代码
	// 由于我们现在没有动态编译机制，返回测试用的虚拟合约
	// 不同合约类型可以通过包名和合约名区分

	// 为特定合约类型返回自定义实现
	return nil, nil
}

// dynamicallyLoadContract 在运行时动态编译和加载合约代码
func (m *Maker) dynamicallyLoadContract(packageName, contractName string, code []byte) (interface{}, error) {
	// 注意：此功能需要在运行时环境中支持Go plugin功能
	// 目前Go plugin仅在Linux、FreeBSD和macOS上受支持

	// 创建一个临时目录来构建插件
	tempDir, err := os.MkdirTemp("", "govm-contract-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建必要的目录结构
	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create source directory: %w", err)
	}

	// 写入合约源代码
	contractFile := filepath.Join(srcDir, "contract.go")
	if err := os.WriteFile(contractFile, code, 0644); err != nil {
		return nil, fmt.Errorf("failed to write contract code: %w", err)
	}

	// 创建一个包装文件，用于导出合约实例
	wrapperCode := fmt.Sprintf(`
package main

import (
	"github.com/govm-net/vm/core"
	// 其他必要的导入
)

// 原始合约代码在contract.go中

// 必须导出的符号
var Contract interface{}

// init函数在插件加载时自动运行
func init() {
	// 创建一个合约实例并将其赋值给导出的Contract变量
	Contract = &%s{}
}
`, contractName)

	wrapperFile := filepath.Join(srcDir, "wrapper.go")
	if err := os.WriteFile(wrapperFile, []byte(wrapperCode), 0644); err != nil {
		return nil, fmt.Errorf("failed to write wrapper code: %w", err)
	}

	// 编译插件
	pluginFile := filepath.Join(tempDir, "contract.so")
	cmd := exec.Command("go", "build", "-buildmode=plugin", "-o", pluginFile, srcDir)
	cmd.Env = append(os.Environ(), "GOPATH="+tempDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to compile plugin: %w\nOutput: %s", err, output)
	}

	// 加载插件
	p, err := plugin.Open(pluginFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %w", err)
	}

	// 获取Contract变量
	symbol, err := p.Lookup("Contract")
	if err != nil {
		return nil, fmt.Errorf("failed to find Contract symbol: %w", err)
	}

	// 返回合约实例
	contract, ok := symbol.(*interface{})
	if !ok {
		return nil, fmt.Errorf("Contract symbol is not of the expected type")
	}

	return *contract, nil
}

// 使用reflection创建合约实例
func (m *Maker) createContractByReflection(packageName, contractName string) (interface{}, error) {
	// 获取合约类型
	return nil, nil
}
