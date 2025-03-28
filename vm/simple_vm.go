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
	vm.blockchainCtx = NewSimpleBlockchainContext()

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
func (vm *SimpleVM) ExecuteContract(ctx types.BlockchainContext, contractAddr, sender core.Address, functionName string, params []byte) ([]byte, error) {
	// 检查合约是否存在
	vm.contractsLock.RLock()
	wasmCode, exists := vm.contracts[contractAddr]
	vm.contractsLock.RUnlock()

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

	// Compile module
	module, err := wasmer.NewModule(store, wasmCode)
	if err != nil {
		log.Fatalf("Failed to compile module: %v", err)
	}

	// Create WASI environment
	wasiEnv, err := wasmer.NewWasiStateBuilder("wasi-program").
		// Add WASI args if needed
		Argument("--verbose").
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
		log.Fatalf("Failed to instantiate module: %v", err)
	}

	m, err := instance.Exports.GetMemory("memory")
	if err != nil {
		log.Fatalf("无法获取内存: %v", err)
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

func (vm *SimpleVM) callWasmFunction(ctx types.BlockchainContext, instance *wasmer.Instance, functionName string, params []byte) ([]byte, error) {
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
		return nil, fmt.Errorf("没有allocate函数")
	}
	processDataFunc, err := instance.Exports.GetFunction("handle_contract_call")
	if err != nil {
		log.Fatalf("handle_contract_call没找到:%v", err)
	}
	var input types.HandleContractCallParams
	input.Contract = ctx.ContractAddress()
	input.Function = functionName
	input.Sender = ctx.Sender()
	input.Args = params
	inputBytes, err := json.Marshal(input)
	if err != nil {
		log.Fatalf("handle_contract_call 序列化失败: %v", err)
	}

	inputAddr, err := allocate(len(inputBytes))
	if err != nil {
		log.Fatalf("fn 内存分配失败: %v", err)
	}
	inputPtr := inputAddr.(int32)
	copy(memory.Data()[int(inputPtr):int(inputPtr)+len(inputBytes)], []byte(inputBytes))

	result, err := processDataFunc(inputPtr, len(inputBytes))
	if err != nil {
		log.Printf("执行%s 失败: %v", functionName, err)
		return nil, err
	}
	resultLen, _ := result.(int32)
	var out []byte
	if resultLen > 0 {
		getBufferAddress, getBufferErr := instance.Exports.GetFunction("get_buffer_address")
		if getBufferErr != nil {
			fmt.Println("没有get_buffer_address函数")
			return nil, fmt.Errorf("没有get_buffer_address函数")
		}
		rst, err := getBufferAddress()
		if err != nil {
			log.Fatalf("get_buffer_address 失败: %v", err)
		}
		bufferPtr := rst.(int32)
		data := readString(memory, bufferPtr, resultLen)
		fmt.Printf("result: %s\n", data)
		out = []byte(data)
	}
	fmt.Printf("执行结束:%s, %v\n", functionName, result)
	//free memory
	free, freeErr := instance.Exports.GetFunction("deallocate")
	if freeErr != nil {
		fmt.Println("没有deallocate函数")
		return nil, fmt.Errorf("没有deallocate函数")
	}
	free(inputPtr, len(inputBytes))
	if resultLen < 0 {
		return nil, fmt.Errorf("执行%s 失败: %v", functionName, result)
	}

	return out, nil
}

// simpleBlockchainContext 实现了默认的区块链上下文
type simpleBlockchainContext struct {
	// 区块信息
	blockHeight uint64
	blockTime   int64

	// 账户余额
	balances map[types.Address]uint64

	// 虚拟机对象存储
	objects        map[core.ObjectID]map[string]interface{}
	objectOwner    map[core.ObjectID]types.Address
	objectContract map[core.ObjectID]types.Address

	// 当前执行上下文
	contractAddr types.Address
	sender       types.Address
	txHash       core.Hash
}

// NewSimpleBlockchainContext 创建一个新的简单区块链上下文
func NewSimpleBlockchainContext() *simpleBlockchainContext {
	return &simpleBlockchainContext{
		blockHeight:    100,
		blockTime:      200,
		balances:       make(map[Address]uint64),
		objects:        make(map[core.ObjectID]map[string]interface{}),
		objectOwner:    make(map[core.ObjectID]Address),
		objectContract: make(map[core.ObjectID]Address),
	}
}

// SetExecutionContext 设置当前执行上下文
func (ctx *simpleBlockchainContext) SetExecutionContext(contractAddr, sender types.Address) {
	ctx.contractAddr = contractAddr
	ctx.sender = sender
}

