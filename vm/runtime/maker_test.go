package runtime

import (
	"testing"

	"github.com/govm-net/vm/vm/api"
)

func TestValidateContract(t *testing.T) {
	// Create a maker with default config
	config := api.ContractConfig{
		MaxCodeSize: 1024 * 1024, // 1MB
		AllowedImports: []string{
			"github.com/govm-net/vm/core",
		},
	}
	maker := NewMaker(config)

	// Valid contract
	validContract := []byte(`
package validcontract

import (
	"github.com/govm-net/vm/core"
)

type ValidContract struct{}

func (c *ValidContract) DoSomething(ctx core.Context) string {
	return "Something"
}
`)

	// Contract with disallowed import
	invalidImportContract := []byte(`
package invalidcontract

import (
	"github.com/govm-net/vm/core"
	"fmt" // This import is not allowed
)

type InvalidContract struct{}

func (c *InvalidContract) DoSomething(ctx core.Context) string {
	return "Something"
}
`)

	// Contract with restricted keyword (go)
	goKeywordContract := []byte(`
package restrictedcontract

import (
	"github.com/govm-net/vm/core"
)

type RestrictedContract struct{}

func (c *RestrictedContract) DoSomething(ctx core.Context) string {
	go func() {} // Using 'go' keyword is restricted
	return "Something"
}
`)

	// Contract with restricted keyword (select)
	selectKeywordContract := []byte(`
package restrictedcontract

import (
	"github.com/govm-net/vm/core"
)

type RestrictedContract struct{}

func (c *RestrictedContract) DoSomething(ctx core.Context) chan string {
	ch1 := make(chan string)
	ch2 := make(chan string)
	go func() {
		select { // Using 'select' keyword is restricted
		case <-ch1:
			return
		case <-ch2:
			return
		}
	}()
	return ch1
}
`)

	// Contract with no exported functions
	noExportedFuncsContract := []byte(`
package noexportedfuncs

import (
	"github.com/govm-net/vm/core"
)

type NoExportedFuncsContract struct{}

func (c *NoExportedFuncsContract) doSomething(ctx core.Context) string {
	return "Something"
}
`)

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

	// Valid contract
	validContract := []byte(`
package validcontract

import (
	"github.com/govm-net/vm/core"
)

type ValidContract struct{}

func (c *ValidContract) DoSomething(ctx core.Context) string {
	return "Something"
}
`)

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

	// Valid contract code
	validContract := []byte(`
package validcontract

import (
	"github.com/govm-net/vm/core"
)

type ValidContract struct{}

func (c *ValidContract) DoSomething(ctx core.Context) string {
	return "Something"
}
`)

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

	// Since the current implementation returns a dummy contract,
	// we can just check that it's not nil for now
	if _, ok := contractInstance.(*TestDummyContract); !ok {
		t.Errorf("Expected a TestDummyContract instance, got something else")
	}
}
