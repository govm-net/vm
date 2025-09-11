package vm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/govm-net/vm/context/memory"
	"github.com/govm-net/vm/interfaces"
)

func TestSimpleContract(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "simple_contract_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建配置
	config := &interfaces.Config{
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

	// 简单的测试合约代码（不使用core包）
	contractCode := []byte(`package testcontract

func Initialize() int32 {
	return 0
}

func GetValue() uint64 {
	return 42
}`)

	// 部署合约
	contractAddr, err := engine.DeployContract(contractCode)
	if err != nil {
		t.Fatalf("DeployContract() error = %v", err)
	}
	defer engine.DeleteContract(contractAddr)

	// 执行Initialize函数
	_, err = engine.ExecuteContract(contractAddr, "Initialize")
	if err != nil {
		t.Fatalf("ExecuteContract(Initialize) error = %v", err)
	}

	// 执行GetValue函数
	result, err := engine.ExecuteContract(contractAddr, "GetValue")
	if err != nil {
		t.Fatalf("ExecuteContract(GetValue) error = %v", err)
	}

	// 验证返回值
	if result != uint64(42) {
		t.Errorf("GetValue returned %v, want 42", result)
	}
}
