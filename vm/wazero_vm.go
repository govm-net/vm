package vm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/types"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// WazeroVM 使用wazero实现的虚拟机
type WazeroVM struct {
	// 存储已部署合约的映射表
	contracts     map[types.Address][]byte
	contractsLock sync.RWMutex

	// 区块信息
	blockHeight uint64
	blockTime   int64

	// 账户余额
	balances map[types.Address]uint64

	// 虚拟机对象存储
	objects        map[core.ObjectID]map[string]interface{}
	objectOwner    map[core.ObjectID]types.Address
	objectContract map[core.ObjectID]types.Address

	// 合约存储目录
	contractDir string

	// wazero运行时
	// runtime wazero.Runtime
	ctx context.Context

	// env模块
	envModule api.Module

	// 当前合约上下文
	// currentContext *wazeroContext
	// contextLock sync.RWMutex
}

// wazeroContext 实现了合约执行上下文
type wazeroContext struct {
	vm           *WazeroVM
	contractAddr types.Address
	sender       types.Address
	memory       api.Memory
}

// NewWazeroVM 创建一个新的wazero虚拟机实例
func NewWazeroVM(contractDir string) (*WazeroVM, error) {
	// 确保合约目录存在
	if contractDir != "" {
		if err := os.MkdirAll(contractDir, 0755); err != nil {
			return nil, fmt.Errorf("创建合约目录失败: %w", err)
		}
	}

	// 创建wazero运行时
	ctx := context.Background()

	vm := &WazeroVM{
		contracts:      make(map[types.Address][]byte),
		balances:       make(map[types.Address]uint64),
		objects:        make(map[core.ObjectID]map[string]interface{}),
		objectOwner:    make(map[core.ObjectID]types.Address),
		objectContract: make(map[core.ObjectID]types.Address),
		contractDir:    contractDir,
		// runtime:        runtime,
		ctx: ctx,
	}

	return vm, nil
}

// SetBlockInfo 设置当前区块信息
func (vm *WazeroVM) SetBlockInfo(height uint64, time int64) {
	vm.blockHeight = height
	vm.blockTime = time
}

// SetBalance 设置账户余额
func (vm *WazeroVM) SetBalance(addr types.Address, balance uint64) {
	vm.balances[addr] = balance
}

// DeployContract 部署新的WebAssembly合约
func (vm *WazeroVM) DeployContract(wasmCode []byte, sender types.Address) (types.Address, error) {
	// 验证WASM代码
	if len(wasmCode) == 0 {
		return types.Address{}, errors.New("合约代码不能为空")
	}
	// ctx := &wazeroContext{
	// 	vm:           vm,
	// 	contractAddr: types.Address{},
	// 	sender:       sender,
	// }

	// // 实例化模块
	// _, err := vm.initContract(ctx, wasmCode, sender)
	// if err != nil {
	// 	return types.Address{}, fmt.Errorf("实例化WebAssembly模块失败: %w", err)
	// }

	// 生成合约地址
	contractAddr := vm.generateContractAddress(wasmCode, sender)

	var objectID core.ObjectID
	copy(objectID[:], contractAddr[:])

	// 存储合约代码
	vm.contractsLock.Lock()
	vm.contracts[contractAddr] = wasmCode
	vm.objects[objectID] = make(map[string]interface{})
	vm.objectOwner[objectID] = contractAddr
	vm.objectContract[objectID] = contractAddr
	vm.contractsLock.Unlock()

	// 如果指定了合约目录，则保存到文件
	if vm.contractDir != "" {
		contractPath := filepath.Join(vm.contractDir, fmt.Sprintf("%x", contractAddr)+".wasm")
		if err := os.WriteFile(contractPath, wasmCode, 0644); err != nil {
			return types.Address{}, fmt.Errorf("存储合约代码失败: %w", err)
		}
	}

	return contractAddr, nil
}

