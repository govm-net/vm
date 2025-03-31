// Package api provides the interfaces for the virtual machine that executes smart contracts.
// This package defines the API between the blockchain and the VM, but is not directly used by smart contracts.
package api

import (
	"fmt"
	"go/ast"

	"github.com/govm-net/vm/core"
)

// VM represents the virtual machine that executes smart contracts
type VM interface {
	// Deploy deploys a new smart contract to the blockchain
	Deploy(code []byte, args ...[]byte) (core.Address, error)

	// Execute executes a function on a deployed contract
	Execute(contract core.Address, function string, args ...[]byte) ([]byte, error)

	// ValidateContract checks if the contract code adheres to the restrictions
	ValidateContract(code []byte) error
}

// Restricted keywords that are not allowed in smart contracts
var RestrictedKeywords = []string{
	"go",      // Prevents concurrent execution
	"select",  // Eliminates channel selection
	"range",   // Restricts iteration over maps
	"cap",     // Prevents capacity checks
	"recover", // Disallows panic recovery
	"package", // Controls package declarations
}

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

// DefaultContractConfig returns a default configuration for contracts
func DefaultContractConfig() ContractConfig {
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
	case *ast.CallExpr:
		if ident, ok := node.(*ast.CallExpr).Fun.(*ast.Ident); ok && ident.Name == "recover" {
			return fmt.Errorf("restricted keyword 'recover' is not allowed")
		}
	}
	return nil
}
