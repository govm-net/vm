// Package vm 实现了基于WebAssembly的虚拟机核心功能
package vm

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/types"
	"github.com/wasmerio/wasmer-go/wasmer"
)

// 函数ID常量定义 - 从types包导入以确保一致性
const (
	FuncGetSender          = int32(types.FuncGetSender)
	FuncGetContractAddress = int32(types.FuncGetContractAddress)
	FuncTransfer           = int32(types.FuncTransfer)
	FuncCreateObject       = int32(types.FuncCreateObject)
	FuncCall               = int32(types.FuncCall)
	FuncGetObject          = int32(types.FuncGetObject)
	FuncGetObjectWithOwner = int32(types.FuncGetObjectWithOwner)
	FuncDeleteObject       = int32(types.FuncDeleteObject)
	FuncLog                = int32(types.FuncLog)
	FuncGetObjectOwner     = int32(types.FuncGetObjectOwner)
	FuncSetObjectOwner     = int32(types.FuncSetObjectOwner)
	FuncGetObjectField     = int32(types.FuncGetObjectField)
	FuncSetObjectField     = int32(types.FuncSetObjectField)
)

// 基础类型定义

// Address 表示区块链上的地址
type Address = types.Address

// ObjectID 表示状态对象的唯一标识符
type ObjectID = types.ObjectID

// ZeroAddress 零地址
var ZeroAddress = Address{}

// ZeroObjectID 零对象ID
var ZeroObjectID = ObjectID{}

// SimpleVM 是一个简化的虚拟机实现，用于演示核心功能
type SimpleVM struct {
	// 存储已部署合约的映射表
	contracts     map[Address][]byte
	contractsLock sync.RWMutex

	// 合约存储目录
	contractDir string

	// 外部区块链上下文
	blockchainCtx types.BlockchainContext
}

// NewSimpleVM 创建一个新的简化虚拟机实例
func NewSimpleVM(contractDir string) (*SimpleVM, error) {
	// 确保合约目录存在
	if contractDir != "" {
		if err := os.MkdirAll(contractDir, 0755); err != nil {
			return nil, fmt.Errorf("创建合约目录失败: %w", err)
		}
	}

	vm := &SimpleVM{
		contracts:   make(map[Address][]byte),
		contractDir: contractDir,
	}

	// 创建默认的区块链上下文
	vm.blockchainCtx = NewDefaultBlockchainContext()

	return vm, nil
}

// DeployContract 部署新的WebAssembly合约
func (vm *SimpleVM) DeployContract(wasmCode []byte, sender core.Address) (core.Address, error) {
	// 验证WASM代码
	if len(wasmCode) == 0 {
		return core.Address{}, errors.New("合约代码不能为空")
	}

	// 编译检查WASM模块
	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)
	_, err := wasmer.NewModule(store, wasmCode)
	if err != nil {
		return core.Address{}, fmt.Errorf("无效的WebAssembly模块: %w", err)
	}

	// 生成合约地址
	contractAddr := vm.generateContractAddress(wasmCode, sender)

	var objectID core.ObjectID
	copy(objectID[:], contractAddr[:])
	// 存储合约代码
	vm.contractsLock.Lock()
	vm.contracts[contractAddr] = wasmCode
	vm.contractsLock.Unlock()
	vm.blockchainCtx.CreateObjectWithID(contractAddr, objectID)

	// 如果指定了合约目录，则保存到文件
	if vm.contractDir != "" {
		contractPath := filepath.Join(vm.contractDir, fmt.Sprintf("%x", contractAddr)+".wasm")
		if err := os.WriteFile(contractPath, wasmCode, 0644); err != nil {
			return core.Address{}, fmt.Errorf("存储合约代码失败: %w", err)
		}
	}

	return contractAddr, nil
}

