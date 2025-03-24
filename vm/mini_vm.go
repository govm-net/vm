// Package vm 提供基于WebAssembly的虚拟机实现
package vm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/wasmerio/wasmer-go/wasmer"
)

// MiniVM 是一个最小化的WebAssembly虚拟机实现
type MiniVM struct {
	// 合约代码存储
	contracts     map[string][]byte
	contractsLock sync.RWMutex

	// 区块信息
	blockHeight uint64
	blockTime   int64

	// 账户余额
	balances map[string]uint64

	// 合约存储目录
	contractDir string
}

// NewMiniVM 创建一个新的最小化虚拟机实例
func NewMiniVM(contractDir string) (*MiniVM, error) {
	// 确保合约目录存在
	if contractDir != "" {
		if err := os.MkdirAll(contractDir, 0755); err != nil {
			return nil, fmt.Errorf("创建合约目录失败: %w", err)
		}
	}

	return &MiniVM{
		contracts:   make(map[string][]byte),
		balances:    make(map[string]uint64),
		contractDir: contractDir,
	}, nil
}

// SetBlockInfo 设置当前区块信息
func (vm *MiniVM) SetBlockInfo(height uint64, time int64) {
	vm.blockHeight = height
	vm.blockTime = time
}

// SetBalance 设置账户余额
func (vm *MiniVM) SetBalance(addr string, balance uint64) {
	vm.balances[addr] = balance
}

// DeployContract 部署新的WebAssembly合约
func (vm *MiniVM) DeployContract(wasmCode []byte) (string, error) {
	// 验证WASM代码
	if len(wasmCode) == 0 {
		return "", errors.New("合约代码不能为空")
	}

	// 编译检查WASM模块
	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)
	_, err := wasmer.NewModule(store, wasmCode)
	if err != nil {
		return "", fmt.Errorf("无效的WebAssembly模块: %w", err)
	}

	// 生成合约地址 (简化版，使用前10字节作为ID)
	contractAddr := fmt.Sprintf("%x", wasmCode[:10])

	// 存储合约代码
	vm.contractsLock.Lock()
	vm.contracts[contractAddr] = wasmCode
	vm.contractsLock.Unlock()

	// 如果指定了合约目录，则保存到文件
	if vm.contractDir != "" {
		contractPath := filepath.Join(vm.contractDir, contractAddr+".wasm")
		if err := os.WriteFile(contractPath, wasmCode, 0644); err != nil {
			return "", fmt.Errorf("存储合约代码失败: %w", err)
		}
	}

	return contractAddr, nil
}

// ExecuteContract 执行已部署的合约函数
func (vm *MiniVM) ExecuteContract(
	contractAddr string,
	sender string,
	functionName string,
	args ...interface{},
) (interface{}, error) {
	// 检查合约是否存在
	vm.contractsLock.RLock()
	wasmCode, exists := vm.contracts[contractAddr]
	vm.contractsLock.RUnlock()

	if !exists {
		return nil, fmt.Errorf("合约不存在: %s", contractAddr)
	}

	// 创建执行上下文
	ctx := &miniContext{
		vm:           vm,
		contractAddr: contractAddr,
		sender:       sender,
	}

	// 创建WASM实例
	instance, err := vm.createWasmInstance(wasmCode, ctx)
	if err != nil {
		return nil, fmt.Errorf("创建WASM实例失败: %w", err)
	}
	defer instance.Close()

	// 找到目标函数
	exports := instance.Exports
	fn, err := exports.GetFunction(functionName)
	if err != nil {
		return nil, fmt.Errorf("合约函数不存在: %s", functionName)
	}

	// 执行函数
	result, err := fn(args...)
	if err != nil {
		return nil, fmt.Errorf("执行合约函数失败: %w", err)
	}

	return result, nil
}

// createWasmInstance 创建WebAssembly实例
func (vm *MiniVM) createWasmInstance(wasmCode []byte, ctx *miniContext) (*wasmer.Instance, error) {
	// 创建WASM引擎和存储
	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)

	// 编译模块
	module, err := wasmer.NewModule(store, wasmCode)
	if err != nil {
		return nil, fmt.Errorf("编译WebAssembly模块失败: %w", err)
	}

	// 创建导入对象，提供宿主函数
	importObject := vm.createImportObject(store, ctx)

	// 实例化模块
	instance, err := wasmer.NewInstance(module, importObject)
	if err != nil {
		return nil, fmt.Errorf("实例化WebAssembly模块失败: %w", err)
	}

	return instance, nil
}

// createImportObject 创建导入对象，为WebAssembly提供宿主函数
func (vm *MiniVM) createImportObject(store *wasmer.Store, ctx *miniContext) *wasmer.ImportObject {
	// 创建导入对象
	importObject := wasmer.NewImportObject()

	// 创建环境函数
	envFunctions := make(map[string]wasmer.IntoExtern)

	// 添加基本环境函数 - 块高度
	envFunctions["get_block_height"] = wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I64)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			return []wasmer.Value{wasmer.NewI64(int64(vm.blockHeight))}, nil
		},
	)

	// 添加基本环境函数 - 块时间
	envFunctions["get_block_time"] = wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I64)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			return []wasmer.Value{wasmer.NewI64(vm.blockTime)}, nil
		},
	)

	// 添加基本环境函数 - 获取合约地址
	envFunctions["get_contract_address"] = wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(
			wasmer.NewValueTypes(wasmer.I32),
			wasmer.NewValueTypes(wasmer.I32),
		),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			// 在实际实现中，需要将合约地址写入WebAssembly内存
			// 此处简化返回写入的字节数
			return []wasmer.Value{wasmer.NewI32(int32(len(ctx.contractAddr)))}, nil
		},
	)

	// 添加基本环境函数 - 获取发送者地址
	envFunctions["get_sender"] = wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(
			wasmer.NewValueTypes(wasmer.I32),
			wasmer.NewValueTypes(wasmer.I32),
		),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			// 在实际实现中，需要将发送者地址写入WebAssembly内存
			// 此处简化返回写入的字节数
			return []wasmer.Value{wasmer.NewI32(int32(len(ctx.sender)))}, nil
		},
	)

	// 注册环境命名空间
	env := make(map[string]wasmer.IntoExtern)
	for k, v := range envFunctions {
		env[k] = v
	}
	importObject.Register("env", env)

	return importObject
}

// miniContext 是合约执行上下文
type miniContext struct {
	vm           *MiniVM
	contractAddr string
	sender       string
}

// GetBlockHeight 获取当前区块高度
func (ctx *miniContext) GetBlockHeight() uint64 {
	return ctx.vm.blockHeight
}

// GetBlockTime 获取当前区块时间戳
func (ctx *miniContext) GetBlockTime() int64 {
	return ctx.vm.blockTime
}

// GetContractAddress 获取当前合约地址
func (ctx *miniContext) GetContractAddress() string {
	return ctx.contractAddr
}

// GetSender 获取交易发送者
func (ctx *miniContext) GetSender() string {
	return ctx.sender
}

// GetBalance 获取账户余额
func (ctx *miniContext) GetBalance(addr string) uint64 {
	return ctx.vm.balances[addr]
}

// Transfer 转账操作
func (ctx *miniContext) Transfer(to string, amount uint64) error {
	// 检查余额
	fromBalance := ctx.vm.balances[ctx.contractAddr]
	if fromBalance < amount {
		return errors.New("余额不足")
	}

	// 执行转账
	ctx.vm.balances[ctx.contractAddr] -= amount
	ctx.vm.balances[to] += amount
	return nil
}
