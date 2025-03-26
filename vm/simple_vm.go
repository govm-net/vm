// Package vm 实现了基于WebAssembly的虚拟机核心功能
package vm

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
	contracts     map[string][]byte
	contractsLock sync.RWMutex

	// 区块信息
	blockHeight uint64
	blockTime   int64

	// 账户余额
	balances map[Address]uint64

	// 虚拟机对象存储
	objects     map[string]map[string]interface{}
	objectOwner map[string]string

	// 合约存储目录
	contractDir string
}

// NewSimpleVM 创建一个新的简化虚拟机实例
func NewSimpleVM(contractDir string) (*SimpleVM, error) {
	// 确保合约目录存在
	if contractDir != "" {
		if err := os.MkdirAll(contractDir, 0755); err != nil {
			return nil, fmt.Errorf("创建合约目录失败: %w", err)
		}
	}

	return &SimpleVM{
		contracts:   make(map[string][]byte),
		balances:    make(map[Address]uint64),
		objects:     make(map[string]map[string]interface{}),
		objectOwner: make(map[string]string),
		contractDir: contractDir,
	}, nil
}

// SetBlockInfo 设置当前区块信息
func (vm *SimpleVM) SetBlockInfo(height uint64, time int64) {
	vm.blockHeight = height
	vm.blockTime = time
}

// SetBalance 设置账户余额
func (vm *SimpleVM) SetBalance(addr core.Address, balance uint64) {
	vm.balances[addr] = balance
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

	// 存储合约代码
	vm.contractsLock.Lock()
	vm.contracts[fmt.Sprintf("%x", contractAddr)] = wasmCode
	vm.contractsLock.Unlock()

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
func (vm *SimpleVM) ExecuteContract(contractAddr core.Address, sender core.Address, functionName string, params []byte) (interface{}, error) {
	// 检查合约是否存在
	vm.contractsLock.RLock()
	wasmCode, exists := vm.contracts[fmt.Sprintf("%x", contractAddr)]
	vm.contractsLock.RUnlock()

	if !exists {
		return nil, fmt.Errorf("合约不存在: %x", contractAddr)
	}

	// 创建执行上下文
	ctx := &vmContext{
		vm:           vm,
		contractAddr: contractAddr,
		sender:       sender,
	}

	// Create Wasmer instance
	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)

	// Compile module
	module, err := wasmer.NewModule(store, wasmCode)
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
			callHostSetHandler(ctx),
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
			callHostGetBufferHandler(ctx),
		),
		// 单独的简单数据类型获取函数
		"get_block_height": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				[]*wasmer.ValueType{},                                // 无参数
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.I64)}, // 返回int64
			),
			getBlockHeightHandler(ctx),
		),
		"get_block_time": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				[]*wasmer.ValueType{},                                // 无参数
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.I64)}, // 返回int64
			),
			getBlockTimeHandler(ctx),
		),
		"get_balance": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				[]*wasmer.ValueType{
					wasmer.NewValueType(wasmer.I32), // addrPtr
				},
				[]*wasmer.ValueType{wasmer.NewValueType(wasmer.F64)}, // 返回float64
			),
			getBalanceHandler(ctx),
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
	ctx.memory = m
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

