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
	TokenBalancePrefix  = "balance_"     // 余额字段前缀
	TokenOwnerKey       = "owner"        // 令牌所有者
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

	// 存储令牌所有者
	err = defaultObj.Set(TokenOwnerKey, ctx.Sender())
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("存储令牌所有者失败: %v", err))
		return core.ZeroObjectID
	}

	// 初始化所有者余额
	err = defaultObj.Set(TokenBalancePrefix+ctx.Sender().String(), totalSupply)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("初始化所有者余额失败: %v", err))
		return core.ZeroObjectID
	}

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

	// 获取令牌所有者
	var owner core.Address
	err = defaultObj.Get(TokenOwnerKey, &owner)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取令牌所有者失败: %v", err))
		return core.ZeroAddress
	}

	return owner
}

// 获取账户余额
func BalanceOf(ctx core.Context, owner core.Address) uint64 {
	// 获取默认Object
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return 0
	}

	// 获取余额
	var balance uint64
	err = defaultObj.Get(TokenBalancePrefix+owner.String(), &balance)
	if err != nil {
		// 如果获取失败，返回0余额
		return 0
	}

	return balance
}

// 转账令牌给其他地址
func Transfer(ctx core.Context, to core.Address, amount uint64) bool {
	from := ctx.Sender()

	// 检查金额有效性
	if amount == 0 {
		ctx.Log("error", "message", "转账金额必须大于0")
		return false
	}

	// 获取默认Object
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return false
	}

	// 获取发送者余额
	var fromBalance uint64
	err = defaultObj.Get(TokenBalancePrefix+from.String(), &fromBalance)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取发送者余额失败: %v", err))
		return false
	}

	// 检查余额充足
	if fromBalance < amount {
		ctx.Log("error", "message", "余额不足")
		return false
	}

	// 获取接收者余额
	var toBalance uint64
	err = defaultObj.Get(TokenBalancePrefix+to.String(), &toBalance)
	if err != nil {
		// 如果接收者没有余额记录，从0开始
		toBalance = 0
	}

	// 更新余额
	err = defaultObj.Set(TokenBalancePrefix+from.String(), fromBalance-amount)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("更新发送者余额失败: %v", err))
		return false
	}

	err = defaultObj.Set(TokenBalancePrefix+to.String(), toBalance+amount)
	if err != nil {
		// 如果更新接收者余额失败，恢复发送者余额
		defaultObj.Set(TokenBalancePrefix+from.String(), fromBalance)
		ctx.Log("error", "message", fmt.Sprintf("更新接收者余额失败: %v", err))
		return false
	}

	// 记录转账事件
	ctx.Log("transfer",
		"from", from,
		"to", to,
		"amount", amount)

	return true
}

// 铸造新令牌（仅限所有者）
func Mint(ctx core.Context, to core.Address, amount uint64) bool {
	sender := ctx.Sender()

	// 获取默认Object
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return false
	}

	// 获取令牌所有者
	var owner core.Address
	err = defaultObj.Get(TokenOwnerKey, &owner)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取令牌所有者失败: %v", err))
		return false
	}

	// 检查是否为令牌所有者
	if sender != owner {
		ctx.Log("error", "message", "只有令牌所有者才能铸造新令牌")
		return false
	}

	// 检查金额有效性
	if amount == 0 {
		ctx.Log("error", "message", "铸造金额必须大于0")
		return false
	}

	// 获取当前总供应量
	var totalSupply uint64
	err = defaultObj.Get(TokenTotalSupplyKey, &totalSupply)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取总供应量失败: %v", err))
		return false
	}

	// 获取接收者余额
	var toBalance uint64
	err = defaultObj.Get(TokenBalancePrefix+to.String(), &toBalance)
	if err != nil {
		// 如果接收者没有余额记录，从0开始
		toBalance = 0
	}

	// 更新余额和总供应量
	err = defaultObj.Set(TokenBalancePrefix+to.String(), toBalance+amount)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("更新接收者余额失败: %v", err))
		return false
	}

	newTotalSupply := totalSupply + amount
	err = defaultObj.Set(TokenTotalSupplyKey, newTotalSupply)
	if err != nil {
		// 如果更新总供应量失败，恢复接收者余额
		defaultObj.Set(TokenBalancePrefix+to.String(), toBalance)
		ctx.Log("error", "message", fmt.Sprintf("更新总供应量失败: %v", err))
		return false
	}

	// 记录铸造事件
	ctx.Log("mint",
		"to", to,
		"amount", amount,
		"total_supply", newTotalSupply)

	return true
}

// 销毁令牌
func Burn(ctx core.Context, amount uint64) bool {
	sender := ctx.Sender()

	// 获取默认Object
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return false
	}

	// 获取令牌所有者
	var owner core.Address
	err = defaultObj.Get(TokenOwnerKey, &owner)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取令牌所有者失败: %v", err))
		return false
	}

	// 检查是否为令牌所有者
	if sender != owner {
		ctx.Log("error", "message", "只有令牌所有者才能销毁令牌")
		return false
	}

	// 检查金额有效性
	if amount == 0 {
		ctx.Log("error", "message", "销毁金额必须大于0")
		return false
	}

	// 获取发送者余额
	var senderBalance uint64
	err = defaultObj.Get(TokenBalancePrefix+sender.String(), &senderBalance)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取发送者余额失败: %v", err))
		return false
	}

	// 检查余额充足
	if senderBalance < amount {
		ctx.Log("error", "message", "余额不足")
		return false
	}

	// 获取当前总供应量
	var totalSupply uint64
	err = defaultObj.Get(TokenTotalSupplyKey, &totalSupply)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取总供应量失败: %v", err))
		return false
	}

	// 更新余额和总供应量
	err = defaultObj.Set(TokenBalancePrefix+sender.String(), senderBalance-amount)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("更新发送者余额失败: %v", err))
		return false
	}

	newTotalSupply := totalSupply - amount
	err = defaultObj.Set(TokenTotalSupplyKey, newTotalSupply)
	if err != nil {
		// 如果更新总供应量失败，恢复发送者余额
		defaultObj.Set(TokenBalancePrefix+sender.String(), senderBalance)
		ctx.Log("error", "message", fmt.Sprintf("更新总供应量失败: %v", err))
		return false
	}

	// 记录销毁事件
	ctx.Log("burn",
		"from", sender,
		"amount", amount,
		"total_supply", newTotalSupply)

	return true
}