func (vm *WazeroVM) initContract(ctx *wazeroContext, wasmCode []byte, sender types.Address) (api.Module, error) {

	ctx1 := context.Background()
	runtime := wazero.NewRuntime(ctx1)
	// 编译WASM模块
	compiled, err := runtime.CompileModule(ctx1, wasmCode)
	if err != nil {
		return nil, fmt.Errorf("编译WebAssembly模块失败: %w", err)
	}
	// 创建导入对象
	builder := runtime.NewHostModuleBuilder("env")

	// 添加内存
	builder.NewFunctionBuilder().
		WithFunc(func() uint32 {
			return 0
		}).Export("memory")

	// 添加宿主函数
	builder.NewFunctionBuilder().
		WithParameterNames("funcID", "argPtr", "argLen", "bufferPtr").
		WithResultNames("result").
		WithFunc(func(_ context.Context, m api.Module, funcID, argPtr, argLen, bufferPtr uint32) int32 {
			fmt.Printf("call_host_set: %d, %d, %d, %d\n", funcID, argPtr, argLen, bufferPtr)
			// 读取参数数据
			mem := m.Memory()
			if mem == nil {
				return 0
			}
			argData, ok := mem.Read(argPtr, argLen)
			if !ok || len(argData) != int(argLen) {
				return 0
			}

			return ctx.handleHostSet(m, funcID, argData)
		}).
		Export("call_host_set")

	builder.NewFunctionBuilder().
		WithParameterNames("funcID", "argPtr", "argLen", "buffer").
		WithResultNames("result").
		WithFunc(func(_ context.Context, m api.Module, funcID, argPtr, argLen, buffer uint32) int32 {
			fmt.Printf("call_host_get_buffer: %d, %d, %d, %d\n", funcID, argPtr, argLen, buffer)
			// 读取参数数据
			mem := m.Memory()
			if mem == nil {
				return 0
			}
			argData, ok := mem.Read(argPtr, argLen)
			if !ok || len(argData) != int(argLen) {
				return 0
			}

			return ctx.handleHostGetBuffer(m, funcID, argData, buffer)
		}).
		Export("call_host_get_buffer")

	builder.NewFunctionBuilder().
		WithResultNames("result").
		WithFunc(func(_ context.Context, _ api.Module) uint32 {
			return uint32(vm.blockHeight)
		}).
		Export("get_block_height")

	builder.NewFunctionBuilder().
		WithResultNames("result").
		WithFunc(func(_ context.Context, _ api.Module) uint32 {
			return uint32(vm.blockTime)
		}).
		Export("get_block_time")

	builder.NewFunctionBuilder().
		WithParameterNames("addrPtr").
		WithResultNames("result").
		WithFunc(func(_ context.Context, m api.Module, addrPtr uint32) uint32 {
			// 读取地址
			mem := m.Memory()
			if mem == nil {
				return 0
			}
			addrData, ok := mem.Read(addrPtr, 20)
			if !ok || len(addrData) != 20 {
				return 0
			}

			var addr types.Address
			copy(addr[:], addrData)

			// 获取余额
			return uint32(vm.balances[addr])
		}).
		Export("get_balance")

	// 实例化导入对象
	envModule, err := builder.Instantiate(vm.ctx)
	if err != nil {
		return nil, fmt.Errorf("实例化导入对象失败: %w", err)
	}
	vm.envModule = envModule

	// 初始化WASI
	wasi_snapshot_preview1.MustInstantiate(vm.ctx, runtime)

	// 创建模块配置，使用合约地址作为模块名称的一部分
	moduleName := fmt.Sprintf("contract_%x", sender)
	config := wazero.NewModuleConfig().
		WithName(moduleName).WithStdout(os.Stdout).WithStderr(os.Stderr)

	// 实例化模块
	module, err := runtime.InstantiateModule(ctx1, compiled, config.WithStartFunctions("_initialize"))
	if err != nil {
		fmt.Printf("实例化模块失败: %v\n", err)
		return nil, fmt.Errorf("实例化模块失败: %w", err)
	}
	return module, nil
}