// ExecuteContract 执行已部署的合约函数
func (vm *SimpleVM) ExecuteContract(ctx types.BlockchainContext, contractAddr, sender core.Address, functionName string, params []byte) (out []byte, err error) {
	// 检查合约是否存在
	vm.contractsLock.RLock()
	wasmCode, exists := vm.contracts[contractAddr]
	vm.contractsLock.RUnlock()
	defer func() {
		if r := recover(); r != nil {
			slog.Error("ExecuteContract panic", "error", "panic")
			err = fmt.Errorf("ExecuteContract panic")
		}
	}()

	if !exists {
		return nil, fmt.Errorf("合约不存在: %x", contractAddr)
	}
	if ctx == nil {
		ctx = vm.blockchainCtx
	}

	// Create Wasmer instance
	config := wasmer.NewConfig()
	engine := wasmer.NewEngineWithConfig(config)
	store := wasmer.NewStore(engine)
	defer store.Close()

	// Compile module
	module, err := wasmer.NewModule(store, wasmCode)
	if err != nil {
		slog.Error("failed to compile module", "error", err)
		return nil, fmt.Errorf("failed to compile module: %w", err)
	}
	defer module.Close()

	// Create WASI environment
	wasiEnv, err := wasmer.NewWasiStateBuilder("wasi-program").
		// Add WASI args if needed
		Argument("--verbose").
		// Capture stdout/stderr
		CaptureStdout().
		CaptureStderr().
		Finalize()
	if err != nil {
		slog.Error("failed to create WASI environment", "error", err)
		return nil, fmt.Errorf("failed to create WASI environment: %w", err)
	}

	// Create import object with WASI imports
	wasiImports, err := wasiEnv.GenerateImportObject(store, module)
	if err != nil {
		slog.Error("failed to generate WASI import object", "error", err)
		return nil, fmt.Errorf("failed to generate WASI import object: %w", err)
	}

	// Create a memory for the instance - 增加初始内存大小
	limits, err := wasmer.NewLimits(16, 128) // 增加初始页数和最大页数
	if err != nil {
		slog.Error("failed to create memory limits", "error", err)
		return nil, fmt.Errorf("failed to create memory limits: %w", err)
	}
	memoryType := wasmer.NewMemoryType(limits)
	memory := wasmer.NewMemory(store, memoryType)
	if memory == nil {
		slog.Error("failed to create memory")
		return nil, fmt.Errorf("failed to create memory")
	}

	slog.Info("Initial memory size", "size", len(memory.Data()))

	// Add host functions to import object
	wasiImports.Register("env", map[string]wasmer.IntoExtern{
		"memory": memory,
		// 使用分离的接口替换原有的统一接口
		"call_host_set": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				[]*wasmer.ValueType{
					wasmer.NewValueType(wasmer.I32), // funcID
					wasmer.NewValueType(wasmer.I32), // argPtr
					wasmer.NewValueType(wasmer.I32), // argLen
					wasmer.NewValueType(wasmer.I32), // bufferPtr
				},
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.I32)}, // 结果编码为int32
			),
			vm.callHostSetHandler(ctx, memory),
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
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.I32)}, // 返回数据大小
			),
			vm.callHostGetBufferHandler(ctx, memory),
		),
		// 单独的简单数据类型获取函数
		"get_block_height": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				[]*wasmer.ValueType{},                                // 无参数
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.I64)}, // 返回int64
			),
			func(args []wasmer.Value) ([]wasmer.Value, error) {
				return []wasmer.Value{wasmer.NewI64(int64(ctx.BlockHeight()))}, nil
			},
		),
		"get_block_time": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				[]*wasmer.ValueType{},                                // 无参数
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.I64)}, // 返回int64
			),
			func(args []wasmer.Value) ([]wasmer.Value, error) {
				return []wasmer.Value{wasmer.NewI64(int64(ctx.BlockTime()))}, nil
			},
		),
		"get_balance": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				[]*wasmer.ValueType{
					wasmer.NewValueType(wasmer.I32), // addrPtr
				},
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.F64)}, // 返回float64
			),
			vm.getBalanceHandler(ctx, memory),
		),
		// 保留其他可能需要的函数...
	})

	// Create instance with all imports
	instance, err := wasmer.NewInstance(module, wasiImports)
	if err != nil {
		slog.Error("failed to instantiate module", "error", err)
		return nil, fmt.Errorf("failed to instantiate module: %w", err)
	}
	defer instance.Close()

	m, err := instance.Exports.GetMemory("memory")
	if err != nil {
		slog.Error("failed to get memory", "error", err)
		return nil, fmt.Errorf("failed to get memory: %w", err)
	}
	*memory = *m
	result, err := vm.callWasmFunction(ctx, instance, functionName, params)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// generateContractAddress 生成合约地址