func (vm *SimpleVM) callWasmFunction(ctx *vmContext, instance *wasmer.Instance, functionName string, params []byte) (interface{}, error) {
	fmt.Printf("调用合约函数:%s, %v\n", functionName, params)
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

	fnAddr, err := allocate(len(functionName))
	if err != nil {
		log.Fatalf("fn 内存分配失败: %v", err)
	}
	fnPtr := fnAddr.(int32)
	copy(memory.Data()[int(fnPtr):int(fnPtr)+len(functionName)], []byte(functionName))

	var paramAddr int32
	if len(params) > 0 {
		paramAddr, err := allocate(len(params))
		if err != nil {
			log.Fatalf("param 内存分配失败: %v", err)
		}
		paramPtr := paramAddr.(int32)
		copy(memory.Data()[int(paramPtr):int(paramPtr)+len(params)], params)
	}

	result, err := processDataFunc(fnPtr, len(functionName), paramAddr, len(params))
	if err != nil {
		log.Fatalf("执行%s 失败: %v", functionName, err)
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
	fmt.Printf("执行结束:%s, %v\n", functionName, result)
	//free memory
	free, freeErr := instance.Exports.GetFunction("deallocate")
	if freeErr != nil {
		fmt.Println("没有deallocate函数")
		return 0, fmt.Errorf("没有deallocate函数")
	}
	free(fnPtr, len(functionName))
	if freeErr != nil {
		fmt.Println("没有free函数")
		return 0, fmt.Errorf("没有free函数")
	}

	return resultLen, nil
}

// vmContext 实现了合约执行上下文
type vmContext struct {
	vm           *SimpleVM
	contractAddr core.Address
	sender       core.Address
	memory       *wasmer.Memory
}

// BlockHeight 获取当前区块高度
func (ctx *vmContext) BlockHeight() uint64 {
	return ctx.vm.blockHeight
}

// BlockTime 获取当前区块时间戳
func (ctx *vmContext) BlockTime() int64 {
	return ctx.vm.blockTime
}

// ContractAddress 获取当前合约地址
func (ctx *vmContext) ContractAddress() core.Address {
	return ctx.contractAddr
}

// Sender 获取交易发送者
func (ctx *vmContext) Sender() core.Address {
	return ctx.sender
}

// Balance 获取账户余额
func (ctx *vmContext) Balance(addr core.Address) uint64 {
	return ctx.vm.balances[addr]
}

// Transfer 转账操作
func (ctx *vmContext) Transfer(from core.Address, to core.Address, amount uint64) error {
	// 检查余额
	fromBalance := ctx.vm.balances[from]
	if fromBalance < amount {
		return errors.New("余额不足")
	}

	// 执行转账
	ctx.vm.balances[from] -= amount
	ctx.vm.balances[to] += amount
	return nil
}

// CreateObject 创建新对象
func (ctx *vmContext) CreateObject() core.Object {
	// 创建对象ID，简化版使用随机数
	id := ctx.generateObjectID()

	// 创建对象存储
	ctx.vm.objects[fmt.Sprintf("%x", id)] = make(map[string]interface{})
	ctx.vm.objectOwner[fmt.Sprintf("%x", id)] = fmt.Sprintf("%x", ctx.sender)

	// 返回对象封装
	return &vmObject{
		vm: ctx.vm,
		id: id,
	}
}

// generateObjectID 生成一个新的对象ID
func (ctx *vmContext) generateObjectID() core.ObjectID {
	// 简化实现，实际应使用哈希或随机数
	var id core.ObjectID
	// 使用合约地址的前16字节和当前时间的后16字节
	copy(id[:16], ctx.contractAddr[:16])
	// 后16字节随机生成
	// 实际实现应使用安全的随机数生成
	return id
}

// GetObject 获取指定对象
func (ctx *vmContext) GetObject(id core.ObjectID) (core.Object, error) {
	_, exists := ctx.vm.objects[fmt.Sprintf("%x", id)]
	if !exists {
		return nil, errors.New("对象不存在")
	}

	return &vmObject{
		vm: ctx.vm,
		id: id,
	}, nil
}

// GetObjectWithOwner 按所有者获取对象
func (ctx *vmContext) GetObjectWithOwner(contract, owner core.Address) (core.Object, error) {
	// 简化版，实际应该维护所有者到对象的索引
	ownerHex := fmt.Sprintf("%x", owner)
	for _, objOwnerStr := range ctx.vm.objectOwner {
		// todo: 验证权限
		if objOwnerStr == ownerHex {
			// 将字符串ID转换为ObjectID
			var id core.ObjectID
			// 简化处理，实际应正确转换
			return &vmObject{
				vm: ctx.vm,
				id: id,
			}, nil
		}
	}
	return nil, errors.New("未找到对象")
}

// DeleteObject 删除对象
func (ctx *vmContext) DeleteObject(id core.ObjectID) {
	idHex := fmt.Sprintf("%x", id)
	delete(ctx.vm.objects, idHex)
	delete(ctx.vm.objectOwner, idHex)
}

// Call 跨合约调用
func (ctx *vmContext) Call(contract core.Address, function string, args ...interface{}) ([]byte, error) {
	// 简化版，实际应实现完整的跨合约调用逻辑
	return nil, errors.New("未实现跨合约调用")
}

// Log 记录事件
func (ctx *vmContext) Log(eventName string, keyValues ...interface{}) {
	// 简化版，仅打印日志
	fmt.Printf("合约日志: %s, 合约: %x, 参数: %v\n", eventName, ctx.contractAddr, keyValues)
}

// vmObject 实现了对象接口
type vmObject struct {
	vm *SimpleVM
	id core.ObjectID
}

// ID 获取对象ID
func (o *vmObject) ID() core.ObjectID {
	return o.id
}

// Owner 获取对象所有者
func (o *vmObject) Owner() core.Address {
	// 对于简化实现，我们只返回空地址
	// 实际应该正确转换字符串到地址
	return core.Address{}
}

func (o *vmObject) Contract() core.Address {
	return core.Address{}
}

// SetOwner 设置对象所有者
func (o *vmObject) SetOwner(addr core.Address) {
	o.vm.objectOwner[fmt.Sprintf("%x", o.id)] = fmt.Sprintf("%x", addr)
}

// Get 获取字段值
func (o *vmObject) Get(field string, value interface{}) error {
	obj, exists := o.vm.objects[fmt.Sprintf("%x", o.id)]
	if !exists {
		return errors.New("对象不存在")
	}

	fieldValue, exists := obj[field]
	if !exists {
		return errors.New("字段不存在")
	}

	// 这里简化处理，实际应根据类型正确转换
	// 应该使用反射或类型断言
	fmt.Printf("获取字段: %s = %v\n", field, fieldValue)
	return nil
}

// Set 设置字段值
func (o *vmObject) Set(field string, value interface{}) error {
	obj, exists := o.vm.objects[fmt.Sprintf("%x", o.id)]
	if !exists {
		return errors.New("对象不存在")
	}

	obj[field] = value
	return nil
}

var state = &HostState{
	Balances:       make(map[Address]uint64),
	Objects:        make(map[ObjectID]core.Object),
	ObjectsByOwner: make(map[Address][]ObjectID),
}

// 合并所有宿主函数到统一的调用处理器 - 用于设置数据的函数
func callHostSetHandler(ctx *vmContext) func([]wasmer.Value) ([]wasmer.Value, error) {
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
			if ctx.memory == nil {
				fmt.Println("内存实例为空")
				return []wasmer.Value{wasmer.NewI64(0)}, fmt.Errorf("内存实例为空")
			}

			// 获取内存大小
			memorySize := int32(len(ctx.memory.Data()))

			// 检查参数指针和长度是否有效
			if argPtr < 0 || argPtr >= memorySize || argPtr+argLen > memorySize {
				fmt.Printf("无效的内存访问: 指针=%d, 长度=%d, 内存大小=%d\n", argPtr, argLen, memorySize)
				return []wasmer.Value{wasmer.NewI64(0)}, fmt.Errorf("无效的内存访问")
			}

			// 安全地读取参数数据
			argData = make([]byte, argLen)
			copy(argData, ctx.memory.Data()[argPtr:argPtr+argLen])
			fmt.Printf("[HOST] Set函数 ID=%d, 参数长度=%d, 参数数据:%s\n", funcID, argLen, string(argData))
		}

		fmt.Printf("调用宿主Set函数 ID=%d, 参数长度=%d\n", funcID, argLen)

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
			// todo:验证权限
			obj, err := ctx.GetObject(params.ID)
			if err != nil {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("获取对象失败: %v", err)
			}
			if obj.Contract() != params.Contract {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("权限不足")
			}
			ctx.DeleteObject(params.ID)
			return []wasmer.Value{wasmer.NewI32(0)}, nil
		case FuncLog: // 记录日志
			// 实现日志记录的逻辑...
			fmt.Printf("[WASM]日志: len:%d %s\n", len(argData), string(argData))
			return []wasmer.Value{wasmer.NewI32(0)}, nil

		case FuncSetObjectOwner: // 设置对象所有者
			var params types.SetOwnerParams
			if err := json.Unmarshal(argData, &params); err != nil {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("解析参数失败: %v", err)
			}
			obj, err := ctx.GetObject(params.ID)
			if err != nil {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("获取对象失败: %v", err)
			}
			if obj.Contract() != params.Contract {
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
			obj, err := ctx.GetObject(params.ID)
			if err != nil {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("获取对象失败: %v", err)
			}
			// todo:验证权限
			if obj.Contract() != params.Contract {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("权限不足, 合约不匹配")
			}
			if obj.Owner() != ctx.Sender() && obj.Owner() != params.Contract && obj.Owner() != params.Sender {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("权限不足, 所有者不匹配")
			}
			obj.Set(params.Field, params.Value)
			return []wasmer.Value{wasmer.NewI32(0)}, nil

		default:
			fmt.Printf("未知的Set函数ID: %d\n", funcID)
			return []wasmer.Value{wasmer.NewI32(-1)}, nil
		}
	}
}

