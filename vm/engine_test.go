package vm

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/govm-net/vm/context/memory"
	"github.com/govm-net/vm/core"
)

//go:embed testdata/counter_contract.go
var counterContractCode []byte

func TestNewEngine(t *testing.T) {
	// 创建临时目录用于测试
	tmpDir, err := os.MkdirTemp("", "engine_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建配置
	config := &Config{
		MaxContractSize:  1024 * 1024, // 1MB
		WASIContractsDir: filepath.Join(tmpDir, "contracts"),
		CodeManagerDir:   filepath.Join(tmpDir, "code"),
		ContextType:      "memory",
	}

	// 创建引擎实例
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}
	if engine == nil {
		t.Fatal("NewEngine() returned nil")
	}

	// 测试关闭引擎
	if err := engine.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestEngine_DeployAndExecuteContract(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "engine_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建配置
	config := &Config{
		MaxContractSize:  1024 * 1024,
		WASIContractsDir: filepath.Join(tmpDir, "contracts"),
		CodeManagerDir:   filepath.Join(tmpDir, "code"),
		ContextType:      "memory",
	}

	// 创建引擎实例
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}
	defer engine.Close()

	ctx := memory.NewBlockchainContext(nil)
	engine = engine.WithContext(ctx)

	// 测试合约代码
	contractCode := counterContractCode

	// 部署合约
	contractAddr, err := engine.DeployContract(contractCode)
	if err != nil {
		t.Fatalf("DeployContract() error = %v", err)
	}
	defer engine.DeleteContract(contractAddr)
	if contractAddr == core.ZeroAddress {
		t.Fatal("DeployContract() returned zero address")
	}

	// 验证ABI文件是否创建
	abiPath := filepath.Join(config.WASIContractsDir, fmt.Sprintf("%x.abi", contractAddr))
	if _, err := os.Stat(abiPath); os.IsNotExist(err) {
		t.Fatal("ABI file was not created")
	}
	// ctx.SetExecutionContext(contractAddr, core.ZeroAddress)

	// 执行Initialize函数
	_, err = engine.Execute(contractAddr, "Initialize", nil)
	if err != nil {
		t.Fatalf("Execute(Initialize) error = %v", err)
	}

	// 执行Increment函数
	_, err = engine.Execute(contractAddr, "Increment", []byte(`{"value": 5}`))
	if err != nil {
		t.Fatalf("Execute(Increment) error = %v", err)
	}

	// GetCounter
	result, err := engine.Execute(contractAddr, "GetCounter", nil)
	if err != nil {
		t.Fatalf("Execute(GetCounter) error = %v", err)
	}
	var hopeResult uint64
	d, _ := json.Marshal(result)
	json.Unmarshal(d, &hopeResult)
	if hopeResult != 5 {
		t.Errorf("GetCounter returned %d, want 5", hopeResult)
	}

	// 测试不存在的函数
	_, err = engine.ExecuteContract(contractAddr, "NonExistentFunction")
	if err == nil {
		t.Error("ExecuteContract(NonExistentFunction) should have returned error")
	}

	// 测试不存在的合约
	_, err = engine.ExecuteContract(core.Address{1}, "GetCounter")
	if err == nil {
		t.Error("ExecuteContract with non-existent address should have returned error")
	}
}

func TestEngine_DeployAndExecuteContract2(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "engine_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建配置
	config := &Config{
		MaxContractSize:  1024 * 1024,
		WASIContractsDir: filepath.Join(tmpDir, "contracts"),
		CodeManagerDir:   filepath.Join(tmpDir, "code"),
		ContextType:      "memory",
	}

	// 创建引擎实例
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}
	defer engine.Close()

	ctx := memory.NewBlockchainContext(nil)
	engine = engine.WithContext(ctx)

	// 测试合约代码
	contractCode := counterContractCode

	// 部署合约
	contractAddr, err := engine.DeployContract(contractCode)
	if err != nil {
		t.Fatalf("DeployContract() error = %v", err)
	}
	defer engine.DeleteContract(contractAddr)
	if contractAddr == core.ZeroAddress {
		t.Fatal("DeployContract() returned zero address")
	}

	// 验证ABI文件是否创建
	abiPath := filepath.Join(config.WASIContractsDir, fmt.Sprintf("%x.abi", contractAddr))
	if _, err := os.Stat(abiPath); os.IsNotExist(err) {
		t.Fatal("ABI file was not created")
	}
	// ctx.SetExecutionContext(contractAddr, core.ZeroAddress)

	// 执行Initialize函数
	_, err = engine.ExecuteContract(contractAddr, "Initialize")
	if err != nil {
		t.Fatalf("ExecuteContract(Initialize) error = %v", err)
	}

	// 执行Increment函数
	_, err = engine.ExecuteContract(contractAddr, "Increment", 5)
	if err != nil {
		t.Fatalf("ExecuteContract(Increment) error = %v", err)
	}

	// GetCounter
	result, err := engine.ExecuteContract(contractAddr, "GetCounter")
	if err != nil {
		t.Fatalf("ExecuteContract(GetCounter) error = %v", err)
	}
	var hopeResult uint64
	d, _ := json.Marshal(result)
	json.Unmarshal(d, &hopeResult)
	if hopeResult != 5 {
		t.Errorf("GetCounter returned %d, want 5", hopeResult)
	}

}
