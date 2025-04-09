package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/govm-net/vm/context"
	_ "github.com/govm-net/vm/context/db"
	_ "github.com/govm-net/vm/context/memory"
	"github.com/govm-net/vm/vm"
)

func runDeploy(sourceFile, repoDir, wasmDir string) error {
	// 检查必需参数
	if sourceFile == "" {
		return fmt.Errorf("source file is required")
	}

	// 读取源代码文件
	code, err := os.ReadFile(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// 获取当前工作目录
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// 如果路径是相对路径，则基于当前目录
	if !filepath.IsAbs(repoDir) {
		repoDir = filepath.Join(currentDir, repoDir)
	}
	if !filepath.IsAbs(wasmDir) {
		wasmDir = filepath.Join(currentDir, wasmDir)
	}

	// 确保目录存在
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return fmt.Errorf("failed to create repo directory: %w", err)
	}
	if err := os.MkdirAll(wasmDir, 0755); err != nil {
		return fmt.Errorf("failed to create wasm directory: %w", err)
	}

	// 创建VM引擎配置
	config := &vm.Config{
		MaxContractSize:  1024 * 1024, // 1MB
		CodeManagerDir:   repoDir,
		WASIContractsDir: wasmDir,
		ContextType:      string(context.DBContextType),
	}

	slog.Info("deploying contract", "config", config)

	// 创建VM引擎
	engine, err := vm.NewEngine(config)
	if err != nil {
		return fmt.Errorf("failed to create VM engine: %w", err)
	}
	defer engine.Close()

	// 部署合约
	address, err := engine.DeployContract(code)
	if err != nil {
		return fmt.Errorf("failed to deploy contract: %w", err)
	}

	fmt.Printf("Contract deployed successfully!\n")
	fmt.Printf("Contract address: %s\n", address)
	fmt.Printf("Contract files are stored in: %s\n", filepath.Join(repoDir, address.String()))

	return nil
}
