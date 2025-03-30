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
	ctx := svm.blockchainCtx.(*defaultBlockchainContext)
	ctx.SetExecutionContext(contractAddr, core.Address{})

	// 测试执行合约
	result, err := svm.ExecuteContract(ctx, contractAddr, core.Address{}, "Initialize", nil)
	if err != nil {
		t.Fatalf("ExecuteContract() error = %v", err)
	}
	fmt.Printf("result: %v, type: %T\n", result, result)

	// // 测试执行合约
	// result, err = svm.ExecuteContract(ctx, contractAddr, core.Address{}, "Increment", []byte(`{"amount": 2}`))
	// if err != nil {
	// 	t.Fatalf("ExecuteContract() error = %v", err)
	// }
	// fmt.Printf("result: %v, type: %T\n", result, result)

	// 测试执行合约
	// result, err = svm.ExecuteContract(ctx, contractAddr, core.Address{}, "Increment", []byte(`{"amount": 2}`))
	// if err != nil {
	// 	t.Fatalf("ExecuteContract() error = %v", err)
	// }
	// fmt.Printf("result: %v, type: %T\n", result, result)
	// var rst types.ExecutionResult
	// err = json.Unmarshal(result, &rst)
	// if err != nil {
	// 	t.Fatalf("Unmarshal() error = %v", err)
	// }
	// fmt.Printf("rst: %v, type: %T\n", rst, rst)
	// if rst.Success != true {
	// 	t.Fatalf("ExecutionStatus = %v, want %v", rst.Success, true)
	// }
	// if rst.Data == nil {
	// 	t.Fatalf("Data = %v, want %v", rst.Data, nil)
	// }
	// if fmt.Sprintf("%v", rst.Data) != "4" {
	// 	t.Fatalf("Data = %v, want %v", rst.Data, "4")
	// }
	// 测试执行合约
	// result, err = svm.ExecuteContract(ctx, contractAddr, core.Address{}, "Panic", nil)
	// if err == nil {
	// 	t.Fatalf("ExecuteContract() error = %v", err)
	// }
	// if result != nil {
	// 	t.Fatalf("result = %v, want %v", result, nil)
	// }

	// result, err = svm.ExecuteContract(ctx, contractAddr, core.Address{}, "Increment", []byte(`{"amount": 2}`))
	// if err != nil {
	// 	t.Fatalf("ExecuteContract() error = %v", err)
	// }
	// fmt.Printf("result: %v, type: %T\n", result, result)
}