// 合并所有宿主函数到统一的调用处理器 - 用于获取缓冲区数据的函数
func callHostGetBufferHandler(ctx *vmContext) func([]wasmer.Value) ([]wasmer.Value, error) {
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
		if ctx.memory == nil {
			fmt.Println("内存实例为空")
			return []wasmer.Value{wasmer.NewI32(0)}, fmt.Errorf("内存实例为空")
		}

		// 获取内存大小
		memorySize := int32(len(ctx.memory.Data()))

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
			copy(argData, ctx.memory.Data()[argPtr:argPtr+argLen])
		}

		// 检查主机缓冲区是否在有效范围内
		if bufferPtr < 0 || bufferPtr >= memorySize || bufferPtr+types.HostBufferSize > memorySize {
			fmt.Printf("无效的缓冲区位置: 指针=%d, 大小=%d, 内存大小=%d\n", bufferPtr, HostBufferSize, memorySize)
			return []wasmer.Value{wasmer.NewI32(0)}, fmt.Errorf("无效的缓冲区位置")
		}

		// 获取全局缓冲区
		hostBuffer := ctx.memory.Data()[bufferPtr : bufferPtr+types.HostBufferSize]

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
			obj := ctx.CreateObject()
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
			obj, err := ctx.GetObject(params.ID)
			if err != nil {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("获取对象失败: %v", err)
			}
			var data any
			err = obj.Get(params.Field, &data)
			if err != nil {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("获取字段失败: %v", err)
			}
			d, _ := json.Marshal(data)
			dataSize := copy(hostBuffer, d)
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
			if params.ID == (types.ObjectID{}) {
				copy(params.ID[:], params.Contract[:])
			}
			obj, err := ctx.GetObject(params.ID)
			if err != nil {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("获取对象失败: %v", err)
			}
			if obj.Contract() != params.Contract {
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
			obj, err := ctx.GetObject(objID)
			if err != nil {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("获取对象失败: %v", err)
			}

			// 获取对象所有者并写入全局缓冲区
			owner := obj.Owner()
			dataSize := copy(hostBuffer, owner[:])
			return []wasmer.Value{wasmer.NewI32(dataSize)}, nil

		default:
			fmt.Printf("未知的GetBuffer函数ID: %d\n", funcID)
			return []wasmer.Value{wasmer.NewI32(0)}, nil
		}
	}
}

