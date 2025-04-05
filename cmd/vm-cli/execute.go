package main

import (
	"encoding/json"
	"fmt"

	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/vm"
)

func runExecute(contractAddr, funcName, argsJSON, sender, repoDir string) error {
	// 检查必需参数
	if contractAddr == "" {
		return fmt.Errorf("contract address is required")
	}
	if funcName == "" {
		return fmt.Errorf("function name is required")
	}
	if sender == "" {
		return fmt.Errorf("sender address is required")
	}
	if repoDir == "" {
		return fmt.Errorf("repo directory is required")
	}

	// 解析合约地址
	address := core.AddressFromString(contractAddr)

	// 创建VM引擎配置
	config := &vm.Config{
		MaxContractSize:  1024 * 1024, // 1MB
		CodeManagerDir:   repoDir,
		WASIContractsDir: repoDir,
		ContextType:      "db",
	}

	// 创建VM引擎
	engine, err := vm.NewEngine(config)
	if err != nil {
		return fmt.Errorf("failed to create VM engine: %w", err)
	}
	defer engine.Close()

	// 解析参数
	var params []byte
	if argsJSON != "" {
		params = []byte(argsJSON)
	}

	// 执行合约函数
	result, err := engine.Execute(address, funcName, params)
	if err != nil {
		return fmt.Errorf("failed to execute contract: %w", err)
	}

	// 打印执行结果
	if result != nil {
		resultJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}
		fmt.Printf("Execution result:\n%s\n", string(resultJSON))
	} else {
		fmt.Printf("Function executed successfully with no return value\n")
	}

	return nil
}
