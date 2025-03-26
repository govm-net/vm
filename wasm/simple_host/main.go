package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"

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

// 全局缓冲区大小
const HostBufferSize int32 = types.HostBufferSize

// Address represents a blockchain address
type Address = core.Address

// ObjectID represents a unique identifier for a state object
type ObjectID = core.ObjectID

// Object represents a state object
type Object struct {
	ID     ObjectID          `json:"id"`
	Type   string            `json:"type"`
	Owner  Address           `json:"owner"`
	Fields map[string][]byte `json:"fields"`
}

// Host state
type HostState struct {
	CurrentSender   Address
	CurrentBlock    uint64
	CurrentTime     int64
	ContractAddress Address
	Balances        map[Address]uint64
	Objects         map[ObjectID]Object
	ObjectsByOwner  map[Address][]ObjectID
}

var state = &HostState{
	Balances:       make(map[Address]uint64),
	Objects:        make(map[ObjectID]Object),
	ObjectsByOwner: make(map[Address][]ObjectID),
}

// 合并所有宿主函数到统一的调用处理器 - 用于设置数据的函数
func callHostSetHandler(memory *wasmer.Memory) func([]wasmer.Value) ([]wasmer.Value, error) {
	return func(args []wasmer.Value) ([]wasmer.Value, error) {
		if len(args) != 4 {
			fmt.Println("参数数量不正确")
			return []wasmer.Value{wasmer.NewI64(0)}, nil
		}

		funcID := args[0].I32()
		argPtr := args[1].I32()
		argLen := args[2].I32()
		fmt.Printf("调用宿主Set函数 ID=%d, 参数指针=%d, 参数长度=%d\n", funcID, argPtr, argLen)

		// 读取参数数据，添加安全检查
		var argData []byte
		if argLen > 0 {
			// 安全检查 - 确保memory不为nil
			if memory == nil {
				fmt.Println("内存实例为空")
				return []wasmer.Value{wasmer.NewI64(0)}, fmt.Errorf("内存实例为空")
			}

			// 获取内存大小
			memorySize := int32(len(memory.Data()))

			// 检查参数指针和长度是否有效
			if argPtr < 0 || argPtr >= memorySize || argPtr+argLen > memorySize {
				fmt.Printf("无效的内存访问: 指针=%d, 长度=%d, 内存大小=%d\n", argPtr, argLen, memorySize)
				return []wasmer.Value{wasmer.NewI64(0)}, fmt.Errorf("无效的内存访问")
			}

			// 安全地读取参数数据
			argData = make([]byte, argLen)
			copy(argData, memory.Data()[argPtr:argPtr+argLen])
			fmt.Printf("[HOST] Set函数 ID=%d, 参数长度=%d, 参数数据:%s\n", funcID, argLen, string(argData))
		}

		fmt.Printf("调用宿主Set函数 ID=%d, 参数长度=%d\n", funcID, argLen)

		// 根据函数ID执行不同的操作 - 主要处理写入/修改类操作
		switch funcID {
		case FuncTransfer: // 转账
			if argLen < 28 { // 20字节地址 + 8字节金额
				return []wasmer.Value{wasmer.NewI64(0)}, nil
			}
			var to Address
			copy(to[:], argData[:20])
			amount := binary.LittleEndian.Uint64(argData[20:28])

			// 检查余额
			if state.Balances[state.ContractAddress] < amount {
				return []wasmer.Value{wasmer.NewI64(0)}, nil
			}

			// 执行转账
			state.Balances[state.ContractAddress] -= amount
			state.Balances[to] += amount
			return []wasmer.Value{wasmer.NewI32(0)}, nil

		case FuncDeleteObject: // 删除对象
			// 实现删除对象的逻辑...
			return []wasmer.Value{wasmer.NewI32(0)}, nil

		case FuncLog: // 记录日志
			// 实现日志记录的逻辑...
			fmt.Printf("[WASM]日志: len:%d %s\n", len(argData), string(argData))
			return []wasmer.Value{wasmer.NewI32(0)}, nil

		case FuncSetObjectOwner: // 设置对象所有者
			// 实现设置对象所有者的逻辑...
			return []wasmer.Value{wasmer.NewI32(0)}, nil

		case FuncSetObjectField: // 设置对象字段
			// 实现设置对象字段的逻辑...
			fmt.Printf("[HOST] SetObjectField: %s\n", string(argData))
			return []wasmer.Value{wasmer.NewI32(0)}, nil

		default:
			fmt.Printf("未知的Set函数ID: %d\n", funcID)
			return []wasmer.Value{wasmer.NewI32(-1)}, nil
		}
	}
}

