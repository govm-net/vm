// Package token 实现一个简单的代币合约示例
package token

import (
	"bytes"
	"errors"

	"github.com/govm-net/vm/core"
)

// 构建辅助函数，生成与Address相关的ObjectID
func getBalanceObjectID(owner core.Address) core.ObjectID {
	// 创建一个新的ObjectID
	var id core.ObjectID

	// 填充前20字节为地址
	copy(id[:20], owner[:])

	// 填充后12字节为"balance"标识
	copy(id[20:], []byte("balance"))

	return id
}

// 构建辅助函数，生成授权对象ID
func getApprovalObjectID(owner, spender core.Address) core.ObjectID {
	// 创建一个新的ObjectID
	var id core.ObjectID

	// 填充前20字节为owner地址
	copy(id[:20], owner[:])

	// 填充中间的字节为spender地址(最多12字节)
	if len(spender) > 12 {
		copy(id[20:], spender[:12])
	} else {
		copy(id[20:], spender[:])
	}

	return id
}

// 检查地址是否为零地址
func isZeroAddress(addr core.Address) bool {
	return bytes.Equal(addr[:], make([]byte, len(addr)))
}

// Initialize 初始化代币合约，创建代币信息对象和创建者的余额对象
func Initialize(ctx core.Context, name string, symbol string, totalSupply uint64) (core.ObjectID, error) {
	// 创建代币信息对象
	infoObj := ctx.CreateObject()

	// 设置代币基本信息
	infoObj.Set("name", name)
	infoObj.Set("symbol", symbol)
	infoObj.Set("total_supply", totalSupply)
	infoObj.Set("decimals", uint8(18)) // 设置默认精度

	// 设置代币信息对象所有者为合约自身地址
	infoObj.SetOwner(ctx.ContractAddress())

	// 创建发行者的余额对象
	creatorBalanceObj := ctx.CreateObject()
	creatorBalanceObj.Set("balance", totalSupply)
	creatorBalanceObj.SetOwner(ctx.Sender())

	// 记录初始化事件
	ctx.Log("TokenInitialized",
		"name", name,
		"symbol", symbol,
		"total_supply", totalSupply,
		"creator", ctx.Sender())

	// 返回代币信息对象ID
	return infoObj.ID(), nil
}

// BalanceOf 查询指定地址的代币余额
func BalanceOf(ctx core.Context, owner core.Address) (uint64, error) {
	// 获取余额对象
	balanceObj, err := ctx.GetObjectWithOwner(owner)
	if err != nil {
		// 如果对象不存在，表示余额为0
		return 0, nil
	}

	// 获取余额值
	var balance uint64
	if err := balanceObj.Get("balance", &balance); err != nil {
		return 0, errors.New("failed to get balance")
	}

	return balance, nil
}

// Transfer 从发送者向接收者转移代币
func Transfer(ctx core.Context, to core.Address, amount uint64) error {
	// 检查接收者地址是否有效
	if isZeroAddress(to) {
		return errors.New("invalid recipient address")
	}

	// 检查金额是否有效
	if amount == 0 {
		return errors.New("transfer amount must be greater than 0")
	}

	// 获取发送者余额对象
	senderObj, err := ctx.GetObjectWithOwner(ctx.Sender())
	if err != nil {
		return errors.New("sender has no balance")
	}

	// 获取发送者余额
	var senderBalance uint64
	if err := senderObj.Get("balance", &senderBalance); err != nil {
		return errors.New("failed to get sender balance")
	}

	// 检查余额是否足够
	if senderBalance < amount {
		return errors.New("insufficient balance")
	}

	// 更新发送者余额
	newSenderBalance := senderBalance - amount
	if err := senderObj.Set("balance", newSenderBalance); err != nil {
		return errors.New("failed to update sender balance")
	}

	// 处理接收者余额
	receiverObj, err := ctx.GetObjectWithOwner(to)
	if err != nil {
		// 如果接收者没有余额对象，创建一个
		receiverObj = ctx.CreateObject()
		receiverObj.SetOwner(to)
		receiverObj.Set("balance", amount)
	} else {
		// 更新接收者余额
		var receiverBalance uint64
		if err := receiverObj.Get("balance", &receiverBalance); err != nil {
			return errors.New("failed to get recipient balance")
		}

		// 更新接收者余额
		newReceiverBalance := receiverBalance + amount
		if err := receiverObj.Set("balance", newReceiverBalance); err != nil {
			return errors.New("failed to update recipient balance")
		}
	}

	// 记录转账事件
	ctx.Log("Transfer",
		"from", ctx.Sender(),
		"to", to,
		"amount", amount)

	return nil
}

