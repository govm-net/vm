package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// HostFunctions 定义了主机函数的常量标识符
type HostFunctions int32

// 主机函数ID
const (
	FuncGetSender          HostFunctions = 1
	FuncGetBlockHeight     HostFunctions = 2
	FuncGetBlockTime       HostFunctions = 3
	FuncGetContractAddress HostFunctions = 4
	FuncGetBalance         HostFunctions = 5
	FuncTransfer           HostFunctions = 6
	FuncCreateObject       HostFunctions = 7
	FuncCall               HostFunctions = 8
	FuncGetObject          HostFunctions = 9
	FuncGetObjectWithOwner HostFunctions = 10
	FuncDeleteObject       HostFunctions = 11
	FuncLog                HostFunctions = 12
	FuncGetObjectOwner     HostFunctions = 13
	FuncSetObjectOwner     HostFunctions = 14
	FuncGetObjectField     HostFunctions = 15
	FuncSetObjectField     HostFunctions = 16
	FuncDbRead             HostFunctions = 17
	FuncDbWrite            HostFunctions = 18
	FuncDbDelete           HostFunctions = 19
	FuncSetHostBuffer      HostFunctions = 20
)

// 主机缓冲区大小
const HostBufferSize = 1024 * 8

// 模拟环境状态
type VMState struct {
	// 主机缓冲区
	HostBuffer []byte

	// 合约地址
	ContractAddress [20]byte

	// 调用者地址
	SenderAddress [20]byte

	// 区块高度
	BlockHeight int64

	// 区块时间
	BlockTime int64

	// 存储虚拟机状态的映射
	Storage map[string][]byte
}

