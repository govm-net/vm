// Package vm 实现了基于WebAssembly的虚拟机核心功能
package vm

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/types"
	"github.com/wasmerio/wasmer-go/wasmer"
)

// WasmEngine 管理WebAssembly模块的加载和执行
type WasmEngine struct {
	// 存储已部署合约的映射表
	contracts     map[core.Address]string
	contractsLock sync.RWMutex

	// 合约存储目录
	contractDir string

	// 资源限制配置
	memoryLimit        uint64
	executionTimeLimit uint64
	instructionsLimit  uint64

	// 全局状态
	state *HostState
}

// HostState 存储全局状态
type HostState struct {
	CurrentSender   core.Address
	CurrentBlock    uint64
	CurrentTime     int64
	ContractAddress core.Address
	Balances        map[core.Address]uint64
	Objects         map[core.ObjectID]core.Object
	ObjectsByOwner  map[core.Address][]core.ObjectID
}

// WasmConfig 包含WebAssembly引擎的配置选项
type WasmConfig struct {
	// 合约文件存储目录
	ContractDir string

	// 资源限制配置
	MemoryLimit        uint64
	ExecutionTimeLimit uint64
	InstructionsLimit  uint64
}

// NewWasmEngine 创建一个新的WebAssembly引擎实例
func NewWasmEngine(config *WasmConfig) (*WasmEngine, error) {
	if config == nil {
		return nil, errors.New("配置不能为空")
	}

	// 确保合约目录存在
	if err := os.MkdirAll(config.ContractDir, 0755); err != nil {
		return nil, fmt.Errorf("创建合约目录失败: %w", err)
	}

	engine := &WasmEngine{
		contracts:          make(map[core.Address]string),
		contractDir:        config.ContractDir,
		memoryLimit:        config.MemoryLimit,
		executionTimeLimit: config.ExecutionTimeLimit,
		instructionsLimit:  config.InstructionsLimit,
		state: &HostState{
			Balances:       make(map[core.Address]uint64),
			Objects:        make(map[core.ObjectID]core.Object),
			ObjectsByOwner: make(map[core.Address][]core.ObjectID),
		},
	}

	return engine, nil
}

// DeployWasmContract 部署新的WebAssembly合约
func (e *WasmEngine) DeployWasmContract(wasmCode []byte, sender core.Address) (core.Address, error) {
	if len(wasmCode) == 0 {
		var zero core.Address
		return zero, errors.New("合约代码不能为空")
	}

	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)
	_, err := wasmer.NewModule(store, wasmCode)
	if err != nil {
		var zero core.Address
		return zero, fmt.Errorf("无效的WebAssembly模块: %w", err)
	}

	contractAddr := wasmGenerateContractAddress(wasmCode, sender)
	contractPath := filepath.Join(e.contractDir, fmt.Sprintf("%x", contractAddr)+".wasm")

	if err := os.WriteFile(contractPath, wasmCode, 0644); err != nil {
		var zero core.Address
		return zero, fmt.Errorf("存储合约代码失败: %w", err)
	}

	e.contractsLock.Lock()
	e.contracts[contractAddr] = contractPath
	e.contractsLock.Unlock()

	return contractAddr, nil
}

// LoadWasmModule 加载WebAssembly模块
func (e *WasmEngine) LoadWasmModule(contractAddr core.Address) ([]byte, error) {
	e.contractsLock.RLock()
	contractPath, exists := e.contracts[contractAddr]
	e.contractsLock.RUnlock()

	if !exists {
		return nil, fmt.Errorf("合约不存在: %x", contractAddr)
	}

	wasmCode, err := os.ReadFile(contractPath)
	if err != nil {
		return nil, fmt.Errorf("读取合约代码失败: %w", err)
	}

	return wasmCode, nil
}