// 合并所有宿主函数到统一的调用处理器 - 用于获取缓冲区数据的函数
func callHostGetBufferHandler(memory *wasmer.Memory) func([]wasmer.Value) ([]wasmer.Value, error) {
	return func(args []wasmer.Value) ([]wasmer.Value, error) {
		if len(args) != 4 {
			fmt.Println("参数数量不正确")
			return []wasmer.Value{wasmer.NewI32(0)}, nil
		}

		funcID := args[0].I32()
		argPtr := args[1].I32()
		argLen := args[2].I32()
		bufferPtr := args[3].I32()
		fmt.Printf("调用宿主GetBuffer函数 ID=%d, 参数指针=%d, 参数长度=%d, 缓冲区指针=%d\n", funcID, argPtr, argLen, bufferPtr)

		// 安全检查 - 确保memory不为nil
		if memory == nil {
			fmt.Println("内存实例为空")
			return []wasmer.Value{wasmer.NewI32(0)}, fmt.Errorf("内存实例为空")
		}

		// 获取内存大小
		memorySize := int32(len(memory.Data()))

		// 读取参数数据，添加安全检查
		var argData []byte
		if argLen > 0 {
			// 检查参数指针和长度是否有效
			if argPtr < 0 || argPtr >= memorySize || argPtr+argLen > memorySize {
				fmt.Printf("无效的内存访问: 指针=%d, 长度=%d, 内存大小=%d\n", argPtr, argLen, memorySize)
				return []wasmer.Value{wasmer.NewI32(0)}, fmt.Errorf("无效的内存访问")
			}

			// 安全地读取参数数据
			argData = make([]byte, argLen)
			copy(argData, memory.Data()[argPtr:argPtr+argLen])
		}

		// 检查主机缓冲区是否在有效范围内
		if bufferPtr < 0 || bufferPtr >= memorySize || bufferPtr+HostBufferSize > memorySize {
			fmt.Printf("无效的缓冲区位置: 指针=%d, 大小=%d, 内存大小=%d\n", bufferPtr, HostBufferSize, memorySize)
			return []wasmer.Value{wasmer.NewI32(0)}, fmt.Errorf("无效的缓冲区位置")
		}

		// 获取全局缓冲区
		hostBuffer := memory.Data()[bufferPtr : bufferPtr+HostBufferSize]

		// 根据函数ID执行不同的操作 - 处理需要返回缓冲区数据的操作
		switch funcID {
		case FuncGetSender: // 获取当前发送者
			data := state.CurrentSender[:]
			dataSize := copy(hostBuffer, data) // 写入全局缓冲区
			return []wasmer.Value{wasmer.NewI32(int32(dataSize))}, nil

		case FuncGetContractAddress: // 获取合约地址
			data := state.ContractAddress[:]
			dataSize := copy(hostBuffer, data) // 写入全局缓冲区
			return []wasmer.Value{wasmer.NewI32(int32(dataSize))}, nil

		case FuncCreateObject: // 创建对象
			// 创建对象ID
			newID := ObjectID{} // 真实实现中应生成唯一ID

			// 创建新对象
			obj := Object{
				ID:     newID,
				Type:   "default",
				Owner:  state.CurrentSender,
				Fields: make(map[string][]byte),
			}

			// 存储对象
			state.Objects[newID] = obj
			state.ObjectsByOwner[state.CurrentSender] = append(state.ObjectsByOwner[state.CurrentSender], newID)

			// 写入对象ID到全局缓冲区
			dataSize := copy(hostBuffer, newID[:])
			return []wasmer.Value{wasmer.NewI32(int32(dataSize))}, nil

		case FuncGetObjectField: // 获取对象字段
			// 实现获取对象字段的逻辑...
			return []wasmer.Value{wasmer.NewI32(1)}, nil

		case FuncCall: // 调用合约
			// 实现合约调用逻辑并将结果写入全局缓冲区
			result := []byte("模拟合约调用结果")
			dataSize := copy(hostBuffer, result)
			return []wasmer.Value{wasmer.NewI32(int32(dataSize))}, nil

		case FuncGetObject: // 获取对象
			var objID ObjectID
			copy(objID[:32], argData)

			// 获取对象数据并写入全局缓冲区
			// 这里简化处理，实际应该序列化对象
			dataSize := copy(hostBuffer, objID[:])
			return []wasmer.Value{wasmer.NewI32(int32(dataSize))}, nil

		case FuncGetObjectWithOwner: // 根据所有者获取对象
			if argLen != 20 {
				return []wasmer.Value{wasmer.NewI32(0)}, nil
			}
			var owner Address
			copy(owner[:], argData)

			// 获取对象数据并写入全局缓冲区
			// 简化处理
			mockObjectID := ObjectID{} // 实际应该查找真实对象
			dataSize := copy(hostBuffer, mockObjectID[:])
			return []wasmer.Value{wasmer.NewI32(int32(dataSize))}, nil

		case FuncGetObjectOwner: // 获取对象所有者
			if argLen != 32 {
				return []wasmer.Value{wasmer.NewI32(0)}, nil
			}
			var objID ObjectID
			copy(objID[:], argData)

			// 获取对象所有者并写入全局缓冲区
			owner := state.Objects[objID].Owner
			dataSize := copy(hostBuffer, owner[:])
			return []wasmer.Value{wasmer.NewI32(int32(dataSize))}, nil

		default:
			fmt.Printf("未知的GetBuffer函数ID: %d\n", funcID)
			return []wasmer.Value{wasmer.NewI32(0)}, nil
		}
	}
}

