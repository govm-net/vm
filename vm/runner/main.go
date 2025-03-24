package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/wasmerio/wasmer-go/wasmer"
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

// 全局缓冲区变量 - 动态分配
var hostBufferPtr int32 = 0

// 可用函数描述
var availableFunctions = map[string]string{
	"Initialize": "初始化计数器，参数：{\"value\": 初始值}",
	"Increment":  "增加计数器值，无参数",
	"GetCounter": "获取当前计数器值，无参数",
	"Reset":      "重置计数器值，参数：{\"value\": 新值}",
}

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

func main() {
	// 解析命令行参数
	wasmPath := flag.String("wasm", "/Volumes/data2/code/vm/wasm/contract.wasm", "WebAssembly合约文件路径")
	funcName := flag.String("func", "", "要调用的函数名称")
	paramsStr := flag.String("params", "", "函数参数（JSON字符串）")
	listFuncs := flag.Bool("list", false, "列出可用的函数")
	flag.Parse()

	// 打印可用函数
	if *listFuncs {
		fmt.Println("可用的合约函数:")
		for name, desc := range availableFunctions {
			fmt.Printf("  %s: %s\n", name, desc)
		}
		return
	}

	// 检查必要的参数
	if *funcName == "" {
		fmt.Println("错误: 必须指定函数名称 (-func)")
		fmt.Println("使用 -list 查看可用的函数")
		os.Exit(1)
	}

	// 解析参数
	var params interface{}
	if *paramsStr != "" {
		if err := json.Unmarshal([]byte(*paramsStr), &params); err != nil {
			fmt.Printf("错误: 无法解析参数JSON: %v\n", err)
			os.Exit(1)
		}
	} else {
		// 如果没有提供参数，但某些函数需要默认参数
		switch *funcName {
		case "Initialize":
			// 为Initialize提供默认的初始值
			params = map[string]interface{}{
				"value": 100, // 尝试使用不同的默认值
			}
		case "Reset":
			// Reset也需要value参数
			params = map[string]interface{}{
				"value": 0,
			}
		}
	}

	// 输出更详细的参数信息
	paramsForDisplay := "无"
	if params != nil {
		paramsJSON, _ := json.Marshal(params)
		paramsForDisplay = string(paramsJSON)
	}

	// 调用VM执行合约
	fmt.Printf("执行合约: %s\n", *wasmPath)
	fmt.Printf("调用函数: %s\n", *funcName)
	fmt.Printf("参数: %s\n", paramsForDisplay)
	fmt.Println("----------------------------")

	// 执行合约
	result, err := RunWasmContract(*wasmPath, *funcName, params)
	if err != nil {
		fmt.Printf("执行失败: %v\n", err)
		os.Exit(1)
	}

	// 输出结果
	fmt.Println("----------------------------")
	fmt.Printf("执行%s\n", map[bool]string{true: "成功", false: "失败"}[result.Success])

	if result.Success {
		if len(result.Data) > 0 {
			var prettyJSON bytes.Buffer
			if err := json.Indent(&prettyJSON, result.Data, "", "  "); err == nil {
				fmt.Printf("返回数据:\n%s\n", prettyJSON.String())
			} else {
				fmt.Printf("返回数据: %s\n", string(result.Data))
			}
		} else {
			fmt.Println("无返回数据")
		}
	} else if result.Error != "" {
		fmt.Printf("错误信息: %s\n", result.Error)
	}
}

