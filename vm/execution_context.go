// Package vm 实现了基于WebAssembly的虚拟机核心功能
package vm

import (
	"errors"
	"fmt"

	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/types"
	"github.com/wasmerio/wasmer-go/wasmer"
)

// ExecutionContext 实现了合约执行上下文，为合约提供与区块链环境交互的接口
type ExecutionContext struct {
	// 父引擎引用
	wasmEngine *WasmEngine

	// 状态管理器引用
	stateManager VMStateManager

	// 合约地址
	contractAddr core.Address

	// 交易发送者
	sender    core.Address
	senderPtr uint32

	// WebAssembly内存实例
	memory *wasmer.Memory

	// 全局缓冲区指针
	hostBufferPtr  uint32
	hostBufferSize uint32

	// 当前调用深度
	callDepth int
}

// VMStateManager 定义了合约状态管理接口
type VMStateManager interface {
	// 获取区块信息
	GetBlockHeight() uint64
	GetBlockTime() int64
	GetSender() core.Address
	GetContractAddress() core.Address
	GetTransactionHash() core.Hash

	// 获取和修改账户余额
	GetBalance(addr core.Address) uint64
	Transfer(from, to core.Address, amount uint64) error

	// 对象操作
	CreateObject() core.ObjectID
	GetObject(id core.ObjectID) (interface{}, error)
	GetObjectByOwner(owner core.Address) (interface{}, error)
	DeleteObject(id core.ObjectID) error
}

// 内存缓冲区大小常量
const (
	// 宿主缓冲区大小 - 用于在宿主和合约间传递数据
	HostBufferSize uint32 = uint32(types.HostBufferSize)

	// 最大调用深度
	MaxCallDepth = 8
)

// NewExecutionContext 创建新的合约执行上下文
func NewExecutionContext(wasmEngine *WasmEngine, stateManager VMStateManager, contractAddr core.Address, sender core.Address) *ExecutionContext {
	return &ExecutionContext{
		wasmEngine:   wasmEngine,
		stateManager: stateManager,
		contractAddr: contractAddr,
		sender:       sender,
		callDepth:    0,
	}
}

// initializeMemory 初始化WebAssembly内存和缓冲区
func (ctx *ExecutionContext) initializeMemory(instance *wasmer.Instance) error {
	// 获取内存实例
	memory, err := instance.Exports.GetMemory("memory")
	if err != nil {
		return fmt.Errorf("无法获取WebAssembly内存: %w", err)
	}
	ctx.memory = memory

	// 分配宿主缓冲区
	exports := instance.Exports
	allocateFn, err := exports.GetFunction("allocate")
	if err != nil {
		return fmt.Errorf("合约未导出内存分配函数: %w", err)
	}

	// 分配宿主缓冲区
	result, err := allocateFn(int32(HostBufferSize))
	if err != nil {
		return fmt.Errorf("内存分配失败: %w", err)
	}
	bufferPtr, ok := result.(int32)
	if !ok {
		return errors.New("内存分配返回无效结果")
	}
	ctx.hostBufferPtr = uint32(bufferPtr)
	ctx.hostBufferSize = HostBufferSize

	// 设置发送者地址
	senderPtr, err := ctx.allocateMemory(len(ctx.sender))
	if err != nil {
		return fmt.Errorf("分配发送者地址内存失败: %w", err)
	}
	ctx.writeMemory(senderPtr, ctx.sender[:])
	ctx.senderPtr = senderPtr

	// 设置合约的宿主缓冲区
	setBufferFn, err := exports.GetFunction("set_host_buffer")
	if err == nil {
		_, setErr := setBufferFn(int32(ctx.hostBufferPtr))
		if setErr != nil {
			return fmt.Errorf("设置宿主缓冲区失败: %w", setErr)
		}
	}

	return nil
}

// allocateMemory 从WebAssembly内存中分配指定大小的内存
func (ctx *ExecutionContext) allocateMemory(size int) (uint32, error) {
	if ctx.memory == nil {
		return 0, errors.New("WebAssembly内存未初始化")
	}

	// 在真实实现中，应该使用合约的内存分配器或管理内存分配
	// 这里使用简化的方法，直接分配内存
	memData := ctx.memory.Data()
	if len(memData) < int(ctx.hostBufferPtr+ctx.hostBufferSize+uint32(size)) {
		// 计算需要增长的页数
		pageSize := 65536 // WebAssembly页大小为64KB
		currentPages := len(memData) / pageSize
		requiredBytes := int(ctx.hostBufferPtr + ctx.hostBufferSize + uint32(size))
		requiredPages := (requiredBytes + pageSize - 1) / pageSize // 向上取整
		pagesToGrow := requiredPages - currentPages

		if pagesToGrow > 0 {
			// 将 uint32 转换为 wasmer.Pages
			var pages wasmer.Pages
			// 设置页数 (这里直接使用类型转换可能不安全，但我们简化处理)
			pages = wasmer.Pages(pagesToGrow)

			grown := ctx.memory.Grow(pages)
			if !grown {
				return 0, errors.New("增长内存失败")
			}
		}
	}

	// 分配地址（简化版：直接使用宿主缓冲区后的内存）
	addr := ctx.hostBufferPtr + ctx.hostBufferSize
	return addr, nil
}