// CreateWasmInstanceWithImports 创建WebAssembly实例并设置导入对象
func (e *WasmEngine) CreateWasmInstanceWithImports(ctx context.Context, wasmCode []byte) (*wasmer.Instance, error) {
	// 创建WASM引擎和存储
	config := wasmer.NewConfig()
	config.UseLLVMCompiler()
	engine := wasmer.NewEngineWithConfig(config)
	store := wasmer.NewStore(engine)

	// 编译WebAssembly模块
	module, err := wasmer.NewModule(store, wasmCode)
	if err != nil {
		return nil, fmt.Errorf("编译WebAssembly模块失败: %w", err)
	}

	// 创建导入对象
	importObject := wasmer.NewImportObject()

	// 创建内存实例
	limits, err := wasmer.NewLimits(16, 128)
	if err != nil {
		return nil, fmt.Errorf("创建内存限制失败: %w", err)
	}
	memoryType := wasmer.NewMemoryType(limits)
	memory := wasmer.NewMemory(store, memoryType)
	if memory == nil {
		return nil, fmt.Errorf("创建内存失败")
	}

	// 创建环境函数
	envFunctions := map[string]wasmer.IntoExtern{
		"memory": memory,
		"call_host_set": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				[]*wasmer.ValueType{
					wasmer.NewValueType(wasmer.I32), // funcID
					wasmer.NewValueType(wasmer.I32), // argPtr
					wasmer.NewValueType(wasmer.I32), // argLen
					wasmer.NewValueType(wasmer.I32), // bufferPtr
				},
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.I32)},
			),
			e.callHostSetHandler(ctx),
		),
		"call_host_get_buffer": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				[]*wasmer.ValueType{
					wasmer.NewValueType(wasmer.I32), // funcID
					wasmer.NewValueType(wasmer.I32), // argPtr
					wasmer.NewValueType(wasmer.I32), // argLen
					wasmer.NewValueType(wasmer.I32), // buffer
				},
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.I32)},
			),
			e.callHostGetBufferHandler(ctx),
		),
		"get_block_height": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				[]*wasmer.ValueType{},
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.I64)},
			),
			e.getBlockHeightHandler(ctx),
		),
		"get_block_time": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				[]*wasmer.ValueType{},
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.I64)},
			),
			e.getBlockTimeHandler(ctx),
		),
	}

	// 注册环境命名空间
	importObject.Register("env", envFunctions)

	// 创建实例
	instance, err := wasmer.NewInstance(module, importObject)
	if err != nil {
		return nil, fmt.Errorf("实例化WebAssembly模块失败: %w", err)
	}

	return instance, nil
}

// callHostSetHandler 处理修改区块链状态的宿主函数调用
func (e *WasmEngine) callHostSetHandler(context.Context) func([]wasmer.Value) ([]wasmer.Value, error) {
	return func(args []wasmer.Value) ([]wasmer.Value, error) {
		if len(args) != 4 {
			return []wasmer.Value{wasmer.NewI32(0)}, nil
		}

		funcID := args[0].I32()
		_ = args[1].I32() // argPtr 暂时未使用
		_ = args[2].I32() // argLen 暂时未使用
		_ = args[3].I32() // bufferPtr 暂时未使用

		// 根据函数ID执行不同的操作
		switch funcID {
		case int32(types.FuncTransfer):
			// TODO: 实现转账逻辑
			return []wasmer.Value{wasmer.NewI32(0)}, nil

		case int32(types.FuncLog):
			// TODO: 实现日志记录逻辑
			return []wasmer.Value{wasmer.NewI32(0)}, nil

		default:
			return []wasmer.Value{wasmer.NewI32(-1)}, nil
		}
	}
}

// callHostGetBufferHandler 处理获取区块链状态的宿主函数调用
func (e *WasmEngine) callHostGetBufferHandler(ctx context.Context) func([]wasmer.Value) ([]wasmer.Value, error) {
	return func(args []wasmer.Value) ([]wasmer.Value, error) {
		if len(args) != 4 {
			return []wasmer.Value{wasmer.NewI32(0)}, nil
		}

		funcID := args[0].I32()
		_ = args[1].I32() // argPtr 暂时未使用
		_ = args[2].I32() // argLen 暂时未使用
		bufferPtr := args[3].I32()

		memory, ok := ctx.Value("memory").(*wasmer.Memory)
		if !ok || memory == nil {
			return []wasmer.Value{wasmer.NewI32(0)}, fmt.Errorf("内存实例为空")
		}

		memorySize := int32(len(memory.Data()))
		if bufferPtr < 0 || bufferPtr >= memorySize || bufferPtr+types.HostBufferSize > memorySize {
			return []wasmer.Value{wasmer.NewI32(0)}, fmt.Errorf("无效的缓冲区位置")
		}

		hostBuffer := memory.Data()[bufferPtr : bufferPtr+types.HostBufferSize]

		switch funcID {
		case int32(types.FuncGetSender):
			data := e.state.CurrentSender[:]
			dataSize := copy(hostBuffer, data)
			return []wasmer.Value{wasmer.NewI32(int32(dataSize))}, nil

		case int32(types.FuncGetContractAddress):
			data := e.state.ContractAddress[:]
			dataSize := copy(hostBuffer, data)
			return []wasmer.Value{wasmer.NewI32(int32(dataSize))}, nil

		default:
			return []wasmer.Value{wasmer.NewI32(0)}, nil
		}
	}
}

