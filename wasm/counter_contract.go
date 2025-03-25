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
func Initialize(ctx *Context) int32 {
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
func Increment(ctx *Context, value uint64) uint64 {
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
func GetCounter(ctx *Context) uint64 {
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
func Reset(ctx *Context) {
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
func handleInitialize(ctx *Context, params []byte) (interface{}, error) {
	fmt.Println("handleInitialize")

	out := Initialize(ctx)

	// 记录初始化事件
	ctx.Log("CounterInitialized", "value", out)

	// 返回成功结果
	return map[string]interface{}{
		"status": "success",
		"value":  out,
	}, nil
}

// 增加计数器函数
func handleIncrement(ctx *Context, params []byte) (interface{}, error) {
	// 解析参数
	var incrParams struct {
		Amount int64 `json:"amount"`
	}

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
	return map[string]interface{}{
		"status":    "success",
		"amount":    incrParams.Amount,
		"new_value": newValue,
	}, nil
}

// 获取计数器当前值
func handleGetCounter(ctx *Context, params []byte) (interface{}, error) {
	// 获取计数器对象
	obj, err := getOrCreateCounterObject(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get counter object: %w", err)
	}

	// 获取当前值
	var currentValue int64
	if err := obj.Get(CounterKey, &currentValue); err != nil {
		// 如果未找到值，初始化为0
		currentValue = 0
	}

	// 返回当前值
	return map[string]interface{}{
		"status": "success",
		"value":  currentValue,
	}, nil
}

// 重置计数器函数
func handleReset(ctx *Context, params []byte) (interface{}, error) {
	// 验证调用者权限
	if ctx.Sender() != ctx.ContractAddress() {
		return nil, fmt.Errorf("permission denied: only contract owner can reset counter")
	}

	// 解析参数
	var resetParams struct {
		Value int64 `json:"value"`
	}

	if len(params) > 0 {
		if err := json.Unmarshal(params, &resetParams); err != nil {
			return nil, fmt.Errorf("invalid reset parameters: %w", err)
		}
	} else {
		// 默认重置为0
		resetParams.Value = 0
	}

	// 获取计数器对象
	obj, err := getOrCreateCounterObject(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get counter object: %w", err)
	}

	// 获取旧值（用于返回）
	var oldValue int64
	if err := obj.Get(CounterKey, &oldValue); err != nil {
		// 如果未找到值，视为0
		oldValue = 0
	}

	// 设置新值
	if err := obj.Set(CounterKey, resetParams.Value); err != nil {
		return nil, fmt.Errorf("failed to reset counter value: %w", err)
	}

	// 记录重置事件
	ctx.Log("CounterReset", "old_value", oldValue, "new_value", resetParams.Value)

	// 返回成功结果
	return map[string]interface{}{
		"status":    "success",
		"old_value": oldValue,
		"new_value": resetParams.Value,
	}, nil
}

// 辅助函数 - 获取或创建计数器对象
func getOrCreateCounterObject(ctx *Context) (core.Object, error) {
	// 尝试获取现有对象
	contractAddr := ctx.ContractAddress()
	obj, err := ctx.GetObjectWithOwner(contractAddr)
	if err == nil {
		return obj, nil
	}

	// 如果对象不存在，创建新对象
	obj = ctx.CreateObject()

	// 设置对象所有者为合约自身，确保只有合约可以修改
	obj.SetOwner(contractAddr)

	return obj, nil
}

// 注册合约函数
func init() {
	// 注册计数器合约的函数处理器
	registerContractFunction("Initialize", handleInitialize)
	registerContractFunction("Increment", handleIncrement)
	registerContractFunction("GetCounter", handleGetCounter)
	registerContractFunction("Reset", handleReset)
}