// ExecuteContract 执行已部署的合约函数
func (vm *WazeroVM) ExecuteContract(contractAddr types.Address, sender types.Address, functionName string, params []byte) (interface{}, error) {
	// 检查合约是否存在
	vm.contractsLock.RLock()
	wasmCode, exists := vm.contracts[contractAddr]
	vm.contractsLock.RUnlock()

	if !exists {
		return nil, fmt.Errorf("合约不存在: %x", contractAddr)
	}
	ctx := &wazeroContext{
		vm:           vm,
		contractAddr: contractAddr,
		sender:       sender,
	}

	module, err := vm.initContract(ctx, wasmCode, sender)
	if err != nil {
		return types.Address{}, fmt.Errorf("实例化WebAssembly模块失败: %w", err)
	}

	// 获取内存
	memory := module.ExportedMemory("memory")
	if memory == nil {
		return nil, fmt.Errorf("无法获取内存")
	}
	ctx.memory = memory

	result, err := vm.callWasmFunction(ctx, module, functionName, params)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// generateContractAddress 生成合约地址
func (vm *WazeroVM) generateContractAddress(code []byte, sender types.Address) types.Address {
	var addr types.Address
	// 简化版实现，实际应使用哈希算法
	if len(code) >= 10 {
		copy(addr[:10], code[:10])
	}
	copy(addr[10:], sender[:10])
	return addr
}

// callWasmFunction 调用WASM函数
func (vm *WazeroVM) callWasmFunction(ctx *wazeroContext, module api.Module, functionName string, params []byte) (interface{}, error) {
	fmt.Printf("调用合约函数:%s, %v\n", functionName, params)

	// 检查是否导出了allocate和deallocate函数
	allocate := module.ExportedFunction("allocate")
	if allocate == nil {
		return nil, fmt.Errorf("没有allocate函数")
	}

	processDataFunc := module.ExportedFunction("handle_contract_call")
	if processDataFunc == nil {
		return nil, fmt.Errorf("handle_contract_call没找到")
	}

	var input types.HandleContractCallParams
	input.Contract = ctx.contractAddr
	input.Function = functionName
	input.Args = params
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("handle_contract_call 序列化失败: %w", err)
	}

	// 分配内存并写入参数
	result, err := allocate.Call(vm.ctx, uint64(len(inputBytes)))
	if err != nil {
		return nil, fmt.Errorf("内存分配失败: %w", err)
	}
	inputAddr := uint32(result[0])

	// 写入参数数据
	if !ctx.memory.Write(inputAddr, inputBytes) {
		return nil, fmt.Errorf("写入内存失败")
	}

	// 调用处理函数
	result, err = processDataFunc.Call(vm.ctx, uint64(inputAddr), uint64(len(inputBytes)))
	if err != nil {
		return nil, fmt.Errorf("执行%s失败: %w", functionName, err)
	}

	resultLen := uint32(result[0])
	if resultLen > 0 {
		getBufferAddress := module.ExportedFunction("get_buffer_address")
		if getBufferAddress == nil {
			return nil, fmt.Errorf("没有get_buffer_address函数")
		}

		result, err = getBufferAddress.Call(vm.ctx)
		if err != nil {
			return nil, fmt.Errorf("get_buffer_address失败: %w", err)
		}
		bufferPtr := uint32(result[0])

		// 读取结果数据
		data, ok := ctx.memory.Read(bufferPtr, resultLen)
		if !ok {
			return nil, fmt.Errorf("读取内存失败")
		}
		fmt.Printf("result: %s\n", string(data))
	}

	// 释放内存
	deallocate := module.ExportedFunction("deallocate")
	if deallocate == nil {
		return nil, fmt.Errorf("没有deallocate函数")
	}
	_, err = deallocate.Call(vm.ctx, uint64(inputAddr), uint64(len(inputBytes)))
	if err != nil {
		return nil, fmt.Errorf("释放内存失败: %w", err)
	}

	fmt.Printf("执行结束:%s, %v\n", functionName, resultLen)
	return resultLen, nil
}

