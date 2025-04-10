package memory

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"

	"github.com/govm-net/vm/context"
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
	objects        map[core.ObjectID]map[string][]byte
	objectOwner    map[core.ObjectID]core.Address
	objectContract map[core.ObjectID]core.Address

	// 当前执行上下文
	contractAddr types.Address
	sender       types.Address
	txHash       core.Hash
	nonce        uint64
	gasLimit     int64
}

func init() {
	context.Register(context.MemoryContextType, NewBlockchainContext)
}

// NewDefaultBlockchainContext 创建一个新的简单区块链上下文
func NewBlockchainContext(params map[string]any) types.BlockchainContext {
	return &defaultBlockchainContext{
		blockHeight:    1,
		blockTime:      2,
		balances:       make(map[core.Address]uint64),
		objects:        make(map[core.ObjectID]map[string][]byte),
		objectOwner:    make(map[core.ObjectID]core.Address),
		objectContract: make(map[core.ObjectID]core.Address),
		gasLimit:       10000000,
	}
}

func (ctx *defaultBlockchainContext) SetBlockInfo(height uint64, time int64, hash core.Hash) error {
	ctx.blockHeight = height
	ctx.blockTime = time
	ctx.txHash = hash
	return nil
}

func (ctx *defaultBlockchainContext) SetTransactionInfo(hash core.Hash, from types.Address, to types.Address, value uint64) error {
	ctx.txHash = hash
	ctx.sender = from
	ctx.contractAddr = to
	// ctx.value = value
	return nil
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

func (ctx *defaultBlockchainContext) SetGasLimit(limit int64) {
	ctx.gasLimit = limit
}

func (ctx *defaultBlockchainContext) GetGas() int64 {
	return ctx.gasLimit
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
	ctx.objects[id] = make(map[string][]byte)
	ctx.objectOwner[id] = ctx.Sender()
	ctx.objectContract[id] = contract

	// 返回对象封装
	return &vmObject{
		ctx:         ctx,
		objOwner:    ctx.Sender(),
		objContract: contract,
		id:          id,
	}, nil
}

// CreateObject 创建新对象
func (ctx *defaultBlockchainContext) CreateObjectWithID(contract types.Address, id types.ObjectID) (types.VMObject, error) {
	// 创建对象存储
	ctx.objects[id] = make(map[string][]byte)
	ctx.objectOwner[id] = contract
	ctx.objectContract[id] = contract

	// 返回对象封装
	return &vmObject{
		ctx:         ctx,
		objOwner:    contract,
		objContract: contract,
		id:          id,
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
		ctx:         ctx,
		objOwner:    ctx.objectOwner[id],
		objContract: ctx.objectContract[id],
		id:          id,
	}, nil
}

// GetObjectWithOwner 按所有者获取对象
func (ctx *defaultBlockchainContext) GetObjectWithOwner(contract, owner types.Address) (types.VMObject, error) {
	for id, objOwner := range ctx.objectOwner {
		if objOwner == owner {
			return &vmObject{
				ctx:         ctx,
				objOwner:    objOwner,
				objContract: ctx.objectContract[id],
				id:          id,
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
	return nil, errors.New("not implemented")
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

func (ctx *defaultBlockchainContext) setObjectField(id core.ObjectID, field string, value []byte) {
	obj, exists := ctx.objects[id]
	if !exists {
		obj = make(map[string][]byte)
	}
	obj[field] = value
	ctx.objects[id] = obj
}

func (ctx *defaultBlockchainContext) getObjectField(id core.ObjectID, field string) []byte {
	obj, exists := ctx.objects[id]
	if !exists {
		return nil
	}
	return obj[field]
}

// vmObject 实现了对象接口
type vmObject struct {
	ctx         *defaultBlockchainContext
	objOwner    types.Address
	objContract types.Address
	id          core.ObjectID
}

// ID 获取对象ID
func (o *vmObject) ID() core.ObjectID {
	return o.id
}

// Owner 获取对象所有者
func (o *vmObject) Owner() types.Address {
	return o.objOwner
}

// Contract 获取对象所属合约
func (o *vmObject) Contract() types.Address {
	return o.objContract
}

// SetOwner 设置对象所有者
func (o *vmObject) SetOwner(contract, sender types.Address, addr types.Address) error {
	if contract != o.objContract {
		return fmt.Errorf("invalid contract")
	}
	if sender != o.objOwner && contract != o.objOwner {
		return fmt.Errorf("not owner")
	}
	o.objOwner = addr
	o.ctx.objectOwner[o.id] = addr
	return nil
}

// Get 获取字段值
func (o *vmObject) Get(contract types.Address, field string) ([]byte, error) {
	if contract != o.objContract {
		return nil, fmt.Errorf("invalid contract")
	}
	fieldValue := o.ctx.getObjectField(o.id, field)
	if fieldValue == nil {
		return nil, errors.New("字段不存在")
	}

	return fieldValue, nil
}

// Set 设置字段值
func (o *vmObject) Set(contract types.Address, sender types.Address, field string, value []byte) error {
	if contract != o.objContract {
		return fmt.Errorf("invalid contract")
	}
	if sender != o.objOwner && contract != o.objOwner {
		return fmt.Errorf("not owner")
	}
	o.ctx.setObjectField(o.id, field, value)
	return nil
}
