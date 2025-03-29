package vm

import (
	"embed"
	"testing"

	"github.com/govm-net/vm/api"
)

//go:embed testdata/*.go
var testContracts embed.FS

func TestValidateContract(t *testing.T) {
	// Create a maker with default config
	config := api.ContractConfig{
		MaxCodeSize: 1024 * 1024, // 1MB
		AllowedImports: []string{
			"github.com/govm-net/vm/core",
		},
	}
	maker := NewMaker(config)

	// Read contract files
	validContract, _ := testContracts.ReadFile("testdata/valid_contract.go")
	invalidImportContract, _ := testContracts.ReadFile("testdata/invalid_import_contract.go")
	goKeywordContract, _ := testContracts.ReadFile("testdata/go_keyword_contract.go")
	selectKeywordContract, _ := testContracts.ReadFile("testdata/select_keyword_contract.go")
	noExportedFuncsContract, _ := testContracts.ReadFile("testdata/no_exported_funcs_contract.go")

	// Contract that exceeds size limit
	largeSizeContract := make([]byte, config.MaxCodeSize+1)

	// Test valid contract
	err := maker.ValidateContract(validContract)
	if err != nil {
		t.Errorf("Valid contract should pass validation, but got error: %v", err)
	}

	// Test contract with disallowed import
	err = maker.ValidateContract(invalidImportContract)
	if err == nil {
		t.Errorf("Contract with disallowed import should fail validation")
	}

	// Test contract with 'go' keyword
	err = maker.ValidateContract(goKeywordContract)
	if err == nil {
		t.Errorf("Contract with 'go' keyword should fail validation")
	}

	// Test contract with 'select' keyword
	err = maker.ValidateContract(selectKeywordContract)
	if err == nil {
		t.Errorf("Contract with 'select' keyword should fail validation")
	}

	// Test contract with no exported functions
	err = maker.ValidateContract(noExportedFuncsContract)
	if err == nil {
		t.Errorf("Contract with no exported functions should fail validation")
	}

	// Test contract that exceeds size limit
	err = maker.ValidateContract(largeSizeContract)
	if err == nil {
		t.Errorf("Contract exceeding size limit should fail validation")
	}
}

func TestCompileContract(t *testing.T) {
	// Create a maker with default config
	config := api.ContractConfig{
		MaxCodeSize: 1024 * 1024, // 1MB
		AllowedImports: []string{
			"github.com/govm-net/vm/core",
		},
	}
	maker := NewMaker(config)

	// Read contract files
	validContract, _ := testContracts.ReadFile("testdata/valid_contract.go")

	// Test compilation of a valid contract
	compiledCode, err := maker.CompileContract(validContract)
	if err != nil {
		t.Errorf("Valid contract should compile successfully, but got error: %v", err)
	}
	if len(compiledCode) == 0 {
		t.Errorf("Compiled code should not be empty")
	}

	// In the current implementation, compiled code is just the source code
	if string(compiledCode) != string(validContract) {
		t.Errorf("Compiled code should match the source code in the current implementation")
	}

	// Test compilation of an invalid contract
	invalidContract := []byte(`
package invalidcontract

import (
	"fmt" // Disallowed import
)

type InvalidContract struct{}

func (c *InvalidContract) DoSomething() {
	fmt.Println("This should not compile")
}
`)

	_, err = maker.CompileContract(invalidContract)
	if err == nil {
		t.Errorf("Invalid contract should fail compilation")
	}
}

func TestInstantiateContract(t *testing.T) {
	// Create a maker with default config
	config := api.ContractConfig{
		MaxCodeSize: 1024 * 1024, // 1MB
		AllowedImports: []string{
			"github.com/govm-net/vm/core",
		},
	}
	maker := NewMaker(config)

	// Read contract file
	validContract, _ := testContracts.ReadFile("testdata/valid_contract.go")

	// Compile the contract
	compiledCode, err := maker.CompileContract(validContract)
	if err != nil {
		t.Fatalf("Failed to compile valid contract: %v", err)
	}

	// Instantiate the contract
	contractInstance, err := maker.InstantiateContract(compiledCode)
	if err != nil {
		t.Errorf("Failed to instantiate contract: %v", err)
	}
	if contractInstance == nil {
		t.Errorf("Instantiated contract should not be nil")
	}
}