// 获取区块高度处理函数
func getBlockHeightHandler(*wasmer.Memory) func([]wasmer.Value) ([]wasmer.Value, error) {
	return func(args []wasmer.Value) ([]wasmer.Value, error) {
		return []wasmer.Value{wasmer.NewI64(int64(state.CurrentBlock))}, nil
	}
}

// 获取区块时间处理函数
func getBlockTimeHandler(*wasmer.Memory) func([]wasmer.Value) ([]wasmer.Value, error) {
	return func(args []wasmer.Value) ([]wasmer.Value, error) {
		return []wasmer.Value{wasmer.NewI64(state.CurrentTime)}, nil
	}
}

// 获取余额处理函数
func getBalanceHandler(memory *wasmer.Memory) func([]wasmer.Value) ([]wasmer.Value, error) {
	return func(args []wasmer.Value) ([]wasmer.Value, error) {
		if len(args) != 1 {
			fmt.Println("参数数量不正确")
			return []wasmer.Value{wasmer.NewI64(0)}, nil
		}

		addrPtr := args[0].I32()

		// 安全检查 - 确保memory不为nil
		if memory == nil {
			fmt.Println("内存实例为空")
			return []wasmer.Value{wasmer.NewI64(0)}, fmt.Errorf("内存实例为空")
		}

		// 获取内存大小
		memorySize := int32(len(memory.Data()))

		// 检查指针是否有效
		if addrPtr < 0 || addrPtr+20 > memorySize {
			fmt.Printf("无效的地址指针: %d\n", addrPtr)
			return []wasmer.Value{wasmer.NewI64(0)}, fmt.Errorf("无效的地址指针")
		}

		// 读取地址
		addrData := make([]byte, 20)
		copy(addrData, memory.Data()[addrPtr:addrPtr+20])

		var addr Address
		copy(addr[:], addrData)

		// 获取余额
		balance := state.Balances[addr]
		return []wasmer.Value{wasmer.NewI64(int64(balance))}, nil
	}
}

func main() {
	// Read WebAssembly module
	wasmPath := "../contract.wasm"
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		log.Fatalf("Failed to read the wasm module: %v", err)
	}

	// Create Wasmer instance
	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)

	// Compile module
	module, err := wasmer.NewModule(store, wasmBytes)
	if err != nil {
		log.Fatalf("Failed to compile module: %v", err)
	}

	// Create WASI environment
	wasiEnv, err := wasmer.NewWasiStateBuilder("wasi-program").
		// Add WASI args if needed
		Argument("--verbose").
		// Map directories if needed
		MapDirectory(".", ".").
		// Capture stdout/stderr
		CaptureStdout().
		CaptureStderr().
		Finalize()
	if err != nil {
		log.Fatalf("Failed to create WASI environment: %v", err)
	}

	// Create import object with WASI imports
	wasiImports, err := wasiEnv.GenerateImportObject(store, module)
	if err != nil {
		log.Fatalf("Failed to generate WASI import object: %v", err)
	}

	// Create a memory for the instance - 增加初始内存大小
	limits, err := wasmer.NewLimits(16, 128) // 增加初始页数和最大页数
	if err != nil {
		log.Fatalf("Failed to create memory limits: %v", err)
	}
	memoryType := wasmer.NewMemoryType(limits)
	memory := wasmer.NewMemory(store, memoryType)
	if memory == nil {
		log.Fatalf("Failed to create memory")
	}

	fmt.Printf("初始内存大小: %d 字节\n", len(memory.Data()))

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
			callHostSetHandler(memory),
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
			callHostGetBufferHandler(memory),
		),
		// 单独的简单数据类型获取函数
		"get_block_height": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				[]*wasmer.ValueType{},                                // 无参数
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.I64)}, // 返回int64
			),
			getBlockHeightHandler(memory),
		),
		"get_block_time": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				[]*wasmer.ValueType{},                                // 无参数
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.I64)}, // 返回int64
			),
			getBlockTimeHandler(memory),
		),
		"get_balance": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				[]*wasmer.ValueType{
					wasmer.NewValueType(wasmer.I32), // addrPtr
				},
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.F64)}, // 返回float64
			),
			getBalanceHandler(memory),
		),
		// 保留其他可能需要的函数...
	})

	// Create instance with all imports
	instance, err := wasmer.NewInstance(module, wasiImports)
	if err != nil {
		log.Fatalf("Failed to instantiate module: %v", err)
	}

	m, err := instance.Exports.GetMemory("memory")
	if err != nil {
		log.Fatalf("无法获取内存: %v", err)
	}
	*memory = *m

	// Set some test state
	state.CurrentBlock = 100
	state.CurrentTime = 1625097600 // 2021-07-01
	state.ContractAddress = Address{1, 2, 3, 4, 5}
	state.CurrentSender = Address{9, 8, 7, 6, 5}
	state.Balances[state.ContractAddress] = 1000
	state.Balances[state.CurrentSender] = 500

	// Get hello function
	helloFn, err := instance.Exports.GetFunction("hello")
	if err != nil {
		log.Fatalf("Failed to get hello function: %v", err)
	}

	// Call hello function
	resultRaw, err := helloFn()
	if err != nil {
		log.Fatalf("Failed to call hello: %v", err)
	}
	fmt.Println(resultRaw)

	fmt.Println("Contract initialized successfully!")

	callContractFunction(instance, "Initialize", nil)
	callContractFunction(instance, "Increment", []byte(`{"amount": 2}`))
}

