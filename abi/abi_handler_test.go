package abi

import (
	_ "embed"
	"encoding/json"
	"strings"
	"testing"
)

//go:embed testdata/struct_contract.handlers.go
var structContractHandlers []byte

//go:embed testdata/multi_return_contract.go
var multiReturnContractCode []byte

//go:embed testdata/multi_return_contract.abi.json
var multiReturnContractABI []byte

//go:embed testdata/multi_return_contract.handlers.go
var multiReturnContractHandlers []byte

// 比较两个代码字符串，忽略格式差异
func compareCode(got, expected string) bool {
	// 移除所有空白字符
	got = strings.ReplaceAll(got, " ", "")
	got = strings.ReplaceAll(got, "\t", "")
	got = strings.ReplaceAll(got, "\n", "")
	got = strings.ReplaceAll(got, "\r", "")

	expected = strings.ReplaceAll(expected, " ", "")
	expected = strings.ReplaceAll(expected, "\t", "")
	expected = strings.ReplaceAll(expected, "\n", "")
	expected = strings.ReplaceAll(expected, "\r", "")

	return got == expected
}

func TestGenerateHandlerFile(t *testing.T) {
	// 创建一个测试用的 ABI
	abi := &ABI{
		PackageName: "testcontract",
		Functions: []Function{
			{
				Name:       "ProcessUser",
				IsExported: true,
				Inputs: []Parameter{
					{Name: "user", Type: "*User"},
				},
				Outputs: []Parameter{
					{Name: "", Type: "error"},
				},
			},
			{
				Name:       "CreateOrder",
				IsExported: true,
				Inputs: []Parameter{
					{Name: "order", Type: "*Order"},
				},
				Outputs: []Parameter{
					{Name: "", Type: "error"},
				},
			},
		},
	}

	// 生成 handler 文件
	code, err := GenerateHandlerFile(abi)
	if err != nil {
		t.Fatalf("Failed to generate handler file: %v", err)
	}

	// 验证生成的代码
	if !compareCode(string(structContractHandlers), string(code)) {
		t.Errorf("Generated code does not match expected:\nGot:\n%s\nExpected:\n%s", code, structContractHandlers)
	}
}

func TestGenerateHandlerFileWithMultipleReturns(t *testing.T) {
	// 加载预期的 ABI 数据
	var expectedABI ABITestData
	if err := json.Unmarshal(multiReturnContractABI, &expectedABI); err != nil {
		t.Fatalf("Failed to parse expected ABI: %v", err)
	}

	// 提取实际 ABI
	abi, err := ExtractABI(multiReturnContractCode)
	if err != nil {
		t.Fatalf("Failed to extract ABI: %v", err)
	}

	// 验证包名
	if abi.PackageName != expectedABI.PackageName {
		t.Errorf("Expected package name '%s', got '%s'", expectedABI.PackageName, abi.PackageName)
	}

	// 验证函数数量
	if len(abi.Functions) != len(expectedABI.Functions) {
		t.Errorf("Expected %d functions, got %d", len(expectedABI.Functions), len(abi.Functions))
	}

	// 验证每个函数
	for i, expectedFn := range expectedABI.Functions {
		if i >= len(abi.Functions) {
			t.Errorf("Missing function %d: %s", i, expectedFn.Name)
			continue
		}

		actualFn := abi.Functions[i]
		if actualFn.Name != expectedFn.Name {
			t.Errorf("Function %d: expected name '%s', got '%s'", i, expectedFn.Name, actualFn.Name)
		}

		// 验证输入参数
		if len(actualFn.Inputs) != len(expectedFn.Inputs) {
			t.Errorf("Function %s: expected %d inputs, got %d",
				actualFn.Name, len(expectedFn.Inputs), len(actualFn.Inputs))
			continue
		}

		// 验证每个输入参数
		for j, expectedInput := range expectedFn.Inputs {
			if j >= len(actualFn.Inputs) {
				t.Errorf("Function %s: missing input %d: %s %s",
					actualFn.Name, j, expectedInput.Name, expectedInput.Type)
				continue
			}

			actualInput := actualFn.Inputs[j]
			if actualInput.Name != expectedInput.Name {
				t.Errorf("Function %s: input %d: expected name '%s', got '%s'",
					actualFn.Name, j, expectedInput.Name, actualInput.Name)
			}
			if actualInput.Type != expectedInput.Type {
				t.Errorf("Function %s: input %d: expected type '%s', got '%s'",
					actualFn.Name, j, expectedInput.Type, actualInput.Type)
			}
		}

		// 验证输出参数
		if len(actualFn.Outputs) != len(expectedFn.Outputs) {
			t.Errorf("Function %s: expected %d outputs, got %d",
				actualFn.Name, len(expectedFn.Outputs), len(actualFn.Outputs))
			continue
		}

		// 验证每个输出参数
		for j, expectedOutput := range expectedFn.Outputs {
			if j >= len(actualFn.Outputs) {
				t.Errorf("Function %s: missing output %d: %s %s",
					actualFn.Name, j, expectedOutput.Name, expectedOutput.Type)
				continue
			}

			actualOutput := actualFn.Outputs[j]
			if actualOutput.Name != expectedOutput.Name {
				t.Errorf("Function %s: output %d: expected name '%s', got '%s'",
					actualFn.Name, j, expectedOutput.Name, actualOutput.Name)
			}
			if actualOutput.Type != expectedOutput.Type {
				t.Errorf("Function %s: output %d: expected type '%s', got '%s'",
					actualFn.Name, j, expectedOutput.Type, actualOutput.Type)
			}
		}
	}

	// 生成 handler 文件
	generatedCode, err := GenerateHandlerFile(abi)
	if err != nil {
		t.Fatalf("Failed to generate handler file: %v", err)
	}

	// 验证生成的代码
	expectedCode := string(multiReturnContractHandlers)
	if !compareCode(generatedCode, expectedCode) {
		t.Errorf("Generated code does not match expected code:\nExpected:\n%s\nGot:\n%s",
			expectedCode, generatedCode)
	}
}

