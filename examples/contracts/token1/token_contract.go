// 基于wasm包装层的简单令牌合约示例
package token

import (
	"fmt"

	"github.com/govm-net/vm/core"
)

// 常量定义 - 对象中的字段名
const (
	// 默认Object中的字段
	TokenNameKey        = "name"         // 令牌名称
	TokenSymbolKey      = "symbol"       // 令牌符号
	TokenDecimalsKey    = "decimals"     // 令牌小数位
	TokenTotalSupplyKey = "total_supply" // 总供应量
	TokenAmountKey      = "amount"       // 余额
)

// 初始化令牌合约
func InitializeToken(ctx core.Context, name string, symbol string, decimals uint8, totalSupply uint64) core.ObjectID {
	// 获取默认Object（空ObjectID）
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return core.ZeroObjectID
	}

	// 存储令牌基本信息
	err = defaultObj.Set(TokenNameKey, name)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("存储令牌名称失败: %v", err))
		return core.ZeroObjectID
	}

	err = defaultObj.Set(TokenSymbolKey, symbol)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("存储令牌符号失败: %v", err))
		return core.ZeroObjectID
	}

	err = defaultObj.Set(TokenDecimalsKey, decimals)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("存储令牌小数位失败: %v", err))
		return core.ZeroObjectID
	}

	err = defaultObj.Set(TokenTotalSupplyKey, totalSupply)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("存储总供应量失败: %v", err))
		return core.ZeroObjectID
	}

	defaultObj.SetOwner(ctx.Sender())

	obj := ctx.CreateObject()
	err = obj.Set(TokenAmountKey, totalSupply)
	core.Request(err)
	obj.SetOwner(ctx.Sender())

	// 记录初始化事件
	ctx.Log("initialize",
		"id", defaultObj.ID(),
		"name", name,
		"symbol", symbol,
		"decimals", decimals,
		"total_supply", totalSupply,
		"owner", ctx.Sender())

	return defaultObj.ID()
}

// 获取令牌信息
func GetTokenInfo(ctx core.Context) (string, string, uint8, uint64) {
	// 获取默认Object
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return "", "", 0, 0
	}

	// 读取令牌基本信息
	var name string
	err = defaultObj.Get(TokenNameKey, &name)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取令牌名称失败: %v", err))
		return "", "", 0, 0
	}

	var symbol string
	err = defaultObj.Get(TokenSymbolKey, &symbol)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取令牌符号失败: %v", err))
		return "", "", 0, 0
	}

	var decimals uint8
	err = defaultObj.Get(TokenDecimalsKey, &decimals)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取令牌小数位失败: %v", err))
		return "", "", 0, 0
	}

	var totalSupply uint64
	err = defaultObj.Get(TokenTotalSupplyKey, &totalSupply)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取总供应量失败: %v", err))
		return "", "", 0, 0
	}

	return name, symbol, decimals, totalSupply
}

// 获取所有者
func GetOwner(ctx core.Context) core.Address {
	// 获取默认Object
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return core.ZeroAddress
	}

	return defaultObj.Owner()
}

// 获取账户余额
func BalanceOf(ctx core.Context, owner core.Address) uint64 {
	obj, err := ctx.GetObjectWithOwner(owner)
	core.Request(err)

	var balance uint64
	err = obj.Get(TokenAmountKey, &balance)
	core.Request(err)

	return balance
}

// 转账令牌给其他地址
func Transfer(ctx core.Context, to core.Address, amount uint64) bool {
	from := ctx.Sender()

	// 检查金额有效性
	core.Request(amount > 0)
	obj, err := ctx.GetObjectWithOwner(from)
	core.Request(err)

	var fromBalance uint64
	err = obj.Get(TokenAmountKey, &fromBalance)
	core.Request(err)

	// 检查余额充足
	core.Request(fromBalance >= amount)

	err = obj.Set(TokenAmountKey, fromBalance-amount)
	core.Request(err)

	toObj := ctx.CreateObject()
	err = toObj.Set(TokenAmountKey, amount)
	core.Request(err)
	toObj.SetOwner(to)

	// 记录转账事件
	ctx.Log("transfer",
		"from", from,
		"to", to,
		"amount", amount)

	return true
}

func Collect(ctx core.Context, ids []core.ObjectID) bool {
	sender := ctx.Sender()

	// 检查ids是否为空
	core.Request(len(ids) > 1)
	var amount uint64
	//将其他的object里的余额迁移到第一个object
	for _, id := range ids[1:] {
		obj, err := ctx.GetObject(id)
		core.Request(err)
		core.Request(obj.Owner() == sender)
		var balance uint64
		err = obj.Get(TokenAmountKey, &balance)
		core.Request(err)
		amount += balance
		ctx.DeleteObject(id)
	}
	obj, err := ctx.GetObject(ids[0])
	core.Request(err)
	core.Request(obj.Owner() == sender)
	err = obj.Set(TokenAmountKey, amount)
	core.Request(err)

	return true
}

// 铸造新令牌（仅限所有者）
func Mint(ctx core.Context, to core.Address, amount uint64) bool {
	sender := ctx.Sender()

	// 检查金额有效性
	core.Request(amount > 0)

	// 获取默认Object
	obj, err := ctx.GetObject(core.ZeroObjectID)
	core.Request(err)
	core.Request(obj.Owner() == sender)

	// 获取当前总供应量
	var totalSupply uint64
	err = obj.Get(TokenTotalSupplyKey, &totalSupply)
	core.Request(err)

	totalSupply += amount
	err = obj.Set(TokenTotalSupplyKey, totalSupply)
	core.Request(err)

	toObj := ctx.CreateObject()
	err = toObj.Set(TokenAmountKey, amount)
	core.Request(err)
	toObj.SetOwner(to)

	// 记录铸造事件
	ctx.Log("mint",
		"to", to,
		"amount", amount,
		"total_supply", totalSupply)

	return true
}

// 销毁令牌
func Burn(ctx core.Context, id core.ObjectID, amount uint64) bool {
	sender := ctx.Sender()

	core.Request(amount > 0)

	// 获取默认Object
	obj, err := ctx.GetObject(core.ZeroObjectID)
	core.Request(err)
	core.Request(obj.Owner() == sender)

	// 获取当前总供应量
	var totalSupply uint64
	err = obj.Get(TokenTotalSupplyKey, &totalSupply)
	core.Request(err)

	totalSupply -= amount
	err = obj.Set(TokenTotalSupplyKey, totalSupply)
	core.Request(err)

	var userObj core.Object
	if id == core.ZeroObjectID {
		userObj, err = ctx.GetObjectWithOwner(sender)
		core.Request(err)
	} else {
		userObj, err = ctx.GetObject(id)
		core.Request(err)
	}

	var userBalance uint64
	err = userObj.Get(TokenAmountKey, &userBalance)
	core.Request(err)

	userBalance -= amount
	err = userObj.Set(TokenAmountKey, userBalance)
	core.Request(err)

	// 记录销毁事件
	ctx.Log("burn",
		"from", sender,
		"amount", amount,
		"total_supply", totalSupply)

	return true
}
