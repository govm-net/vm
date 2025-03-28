// Package vm 实现了基于WebAssembly的虚拟机核心功能
package vm

import (
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

	// 测试部署合约
	contractAddr, err := svm.DeployContract(code, core.Address{})
	if err != nil {
		t.Fatalf("DeployContract() error = %v", err)
	}

	// 测试执行合约
	result, err := svm.ExecuteContract(contractAddr, core.Address{}, "Initialize", nil)
	if err != nil {
		t.Fatalf("ExecuteContract() error = %v", err)
	}
	fmt.Printf("result: %v, type: %T\n", result, result)

	// 测试执行合约
	result, err = svm.ExecuteContract(contractAddr, core.Address{}, "Increment", []byte(`{"amount": 2}`))
	if err != nil {
		t.Fatalf("ExecuteContract() error = %v", err)
	}
	fmt.Printf("result: %v, type: %T\n", result, result)

	// 测试执行合约
	result, err = svm.ExecuteContract(contractAddr, core.Address{}, "Increment", []byte(`{"amount": 2}`))
	if err != nil {
		t.Fatalf("ExecuteContract() error = %v", err)
	}
	fmt.Printf("result: %v, type: %T\n", result, result)

	t.Error(result)
}
