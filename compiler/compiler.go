package compiler

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/govm-net/vm/abi"
	"github.com/govm-net/vm/config"
	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/interfaces"
)

//go:embed template/contract1.template
var contractMainCode string

// DefaultCompiler 是默认的合约编译器实现
type DefaultCompiler struct {
	addressGenerator interfaces.ContractAddressGenerator
	allowImports     []string
}

// NewDefaultCompiler 创建一个新的默认合约编译器
func NewDefaultCompiler() *DefaultCompiler {
	var out DefaultCompiler
	out.addressGenerator = AddressGenerator{}
	os.MkdirAll(config.RootDirOfContracts, 0755)
	out.allowImports = config.AllowImports

	return &out
}

// Compile 将源码编译为 WASM 二进制代码
func (c *DefaultCompiler) CompileContract(ctx context.Context, code []byte) (core.Address, error) {
	address := c.addressGenerator.Generate(ctx, code)
	contractDir := filepath.Join(config.RootDirOfContracts, address.String())
	// 1. 验证合约代码
	fiel, err := c.parseCode(code)
	if err != nil {
		return core.ZeroAddress, err
	}
	if err := c.validateContract(fiel); err != nil {
		return core.ZeroAddress, err
	}

	if err := c.tryBuild(address, code); err != nil {
		return core.ZeroAddress, err
	}

	os.MkdirAll(contractDir, 0755)

	abi, err := abi.ExtractABI(code)
	if err != nil {
		return core.ZeroAddress, err
	}
	data, err := json.MarshalIndent(abi, "", "  ")
	if err != nil {
		return core.ZeroAddress, err
	}
	abiFile := filepath.Join(contractDir, "abi.json")
	if err := os.WriteFile(abiFile, data, 0644); err != nil {
		return core.ZeroAddress, err
	}

	// 2. 注入gas计费代码
	if err := c.addGasCode(address, code); err != nil {
		return core.ZeroAddress, err
	}
	// 4. 编译代码
	if err := c.buildContract(address, abi); err != nil {
		return core.ZeroAddress, err
	}

	return address, nil
}

func (c *DefaultCompiler) parseCode(code []byte) (*ast.File, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", code, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract: %w", err)
	}
	return file, nil
}

func (c *DefaultCompiler) validateContract(file *ast.File) error {
	if err := c.validateImports(file); err != nil {
		return err
	}

	if err := c.validateComment(file); err != nil {
		return err
	}

	if err := c.validateKeywords(file); err != nil {
		return err
	}

	return nil
}

func (c *DefaultCompiler) validateImports(file *ast.File) error {
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		var allowed bool
		for _, k := range c.allowImports {
			if strings.HasPrefix(importPath, k) {
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

func (c *DefaultCompiler) validateKeywords(file *ast.File) error {
	var err error
	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			return true
		}
		switch n := node.(type) {
		case *ast.EmptyStmt:
			if n.Implicit {
				err = fmt.Errorf("restricted keyword ';' is not allowed")
				return false
			}
		case *ast.GoStmt:
			err = fmt.Errorf("restricted keyword 'go' is not allowed")
			return false
		case *ast.SelectStmt:
			err = fmt.Errorf("restricted keyword 'select' is not allowed")
		case *ast.ChanType:
			err = fmt.Errorf("restricted keyword 'chan' is not allowed")
		case *ast.MapType:
			err = fmt.Errorf("restricted keyword 'map' is not allowed")
		case *ast.CallExpr:
			ident, ok := n.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			switch ident.Name {
			case "make", "cap", "recover":
				err = fmt.Errorf("restricted keyword '%s' is not allowed", ident.Name)
			}
		case *ast.FuncDecl:
			name := n.Name.String()
			if name == "main" || name == "init" {
				err = fmt.Errorf("function '%s' is not allowed", name)
			}
		}

		return err == nil
	})
	return err
}

func (c *DefaultCompiler) validateComment(file *ast.File) error {
	for _, commentGroup := range file.Comments {
		for _, comment := range commentGroup.List {
			if constraint.IsGoBuild(comment.Text) {
				return fmt.Errorf("'%s' is not allowed", comment.Text)
			}
		}
	}

	return nil
}

func (c *DefaultCompiler) tryBuild(address core.Address, code []byte) error {
	tmpDir := "./temp/vm-build1/"
	err := os.MkdirAll(tmpDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	contractFile := filepath.Join(tmpDir, address.String()+".go")
	if err := os.WriteFile(contractFile, []byte(code), 0644); err != nil {
		return fmt.Errorf("failed to write contract code: %w", err)
	}
	defer os.Remove(contractFile)
	cmd := exec.Command("go", "build", contractFile)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build contract: %w", err)
	}
	return nil
}

// add cover:go tool cover -mode=set file1.go
func (c *DefaultCompiler) addGasCode(address core.Address, code []byte) error {
	tmpDir := "./temp/vm-build2/"
	err := os.MkdirAll(tmpDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	contractFile := filepath.Join(tmpDir, address.String()+"1.go")
	if err := os.WriteFile(contractFile, code, 0644); err != nil {
		return fmt.Errorf("failed to write contract code: %w", err)
	}
	defer os.Remove(contractFile)
	dstFile := filepath.Join(tmpDir, address.String()+"2.go")
	os.Remove(dstFile)
	cmd := exec.Command("go", "tool", "cover", "-mode=atomic", "-var=goCover", "-o", dstFile, contractFile)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build contract: %w", err)
	}
	defer os.Remove(dstFile)

	code1, err := os.ReadFile(dstFile)
	if err != nil {
		return fmt.Errorf("failed to read contract: %w", err)
	}
	code2 := strings.Replace(string(code1), `import _cover_atomic_ "sync/atomic"`, config.GasLimitPackage, 1)
	reg := regexp.MustCompile(`AddUint32\(&goCover.*],`)
	code2 = reg.ReplaceAllString(string(code2), "ConsumeGas(")
	id := strings.Index(code2, "var goCover")
	code2 = code2[:id]
	dstDir := filepath.Join(config.RootDirOfContracts, address.String())
	os.MkdirAll(dstDir, 0755)
	lastFile := filepath.Join(dstDir, "contract.go")
	err = os.WriteFile(lastFile, []byte(code2), 0644)
	if err != nil {
		return fmt.Errorf("failed to write contract: %w", err)
	}
	cmd = exec.Command("go", "build", lastFile)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build contract, %s: %w", lastFile, err)
	}

	return nil
}

func (c *DefaultCompiler) buildContract(address core.Address, abi *abi.ABI) error {
	args := abi2args(address, abi)
	t := template.Must(template.New("code").Funcs(template.FuncMap{
		"join": func(v []Parameter, sep string) string {
			keys := make([]string, len(v))
			for i, v := range v {
				keys[i] = v.Name
			}
			return strings.Join(keys, sep)
		},
	}).Parse(contractMainCode))

	buff := bytes.NewBuffer(nil)
	err := t.Execute(buff, args)
	if err != nil {
		log.Println("executing template:", err)
		return err
	}
	fn := filepath.Join(config.RootDirOfContracts, address.String()+".go")
	if err := os.WriteFile(fn, buff.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write contract: %w", err)
	}
	defer os.Remove(fn)
	binFile := filepath.Join(config.RootDirOfContracts, address.String(), "contract.bin")
	cmd := exec.Command("go", "build", "-o", binFile, fn)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build contract, %s: %w", fn, err)
	}
	return nil
}