// Transfer 转账操作
func (ctx *wazeroContext) Transfer(from types.Address, to types.Address, amount uint64) error {
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
func (ctx *wazeroContext) CreateObject() core.Object {
	// 创建对象ID，简化版使用随机数
	id := ctx.generateObjectID()

	// 创建对象存储
	ctx.vm.objects[id] = make(map[string]interface{})
	ctx.vm.objectOwner[id] = ctx.sender
	ctx.vm.objectContract[id] = ctx.contractAddr

	// 返回对象封装
	return &wazeroVmObject{
		vm: ctx.vm,
		id: id,
	}
}

// generateObjectID 生成一个新的对象ID
func (ctx *wazeroContext) generateObjectID() core.ObjectID {
	// 简化实现，实际应使用哈希或随机数
	var id core.ObjectID
	// 使用合约地址的前16字节和当前时间的后16字节
	copy(id[:16], ctx.contractAddr[:16])
	// 后16字节随机生成
	// 实际实现应使用安全的随机数生成
	return id
}

// GetObject 获取指定对象
func (ctx *wazeroContext) GetObject(id core.ObjectID) (core.Object, error) {
	_, exists := ctx.vm.objects[id]
	if !exists {
		return nil, errors.New("对象不存在")
	}

	return &wazeroVmObject{
		vm: ctx.vm,
		id: id,
	}, nil
}

// GetObjectWithOwner 按所有者获取对象
func (ctx *wazeroContext) GetObjectWithOwner(contract, owner types.Address) (core.Object, error) {
	// 简化版，实际应该维护所有者到对象的索引
	for id, objOwner := range ctx.vm.objectOwner {
		if objOwner == owner {
			return &wazeroVmObject{
				vm: ctx.vm,
				id: id,
			}, nil
		}
	}
	return nil, errors.New("未找到对象")
}

// DeleteObject 删除对象
func (ctx *wazeroContext) DeleteObject(id core.ObjectID) {
	delete(ctx.vm.objects, id)
	delete(ctx.vm.objectOwner, id)
	delete(ctx.vm.objectContract, id)
}

// Call 跨合约调用
func (ctx *wazeroContext) Call(contract types.Address, function string, args ...interface{}) ([]byte, error) {
	// 简化版，实际应实现完整的跨合约调用逻辑
	return nil, errors.New("未实现跨合约调用")
}

// Log 记录事件
func (ctx *wazeroContext) Log(eventName string, keyValues ...interface{}) {
	// 简化版，仅打印日志
	fmt.Printf("合约日志: %s, 合约: %x, 参数: %v\n", eventName, ctx.contractAddr, keyValues)
}

// wazeroVmObject 实现了对象接口
type wazeroVmObject struct {
	vm *WazeroVM
	id core.ObjectID
}

// ID 获取对象ID
func (o *wazeroVmObject) ID() core.ObjectID {
	return o.id
}

// Owner 获取对象所有者
func (o *wazeroVmObject) Owner() types.Address {
	return o.vm.objectOwner[o.id]
}

// Contract 获取对象所属合约
func (o *wazeroVmObject) Contract() types.Address {
	return o.vm.objectContract[o.id]
}

// SetOwner 设置对象所有者
func (o *wazeroVmObject) SetOwner(addr types.Address) {
	o.vm.objectOwner[o.id] = addr
}

// Get 获取字段值
func (o *wazeroVmObject) Get(field string, value interface{}) error {
	obj, exists := o.vm.objects[o.id]
	if !exists {
		fmt.Printf("obj.get 对象不存在: %x\n", o.id)
		return errors.New("对象不存在")
	}

	fieldValue, exists := obj[field]
	if !exists {
		fmt.Printf("obj.get 字段不存在: %s\n", field)
		return errors.New("字段不存在")
	}
	d, _ := json.Marshal(fieldValue)
	err := json.Unmarshal(d, value)
	if err != nil {
		fmt.Printf("obj.get 反序列化失败: %v\n", err)
		return err
	}

	// 这里简化处理，实际应根据类型正确转换
	fmt.Printf("obj.get 获取字段: %s = %v\n", field, fieldValue)
	return nil
}

// Set 设置字段值
func (o *wazeroVmObject) Set(field string, value interface{}) error {
	obj, exists := o.vm.objects[o.id]
	if !exists {
		fmt.Printf("obj.set 对象不存在: %x\n", o.id)
		return errors.New("对象不存在")
	}
	fmt.Printf("obj.set 设置字段: %s = %v\n", field, value)

	obj[field] = value
	return nil
}

// 宿主函数处理器
func (ctx *wazeroContext) handleHostSet(_ api.Module, funcID uint32, argData []byte) int32 {
	// 根据函数ID处理不同的操作
	switch types.WasmFunctionID(funcID) {
	case types.FuncTransfer:
		var params types.TransferParams
		if err := json.Unmarshal(argData, &params); err != nil {
			return -1
		}
		err := ctx.Transfer(params.From, params.To, params.Amount)
		if err != nil {
			return -1
		}
		return 0

	case types.FuncDeleteObject:
		var params types.DeleteObjectParams
		if err := json.Unmarshal(argData, &params); err != nil {
			return -1
		}
		if params.ID == (types.ObjectID{}) {
			copy(params.ID[:], params.Contract[:])
		}
		obj, err := ctx.GetObject(params.ID)
		if err != nil {
			return -1
		}
		if obj.Contract() != params.Contract {
			return -1
		}
		ctx.DeleteObject(params.ID)
		return 0

	case types.FuncLog:
		fmt.Printf("[WASM]日志: %s\n", string(argData))
		return 0

	case types.FuncSetObjectOwner:
		var params types.SetOwnerParams
		if err := json.Unmarshal(argData, &params); err != nil {
			return -1
		}
		if params.ID == (types.ObjectID{}) {
			copy(params.ID[:], params.Contract[:])
		}
		obj, err := ctx.GetObject(params.ID)
		if err != nil {
			return -1
		}
		if obj.Contract() != params.Contract {
			return -1
		}
		if obj.Owner() != ctx.sender && obj.Owner() != params.Contract && obj.Owner() != params.Sender {
			return -1
		}
		obj.SetOwner(params.Owner)
		return 0

	case types.FuncSetObjectField:
		var params types.SetObjectFieldParams
		if err := json.Unmarshal(argData, &params); err != nil {
			return -1
		}
		if params.ID == (types.ObjectID{}) {
			copy(params.ID[:], params.Contract[:])
		}
		obj, err := ctx.GetObject(params.ID)
		if err != nil {
			return -1
		}
		if obj.Contract() != params.Contract {
			return -1
		}
		if obj.Owner() != ctx.sender && obj.Owner() != params.Contract && obj.Owner() != params.Sender {
			return -1
		}
		obj.Set(params.Field, params.Value)
		return 0

	default:
		return -1
	}
}

func (ctx *wazeroContext) handleHostGetBuffer(m api.Module, funcID uint32, argData []byte, offset uint32) int32 {
	mem := m.Memory()
	if mem == nil {
		return -1
	}
	// 根据函数ID处理不同的操作
	switch types.WasmFunctionID(funcID) {
	case types.FuncGetSender:
		mem.Write(offset, ctx.sender[:])
		return int32(len(ctx.sender))

	case types.FuncGetContractAddress:
		mem.Write(offset, ctx.contractAddr[:])
		return int32(len(ctx.contractAddr))

	case types.FuncCreateObject:
		obj := ctx.CreateObject()
		id := obj.ID()
		mem.Write(offset, id[:])
		return int32(len(id))

	case types.FuncGetObjectField:
		var params types.GetObjectFieldParams
		if err := json.Unmarshal(argData, &params); err != nil {
			fmt.Printf("obj.getfield 反序列化失败: %v\n", err)
			return -1
		}
		if params.ID == (types.ObjectID{}) {
			copy(params.ID[:], params.Contract[:])
		}
		obj, err := ctx.GetObject(params.ID)
		if err != nil {
			fmt.Printf("obj.getfield 获取对象失败:id:%x, %v\n", params.ID, err)
			return -1
		}
		var data any
		err = obj.Get(params.Field, &data)
		if err != nil {
			fmt.Printf("obj.getfield 获取字段失败:id:%x, %v\n", params.ID, err)
			return -1
		}
		d, _ := json.Marshal(data)
		mem.Write(offset, d)
		fmt.Printf("obj.getfield 获取字段成功:id:%x, %s\n", params.ID, string(d))
		return int32(len(d))

	case types.FuncGetObject:
		var params types.GetObjectParams
		if err := json.Unmarshal(argData, &params); err != nil {
			return -1
		}
		if params.ID == (types.ObjectID{}) {
			copy(params.ID[:], params.Contract[:])
		}
		obj, err := ctx.GetObject(params.ID)
		if err != nil {
			return -1
		}
		if obj.Contract() != params.Contract {
			return -1
		}
		id := obj.ID()
		mem.Write(offset, id[:])
		return int32(len(id))

	case types.FuncGetObjectWithOwner:
		var params types.GetObjectWithOwnerParams
		if err := json.Unmarshal(argData, &params); err != nil {
			return -1
		}
		obj, err := ctx.GetObjectWithOwner(params.Contract, params.Owner)
		if err != nil {
			return -1
		}
		id := obj.ID()
		mem.Write(offset, id[:])
		return int32(len(id))

	case types.FuncGetObjectOwner:
		if len(argData) != 32 {
			return -1
		}
		var objID types.ObjectID
		copy(objID[:], argData)
		obj, err := ctx.GetObject(objID)
		if err != nil {
			return -1
		}
		owner := obj.Owner()
		mem.Write(offset, owner[:])
		return int32(len(owner))

	default:
		return -1
	}
}