// 从 WebAssembly 内存中读取字符串
func readString(memory *wasmer.Memory, ptr, len int32) string {
	data := memory.Data()[ptr : ptr+len]
	return string(data)
}

func callContractFunction(instance *wasmer.Instance, fn string, params []byte) (int32, error) {
	fmt.Printf("调用合约函数:%s, %v\n", fn, params)
	memory, err := instance.Exports.GetMemory("memory")
	if err != nil {
		log.Fatalf("无法获取内存: %v", err)
	}
	// 检查是否导出了allocate和deallocate函数
	allocate, allocErr := instance.Exports.GetFunction("allocate")
	// deallocate, dealErr := instance.Exports.GetFunction("deallocate")
	if allocErr != nil {
		fmt.Println("没有allocate函数")
		return 0, fmt.Errorf("没有allocate函数")
	}
	processDataFunc, err := instance.Exports.GetFunction("handle_contract_call")
	if err != nil {
		log.Fatalf("handle_contract_call没找到:%v", err)
	}

	fnAddr, err := allocate(len(fn))
	if err != nil {
		log.Fatalf("fn 内存分配失败: %v", err)
	}
	fnPtr := fnAddr.(int32)
	copy(memory.Data()[int(fnPtr):int(fnPtr)+len(fn)], []byte(fn))

	var paramAddr int32
	if len(params) > 0 {
		paramAddr, err := allocate(len(params))
		if err != nil {
			log.Fatalf("param 内存分配失败: %v", err)
		}
		paramPtr := paramAddr.(int32)
		copy(memory.Data()[int(paramPtr):int(paramPtr)+len(params)], params)
	}

	result, err := processDataFunc(fnPtr, len(fn), paramAddr, len(params))
	if err != nil {
		log.Fatalf("执行%s 失败: %v", fn, err)
	}
	resultLen, _ := result.(int32)
	if resultLen > 0 {
		getBufferAddress, getBufferErr := instance.Exports.GetFunction("get_buffer_address")
		if getBufferErr != nil {
			fmt.Println("没有get_buffer_address函数")
			return 0, fmt.Errorf("没有get_buffer_address函数")
		}
		rst, err := getBufferAddress()
		if err != nil {
			log.Fatalf("get_buffer_address 失败: %v", err)
		}
		bufferPtr := rst.(int32)
		data := readString(memory, bufferPtr, resultLen)
		fmt.Printf("result: %s\n", data)
	}
	fmt.Printf("执行结束:%s, %v\n", fn, result)
	//free memory
	free, freeErr := instance.Exports.GetFunction("deallocate")
	if freeErr != nil {
		fmt.Println("没有deallocate函数")
		return 0, fmt.Errorf("没有deallocate函数")
	}
	free(fnPtr, len(fn))
	if freeErr != nil {
		fmt.Println("没有free函数")
		return 0, fmt.Errorf("没有free函数")
	}

	return resultLen, nil
}