func (vm *SimpleVM) generateContractAddress(code []byte, sender core.Address) core.Address {
	var addr core.Address
	// 简化版实现，实际应使用哈希算法
	if len(code) >= 10 {
		copy(addr[:10], code[:10])
	}
	copy(addr[10:], sender[:10])
	return addr
}

// 从 WebAssembly 内存中读取字符串
func readString(memory *wasmer.Memory, ptr, len int32) string {
	data := memory.Data()[ptr : ptr+len]
	return string(data)
}

func (vm *SimpleVM) callWasmFunction(ctx types.BlockchainContext, instance *wasmer.Instance, functionName string, params []byte) (out []byte, err error) {
	slog.Info("Calling contract function", "function", functionName, "params", params)
	memory, err := instance.Exports.GetMemory("memory")
	if err != nil {
		slog.Error("failed to get memory", "error", err)
		return nil, fmt.Errorf("failed to get memory: %w", err)
	}

	allocate, allocErr := instance.Exports.GetFunction("allocate")
	if allocErr != nil {
		slog.Error("No allocate function found")
		return nil, fmt.Errorf("no allocate function found")
	}

	processDataFunc, err := instance.Exports.GetFunction("handle_contract_call")
	if err != nil {
		slog.Error("handle_contract_call not found", "error", err)
		return nil, fmt.Errorf("handle_contract_call not found: %w", err)
	}

	var input types.HandleContractCallParams
	input.Contract = ctx.ContractAddress()
	input.Function = functionName
	input.Sender = ctx.Sender()
	input.Args = params
	inputBytes, err := json.Marshal(input)
	if err != nil {
		slog.Error("failed to serialize handle_contract_call", "error", err)
		return nil, fmt.Errorf("failed to serialize handle_contract_call: %w", err)
	}

	inputAddr, err := allocate(len(inputBytes))
	if err != nil {
		slog.Error("failed to allocate memory", "error", err)
		return nil, fmt.Errorf("failed to allocate memory: %w", err)
	}

	inputPtr := inputAddr.(int32)
	copy(memory.Data()[int(inputPtr):int(inputPtr)+len(inputBytes)], []byte(inputBytes))

	result, err := processDataFunc(inputPtr, len(inputBytes))
	if err != nil {
		slog.Error("failed to execute function", "function", functionName, "error", err)
		return nil, err
	}

	resultLen, _ := result.(int32)

	if resultLen > 0 {
		getBufferAddress, getBufferErr := instance.Exports.GetFunction("get_buffer_address")
		if getBufferErr != nil {
			slog.Error("No get_buffer_address function found")
			return nil, fmt.Errorf("no get_buffer_address function found")
		}
		rst, err := getBufferAddress()
		if err != nil {
			slog.Error("failed to get buffer address", "error", err)
			return nil, fmt.Errorf("failed to get buffer address: %w", err)
		}
		bufferPtr := rst.(int32)
		data := readString(memory, bufferPtr, resultLen)
		slog.Info("Function result", "result", data)
		out = []byte(data)
	}

	slog.Info("Function execution completed", "function", functionName, "result", result)

	//free memory
	free, freeErr := instance.Exports.GetFunction("deallocate")
	if freeErr != nil {
		slog.Error("No deallocate function found")
		return nil, fmt.Errorf("no deallocate function found")
	}
	free(inputPtr, len(inputBytes))
	if resultLen < 0 {
		return nil, fmt.Errorf("执行%s 失败: %v", functionName, result)
	}

	return out, nil
}

