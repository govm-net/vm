// Package core 定义了智能合约与VM系统交互所需的核心接口
// 合约开发者只需了解并使用此文件中的接口即可编写智能合约
package core

import "encoding/hex"

// Address 表示区块链上的地址
type Address [20]byte

// ObjectID 表示状态对象的唯一标识符
type ObjectID [32]byte

type Hash [32]byte

var ZeroAddress = Address{}
var ZeroObjectID = ObjectID{}
var ZeroHash = Hash{}

func (id ObjectID) String() string {
	return hex.EncodeToString(id[:])
}

func IDFromString(str string) ObjectID {
	id, err := hex.DecodeString(str)
	if err != nil {
		return ZeroObjectID
	}
	return ObjectID(id)
}

func (addr Address) String() string {
	return hex.EncodeToString(addr[:])
}

func AddressFromString(str string) Address {
	addr, err := hex.DecodeString(str)
	if err != nil {
		return ZeroAddress
	}
	return Address(addr)
}

func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

func HashFromString(str string) Hash {
	h, err := hex.DecodeString(str)
	if err != nil {
		return ZeroHash
	}
	return Hash(h)
}

// Context 是合约与区块链环境交互的主要接口
type Context interface {
	// 区块链信息相关
	BlockHeight() uint64      // 获取当前区块高度
	BlockTime() int64         // 获取当前区块时间戳
	ContractAddress() Address // 获取当前合约地址

	// 账户操作相关
	Sender() Address                          // 获取交易发送者或调用合约
	Balance(addr Address) uint64              // 获取账户余额
	Transfer(to Address, amount uint64) error // 转账操作

	// 对象存储相关 - 基础状态操作使用panic而非返回error
	CreateObject() Object                             // 创建新对象，失败时panic
	GetObject(id ObjectID) (Object, error)            // 获取指定对象，可能返回error
	GetObjectWithOwner(owner Address) (Object, error) // 按所有者获取对象，可能返回error
	DeleteObject(id ObjectID)                         // 删除对象，失败时panic

	// 跨合约调用
	Call(contract Address, function string, args ...any) ([]byte, error)

	// 日志与事件
	Log(eventName string, keyValues ...interface{}) // 记录事件
}

// Object 接口用于管理区块链状态对象
type Object interface {
	ID() ObjectID          // 获取对象ID
	Owner() Address        // 获取对象所有者
	Contract() Address     // 获取对象所属合约
	SetOwner(addr Address) // 设置对象所有者，失败时panic

	// 字段操作
	Get(field string, value any) error // 获取字段值
	Set(field string, value any) error // 设置字段值
}

func Request(condition any) {
	switch v := condition.(type) {
	case bool:
		if !v {
			panic("request failed")
		}
	case error:
		if v != nil {
			panic(v)
		}
	}
}
