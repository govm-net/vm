package main

import (
	"encoding/json"
	"os"
	"sync"
	"testing"

	_ "embed"

	_ "github.com/govm-net/vm/context/db"
	_ "github.com/govm-net/vm/context/memory"
	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/vm"
)

//go:embed testdata/counter.go
var counterCode []byte

// BenchmarkParallelExecution 基准测试并行执行性能
func BenchmarkParallelExecution(b *testing.B) {
	// 配置参数
	// contractAddr := "0x1234567890123456789012345678901234567890"
	funcName := "Increment"
	wasmDir := "wasm"
	sender := "0xabcdef1234567890abcdef1234567890abcdef12"
	defer func() {
		os.RemoveAll("wasm")
		os.RemoveAll("code")
	}()

	// 创建VM引擎配置
	config := &vm.Config{
		MaxContractSize:  1024 * 1024, // 1MB
		WASIContractsDir: wasmDir,
		CodeManagerDir:   "code",
		ContextType:      "memory",
	}

	// 创建VM引擎
	engine, err := vm.NewEngine(config)
	if err != nil {
		b.Fatalf("failed to create VM engine: %v", err)
	}
	defer engine.Close()

	contractAddr, err := engine.DeployContract(counterCode)
	if err != nil {
		b.Fatalf("failed to deploy contract: %v", err)
	}

	// 设置上下文
	ctx := engine.GetContext()
	ctx.SetBlockInfo(1, 1, core.HashFromString("0x1234567890"))
	ctx.SetTransactionInfo(
		core.HashFromString("0x1234567890ab"),
		core.AddressFromString(sender),
		contractAddr,
		1000,
	)

	_, err = engine.Execute(
		contractAddr,
		"Initialize",
		nil,
	)
	if err != nil {
		b.Fatalf("failed to initialize contract: %v", err)
	}

	// 准备测试数据
	args := map[string]interface{}{
		"value": 5,
	}
	argsJSON, _ := json.Marshal(args)

	// 重置计时器
	b.ResetTimer()

	// 运行基准测试
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < 5; j++ { // 每次迭代执行5个并行交易
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := engine.Execute(
					contractAddr,
					funcName,
					argsJSON,
				)
				if err != nil {
					b.Errorf("parallel execution failed: %v", err)
				}
			}()
		}
		wg.Wait()
	}
}
