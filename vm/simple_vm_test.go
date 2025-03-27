// Package vm 实现了基于WebAssembly的虚拟机核心功能
package vm

import (
	"fmt"
	"os"
	"testing"

	"github.com/govm-net/vm/core"
)

func TestNewSimpleVM(t *testing.T) {
	svm, err := NewSimpleVM("")
	if err != nil {
		t.Fatalf("NewSimpleVM() error = %v", err)
	}
	if svm == nil {
		t.Fatalf("NewSimpleVM() = nil, want non-nil")
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
	fmt.Println(result)
	t.Error(result)
}
