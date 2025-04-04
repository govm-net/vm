// 基于wasm包装层的简单计数器合约示例
package main

import (
	"encoding/json"
	"fmt"

	"github.com/govm-net/vm/core"
)

// 计数器合约的状态键
const (
	CounterKey = "counter_value"
)

// 初始化合约
// 此函数是大写开头的，因此会被自动导出并在合约部署时调用
func Initialize(ctx core.Context) int32 {
	// 获取合约的默认Object（空ObjectID）
	defaultObj, err := ctx.GetObject(ObjectID{})
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return -1
	}

	// 初始化计数器值为0
	err = defaultObj.Set(CounterKey, uint64(0))
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("初始化失败: %v", err))
		return -1
	}

	ctx.Log("initialize", "contract_address", ctx.ContractAddress())
	return 0
}

// 增加计数器
func Increment(ctx core.Context, value uint64) uint64 {
	// 获取默认Object
	defaultObj, err := ctx.GetObject(ObjectID{})
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return 0
	}

	// 获取当前计数器值
	var currentValue uint64
	err = defaultObj.Get(CounterKey, &currentValue)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取计数器值失败: %v", err))
		return 0
	}

	// 增加计数器值
	newValue := currentValue + value

	// 更新计数器值
	err = defaultObj.Set(CounterKey, newValue)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("更新计数器值失败: %v", err))
		return 0
	}

	// 记录事件
	ctx.Log("increment",
		"from", currentValue,
		"add", value,
		"to", newValue,
		"sender", ctx.Sender())

	return newValue
}

// 获取计数器当前值
func GetCounter(ctx core.Context) uint64 {
	// 获取默认Object
	defaultObj, err := ctx.GetObject(ObjectID{})
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return 0
	}

	// 获取当前计数器值
	var currentValue uint64
	err = defaultObj.Get(CounterKey, &currentValue)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取计数器值失败: %v", err))
		return 0
	}

	return currentValue
}

// 重置计数器值为0
func Reset(ctx core.Context) {
	// 检查调用者是否为合约所有者
	if ctx.Sender() != ctx.ContractAddress() {
		ctx.Log("error", "message", "无权限重置计数器")
		return
	}

	// 获取默认Object
	defaultObj, err := ctx.GetObject(ObjectID{})
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return
	}

	// 重置计数器值为0
	err = defaultObj.Set(CounterKey, uint64(0))
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("重置计数器值失败: %v", err))
		return
	}

	// 记录事件
	ctx.Log("reset", "sender", ctx.Sender())
}

// 初始化计数器函数
func handleInitialize(ctx core.Context, params []byte) (any, error) {
	fmt.Println("handleInitialize")

	out := Initialize(ctx)

	// 记录初始化事件
	ctx.Log("CounterInitialized", "value", out)

	// 返回成功结果
	return out, nil
}

// 增加计数器函数
func handleIncrement(ctx core.Context, params []byte) (any, error) {
	// 解析参数
	var incrParams struct {
		Amount int64 `json:"amount"`
	}
	fmt.Printf("handleIncrement params: %s\n", string(params))

	if len(params) > 0 {
		if err := json.Unmarshal(params, &incrParams); err != nil {
			return nil, fmt.Errorf("invalid increment parameters: %w", err)
		}
	} else {
		// 默认增加1
		incrParams.Amount = 1
	}

	newValue := Increment(ctx, uint64(incrParams.Amount))
	// 记录增加事件
	ctx.Log("CounterIncremented", "amount", incrParams.Amount, "new_value", newValue)

	// 返回成功结果
	return newValue, nil
}

// 获取计数器当前值
func handleGetCounter(ctx core.Context, params []byte) (any, error) {
	value := GetCounter(ctx)

	// 返回当前值
	return value, nil
}

// 重置计数器函数
func handleReset(ctx core.Context, params []byte) (any, error) {
	// 验证调用者权限
	Reset(ctx)
	// 返回成功结果
	return nil, nil
}

// 注册合约函数
func init() {
	// 注册计数器合约的函数处理器
	registerContractFunction("Initialize", handleInitialize)
	registerContractFunction("Increment", handleIncrement)
	registerContractFunction("GetCounter", handleGetCounter)
	registerContractFunction("Reset", handleReset)
	registerContractFunction("Panic", func(ctx core.Context, params []byte) (any, error) {
		panic("test panic")
	})
}