// Approve 授权指定地址可以从发送者账户转移的代币数量
func Approve(ctx core.Context, spender core.Address, amount uint64) error {
	// 检查授权地址是否有效
	if isZeroAddress(spender) {
		return errors.New("invalid spender address")
	}

	// 创建或获取授权对象ID
	approvalID := getApprovalObjectID(ctx.Sender(), spender)
	approvalObj, err := ctx.GetObject(approvalID)
	if err != nil {
		// 创建新的授权对象
		approvalObj = ctx.CreateObject()
		approvalObj.SetOwner(ctx.Sender())
	}

	// 设置授权金额
	if err := approvalObj.Set("amount", amount); err != nil {
		return errors.New("failed to set approval amount")
	}

	// 记录授权事件
	ctx.Log("Approval",
		"owner", ctx.Sender(),
		"spender", spender,
		"amount", amount)

	return nil
}

// TransferFrom 从一个地址向另一个地址转移代币（需要预先授权）
func TransferFrom(ctx core.Context, from core.Address, to core.Address, amount uint64) error {
	// 检查地址是否有效
	if isZeroAddress(from) || isZeroAddress(to) {
		return errors.New("invalid address")
	}

	// 检查金额是否有效
	if amount == 0 {
		return errors.New("transfer amount must be greater than 0")
	}

	// 获取授权对象ID
	approvalID := getApprovalObjectID(from, ctx.Sender())
	approvalObj, err := ctx.GetObject(approvalID)
	if err != nil {
		return errors.New("no approval found")
	}

	// 获取授权金额
	var allowance uint64
	if err := approvalObj.Get("amount", &allowance); err != nil {
		return errors.New("failed to get allowance")
	}

	// 检查授权金额是否足够
	if allowance < amount {
		return errors.New("insufficient allowance")
	}

	// 获取发送者余额对象
	fromObj, err := ctx.GetObjectWithOwner(from)
	if err != nil {
		return errors.New("sender has no balance")
	}

	// 获取发送者余额
	var fromBalance uint64
	if err := fromObj.Get("balance", &fromBalance); err != nil {
		return errors.New("failed to get sender balance")
	}

	// 检查余额是否足够
	if fromBalance < amount {
		return errors.New("insufficient balance")
	}

	// 更新发送者余额
	newFromBalance := fromBalance - amount
	if err := fromObj.Set("balance", newFromBalance); err != nil {
		return errors.New("failed to update sender balance")
	}

	// 处理接收者余额
	toObj, err := ctx.GetObjectWithOwner(to)
	if err != nil {
		// 如果接收者没有余额对象，创建一个
		toObj = ctx.CreateObject()
		toObj.SetOwner(to)
		toObj.Set("balance", amount)
	} else {
		// 更新接收者余额
		var toBalance uint64
		if err := toObj.Get("balance", &toBalance); err != nil {
			return errors.New("failed to get recipient balance")
		}

		newToBalance := toBalance + amount
		if err := toObj.Set("balance", newToBalance); err != nil {
			return errors.New("failed to update recipient balance")
		}
	}

	// 更新授权金额
	newAllowance := allowance - amount
	if err := approvalObj.Set("amount", newAllowance); err != nil {
		return errors.New("failed to update allowance")
	}

	// 记录转账事件
	ctx.Log("TransferFrom",
		"from", from,
		"to", to,
		"spender", ctx.Sender(),
		"amount", amount)

	return nil
}