func (ctx *simpleBlockchainContext) WithTransaction(txHash core.Hash) types.BlockchainContext {
	ctx.txHash = txHash
	return ctx
}

func (ctx *simpleBlockchainContext) WithBlock(height uint64, time int64) types.BlockchainContext {
	ctx.blockHeight = height
	ctx.blockTime = time
	return ctx
}

// BlockHeight 获取当前区块高度
func (ctx *simpleBlockchainContext) BlockHeight() uint64 {
	return ctx.blockHeight
}

// BlockTime 获取当前区块时间戳
func (ctx *simpleBlockchainContext) BlockTime() int64 {
	return ctx.blockTime
}

// ContractAddress 获取当前合约地址
func (ctx *simpleBlockchainContext) ContractAddress() types.Address {
	return ctx.contractAddr
}

// TransactionHash 获取当前交易哈希
func (ctx *simpleBlockchainContext) TransactionHash() core.Hash {
	return core.Hash{} // 简化实现
}

// Sender 获取交易发送者
func (ctx *simpleBlockchainContext) Sender() types.Address {
	return ctx.sender
}

// Balance 获取账户余额
func (ctx *simpleBlockchainContext) Balance(addr types.Address) uint64 {
	return ctx.balances[addr]
}

// Transfer 转账操作
func (ctx *simpleBlockchainContext) Transfer(from, to types.Address, amount uint64) error {
	// 检查余额
	fromBalance := ctx.balances[from]
	if fromBalance < amount {
		return errors.New("余额不足")
	}

	// 执行转账
	ctx.balances[from] -= amount
	ctx.balances[to] += amount
	return nil
}

// CreateObject 创建新对象
func (ctx *simpleBlockchainContext) CreateObject(contract types.Address) (types.VMObject, error) {
	// 创建对象ID，简化版使用随机数
	id := ctx.generateObjectID(contract)

	// 创建对象存储
	ctx.objects[id] = make(map[string]interface{})
	ctx.objectOwner[id] = ctx.Sender()
	ctx.objectContract[id] = contract

	// 返回对象封装
	return &vmObject{
		objects:        ctx.objects,
		objectOwner:    ctx.objectOwner,
		objectContract: ctx.objectContract,
		id:             id,
	}, nil
}

// CreateObject 创建新对象
func (ctx *simpleBlockchainContext) CreateObjectWithID(contract types.Address, id types.ObjectID) (types.VMObject, error) {
	// 创建对象存储
	ctx.objects[id] = make(map[string]interface{})
	ctx.objectOwner[id] = ctx.Sender()
	ctx.objectContract[id] = contract

	// 返回对象封装
	return &vmObject{
		objects:        ctx.objects,
		objectOwner:    ctx.objectOwner,
		objectContract: ctx.objectContract,
		id:             id,
	}, nil
}

// generateObjectID 生成一个新的对象ID
func (ctx *simpleBlockchainContext) generateObjectID(contract types.Address) core.ObjectID {
	var id core.ObjectID
	copy(id[:16], contract[:16])
	return id
}

// GetObject 获取指定对象
func (ctx *simpleBlockchainContext) GetObject(contract types.Address, id core.ObjectID) (types.VMObject, error) {
	_, exists := ctx.objects[id]
	if !exists {
		return nil, errors.New("对象不存在")
	}

	return &vmObject{
		objects:        ctx.objects,
		objectOwner:    ctx.objectOwner,
		objectContract: ctx.objectContract,
		id:             id,
	}, nil
}

// GetObjectWithOwner 按所有者获取对象
func (ctx *simpleBlockchainContext) GetObjectWithOwner(contract, owner types.Address) (types.VMObject, error) {
	for id, objOwner := range ctx.objectOwner {
		if objOwner == owner {
			return &vmObject{
				objects:        ctx.objects,
				objectOwner:    ctx.objectOwner,
				objectContract: ctx.objectContract,
				id:             id,
			}, nil
		}
	}
	return nil, errors.New("未找到对象")
}

// DeleteObject 删除对象
func (ctx *simpleBlockchainContext) DeleteObject(contract types.Address, id core.ObjectID) error {
	delete(ctx.objects, id)
	delete(ctx.objectOwner, id)
	delete(ctx.objectContract, id)
	return nil
}

// Call 跨合约调用
func (ctx *simpleBlockchainContext) Call(contract types.Address, function string, args ...any) ([]byte, error) {
	return nil, errors.New("未实现跨合约调用")
}

// Log 记录事件
func (ctx *simpleBlockchainContext) Log(contract types.Address, eventName string, keyValues ...any) {
	fmt.Printf("合约日志: %s, 合约: %x, 参数: %v\n", eventName, contract, keyValues)
}

