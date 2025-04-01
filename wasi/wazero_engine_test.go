// Package vm 实现了基于WebAssembly的虚拟机核心功能
package wasi

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/govm-net/vm/core"
)

func TestNewWazeroVM(t *testing.T) {
	svm, err := NewWazeroVM("")
	if err != nil {
		t.Fatalf("NewWazeroVM() error = %v", err)
	}
	if svm == nil {
		t.Fatalf("NewWazeroVM() = nil, want non-nil")
	}

	code, err := os.ReadFile("../wasm/contract.wasm")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var resultValue uint64

	ctx := NewDefaultBlockchainContext()
	sender := core.Address{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14}

	// 测试部署合约
	contractAddr, err := svm.DeployContract(ctx, code, sender)
	if err != nil {
		t.Fatalf("DeployContract() error = %v", err)
	}
	defer svm.DeleteContract(ctx, contractAddr)
	ctx.SetExecutionContext(contractAddr, sender)

	// 测试执行合约
	result, err := svm.ExecuteContract(ctx, contractAddr, "Initialize", nil)
	if err != nil {
		t.Fatalf("ExecuteContract() error = %v", err)
	}
	fmt.Printf("result1: %s, type: %T\n", result, result)
	d, _ := json.Marshal(result)
	json.Unmarshal(d, &resultValue)
	if resultValue != 0 {
		t.Fatalf("Initialize() error = %v", resultValue)
	}

	// 测试执行合约
	result, err = svm.ExecuteContract(ctx, contractAddr, "Increment", []byte(`{"amount": 2}`))
	if err != nil {
		t.Fatalf("ExecuteContract() error = %v", err)
	}

	fmt.Printf("result2: %s, type: %T\n", result, result)
	d, _ = json.Marshal(result)
	json.Unmarshal(d, &resultValue)
	if resultValue != 2 {
		t.Fatalf("Increment() error = %v", resultValue)
	}

	// 测试执行合约
	result, err = svm.ExecuteContract(ctx, contractAddr, "Increment", []byte(`{"amount": 2}`))
	if err != nil {
		t.Fatalf("ExecuteContract() error = %v", err)
	}
	fmt.Printf("result3: %s, type: %T\n", result, result)
	d, _ = json.Marshal(result)
	json.Unmarshal(d, &resultValue)
	if resultValue != 4 {
		t.Fatalf("Increment() error = %v", resultValue)
	}

	result, err = svm.ExecuteContract(ctx, contractAddr, "Panic", nil)
	if err == nil {
		t.Fatalf("ExecuteContract() error = %v", err)
	}
	if result != nil {
		t.Fatalf("result4 = %s, want %v", result, nil)
	}
	// t.Error(err)
}
