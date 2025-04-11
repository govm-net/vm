package main

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	_ "embed"

	_ "github.com/govm-net/vm/context/db"
	_ "github.com/govm-net/vm/context/memory"
	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/vm"
)

//go:embed testdata/counter.go
var counterCode []byte

// TestParallelExecution 测试并行执行合约的性能
func TestParallelExecution(t *testing.T) {
	// 配置参数
	contractAddr := "0x1234567890123456789012345678901234567890"
	funcName := "Transfer"
	wasmDir := "wasm"
	sender := "0xabcdef1234567890abcdef1234567890abcdef12"

	// 创建测试数据
	transfers := []struct {
		to     string
		amount uint64
	}{
		{"0x1111111111111111111111111111111111111111", 100},
		{"0x2222222222222222222222222222222222222222", 200},
		{"0x3333333333333333333333333333333333333333", 300},
		{"0x4444444444444444444444444444444444444444", 400},
		{"0x5555555555555555555555555555555555555555", 500},
	}

	// 创建VM引擎配置
	config := &vm.Config{
		MaxContractSize:  1024 * 1024, // 1MB
		WASIContractsDir: wasmDir,
		CodeManagerDir:   "code",
		ContextType:      "db",
	}

	// 创建VM引擎
	engine, err := vm.NewEngine(config)
	if err != nil {
		t.Fatalf("failed to create VM engine: %v", err)
	}
	defer engine.Close()

	// 设置上下文
	ctx := engine.GetContext()
	ctx.SetBlockInfo(1, 1, core.HashFromString("0x1234567890"))
	ctx.SetTransactionInfo(
		core.HashFromString("0x1234567890ab"),
		core.AddressFromString(sender),
		core.AddressFromString(contractAddr),
		1000,
	)

	// 测试串行执行
	startTime := time.Now()
	for _, transfer := range transfers {
		args := map[string]interface{}{
			"to":     transfer.to,
			"amount": transfer.amount,
		}
		argsJSON, _ := json.Marshal(args)

		_, err := engine.Execute(
			core.AddressFromString(contractAddr),
			funcName,
			argsJSON,
		)
		if err != nil {
			t.Errorf("serial execution failed: %v", err)
		}
	}
	serialDuration := time.Since(startTime)
	t.Logf("Serial execution took: %v", serialDuration)

	// 测试并行执行
	startTime = time.Now()
	var wg sync.WaitGroup
	for _, transfer := range transfers {
		wg.Add(1)
		go func(to string, amount uint64) {
			defer wg.Done()

			args := map[string]interface{}{
				"to":     to,
				"amount": amount,
			}
			argsJSON, _ := json.Marshal(args)

			_, err := engine.Execute(
				core.AddressFromString(contractAddr),
				funcName,
				argsJSON,
			)
			if err != nil {
				t.Errorf("parallel execution failed: %v", err)
			}
		}(transfer.to, transfer.amount)
	}
	wg.Wait()
	parallelDuration := time.Since(startTime)
	t.Logf("Parallel execution took: %v", parallelDuration)

	// 计算性能提升
	speedup := float64(serialDuration) / float64(parallelDuration)
	t.Logf("Speedup: %.2fx", speedup)

	// 输出详细统计信息
	t.Logf("\nPerformance Statistics:")
	t.Logf("Number of transactions: %d", len(transfers))
	t.Logf("Serial execution time: %v", serialDuration)
	t.Logf("Parallel execution time: %v", parallelDuration)
	t.Logf("Average serial time per transaction: %v", serialDuration/time.Duration(len(transfers)))
	t.Logf("Average parallel time per transaction: %v", parallelDuration/time.Duration(len(transfers)))
	t.Logf("Speedup factor: %.2fx", speedup)
}

// BenchmarkParallelExecution 基准测试并行执行性能
func BenchmarkParallelExecution(b *testing.B) {
	// 配置参数
	// contractAddr := "0x1234567890123456789012345678901234567890"
	funcName := "Increment"
	wasmDir := "wasm"
	sender := "0xabcdef1234567890abcdef1234567890abcdef12"

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