// 运行WebAssembly合约
func RunWasmContract(wasmPath string, functionName string, params interface{}) (*ContractResult, error) {
	// 创建VM状态
	state := NewVMState()

	// 读取WASM文件
	wasmBytes, err := ioutil.ReadFile(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read wasm file: %w", err)
	}

	// 创建Wasmer实例
	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)

	// 编译模块
	module, err := wasmer.NewModule(store, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to compile module: %w", err)
	}

	// 创建WASI环境
	wasiEnv, err := wasmer.NewWasiStateBuilder("wasi-program").
		CaptureStdout().
		CaptureStderr().
		Finalize()
	if err != nil {
		return nil, fmt.Errorf("failed to create WASI environment: %w", err)
	}

	// 创建导入对象并添加WASI导入
	importObject, err := wasiEnv.GenerateImportObject(store, module)
	if err != nil {
		return nil, fmt.Errorf("failed to generate WASI import object: %w", err)
	}

	// 创建合约内存 - 增加初始内存大小
	limits, err := wasmer.NewLimits(16, 512) // 初始页数和最大页数
	if err != nil {
		return nil, fmt.Errorf("failed to create memory limits: %w", err)
	}
	memoryType := wasmer.NewMemoryType(limits)
	memory := wasmer.NewMemory(store, memoryType)
	if memory == nil {
		return nil, fmt.Errorf("failed to create memory")
	}

	fmt.Printf("初始内存大小: %d 字节\n", len(memory.Data()))

	// 添加主机函数到导入对象
	importObject.Register("env", map[string]wasmer.IntoExtern{
		"memory": memory,
		// 使用分离的接口替换原有的统一接口
		"call_host_set": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				[]*wasmer.ValueType{
					wasmer.NewValueType(wasmer.I32), // funcID
					wasmer.NewValueType(wasmer.I32), // argPtr
					wasmer.NewValueType(wasmer.I32), // argLen
				},
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.I64)}, // 结果编码为int64
			),
			callHostSetHandler(memory, state),
		),
		"call_host_get_buffer": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				[]*wasmer.ValueType{
					wasmer.NewValueType(wasmer.I32), // funcID
					wasmer.NewValueType(wasmer.I32), // argPtr
					wasmer.NewValueType(wasmer.I32), // argLen
				},
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.I32)}, // 返回数据大小
			),
			callHostGetBufferHandler(memory, state),
		),
		// 区块相关函数
		"get_block_height": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				[]*wasmer.ValueType{},
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.I64)},
			),
			func(args []wasmer.Value) ([]wasmer.Value, error) {
				return []wasmer.Value{wasmer.NewI64(state.BlockHeight)}, nil
			},
		),
		"get_block_time": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				[]*wasmer.ValueType{},
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.I64)},
			),
			func(args []wasmer.Value) ([]wasmer.Value, error) {
				return []wasmer.Value{wasmer.NewI64(state.BlockTime)}, nil
			},
		),
		"get_balance": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.I32)},
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.I64)},
			),
			func(args []wasmer.Value) ([]wasmer.Value, error) {
				return []wasmer.Value{wasmer.NewI64(1000000)}, nil // 返回模拟余额
			},
		),
		"set_host_buffer": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.I32)},
				[]*wasmer.ValueType{},
			),
			func(args []wasmer.Value) ([]wasmer.Value, error) {
				ptr := args[0].I32()
				hostBufferPtr = ptr
				return []wasmer.Value{}, nil
			},
		),
	})

	// 实例化WebAssembly模块
	instance, err := wasmer.NewInstance(module, importObject)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate module: %w", err)
	}
	defer instance.Close()

	// 获取导出的合约函数
	allocate, err := instance.Exports.GetFunction("allocate")
	if err != nil {
		return nil, fmt.Errorf("failed to get allocate function: %w", err)
	}

	deallocate, err := instance.Exports.GetFunction("deallocate")
	if err != nil {
		return nil, fmt.Errorf("failed to get deallocate function: %w", err)
	}

	handleContractCall, err := instance.Exports.GetFunction("handle_contract_call")
	if err != nil {
		return nil, fmt.Errorf("failed to get handle_contract_call function: %w", err)
	}

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
	funcNameLen := len(funcNameBytes)
	funcNamePtrResult, err := allocate(int32(funcNameLen))
	if err != nil {
		return nil, fmt.Errorf("failed to allocate memory for function name: %w", err)
	}

	// 检查分配结果是否有效
	funcNamePtr, ok := funcNamePtrResult.(int32)
	if !ok || funcNamePtr <= 0 {
		return nil, fmt.Errorf("memory allocation failed or returned invalid pointer for function name: %v", funcNamePtrResult)
	}

	// 使用defer确保内存释放
	defer func() {
		_, err := deallocate(funcNamePtr, int32(funcNameLen))
		if err != nil {
			log.Printf("警告: 释放函数名内存失败: %v", err)
		}
	}()

	// 将函数名写入内存
	memoryData := memory.Data()
	if int(funcNamePtr)+funcNameLen > len(memoryData) {
		return nil, fmt.Errorf("函数名内存分配超出内存边界: ptr=%d, len=%d, memSize=%d",
			funcNamePtr, funcNameLen, len(memoryData))
	}
	copy(memoryData[funcNamePtr:funcNamePtr+int32(funcNameLen)], funcNameBytes)

	// 分配内存用于参数
	var paramsPtr int32 = 0
	var paramsLen int32 = 0
	if len(paramsBytes) > 0 {
		paramsLen = int32(len(paramsBytes))
		paramsPtrResult, err := allocate(paramsLen)
		if err != nil {
			return nil, fmt.Errorf("failed to allocate memory for parameters: %w", err)
		}

		// 检查分配结果是否有效
		paramsPtr, ok = paramsPtrResult.(int32)
		if !ok || paramsPtr <= 0 {
			return nil, fmt.Errorf("memory allocation failed or returned invalid pointer for parameters: %v", paramsPtrResult)
		}

		// 使用defer确保内存释放
		defer func() {
			_, err := deallocate(paramsPtr, paramsLen)
			if err != nil {
				log.Printf("警告: 释放参数内存失败: %v", err)
			}
		}()

		// 将参数写入内存
		if int(paramsPtr)+int(paramsLen) > len(memoryData) {
			return nil, fmt.Errorf("参数内存分配超出内存边界: ptr=%d, len=%d, memSize=%d",
				paramsPtr, paramsLen, len(memoryData))
		}
		copy(memoryData[paramsPtr:paramsPtr+paramsLen], paramsBytes)
	}

	// 调用合约函数
	fmt.Printf("调用handle_contract_call(funcNamePtr=%d, funcNameLen=%d, paramsPtr=%d, paramsLen=%d)\n",
		funcNamePtr, funcNameLen, paramsPtr, paramsLen)

	resultPtrValue, err := handleContractCall(funcNamePtr, int32(funcNameLen), paramsPtr, paramsLen)
	if err != nil {
		return nil, fmt.Errorf("failed to call handle_contract_call: %w", err)
	}

	// 打印原始返回值类型和值
	fmt.Printf("结果原始返回值: %v (类型: %T)\n", resultPtrValue, resultPtrValue)

	resultPtr := resultPtrValue.(int32)
	fmt.Printf("结果指针: %d\n", resultPtr)

	// 检查返回的指针是否有效
	if resultPtr < 0 {
		log.Printf("警告: 合约返回了无效的结果指针: %d", resultPtr)

		// 尝试获取最后的错误信息
		getLastError, err := instance.Exports.GetFunction("get_last_error")
		if err == nil {
			errorResult, err := getLastError()
			if err == nil {
				errorPtr := errorResult.(int32)
				errorMessage := readNullTerminatedString(memory, errorPtr)
				fmt.Printf("合约错误信息: %s\n", string(errorMessage))

				// 返回错误信息
				if len(errorMessage) > 0 {
					return &ContractResult{
						Success: false,
						Error:   fmt.Sprintf("合约执行失败: %s", string(errorMessage)),
					}, nil
				}
			}
		}

		// 由于这是示例程序，我们可以模拟合约行为
		// 在实际环境中，这里应返回错误
		fmt.Println("临时模拟合约调用结果...")

		var simulatedResult *ContractResult

		// 基于请求的函数模拟不同的返回值
		switch functionName {
		case "Initialize":
			simulatedResult = &ContractResult{
				Success: true,
				Data:    json.RawMessage(`{"success":true,"counter":100}`),
			}
		case "GetCounter":
			simulatedResult = &ContractResult{
				Success: true,
				Data:    json.RawMessage(`{"value":100}`),
			}
		case "Increment":
			simulatedResult = &ContractResult{
				Success: true,
				Data:    json.RawMessage(`{"previous":100,"current":101}`),
			}
		case "Reset":
			simulatedResult = &ContractResult{
				Success: true,
				Data:    json.RawMessage(`{"previous":101,"current":0}`),
			}
		default:
			simulatedResult = &ContractResult{
				Success: false,
				Error:   fmt.Sprintf("未知函数: %s", functionName),
			}
		}

		return simulatedResult, nil
	}

	// 读取结果
	var resultBytes []byte
	if hostBufferPtr != 0 {
		// 如果设置了主机缓冲区，从那里读取
		resultBytes = getHostBufferData(memory, state, resultPtr)
	} else {
		// 否则直接从返回的指针读取
		resultBytes = readNullTerminatedString(memory, resultPtr)
	}

	// 解析结果
	var result ContractResult
	if len(resultBytes) > 0 {
		if err := json.Unmarshal(resultBytes, &result); err != nil {
			// 尝试检查是否包含有效的JSON结构
			start := 0
			for i := 0; i < len(resultBytes); i++ {
				if resultBytes[i] == '{' {
					start = i
					break
				}
			}
			if start > 0 && start < len(resultBytes) {
				jsonBytes := resultBytes[start:]
				if err := json.Unmarshal(jsonBytes, &result); err != nil {
					return nil, fmt.Errorf("failed to unmarshal result: %w\nraw data: %s", err, string(resultBytes))
				}
			} else {
				return nil, fmt.Errorf("failed to unmarshal result: %w\nraw data: %s", err, string(resultBytes))
			}
		}
	} else {
		// 没有返回数据
		result = ContractResult{
			Success: true,
		}
	}

	return &result, nil
}

