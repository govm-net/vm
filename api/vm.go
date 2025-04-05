// Package api provides the interfaces for the virtual machine that executes smart contracts.
// This package defines the API between the blockchain and the VM, but is not directly used by smart contracts.
package api

import (
	"crypto/sha256"
	"fmt"
	"go/ast"

	"github.com/govm-net/vm/types"
)

// ContractConfig defines configuration for contract validation and execution
type ContractConfig struct {
	// MaxGas is the maximum amount of gas that can be used by a contract
	MaxGas uint64

	// MaxCallDepth is the maximum depth of contract calls
	MaxCallDepth uint8

	// MaxCodeSize is the maximum size of contract code in bytes
	MaxCodeSize uint64

	// AllowedImports contains the packages that can be imported by contracts
	AllowedImports []string
}

type IContractConfigGenerator func() ContractConfig

var DefaultContractConfig IContractConfigGenerator = func() ContractConfig {
	return ContractConfig{
		MaxGas:       1000000,
		MaxCallDepth: 8,
		MaxCodeSize:  1024 * 1024, // 1MB
		AllowedImports: []string{
			"github.com/govm-net/vm/core",
			// Additional allowed imports would be listed here
		},
	}
}

type IKeywordValidator func(node ast.Node) error

var DefaultKeywordValidator IKeywordValidator = func(node ast.Node) error {
	if node == nil {
		return nil
	}
	switch node.(type) {
	case *ast.GoStmt:
		return fmt.Errorf("restricted keyword 'go' is not allowed")
	case *ast.SelectStmt:
		return fmt.Errorf("restricted keyword 'select' is not allowed")
	case *ast.RangeStmt:
		return fmt.Errorf("restricted keyword 'range' is not allowed")
	case *ast.ChanType:
		return fmt.Errorf("restricted keyword 'chan' is not allowed")
	case *ast.CallExpr:
		if ident, ok := node.(*ast.CallExpr).Fun.(*ast.Ident); ok && ident.Name == "recover" {
			return fmt.Errorf("restricted keyword 'recover' is not allowed")
		}
	}
	return nil
}

type IContractAddressGenerator func(code []byte) types.Address

var DefaultContractAddressGenerator IContractAddressGenerator = func(code []byte) types.Address {
	var addr types.Address
	hash := sha256.Sum256(code)
	copy(addr[:], hash[:])
	return addr
}

type IGoModGenerator func(moduleName string, imports map[string]string, replaces map[string]string) string

var DefaultGoModGenerator IGoModGenerator = func(moduleName string, imports, replaces map[string]string) string {
	return fmt.Sprintf(`
	module %s

go 1.23.0

require (
	github.com/govm-net/vm v1.0.0
)

// replace github.com/govm-net/vm => ./
`, moduleName)
}

var Builder = "tinygo"
var BuildParams = []string{"build", "-o", "contract.wasm", "-target", "wasi", "-opt", "z", "-no-debug", "./"}
