package main

import (
	"encoding/json"
	"fmt"

	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/vm"
)

func runExecute(contractAddr, funcName, argsJSON, sender, wasmDir string) error {
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
	if wasmDir == "" {
		return fmt.Errorf("wasm directory is required")
	}

	// 解析合约地址
	address := core.AddressFromString(contractAddr)
	fmt.Println("address", address)

	// 创建VM引擎配置
	config := &vm.Config{
		MaxContractSize:  1024 * 1024, // 1MB
		WASIContractsDir: wasmDir,
		CodeManagerDir:   ".code",
		ContextType:      "db",
	}

	// 创建VM引擎
	engine, err := vm.NewEngine(config)
	if err != nil {
		return fmt.Errorf("failed to create VM engine: %w", err)
	}
	defer engine.Close()

	ctx := engine.GetContext()
	ctx.SetBlockInfo(1, 1, core.HashFromString("0x1234567890"))
	ctx.SetTransactionInfo(core.HashFromString("0x1234567890ab"), core.AddressFromString(sender), core.AddressFromString(contractAddr), 1000)

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
