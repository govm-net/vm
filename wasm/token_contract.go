// 基于wasm包装层的简单令牌合约示例
package main

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
	TokenOwnerKey       = "owner"        // 令牌所有者

	// 余额对象前缀
	BalancePrefix = "balance:" // 余额对象前缀
)

// 初始化令牌合约
func InitializeToken(ctx *Context, name string, symbol string, decimals uint8, totalSupply uint64) core.ObjectID {
	// 获取默认Object（空ObjectID）
	defaultObj, err := ctx.GetObject(ObjectID{})
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return core.ObjectID{}
	}

	// 存储令牌基本信息
	err = defaultObj.Set(TokenNameKey, name)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("存储令牌名称失败: %v", err))
		return core.ObjectID{}
	}

	err = defaultObj.Set(TokenSymbolKey, symbol)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("存储令牌符号失败: %v", err))
		return core.ObjectID{}
	}

	err = defaultObj.Set(TokenDecimalsKey, decimals)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("存储令牌小数位失败: %v", err))
		return core.ObjectID{}
	}

	err = defaultObj.Set(TokenTotalSupplyKey, totalSupply)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("存储总供应量失败: %v", err))
		return core.ObjectID{}
	}

	// 存储令牌所有者（部署者）
	owner := ctx.Sender()
	err = defaultObj.Set(TokenOwnerKey, owner)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("存储令牌所有者失败: %v", err))
		return core.ObjectID{}
	}

	// 创建所有者余额对象
	err = setBalance(ctx, owner, totalSupply)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("初始化所有者余额失败: %v", err))
		return core.ObjectID{}
	}

	// 记录初始化事件
	ctx.Log("token_initialize",
		"name", name,
		"symbol", symbol,
		"decimals", decimals,
		"total_supply", totalSupply,
		"owner", owner)

	return defaultObj.ID()
}

// 获取令牌信息
func GetTokenInfo(ctx *Context) (string, string, uint8, uint64) {
	// 获取默认Object
	defaultObj, err := ctx.GetObject(ObjectID{})
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
func GetOwner(ctx *Context) Address {
	// 获取默认Object
	defaultObj, err := ctx.GetObject(ObjectID{})
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return ZeroAddress
	}

	// 读取所有者地址
	var owner Address
	err = defaultObj.Get(TokenOwnerKey, &owner)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取令牌所有者失败: %v", err))
		return ZeroAddress
	}

	return owner
}

// 获取账户余额
func BalanceOf(ctx *Context, owner Address) uint64 {
	balance, _ := getBalance(ctx, owner)
	return balance
}

// 转账令牌给其他地址
func Transfer(ctx *Context, to Address, amount uint64) bool {
	from := ctx.Sender()

	// 检查金额有效性
	if amount == 0 {
		ctx.Log("error", "message", "转账金额必须大于0")
		return false
	}

	// 获取发送者余额
	fromBalance, err := getBalance(ctx, from)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取发送者余额失败: %v", err))
		return false
	}

	// 检查余额充足
	if fromBalance < amount {
		ctx.Log("error", "message", "余额不足")
		return false
	}

	// 更新发送者余额
	err = setBalance(ctx, from, fromBalance-amount)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("更新发送者余额失败: %v", err))
		return false
	}

	// 获取接收者余额
	toBalance, _ := getBalance(ctx, to)

	// 更新接收者余额
	err = setBalance(ctx, to, toBalance+amount)
	if err != nil {
		// 如果更新接收者余额失败，恢复发送者余额
		setBalance(ctx, from, fromBalance)
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
func Mint(ctx *Context, to Address, amount uint64) bool {
	sender := ctx.Sender()

	// 获取令牌所有者
	owner := GetOwner(ctx)

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

	// 获取默认Object
	defaultObj, err := ctx.GetObject(ObjectID{})
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
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
	toBalance, _ := getBalance(ctx, to)

	// 更新接收者余额
	err = setBalance(ctx, to, toBalance+amount)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("更新接收者余额失败: %v", err))
		return false
	}

	// 更新总供应量
	newTotalSupply := totalSupply + amount
	err = defaultObj.Set(TokenTotalSupplyKey, newTotalSupply)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("更新总供应量失败: %v", err))

		// 回滚余额变更
		setBalance(ctx, to, toBalance)
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
func Burn(ctx *Context, amount uint64) bool {
	sender := ctx.Sender()

	// 检查金额有效性
	if amount == 0 {
		ctx.Log("error", "message", "销毁金额必须大于0")
		return false
	}

	// 获取发送者余额
	senderBalance, err := getBalance(ctx, sender)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取发送者余额失败: %v", err))
		return false
	}

	// 检查余额充足
	if senderBalance < amount {
		ctx.Log("error", "message", "余额不足")
		return false
	}

	// 更新发送者余额
	err = setBalance(ctx, sender, senderBalance-amount)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("更新发送者余额失败: %v", err))
		return false
	}

	// 获取默认Object
	defaultObj, err := ctx.GetObject(ObjectID{})
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))

		// 回滚余额变更
		setBalance(ctx, sender, senderBalance)
		return false
	}

	// 获取当前总供应量
	var totalSupply uint64
	err = defaultObj.Get(TokenTotalSupplyKey, &totalSupply)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取总供应量失败: %v", err))

		// 回滚余额变更
		setBalance(ctx, sender, senderBalance)
		return false
	}

	// 更新总供应量
	newTotalSupply := totalSupply - amount
	err = defaultObj.Set(TokenTotalSupplyKey, newTotalSupply)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("更新总供应量失败: %v", err))

		// 回滚余额变更
		setBalance(ctx, sender, senderBalance)
		return false
	}

	// 记录销毁事件
	ctx.Log("burn",
		"from", sender,
		"amount", amount,
		"total_supply", newTotalSupply)

	return true
}

// 辅助函数 - 获取账户余额
func getBalance(ctx *Context, owner Address) (uint64, error) {
	// 尝试获取余额对象
	obj, err := ctx.GetObjectWithOwner(owner)
	if err != nil {
		// 如果对象不存在，返回0余额
		return 0, nil
	}

	// 获取余额值
	var balance uint64
	err = obj.Get("amount", &balance)
	if err != nil {
		// 如果获取失败，返回0余额
		return 0, nil
	}

	return balance, nil
}

// 辅助函数 - 设置账户余额
func setBalance(ctx *Context, owner Address, amount uint64) error {
	// 尝试获取余额对象
	obj, err := ctx.GetObjectWithOwner(owner)
	if err != nil {
		// 如果对象不存在，创建新对象
		obj = ctx.CreateObject()
		// 设置对象所有者
		obj.SetOwner(owner)
	}

	// 设置余额值
	err = obj.Set("amount", amount)
	if err != nil {
		return fmt.Errorf("设置余额失败: %w", err)
	}

	return nil
}