// 合并所有宿主函数到统一的调用处理器 - 用于设置数据的函数
func (vm *SimpleVM) callHostSetHandler(ctx types.BlockchainContext, memory *wasmer.Memory) func([]wasmer.Value) ([]wasmer.Value, error) {
	return func(args []wasmer.Value) ([]wasmer.Value, error) {
		if len(args) != 4 {
			slog.Error("Invalid argument count")
			return []wasmer.Value{wasmer.NewI64(0)}, nil
		}

		funcID := args[0].I32()
		argPtr := args[1].I32()
		argLen := args[2].I32()
		slog.Debug("Calling host Set function",
			"funcID", funcID,
			"argPtr", argPtr,
			"argLen", argLen,
		)

		// 读取参数数据，添加安全检查
		var argData []byte
		if argLen > 0 {
			// 安全检查 - 确保memory不为nil
			if memory == nil {
				slog.Error("Memory instance is nil")
				return []wasmer.Value{wasmer.NewI64(0)}, fmt.Errorf("memory instance is nil")
			}

			// 获取内存大小
			memorySize := int32(len(memory.Data()))

			// 检查参数指针和长度是否有效
			if argPtr < 0 || argPtr >= memorySize || argPtr+argLen > memorySize {
				slog.Error("Invalid memory access",
					"ptr", argPtr,
					"len", argLen,
					"memorySize", memorySize,
				)
				return []wasmer.Value{wasmer.NewI64(0)}, fmt.Errorf("无效的内存访问")
			}

			// 安全地读取参数数据
			argData = make([]byte, argLen)
			copy(argData, memory.Data()[argPtr:argPtr+argLen])
			slog.Debug("Host Set function parameters",
				"funcID", funcID,
				"argLen", argLen,
				"data", string(argData),
			)
		}

		slog.Debug("调用宿主Set函数", "ID", funcID, "length", argLen)

		// 根据函数ID执行不同的操作 - 主要处理写入/修改类操作
		switch funcID {
		case FuncTransfer: // 转账
			var params types.TransferParams
			if err := json.Unmarshal(argData, &params); err != nil {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("解析参数失败: %v", err)
			}
			err := ctx.Transfer(params.From, params.To, params.Amount)
			if err != nil {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("转账失败: %v", err)
			}
			return []wasmer.Value{wasmer.NewI32(0)}, nil

		case FuncDeleteObject: // 删除对象
			var params types.DeleteObjectParams
			if err := json.Unmarshal(argData, &params); err != nil {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("解析参数失败: %v", err)
			}
			if params.ID == (types.ObjectID{}) {
				copy(params.ID[:], params.Contract[:])
			}
			// todo:验证权限
			obj, err := ctx.GetObject(params.Contract, params.ID)
			if err != nil {
				slog.Error("deleteObject获取对象失败", "ID", params.ID)
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("获取对象失败: %v", err)
			}
			if obj.Contract() != params.Contract {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("权限不足")
			}
			ctx.DeleteObject(params.Contract, params.ID)
			return []wasmer.Value{wasmer.NewI32(0)}, nil
		case FuncLog: // 记录日志
			slog.Info("WASM log", "data", string(argData))
			return []wasmer.Value{wasmer.NewI32(0)}, nil

		case FuncSetObjectOwner: // 设置对象所有者
			var params types.SetOwnerParams
			if err := json.Unmarshal(argData, &params); err != nil {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("解析参数失败: %v", err)
			}
			if params.ID == (types.ObjectID{}) {
				copy(params.ID[:], params.Contract[:])
			}
			obj, err := ctx.GetObject(params.Contract, params.ID)
			if err != nil {
				slog.Error("setOwner获取对象失败", "ID", params.ID)
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("获取对象失败: %v", err)
			}
			if obj.Contract() != params.Contract {
				slog.Error("权限不足, 合约不匹配", "contract", obj.Contract(), "expected", params.Contract)
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("权限不足, 合约不匹配")
			}
			if obj.Owner() != ctx.Sender() && obj.Owner() != params.Contract && obj.Owner() != params.Sender {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("权限不足, 所有者不匹配")
			}
			obj.SetOwner(params.Owner)
			return []wasmer.Value{wasmer.NewI32(0)}, nil

		case FuncSetObjectField: // 设置对象字段
			var params types.SetObjectFieldParams
			if err := json.Unmarshal(argData, &params); err != nil {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("解析参数失败: %v", err)
			}
			if params.ID == (types.ObjectID{}) {
				copy(params.ID[:], params.Contract[:])
			}
			obj, err := ctx.GetObject(params.Contract, params.ID)
			if err != nil {
				slog.Error("setObjectField获取对象失败", "ID", params.ID)
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("获取对象失败: %v", err)
			}
			// todo:验证权限
			if obj.Contract() != params.Contract {
				slog.Error("权限不足, 合约不匹配", "contract", obj.Contract(), "expected", params.Contract)
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("权限不足, 合约不匹配")
			}
			if obj.Owner() != ctx.Sender() && obj.Owner() != params.Contract && obj.Owner() != params.Sender {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("权限不足, 所有者不匹配")
			}
			value, err := json.Marshal(params.Value)
			if err != nil {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("序列化失败: %v", err)
			}
			obj.Set(params.Field, value)
			return []wasmer.Value{wasmer.NewI32(0)}, nil

		default:
			slog.Error("Unknown Set function ID", "funcID", funcID)
			return []wasmer.Value{wasmer.NewI32(-1)}, nil
		}
	}
}