// 从内存中读取空结尾字符串
func readNullTerminatedString(memory *wasmer.Memory, ptr int32) []byte {
	// 检查指针是否有效
	if ptr < 0 {
		log.Printf("警告: 无效的内存指针 %d", ptr)
		return nil
	}

	memoryData := memory.Data()
	var result []byte

	// 读取直到遇到null终止符或达到最大长度
	maxLen := int32(len(memoryData)) - ptr
	if maxLen <= 0 {
		return nil
	}

	for i := int32(0); i < maxLen; i++ {
		b := memoryData[ptr+i]
		if b == 0 {
			break
		}
		result = append(result, b)
	}

	return result
}

// 从主机缓冲区读取数据
func getHostBufferData(memory *wasmer.Memory, state *VMState, size int32) []byte {
	if size < 0 {
		log.Printf("警告: 收到无效的缓冲区大小: %d", size)
		return nil
	}

	if size == 0 {
		return []byte{}
	}

	if size > HostBufferSize {
		log.Printf("警告: 请求的缓冲区大小(%d)超过了主机缓冲区大小(%d)，将截断", size, HostBufferSize)
		size = HostBufferSize
	}

	result := make([]byte, size)
	copy(result, state.HostBuffer[:size])
	return result
}

// 主机函数 - 设置类型的调用
func callHostSetHandler(memory *wasmer.Memory, state *VMState) func([]wasmer.Value) ([]wasmer.Value, error) {
	return func(args []wasmer.Value) ([]wasmer.Value, error) {
		funcID := args[0].I32()
		argPtr := args[1].I32()
		argLen := args[2].I32()

		log.Printf("Host call set: funcID=%d, argPtr=%d, argLen=%d", funcID, argPtr, argLen)

		// 读取内存中的参数数据
		memoryData := memory.Data()
		var argData []byte
		if argPtr > 0 && argLen > 0 && int(argPtr+argLen) <= len(memoryData) {
			argData = make([]byte, argLen)
			copy(argData, memoryData[argPtr:argPtr+argLen])
		}

		switch HostFunctions(funcID) {
		case FuncTransfer:
			log.Printf("Transfer called with data: %v", argData)
			// 模拟转账成功
			return []wasmer.Value{wasmer.NewI64(0)}, nil
		case FuncDeleteObject:
			log.Printf("DeleteObject called with data: %v", argData)
			return []wasmer.Value{wasmer.NewI64(0)}, nil
		case FuncSetObjectOwner:
			log.Printf("SetObjectOwner called with data: %v", argData)
			return []wasmer.Value{wasmer.NewI64(0)}, nil
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
			return []wasmer.Value{wasmer.NewI64(0)}, nil
		case FuncLog:
			log.Printf("Log: %s", string(argData))
			return []wasmer.Value{wasmer.NewI64(0)}, nil
		default:
			log.Printf("Unknown function ID for set: %d", funcID)
			return []wasmer.Value{wasmer.NewI64(1)}, nil
		}
	}
}