// writeMemory 将数据写入WebAssembly内存
func (ctx *ExecutionContext) writeMemory(ptr uint32, data []byte) {
	if ptr == 0 || len(data) == 0 || ctx.memory == nil {
		return
	}

	// 检查内存范围
	memLen := uint32(len(ctx.memory.Data()))
	if ptr >= memLen || ptr+uint32(len(data)) > memLen {
		// 越界访问，实际应该报错
		return
	}

	// 写入数据
	copy(ctx.memory.Data()[ptr:ptr+uint32(len(data))], data)
}

// readMemory 从WebAssembly内存读取数据
func (ctx *ExecutionContext) readMemory(ptr uint32, size uint32) []byte {
	if ptr == 0 || size == 0 || ctx.memory == nil {
		return []byte{}
	}

	// 检查内存范围
	memLen := uint32(len(ctx.memory.Data()))
	if ptr >= memLen || ptr+size > memLen {
		// 越界访问，实际应该报错
		return []byte{}
	}

	// 读取数据
	data := make([]byte, size)
	copy(data, ctx.memory.Data()[ptr:ptr+size])
	return data
}

// 实现Context接口方法

// BlockHeight 获取当前区块高度
func (ctx *ExecutionContext) BlockHeight() uint64 {
	return ctx.stateManager.GetBlockHeight()
}

// BlockTime 获取当前区块时间戳
func (ctx *ExecutionContext) BlockTime() int64 {
	return ctx.stateManager.GetBlockTime()
}

// ContractAddress 获取当前合约地址
func (ctx *ExecutionContext) ContractAddress() core.Address {
	return ctx.contractAddr
}

// Sender 获取交易发送者
func (ctx *ExecutionContext) Sender() core.Address {
	return ctx.sender
}

// Balance 获取账户余额
func (ctx *ExecutionContext) Balance(addr core.Address) uint64 {
	return ctx.stateManager.GetBalance(addr)
}

// Transfer 转账操作
func (ctx *ExecutionContext) Transfer(to core.Address, amount uint64) error {
	return ctx.stateManager.Transfer(ctx.contractAddr, to, amount)
}

// CreateObject 创建新对象
func (ctx *ExecutionContext) CreateObject() core.Object {
	objectID := ctx.stateManager.CreateObject()
	// 返回对象封装
	return &StateObject{
		ctx: ctx,
		id:  objectID,
	}
}

// GetObject 获取指定对象
func (ctx *ExecutionContext) GetObject(id core.ObjectID) (core.Object, error) {
	// 检查对象是否存在
	_, err := ctx.stateManager.GetObject(id)
	if err != nil {
		return nil, err
	}

	// 返回对象封装
	return &StateObject{
		ctx: ctx,
		id:  id,
	}, nil
}

// GetObjectWithOwner 按所有者获取对象
func (ctx *ExecutionContext) GetObjectWithOwner(owner core.Address) (core.Object, error) {
	// 检查是否存在具有该所有者的对象
	_, err := ctx.stateManager.GetObjectByOwner(owner)
	if err != nil {
		return nil, err
	}

	// 这里简化处理，实际上应该返回找到的对象ID
	var id core.ObjectID
	// TODO: 从找到的对象中获取ID

	// 返回对象封装
	return &StateObject{
		ctx: ctx,
		id:  id,
	}, nil
}

// DeleteObject 删除对象
func (ctx *ExecutionContext) DeleteObject(id core.ObjectID) {
	_ = ctx.stateManager.DeleteObject(id)
}

// Call 跨合约调用
func (ctx *ExecutionContext) Call(contract core.Address, function string, args ...interface{}) ([]byte, error) {
	// 检查调用深度
	if ctx.callDepth >= MaxCallDepth {
		return nil, errors.New("超出最大调用深度")
	}

	// TODO: 实现跨合约调用
	return nil, errors.New("暂未实现跨合约调用")
}

// Log 记录事件
func (ctx *ExecutionContext) Log(eventName string, keyValues ...interface{}) {
	// TODO: 实现事件记录
	fmt.Printf("合约日志: %s, 参数: %v\n", eventName, keyValues)
}

// StateObject 实现了状态对象接口
type StateObject struct {
	ctx *ExecutionContext
	id  core.ObjectID
}

// ID 获取对象ID
func (o *StateObject) ID() core.ObjectID {
	return o.id
}

// Owner 获取对象所有者
func (o *StateObject) Owner() core.Address {
	// 从状态管理器获取对象所有者
	// 简化实现，实际应该从对象中获取
	var owner core.Address
	// TODO: 从对象中获取所有者
	return owner
}

// Contract 获取对象所属合约
func (o *StateObject) Contract() core.Address {
	// 从状态管理器获取对象所属合约
	// 简化实现，实际应该从对象中获取
	var contract core.Address
	// TODO: 从对象中获取合约
	return contract
}

// SetOwner 设置对象所有者
func (o *StateObject) SetOwner(addr core.Address) {
	// 更新对象所有者
	// TODO: 实现设置所有者
}

// Get 获取字段值
func (o *StateObject) Get(field string, value interface{}) error {
	// 从对象获取字段值
	// TODO: 实现获取字段值
	return nil
}

// Set 设置字段值
func (o *StateObject) Set(field string, value interface{}) error {
	// 更新对象字段值
	// TODO: 实现设置字段值
	return nil
}