// 获取区块高度处理函数
func getBlockHeightHandler(ctx *vmContext) func([]wasmer.Value) ([]wasmer.Value, error) {
	return func(args []wasmer.Value) ([]wasmer.Value, error) {
		return []wasmer.Value{wasmer.NewI64(int64(ctx.vm.blockHeight))}, nil
	}
}

// 获取区块时间处理函数
func getBlockTimeHandler(ctx *vmContext) func([]wasmer.Value) ([]wasmer.Value, error) {
	return func(args []wasmer.Value) ([]wasmer.Value, error) {
		return []wasmer.Value{wasmer.NewI64(ctx.vm.blockTime)}, nil
	}
}

// 获取余额处理函数
func getBalanceHandler(ctx *vmContext) func([]wasmer.Value) ([]wasmer.Value, error) {
	return func(args []wasmer.Value) ([]wasmer.Value, error) {
		if len(args) != 1 {
			fmt.Println("参数数量不正确")
			return []wasmer.Value{wasmer.NewI64(0)}, nil
		}

		addrPtr := args[0].I32()

		// 安全检查 - 确保memory不为nil
		if ctx.memory == nil {
			fmt.Println("内存实例为空")
			return []wasmer.Value{wasmer.NewI64(0)}, fmt.Errorf("内存实例为空")
		}

		// 获取内存大小
		memorySize := int32(len(ctx.memory.Data()))

		// 检查指针是否有效
		if addrPtr < 0 || addrPtr+20 > memorySize {
			fmt.Printf("无效的地址指针: %d\n", addrPtr)
			return []wasmer.Value{wasmer.NewI64(0)}, fmt.Errorf("无效的地址指针")
		}

		// 读取地址
		addrData := make([]byte, 20)
		copy(addrData, ctx.memory.Data()[addrPtr:addrPtr+20])

		var addr Address
		copy(addr[:], addrData)

		// 获取余额
		balance := state.Balances[addr]
		return []wasmer.Value{wasmer.NewI64(int64(balance))}, nil
	}
}