// 创建新的VM状态
func NewVMState() *VMState {
	return &VMState{
		HostBuffer:      make([]byte, HostBufferSize),
		ContractAddress: [20]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
		SenderAddress:   [20]byte{19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
		BlockHeight:     12345,
		BlockTime:       1647312000,
		Storage:         make(map[string][]byte),
	}
}

// 导出的合约调用结果
type ContractResult struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// 将内存中的数据读取为字节切片
func readMemory(mem api.Memory, offset, size uint32) []byte {
	if size == 0 {
		return nil
	}
	data, ok := mem.Read(offset, size)
	if !ok {
		log.Printf("Error reading memory at offset %d with size %d", offset, size)
		return nil
	}
	return data
}

// 将数据写入内存
func writeMemory(mem api.Memory, data []byte, offset uint32) error {
	if len(data) == 0 {
		return nil
	}
	ok := mem.Write(offset, data)
	if !ok {
		return fmt.Errorf("failed to write memory at offset %d with size %d", offset, len(data))
	}
	return nil
}

// 运行WebAssembly合约
func RunWasmContract(wasmPath string, functionName string, params interface{}) (*ContractResult, error) {
	// 创建VM状态
	state := NewVMState()

	// 设置上下文
	ctx := context.Background()

	// 创建新的运行时
	runtime := wazero.NewRuntime(ctx)
	defer runtime.Close(ctx)

	// 注册WASI
	wasi_snapshot_preview1.MustInstantiate(ctx, runtime)

	// 读取WASM文件
	wasmBytes, err := ioutil.ReadFile(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read wasm file: %w", err)
	}

	// 准备环境导入
	builder := runtime.NewHostModuleBuilder("env")

	// 导出主机函数
	builder.NewFunctionBuilder().
		WithFunc(func(mod api.Module, funcID, argPtr, argLen int32) int64 {
			return hostCallSet(mod, state, funcID, argPtr, argLen)
		}).
		Export("call_host_set")

	builder.NewFunctionBuilder().
		WithFunc(func(mod api.Module, funcID, argPtr, argLen int32) int32 {
			return hostCallGetBuffer(mod, state, funcID, argPtr, argLen)
		}).
		Export("call_host_get_buffer")

	builder.NewFunctionBuilder().
		WithFunc(func() int64 {
			return state.BlockHeight
		}).
		Export("get_block_height")

	builder.NewFunctionBuilder().
		WithFunc(func() int64 {
			return state.BlockTime
		}).
		Export("get_block_time")

	builder.NewFunctionBuilder().
		WithFunc(func(addr int32) uint64 {
			return 1000000 // 模拟余额
		}).
		Export("get_balance")

	builder.NewFunctionBuilder().
		WithFunc(func(ptr int32) {
			fmt.Printf("Set host buffer to pointer: %d\n", ptr)
		}).
		Export("set_host_buffer")

	// 实例化导入模块
	hostModule, err := builder.Instantiate(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate host module: %w", err)
	}
	defer hostModule.Close(ctx)

	// 创建带有导入函数的模块配置
	config := wazero.NewModuleConfig().
		WithName("wasm").
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithSysWalltime().
		WithSysNanotime()

	// 编译并实例化合约模块
	contractModule, err := runtime.InstantiateWithConfig(ctx, wasmBytes, config)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate contract module: %w", err)
	}
	defer contractModule.Close(ctx)

	// 分配内存用于函数名和参数
	allocate := contractModule.ExportedFunction("allocate")
	deallocate := contractModule.ExportedFunction("deallocate")
	handleContractCall := contractModule.ExportedFunction("handle_contract_call")

	// 序列化参数
	var paramsBytes []byte
	if params != nil {
		paramsBytes, err = json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal parameters: %w", err)
		}
	}

	// 分配内存用于函数名
	funcNameBytes := []byte(functionName)
	funcNameSize := uint64(len(funcNameBytes))
	funcNamePtrResult, err := allocate.Call(ctx, uint64(funcNameSize))
	if err != nil {
		return nil, fmt.Errorf("failed to allocate memory for function name: %w", err)
	}
	funcNamePtr := uint32(funcNamePtrResult[0])
	defer deallocate.Call(ctx, uint64(funcNamePtr), funcNameSize)

	// 写入函数名
	if err := writeMemory(contractModule.Memory(), funcNameBytes, funcNamePtr); err != nil {
		return nil, fmt.Errorf("failed to write function name: %w", err)
	}

	// 分配内存用于参数
	var paramsPtr uint32
	var paramsSize uint64
	if len(paramsBytes) > 0 {
		paramsSize = uint64(len(paramsBytes))
		paramsPtrResult, err := allocate.Call(ctx, paramsSize)
		if err != nil {
			return nil, fmt.Errorf("failed to allocate memory for parameters: %w", err)
		}
		paramsPtr = uint32(paramsPtrResult[0])
		defer deallocate.Call(ctx, uint64(paramsPtr), paramsSize)

		// 写入参数
		if err := writeMemory(contractModule.Memory(), paramsBytes, paramsPtr); err != nil {
			return nil, fmt.Errorf("failed to write parameters: %w", err)
		}
	}

	// 调用合约函数
	resultPtrResult, err := handleContractCall.Call(ctx,
		uint64(funcNamePtr), uint64(len(funcNameBytes)),
		uint64(paramsPtr), uint64(len(paramsBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to call handle_contract_call: %w", err)
	}
	resultPtr := uint32(resultPtrResult[0])

	// 读取结果数据
	resultBytes := readMemory(contractModule.Memory(), resultPtr, uint32(HostBufferSize))
	if resultBytes == nil {
		return nil, fmt.Errorf("failed to read result data")
	}

	// 解析返回的JSON数据
	var result ContractResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w\nraw data: %s", err, string(resultBytes))
	}

	return &result, nil
}

