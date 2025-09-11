package vm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/govm-net/vm/context/memory"
	"github.com/govm-net/vm/interfaces"
)

func TestNewEngineWithCustomModules(t *testing.T) {
	// 创建临时目录用于测试
	tmpDir, err := os.MkdirTemp("", "engine_options_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建配置
	config := &interfaces.Config{
		MaxContractSize:  1024 * 1024, // 1MB
		WASIContractsDir: filepath.Join(tmpDir, "contracts"),
		CodeManagerDir:   filepath.Join(tmpDir, "code"),
		ContextType:      "memory",
	}

	// 测试使用自定义上下文创建引擎
	customContext := memory.NewBlockchainContext(nil)
	engine, err := NewEngine(config, WithContext(customContext))
	if err != nil {
		t.Fatalf("NewEngine() with custom context error = %v", err)
	}
	if engine == nil {
		t.Fatal("NewEngine() returned nil")
	}

	// 验证使用的是否是自定义上下文
	if engine.GetContext() != customContext {
		t.Error("Engine did not use the custom context")
	}

	// 测试关闭引擎
	if err := engine.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestNewEngineWithDefaultModules(t *testing.T) {
	// 创建临时目录用于测试
	tmpDir, err := os.MkdirTemp("", "engine_options_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建配置
	config := &interfaces.Config{
		MaxContractSize:  1024 * 1024, // 1MB
		WASIContractsDir: filepath.Join(tmpDir, "contracts"),
		CodeManagerDir:   filepath.Join(tmpDir, "code"),
		ContextType:      "memory",
	}

	// 测试使用默认模块创建引擎
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("NewEngine() with default modules error = %v", err)
	}
	if engine == nil {
		t.Fatal("NewEngine() returned nil")
	}

	// 测试关闭引擎
	if err := engine.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
