package compiler

import (
	"embed"
	"go/parser"
	"go/token"
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

	// Test compilation of an invalid contract
	invalidContract, _ := testContracts.ReadFile("testdata/invalid_import_contract.go")

	_, err = maker.CompileContract(invalidContract)
	if err == nil {
		t.Errorf("Invalid contract should fail compilation")
	}
	err = maker.ValidateContract(invalidContract)
	if err == nil {
		t.Errorf("Invalid contract should fail validation")
	}
}

func TestValidateNoMaliciousCommands(t *testing.T) {
	// Create a maker with default config
	config := api.ContractConfig{
		MaxCodeSize: 1024 * 1024, // 1MB
		AllowedImports: []string{
			"github.com/govm-net/vm/core",
		},
	}
	maker := NewMaker(config)

	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name: "valid single-line comment",
			code: `package test
// This is a normal comment
func main() {}`,
			wantErr: false,
		},
		{
			name: "valid multi-line comment",
			code: `package test
/* This is a normal
   multi-line comment */
func main() {}`,
			wantErr: false,
		},
		{
			name: "restricted single-line comment - go build",
			code: `package test
// go build -o test
func main() {}`,
			wantErr: true,
		},
		{
			name: "restricted single-line comment - +build",
			code: `package test
// +build linux,amd64
func main() {}`,
			wantErr: true,
		},
		{
			name: "restricted multi-line comment - go build",
			code: `package test
/* go build -o test
   some other text */
func main() {}`,
			wantErr: true,
		},
		{
			name: "restricted multi-line comment - +build",
			code: `package test
/* +build linux,amd64
   some other text */
func main() {}`,
			wantErr: true,
		},
		{
			name: "normal word containing restricted prefix",
			code: `package test
// going to implement something
func main() {}`,
			wantErr: false,
		},
		{
			name: "normal multi-line comment containing word with restricted prefix",
			code: `package test
/* going to implement
   something */
func main() {}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("failed to parse test code: %v", err)
			}

			err = maker.validateNoMaliciousCommands(file)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateNoMaliciousCommands() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