// 主机函数 - 设置类型的调用
func hostCallSet(mod api.Module, state *VMState, funcID, argPtr, argLen int32) int64 {
	log.Printf("Host call set: funcID=%d, argPtr=%d, argLen=%d", funcID, argPtr, argLen)

	argData := readMemory(mod.Memory(), uint32(argPtr), uint32(argLen))

	switch HostFunctions(funcID) {
	case FuncTransfer:
		log.Printf("Transfer called with data: %v", argData)
		// 模拟转账成功
		return 0
	case FuncDeleteObject:
		log.Printf("DeleteObject called with data: %v", argData)
		return 0
	case FuncSetObjectOwner:
		log.Printf("SetObjectOwner called with data: %v", argData)
		return 0
	case FuncSetObjectField:
		log.Printf("SetObjectField called with data: %v", argData)
		if len(argData) > 0 {
			var fieldData struct {
				ID    [32]byte        `json:"id"`
				Field string          `json:"field"`
				Value json.RawMessage `json:"value"`
			}
			if err := json.Unmarshal(argData, &fieldData); err == nil {
				key := fmt.Sprintf("%x:%s", fieldData.ID, fieldData.Field)
				state.Storage[key] = fieldData.Value
				log.Printf("Set object field: %s = %s", key, string(fieldData.Value))
			}
		}
		return 0
	case FuncLog:
		log.Printf("Log: %s", string(argData))
		return 0
	default:
		log.Printf("Unknown function ID for set: %d", funcID)
		return 1
	}
}

// 主机函数 - 获取类型的调用
func hostCallGetBuffer(mod api.Module, state *VMState, funcID, argPtr, argLen int32) int32 {
	log.Printf("Host call get buffer: funcID=%d, argPtr=%d, argLen=%d", funcID, argPtr, argLen)

	argData := readMemory(mod.Memory(), uint32(argPtr), uint32(argLen))

	switch HostFunctions(funcID) {
	case FuncGetSender:
		// 写入发送者地址到缓冲区
		copy(state.HostBuffer, state.SenderAddress[:])
		log.Printf("Get sender: %x", state.SenderAddress)
		return int32(len(state.SenderAddress))

	case FuncGetContractAddress:
		// 写入合约地址到缓冲区
		copy(state.HostBuffer, state.ContractAddress[:])
		log.Printf("Get contract address: %x", state.ContractAddress)
		return int32(len(state.ContractAddress))

	case FuncGetObject:
		log.Printf("GetObject called with data: %v", argData)
		// 模拟返回一个空对象
		objectID := [32]byte{1, 2, 3}
		copy(state.HostBuffer, objectID[:])
		return int32(len(objectID))

	case FuncGetObjectWithOwner:
		log.Printf("GetObjectWithOwner called with data: %v", argData)
		// 模拟返回一个对象ID
		objectID := [32]byte{1, 2, 3}
		copy(state.HostBuffer, objectID[:])
		return int32(len(objectID))

	case FuncCreateObject:
		// 模拟创建对象
		objectID := [32]byte{1, 2, 3}
		copy(state.HostBuffer, objectID[:])
		log.Printf("Created object: %x", objectID)
		return int32(len(objectID))

	case FuncGetObjectOwner:
		log.Printf("GetObjectOwner called with data: %v", argData)
		// 模拟返回所有者为合约地址
		copy(state.HostBuffer, state.ContractAddress[:])
		return int32(len(state.ContractAddress))

	case FuncGetObjectField:
		log.Printf("GetObjectField called with data: %v", argData)
		if len(argData) > 0 {
			var fieldData struct {
				ID    [32]byte `json:"id"`
				Field string   `json:"field"`
			}
			if err := json.Unmarshal(argData, &fieldData); err == nil {
				key := fmt.Sprintf("%x:%s", fieldData.ID, fieldData.Field)
				if value, exists := state.Storage[key]; exists {
					copy(state.HostBuffer, value)
					log.Printf("Get object field: %s = %s", key, string(value))
					return int32(len(value))
				} else {
					// 返回默认值0
					defaultValue := []byte("0")
					copy(state.HostBuffer, defaultValue)
					return int32(len(defaultValue))
				}
			}
		}
		return 0

	default:
		log.Printf("Unknown function ID for get buffer: %d", funcID)
		return 0
	}
}
