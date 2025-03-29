package vm

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/types"
)

// defaultBlockchainContext 实现了默认的区块链上下文
type defaultBlockchainContext struct {
	// 区块信息
	blockHeight uint64
	blockTime   int64

	// 账户余额
	balances map[types.Address]uint64

	// 虚拟机对象存储
	objects        map[core.ObjectID]map[string]any
	objectOwner    map[core.ObjectID]types.Address
	objectContract map[core.ObjectID]types.Address

	// 当前执行上下文
	contractAddr types.Address
	sender       types.Address
	txHash       core.Hash
	nonce        uint64
}

// NewDefaultBlockchainContext 创建一个新的简单区块链上下文
func NewDefaultBlockchainContext() *defaultBlockchainContext {
	return &defaultBlockchainContext{
		blockHeight:    1,
		blockTime:      2,
		balances:       make(map[Address]uint64),
		objects:        make(map[core.ObjectID]map[string]any),
		objectOwner:    make(map[core.ObjectID]Address),
		objectContract: make(map[core.ObjectID]Address),
	}
}

// SetExecutionContext 设置当前执行上下文
func (ctx *defaultBlockchainContext) SetExecutionContext(contractAddr, sender types.Address) {
	ctx.contractAddr = contractAddr
	ctx.sender = sender
}

func (ctx *defaultBlockchainContext) WithTransaction(txHash core.Hash) types.BlockchainContext {
	ctx.txHash = txHash
	return ctx
}

func (ctx *defaultBlockchainContext) WithBlock(height uint64, time int64) types.BlockchainContext {
	ctx.blockHeight = height
	ctx.blockTime = time
	return ctx
}

// BlockHeight 获取当前区块高度
func (ctx *defaultBlockchainContext) BlockHeight() uint64 {
	return ctx.blockHeight
}

// BlockTime 获取当前区块时间戳
func (ctx *defaultBlockchainContext) BlockTime() int64 {
	return ctx.blockTime
}

// ContractAddress 获取当前合约地址
func (ctx *defaultBlockchainContext) ContractAddress() types.Address {
	return ctx.contractAddr
}

// TransactionHash 获取当前交易哈希
func (ctx *defaultBlockchainContext) TransactionHash() core.Hash {
	return core.Hash{} // 简化实现
}

// Sender 获取交易发送者
func (ctx *defaultBlockchainContext) Sender() types.Address {
	return ctx.sender
}

// Balance 获取账户余额
func (ctx *defaultBlockchainContext) Balance(addr types.Address) uint64 {
	return ctx.balances[addr]
}

// Transfer 转账操作
func (ctx *defaultBlockchainContext) Transfer(from, to types.Address, amount uint64) error {
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
func (ctx *defaultBlockchainContext) CreateObject(contract types.Address) (types.VMObject, error) {
	// 创建对象ID，简化版使用随机数
	id := ctx.generateObjectID(contract, ctx.sender)

	// 创建对象存储
	ctx.objects[id] = make(map[string]any)
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
func (ctx *defaultBlockchainContext) CreateObjectWithID(contract types.Address, id types.ObjectID) (types.VMObject, error) {
	// 创建对象存储
	ctx.objects[id] = make(map[string]any)
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
func (ctx *defaultBlockchainContext) generateObjectID(contract types.Address, sender types.Address) core.ObjectID {
	ctx.nonce++
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%s-%d", contract, sender, ctx.txHash, ctx.nonce)))
	var id core.ObjectID
	copy(id[:], hash[:])
	return id
}

// GetObject 获取指定对象
func (ctx *defaultBlockchainContext) GetObject(contract types.Address, id core.ObjectID) (types.VMObject, error) {
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
func (ctx *defaultBlockchainContext) GetObjectWithOwner(contract, owner types.Address) (types.VMObject, error) {
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
func (ctx *defaultBlockchainContext) DeleteObject(contract types.Address, id core.ObjectID) error {
	delete(ctx.objects, id)
	delete(ctx.objectOwner, id)
	delete(ctx.objectContract, id)
	return nil
}

// Call 跨合约调用
func (ctx *defaultBlockchainContext) Call(caller types.Address, contract types.Address, function string, args ...any) ([]byte, error) {
	return nil, errors.New("未实现跨合约调用")
}

// Log 记录事件
func (ctx *defaultBlockchainContext) Log(contract types.Address, eventName string, keyValues ...any) {
	params := []any{
		"contract", contract,
		"event", eventName,
	}
	params = append(params, keyValues...)
	slog.Info("Contract log", params...)
}

// vmObject 实现了对象接口
type vmObject struct {
	objects        map[core.ObjectID]map[string]any
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

	var fieldValue any
	if err := json.Unmarshal(value, &fieldValue); err != nil {
		return fmt.Errorf("反序列化失败: %w", err)
	}

	obj[field] = fieldValue
	return nil
}