// getBlockHeightHandler 获取区块高度处理函数
func (e *WasmEngine) getBlockHeightHandler(context.Context) func([]wasmer.Value) ([]wasmer.Value, error) {
	return func(args []wasmer.Value) ([]wasmer.Value, error) {
		return []wasmer.Value{wasmer.NewI64(int64(e.state.CurrentBlock))}, nil
	}
}

// getBlockTimeHandler 获取区块时间处理函数
func (e *WasmEngine) getBlockTimeHandler(context.Context) func([]wasmer.Value) ([]wasmer.Value, error) {
	return func(args []wasmer.Value) ([]wasmer.Value, error) {
		return []wasmer.Value{wasmer.NewI64(e.state.CurrentTime)}, nil
	}
}

// CallWasmFunction 调用WebAssembly函数
func (e *WasmEngine) CallWasmFunction(ctx context.Context, instance *wasmer.Instance, functionName string, params []byte) (int32, error) {
	// 检查上下文是否已取消
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("上下文已取消: %w", err)
	}

	// 获取入口函数
	handleFn, err := instance.Exports.GetFunction("handle_contract_call")
	if err != nil {
		return 0, fmt.Errorf("获取入口函数失败: %w", err)
	}

	// 获取内存实例
	memory, ok := ctx.Value("memory").(*wasmer.Memory)
	if !ok || memory == nil {
		return 0, fmt.Errorf("内存实例为空")
	}

	// 获取allocate函数
	allocateFn, err := instance.Exports.GetFunction("allocate")
	if err != nil {
		return 0, fmt.Errorf("获取allocate函数失败: %w", err)
	}

	// 写入函数名到WASM内存
	var fnNamePtr int32
	if len(functionName) > 0 {
		// 申请内存
		result, err := allocateFn(int32(len(functionName)))
		if err != nil {
			return 0, fmt.Errorf("申请函数名内存失败: %w", err)
		}
		fnNamePtr = result.(int32)

		// 写入数据
		memorySize := len(memory.Data())
		if fnNamePtr < 0 || fnNamePtr >= int32(memorySize) || fnNamePtr+int32(len(functionName)) > int32(memorySize) {
			return 0, fmt.Errorf("无效的内存位置")
		}
		copy(memory.Data()[fnNamePtr:], []byte(functionName))
	}

	// 写入参数到WASM内存
	var paramPtr int32
	if len(params) > 0 {
		// 申请内存
		result, err := allocateFn(int32(len(params)))
		if err != nil {
			return 0, fmt.Errorf("申请参数内存失败: %w", err)
		}
		paramPtr = result.(int32)

		// 写入数据
		memorySize := len(memory.Data())
		if paramPtr < 0 || paramPtr >= int32(memorySize) || paramPtr+int32(len(params)) > int32(memorySize) {
			return 0, fmt.Errorf("无效的内存位置")
		}
		copy(memory.Data()[paramPtr:], params)
	}

	// 调用入口函数
	result, err := handleFn(fnNamePtr, int32(len(functionName)), paramPtr, int32(len(params)))
	if err != nil {
		return 0, fmt.Errorf("调用入口函数失败: %w", err)
	}

	// 转换结果为int32
	if result == nil {
		return 0, nil
	}

	switch v := result.(type) {
	case int32:
		return v, nil
	case int64:
		return int32(v), nil
	default:
		return 0, fmt.Errorf("不支持的返回值类型: %T", v)
	}
}

// 生成合约地址
func wasmGenerateContractAddress(code []byte, _ core.Address) core.Address {
	codeHash := sha256.Sum256(code)

	var addr core.Address
	copy(addr[:], codeHash[:])
	return addr
}