// 合并所有宿主函数到统一的调用处理器 - 用于获取缓冲区数据的函数
func (vm *SimpleVM) callHostGetBufferHandler(ctx types.BlockchainContext, memory *wasmer.Memory) func([]wasmer.Value) ([]wasmer.Value, error) {
	return func(args []wasmer.Value) ([]wasmer.Value, error) {
		if len(args) != 4 {
			slog.Error("Invalid argument count")
			return []wasmer.Value{wasmer.NewI32(0)}, nil
		}

		funcID := args[0].I32()
		argPtr := args[1].I32()
		argLen := args[2].I32()
		bufferPtr := args[3].I32()
		slog.Debug("Calling host GetBuffer function",
			"funcID", funcID,
			"argPtr", argPtr,
			"argLen", argLen,
			"bufferPtr", bufferPtr,
		)

		// 安全检查 - 确保memory不为nil
		if memory == nil {
			slog.Error("Memory instance is nil")
			return []wasmer.Value{wasmer.NewI32(0)}, fmt.Errorf("memory instance is nil")
		}

		// 获取内存大小
		memorySize := int32(len(memory.Data()))

		// 读取参数数据，添加安全检查
		var argData []byte
		if argLen > 0 {
			// 检查参数指针和长度是否有效
			if argPtr < 0 || argPtr >= memorySize || argPtr+argLen > memorySize {
				slog.Error("Invalid memory access",
					"ptr", argPtr,
					"len", argLen,
					"memorySize", memorySize,
				)
				return []wasmer.Value{wasmer.NewI32(0)}, fmt.Errorf("无效的内存访问")
			}

			// 安全地读取参数数据
			argData = make([]byte, argLen)
			copy(argData, memory.Data()[argPtr:argPtr+argLen])
		}

		// 检查主机缓冲区是否在有效范围内
		if bufferPtr < 0 || bufferPtr >= memorySize || bufferPtr+types.HostBufferSize > memorySize {
			slog.Error("Invalid buffer position",
				"ptr", bufferPtr,
				"size", types.HostBufferSize,
				"memorySize", memorySize,
			)
			return []wasmer.Value{wasmer.NewI32(0)}, fmt.Errorf("无效的缓冲区位置")
		}

		// 获取全局缓冲区
		hostBuffer := memory.Data()[bufferPtr : bufferPtr+types.HostBufferSize]

		// 根据函数ID执行不同的操作 - 处理需要返回缓冲区数据的操作
		switch funcID {
		case FuncGetSender: // 获取当前发送者
			data := ctx.Sender()
			dataSize := copy(hostBuffer, data[:]) // 写入全局缓冲区
			return []wasmer.Value{wasmer.NewI32(int32(dataSize))}, nil

		case FuncGetContractAddress: // 获取合约地址
			data := ctx.ContractAddress()
			dataSize := copy(hostBuffer, data[:]) // 写入全局缓冲区
			return []wasmer.Value{wasmer.NewI32(int32(dataSize))}, nil

		case FuncCreateObject: // 创建对象
			var addr Address
			copy(addr[:], argData)
			obj, err := ctx.CreateObject(addr)
			if err != nil {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("创建对象失败: %v", err)
			}
			id := obj.ID()

			// 写入对象ID到全局缓冲区
			dataSize := copy(hostBuffer, id[:])
			return []wasmer.Value{wasmer.NewI32(int32(dataSize))}, nil

		case FuncGetObjectField: // 获取对象字段
			// 实现获取对象字段的逻辑...
			var params types.GetObjectFieldParams
			if err := json.Unmarshal(argData, &params); err != nil {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("解析参数失败: %v", err)
			}
			if params.ID == (types.ObjectID{}) {
				copy(params.ID[:], params.Contract[:])
			}
			obj, err := ctx.GetObject(params.Contract, params.ID)
			if err != nil {
				slog.Error("getObjectField获取对象失败", "ID", params.ID)
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("获取对象失败: %v", err)
			}
			data, err := obj.Get(params.Field)
			if err != nil {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("获取字段失败: %v", err)
			}
			dataSize := copy(hostBuffer, data)
			return []wasmer.Value{wasmer.NewI32(int32(dataSize))}, nil
		case FuncCall: // 调用合约
			// 实现合约调用逻辑并将结果写入全局缓冲区
			result := []byte("模拟合约调用结果")
			dataSize := copy(hostBuffer, result)
			return []wasmer.Value{wasmer.NewI32(int32(dataSize))}, nil

		case FuncGetObject: // 获取对象
			var params types.GetObjectParams
			if err := json.Unmarshal(argData, &params); err != nil {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("解析参数失败: %v", err)
			}
			slog.Info("获取对象", "ID", params.ID, "contract", params.Contract)
			if params.ID == (types.ObjectID{}) {
				copy(params.ID[:], params.Contract[:])
			}
			obj, err := ctx.GetObject(params.Contract, params.ID)
			if err != nil {
				slog.Error("getObject获取对象失败", "ID", params.ID)
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("获取对象失败: %v", err)
			}
			if obj.Contract() != params.Contract {
				slog.Error("权限不足, 合约不匹配", "contract", obj.Contract(), "expected", params.Contract)
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("权限不足, 合约不匹配")
			}
			id := obj.ID()
			dataSize := copy(hostBuffer, id[:])
			return []wasmer.Value{wasmer.NewI32(int32(dataSize))}, nil

		case FuncGetObjectWithOwner: // 根据所有者获取对象
			var params types.GetObjectWithOwnerParams
			if err := json.Unmarshal(argData, &params); err != nil {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("解析参数失败: %v", err)
			}
			obj, err := ctx.GetObjectWithOwner(params.Contract, params.Owner)
			if err != nil {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("获取对象失败: %v", err)
			}
			id := obj.ID()
			dataSize := copy(hostBuffer, id[:])
			return []wasmer.Value{wasmer.NewI32(dataSize)}, nil

		case FuncGetObjectOwner: // 获取对象所有者
			if argLen != 32 {
				return []wasmer.Value{wasmer.NewI32(0)}, nil
			}
			var objID ObjectID
			copy(objID[:], argData)
			if objID == (types.ObjectID{}) {
				copy(objID[:], argData)
			}
			obj, err := ctx.GetObject(Address{}, objID)
			if err != nil {
				slog.Error("getObjectOwner获取对象失败", "ID", objID)
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("获取对象失败: %v", err)
			}

			// 获取对象所有者并写入全局缓冲区
			owner := obj.Owner()
			dataSize := copy(hostBuffer, owner[:])
			return []wasmer.Value{wasmer.NewI32(dataSize)}, nil

		default:
			slog.Error("Unknown GetBuffer function ID", "funcID", funcID)
			return []wasmer.Value{wasmer.NewI32(0)}, nil
		}
	}
}