func TestGenerateHandlerFileWithNoInputs(t *testing.T) {
	// 创建一个测试用的 ABI，包含无入参的函数
	abi := &ABI{
		PackageName: "testcontract",
		Functions: []Function{
			{
				Name:       "GetBalance",
				IsExported: true,
				Inputs:     []Parameter{}, // 无入参
				Outputs: []Parameter{
					{Name: "", Type: "uint64"},
				},
			},
			{
				Name:       "GetTimestamp",
				IsExported: true,
				Inputs:     []Parameter{}, // 无入参
				Outputs: []Parameter{
					{Name: "", Type: "int64"},
				},
			},
		},
	}

	// 生成 handler 文件
	code, err := GenerateHandlerFile(abi)
	if err != nil {
		t.Fatalf("Failed to generate handler file: %v", err)
	}

	// 验证生成的代码包含正确的函数签名
	expectedCode := `package testcontract
        
        import (
                "encoding/json"
                "fmt"
                "github.com/govm-net/vm/core"
        )
        
        type GetBalanceParams struct {
        }
        
        func handleGetBalance(ctx core.Context, params []byte) (any, error) {
                // 调用原始函数
                result0 := GetBalance()
        
                return result0, nil
        }
        
        type GetTimestampParams struct {
        }
        
        func handleGetTimestamp(ctx core.Context, params []byte) (any, error) {
                // 调用原始函数
                result0 := GetTimestamp()
        
                return result0, nil
        }
`
	if !compareCode(string(code), expectedCode) {
		t.Errorf("Generated code does not match expected:\nGot:\n%s\nExpected:\n%s", code, expectedCode)
	}
}

func TestFindImportForType(t *testing.T) {
	// 创建一个测试用的 ABI
	abi := &ABI{
		Imports: []Import{
			{Path: "github.com/govm-net/vm/core", Name: "core"},
			{Path: "github.com/govm-net/vm/types", Name: "vmtypes"},
			{Path: "math/big"},
		},
	}
	generator := NewHandlerGenerator(abi)

	tests := []struct {
		name     string
		typeStr  string
		wantPath string
		wantName string
	}{
		{
			name:     "basic type with alias",
			typeStr:  "core.Context",
			wantPath: "github.com/govm-net/vm/core",
			wantName: "core",
		},
		{
			name:     "pointer type with alias",
			typeStr:  "*core.Context",
			wantPath: "github.com/govm-net/vm/core",
			wantName: "core",
		},
		{
			name:     "array type with alias",
			typeStr:  "[]vmtypes.Address",
			wantPath: "github.com/govm-net/vm/types",
			wantName: "vmtypes",
		},
		{
			name:     "type without alias",
			typeStr:  "big.Int",
			wantPath: "math/big",
			wantName: "",
		},
		{
			name:     "pointer type without alias",
			typeStr:  "*big.Int",
			wantPath: "math/big",
			wantName: "",
		},
		{
			name:     "unknown type",
			typeStr:  "unknown.Type",
			wantPath: "",
			wantName: "",
		},
		{
			name:     "builtin type",
			typeStr:  "string",
			wantPath: "",
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generator.findImportForType(tt.typeStr)
			if tt.wantPath == "" {
				if got != nil {
					t.Errorf("findImportForType() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Errorf("findImportForType() = nil, want %v", tt.wantPath)
				return
			}
			if got.Path != tt.wantPath {
				t.Errorf("findImportForType() path = %v, want %v", got.Path, tt.wantPath)
			}
			if got.Name != tt.wantName {
				t.Errorf("findImportForType() name = %v, want %v", got.Name, tt.wantName)
			}
		})
	}
}