// vmObject 实现了对象接口
type vmObject struct {
	objects        map[core.ObjectID]map[string]interface{}
	objectOwner    map[core.ObjectID]types.Address
	objectContract map[core.ObjectID]types.Address
	id             core.ObjectID
}

// ID 获取对象ID
func (o *vmObject) ID() core.ObjectID {
	return o.id
}

// Owner 获取对象所有者
func (o *vmObject) Owner() types.Address {
	return o.objectOwner[o.id]
}

// Contract 获取对象所属合约
func (o *vmObject) Contract() types.Address {
	return o.objectContract[o.id]
}

// SetOwner 设置对象所有者
func (o *vmObject) SetOwner(addr types.Address) error {
	o.objectOwner[o.id] = addr
	return nil
}

// Get 获取字段值
func (o *vmObject) Get(field string) ([]byte, error) {
	obj, exists := o.objects[o.id]
	if !exists {
		return nil, errors.New("对象不存在")
	}

	fieldValue, exists := obj[field]
	if !exists {
		return nil, errors.New("字段不存在")
	}

	data, err := json.Marshal(fieldValue)
	if err != nil {
		return nil, fmt.Errorf("序列化失败: %w", err)
	}

	return data, nil
}

// Set 设置字段值
func (o *vmObject) Set(field string, value []byte) error {
	obj, exists := o.objects[o.id]
	if !exists {
		return errors.New("对象不存在")
	}

	var fieldValue interface{}
	if err := json.Unmarshal(value, &fieldValue); err != nil {
		return fmt.Errorf("反序列化失败: %w", err)
	}

	obj[field] = fieldValue
	return nil
}

// 合并所有宿主函数到统一的调用处理器 - 用于设置数据的函数
func (vm *SimpleVM) callHostSetHandler(ctx types.BlockchainContext, memory *wasmer.Memory) func([]wasmer.Value) ([]wasmer.Value, error) {
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
				fmt.Println("deleteObject获取对象失败", params.ID)
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("获取对象失败: %v", err)
			}
			if obj.Contract() != params.Contract {
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("权限不足")
			}
			ctx.DeleteObject(params.Contract, params.ID)
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
			if params.ID == (types.ObjectID{}) {
				copy(params.ID[:], params.Contract[:])
			}
			obj, err := ctx.GetObject(params.Contract, params.ID)
			if err != nil {
				fmt.Println("setOwner获取对象失败", params.ID)
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("获取对象失败: %v", err)
			}
			if obj.Contract() != params.Contract {
				fmt.Println("权限不足, 合约不匹配", obj.Contract(), params.Contract)
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
				fmt.Println("setObjectField获取对象失败", params.ID)
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("获取对象失败: %v", err)
			}
			// todo:验证权限
			if obj.Contract() != params.Contract {
				fmt.Println("权限不足, 合约不匹配", obj.Contract(), params.Contract)
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
			fmt.Printf("未知的Set函数ID: %d\n", funcID)
			return []wasmer.Value{wasmer.NewI32(-1)}, nil
		}
	}
}

// 合并所有宿主函数到统一的调用处理器 - 用于获取缓冲区数据的函数
func (vm *SimpleVM) callHostGetBufferHandler(ctx types.BlockchainContext, memory *wasmer.Memory) func([]wasmer.Value) ([]wasmer.Value, error) {
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
		if bufferPtr < 0 || bufferPtr >= memorySize || bufferPtr+types.HostBufferSize > memorySize {
			fmt.Printf("无效的缓冲区位置: 指针=%d, 大小=%d, 内存大小=%d\n", bufferPtr, HostBufferSize, memorySize)
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
				fmt.Println("getObjectField获取对象失败", params.ID)
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
			fmt.Println("获取对象", params.ID, params.Contract)
			if params.ID == (types.ObjectID{}) {
				copy(params.ID[:], params.Contract[:])
			}
			obj, err := ctx.GetObject(params.Contract, params.ID)
			if err != nil {
				fmt.Println("getObject获取对象失败", params.ID)
				return []wasmer.Value{wasmer.NewI64(-1)}, fmt.Errorf("获取对象失败: %v", err)
			}
			if obj.Contract() != params.Contract {
				fmt.Println("权限不足, 合约不匹配", obj.Contract(), params.Contract)
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
				fmt.Println("getObjectOwner获取对象失败", objID)
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

// 获取余额处理函数
func (vm *SimpleVM) getBalanceHandler(ctx types.BlockchainContext, memory *wasmer.Memory) func([]wasmer.Value) ([]wasmer.Value, error) {
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
		balance := ctx.Balance(addr)
		return []wasmer.Value{wasmer.NewI64(int64(balance))}, nil
	}
}