// 获取余额处理函数
func (vm *SimpleVM) getBalanceHandler(ctx types.BlockchainContext, memory *wasmer.Memory) func([]wasmer.Value) ([]wasmer.Value, error) {
	return func(args []wasmer.Value) ([]wasmer.Value, error) {
		if len(args) != 1 {
			slog.Error("Invalid argument count")
			return []wasmer.Value{wasmer.NewI64(0)}, nil
		}

		addrPtr := args[0].I32()

		// 安全检查 - 确保memory不为nil
		if memory == nil {
			slog.Error("Memory instance is nil")
			return []wasmer.Value{wasmer.NewI64(0)}, fmt.Errorf("memory instance is nil")
		}

		// 获取内存大小
		memorySize := int32(len(memory.Data()))

		// 检查指针是否有效
		if addrPtr < 0 || addrPtr+20 > memorySize {
			slog.Error("Invalid address pointer", "ptr", addrPtr)
			return []wasmer.Value{wasmer.NewI64(0)}, fmt.Errorf("无效的地址指针")
		}

		// 读取地址
		addrData := make([]byte, 20)
		copy(addrData, memory.Data()[addrPtr:addrPtr+20])

		var addr Address
		copy(addr[:], addrData)

		// 获取余额
		balance := ctx.Balance(addr)
		return []wasmer.Value{wasmer.NewI64(int64(balance))}, nil
	}
}