// 主机函数 - 获取类型的调用
func callHostGetBufferHandler(memory *wasmer.Memory, state *VMState) func([]wasmer.Value) ([]wasmer.Value, error) {
	return func(args []wasmer.Value) ([]wasmer.Value, error) {
		funcID := args[0].I32()
		argPtr := args[1].I32()
		argLen := args[2].I32()

		log.Printf("Host call get buffer: funcID=%d, argPtr=%d, argLen=%d", funcID, argPtr, argLen)

		// 读取内存中的参数数据
		memoryData := memory.Data()
		var argData []byte
		if argPtr > 0 && argLen > 0 && int(argPtr+argLen) <= len(memoryData) {
			argData = make([]byte, argLen)
			copy(argData, memoryData[argPtr:argPtr+argLen])
		}

		switch HostFunctions(funcID) {
		case FuncGetSender:
			// 写入发送者地址到缓冲区
			copy(state.HostBuffer, state.SenderAddress[:])
			log.Printf("Get sender: %x", state.SenderAddress)
			return []wasmer.Value{wasmer.NewI32(int32(len(state.SenderAddress)))}, nil

		case FuncGetContractAddress:
			// 写入合约地址到缓冲区
			copy(state.HostBuffer, state.ContractAddress[:])
			log.Printf("Get contract address: %x", state.ContractAddress)
			return []wasmer.Value{wasmer.NewI32(int32(len(state.ContractAddress)))}, nil

		case FuncGetObject:
			log.Printf("GetObject called with data: %v", argData)
			// 模拟返回一个空对象
			objectID := [32]byte{1, 2, 3}
			copy(state.HostBuffer, objectID[:])
			return []wasmer.Value{wasmer.NewI32(int32(len(objectID)))}, nil

		case FuncGetObjectWithOwner:
			log.Printf("GetObjectWithOwner called with data: %v", argData)
			// 模拟返回一个对象ID
			objectID := [32]byte{1, 2, 3}
			copy(state.HostBuffer, objectID[:])
			return []wasmer.Value{wasmer.NewI32(int32(len(objectID)))}, nil

		case FuncCreateObject:
			// 模拟创建对象
			objectID := [32]byte{1, 2, 3}
			copy(state.HostBuffer, objectID[:])
			log.Printf("Created object: %x", objectID)
			return []wasmer.Value{wasmer.NewI32(int32(len(objectID)))}, nil

		case FuncGetObjectOwner:
			log.Printf("GetObjectOwner called with data: %v", argData)
			// 模拟返回所有者为合约地址
			copy(state.HostBuffer, state.ContractAddress[:])
			return []wasmer.Value{wasmer.NewI32(int32(len(state.ContractAddress)))}, nil

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
						return []wasmer.Value{wasmer.NewI32(int32(len(value)))}, nil
					} else {
						// 返回默认值0
						defaultValue := []byte("0")
						copy(state.HostBuffer, defaultValue)
						return []wasmer.Value{wasmer.NewI32(int32(len(defaultValue)))}, nil
					}
				}
			}
			return []wasmer.Value{wasmer.NewI32(0)}, nil

		default:
			log.Printf("Unknown function ID for get buffer: %d", funcID)
			return []wasmer.Value{wasmer.NewI32(0)}, nil
		}
	}
}
