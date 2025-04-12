// Package runtime provides the execution environment for smart contracts.
package compiler

import (
	_ "embed"
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

//go:embed wasm/contract.go
var WASM_CONTRACT_TEMPLATE string

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

	// Preprocess restricted comment prefixes
	// Remove "//" or "// " from prefix for comparison
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

	// Create temporary directory for compilation verification
	tmpDir, err := os.MkdirTemp("", "vm-contract-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source code directory
	srcDir := tmpDir

	// Write contract code
	contractFile := filepath.Join(srcDir, "contract.go")
	if err := os.WriteFile(contractFile, code, 0644); err != nil {
		return fmt.Errorf("failed to write contract code: %w", err)
	}

	// Create go.mod file
	goModContent := api.DefaultGoModGenerator(file.Name.Name, nil, nil)

	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		return fmt.Errorf("failed to write go.mod: %w", err)
	}
	// Try to compile code
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tmpDir
	// cmd.Env = append(os.Environ(), "GOPATH="+tmpDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go mod tidy failed: %s\nOutput: %s", err, string(output))
	}

	// Try to compile code
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
	restrictedKeywordVisitor := &restrictedKeywordVisitor{}

	ast.Walk(restrictedKeywordVisitor, file)

	if restrictedKeywordVisitor.err != nil {
		return restrictedKeywordVisitor.err
	}

	return nil
}

// restrictedKeywordVisitor is an AST visitor that detects restricted keywords.
type restrictedKeywordVisitor struct {
	err error
}

// Visit implements the ast.Visitor interface.
func (v *restrictedKeywordVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	err := api.DefaultKeywordValidator(node)
	if err != nil {
		v.err = err
		return nil
	}

	return v
}

// validateNoMaliciousCommands ensures the contract doesn't contain malicious commands in comments.
func (m *Maker) validateNoMaliciousCommands(file *ast.File) error {
	fset := token.NewFileSet()
	// Check all comments
	for _, commentGroup := range file.Comments {
		for _, comment := range commentGroup.List {
			// Remove comment markers
			commentText := strings.TrimPrefix(comment.Text, "//")
			commentText = strings.TrimPrefix(commentText, "/*")
			commentText = strings.TrimSuffix(commentText, "*/")
			commentText = strings.TrimSpace(commentText)

			// Check each line
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
	// 1. Extract code ABI
	abiInfo, err := abi.ExtractABI(code)
	if err != nil {
		return nil, fmt.Errorf("failed to extract ABI: %w", err)
	}
	// Return error if no functions found
	if len(abiInfo.Functions) == 0 {
		return nil, errors.New("contract must have at least one exported (public) function")
	}

	// 2. Create temporary directory
	var tmpDir string
	if true {
		tmpDir, err = os.MkdirTemp("", "vm-contract-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)
	} else {
		tmpDir = "./tmp"
		os.MkdirAll(tmpDir, 0755)
	}

	// 3. Place code in temporary folder and change package name to main
	contractCode := string(code)
	contractCode = strings.Replace(contractCode, "package "+abiInfo.PackageName, "package main", 1)
	contractFile := filepath.Join(tmpDir, "source.go")
	if err := os.WriteFile(contractFile, []byte(contractCode), 0644); err != nil {
		return nil, fmt.Errorf("failed to write contract code: %w", err)
	}

	// 4. Generate handler functions
	handlerGenerator := abi.NewHandlerGenerator(abiInfo)
	handlerCode := handlerGenerator.GenerateHandlers()

	// Modify generated code to use main package
	handlerCode = strings.Replace(handlerCode, "package "+abiInfo.PackageName, "package main", 1)

	// Modify contract code to use main package
	contractCode = string(code)
	contractCode = strings.Replace(contractCode, "package "+abiInfo.PackageName, "package main", 1)

	// Write modified contract code
	if err := os.WriteFile(contractFile, []byte(contractCode), 0644); err != nil {
		return nil, fmt.Errorf("failed to write contract code: %w", err)
	}

	handlerFile := filepath.Join(tmpDir, "handlers.go")
	if err := os.WriteFile(handlerFile, []byte(handlerCode), 0644); err != nil {
		return nil, fmt.Errorf("failed to write handler code: %w", err)
	}

	// Modify contract.go to add handler function registration
	registerCode := "\nfunc init() {\n"
	for _, fn := range abiInfo.Functions {
		registerCode += fmt.Sprintf("\tregisterContractFunction(\"%s\", handle%s)\n", fn.Name, fn.Name)
	}
	registerCode += "}\n"

	modifiedContractGo := WASM_CONTRACT_TEMPLATE + registerCode
	if err := os.WriteFile(filepath.Join(tmpDir, "contract.go"), []byte(modifiedContractGo), 0644); err != nil {
		return nil, fmt.Errorf("failed to write modified contract.go: %w", err)
	}

	// Create go.mod file
	goModContent := api.DefaultGoModGenerator("main", nil, nil)
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write go.mod: %w", err)
	}

	// Run go mod tidy
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go mod tidy failed: %s\nOutput: %s", err, string(output))
	}

	// 6. Compile with tinygo
	cmd = exec.Command(api.Builder, api.BuildParams...)
	cmd.Dir = tmpDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tinygo build failed: %s\nOutput: %s", err, string(output))
	}

	// Read compiled wasm file
	wasmFile := filepath.Join(tmpDir, "contract.wasm")
	wasmCode, err := os.ReadFile(wasmFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read compiled wasm: %w", err)
	}

	return wasmCode, nil
}
