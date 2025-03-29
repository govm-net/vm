package vm

import (
	_ "embed"
	"encoding/json"
	"testing"
)

//go:embed testdata/test_contract.go
var testContractCode []byte

//go:embed testdata/counter_contract.go
var counterContractCode []byte

//go:embed testdata/counter_contract.events.json
var counterContractEvents []byte

//go:embed testdata/struct_contract.go
var structContractCode []byte

//go:embed testdata/struct_contract.abi.json
var structContractABI []byte

// EventTestData represents the test data for events
type EventTestData struct {
	Events []Event `json:"events"`
}

// ABITestData represents the test data for ABI
type ABITestData struct {
	PackageName string     `json:"package_name"`
	Functions   []Function `json:"functions"`
	Events      []Event    `json:"events"`
}

func TestExtractABI(t *testing.T) {
	// 测试简单合约
	abi, err := ExtractABI(testContractCode)
	if err != nil {
		t.Fatalf("Failed to extract ABI: %v", err)
	}

	// 验证包名
	if abi.PackageName != "testcontract" {
		t.Errorf("Expected package name 'testcontract', got '%s'", abi.PackageName)
	}

	// 验证函数
	if len(abi.Functions) != 2 {
		t.Errorf("Expected 2 function, got %d", len(abi.Functions))
	}

	// 验证事件
	if len(abi.Events) != 0 {
		t.Errorf("Expected 0 event, got %d", len(abi.Events))
	}

	// 测试计数器合约
	abi, err = ExtractABI(counterContractCode)
	if err != nil {
		t.Fatalf("Failed to extract counter contract ABI: %v", err)
	}

	// 验证包名
	if abi.PackageName != "countercontract" {
		t.Errorf("Expected package name 'countercontract', got '%s'", abi.PackageName)
	}

	// 验证函数
	if len(abi.Functions) != 4 {
		t.Errorf("Expected 4 functions, got %d", len(abi.Functions))
	}

	// 加载预期的事件数据
	var expectedEvents EventTestData
	if err := json.Unmarshal(counterContractEvents, &expectedEvents); err != nil {
		t.Fatalf("Failed to parse expected events: %v", err)
	}

	// 验证事件数量
	if len(abi.Events) != len(expectedEvents.Events) {
		t.Errorf("Expected %d events, got %d", len(expectedEvents.Events), len(abi.Events))
	}

	// 验证每个事件
	for i, expectedEvent := range expectedEvents.Events {
		if i >= len(abi.Events) {
			t.Errorf("Missing event %d: %s", i, expectedEvent.Name)
			continue
		}

		actualEvent := abi.Events[i]
		if actualEvent.Name != expectedEvent.Name {
			t.Errorf("Event %d: expected name '%s', got '%s'", i, expectedEvent.Name, actualEvent.Name)
		}

		// 验证参数数量
		if len(actualEvent.Parameters) != len(expectedEvent.Parameters) {
			t.Errorf("Event %s: expected %d parameters, got %d",
				actualEvent.Name, len(expectedEvent.Parameters), len(actualEvent.Parameters))
			continue
		}

		// 验证每个参数
		for j, expectedParam := range expectedEvent.Parameters {
			if j >= len(actualEvent.Parameters) {
				t.Errorf("Event %s: missing parameter %d: %s %s",
					actualEvent.Name, j, expectedParam.Name, expectedParam.Type)
				continue
			}

			actualParam := actualEvent.Parameters[j]
			if actualParam.Name != expectedParam.Name {
				t.Errorf("Event %s: parameter %d: expected name '%s', got '%s'",
					actualEvent.Name, j, expectedParam.Name, actualParam.Name)
			}
		}
	}
}

// 测试自定义结构体参数
func TestCustomStructParams(t *testing.T) {
	// 加载预期的 ABI 数据
	var expectedABI ABITestData
	if err := json.Unmarshal(structContractABI, &expectedABI); err != nil {
		t.Fatalf("Failed to parse expected ABI: %v", err)
	}

	// 提取实际 ABI
	abi, err := ExtractABI(structContractCode)
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

	// 验证事件
	if len(abi.Events) != len(expectedABI.Events) {
		t.Errorf("Expected %d events, got %d", len(expectedABI.Events), len(abi.Events))
	}
}
