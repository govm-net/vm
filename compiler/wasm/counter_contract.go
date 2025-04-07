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
func Initialize() int32 {
	// 获取合约的默认Object（空ObjectID）
	defaultObj, err := core.GetObject(ObjectID{})
	if err != nil {
		core.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return -1
	}

	// 初始化计数器值为0
	err = defaultObj.Set(CounterKey, uint64(0))
	if err != nil {
		core.Log("error", "message", fmt.Sprintf("初始化失败: %v", err))
		return -1
	}

	core.Log("initialize", "contract_address", core.ContractAddress())
	return 0
}

// 增加计数器
func Increment(value uint64) uint64 {
	// 获取默认Object
	defaultObj, err := core.GetObject(ObjectID{})
	if err != nil {
		core.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return 0
	}

	// 获取当前计数器值
	var currentValue uint64
	err = defaultObj.Get(CounterKey, &currentValue)
	if err != nil {
		core.Log("error", "message", fmt.Sprintf("获取计数器值失败: %v", err))
		return 0
	}

	// 增加计数器值
	newValue := currentValue + value

	// 更新计数器值
	err = defaultObj.Set(CounterKey, newValue)
	if err != nil {
		core.Log("error", "message", fmt.Sprintf("更新计数器值失败: %v", err))
		return 0
	}

	// 记录事件
	core.Log("increment",
		"from", currentValue,
		"add", value,
		"to", newValue,
		"sender", core.Sender())

	return newValue
}

// 获取计数器当前值
func GetCounter() uint64 {
	// 获取默认Object
	defaultObj, err := core.GetObject(ObjectID{})
	if err != nil {
		core.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return 0
	}

	// 获取当前计数器值
	var currentValue uint64
	err = defaultObj.Get(CounterKey, &currentValue)
	if err != nil {
		core.Log("error", "message", fmt.Sprintf("获取计数器值失败: %v", err))
		return 0
	}

	return currentValue
}

// 重置计数器值为0
func Reset() {
	// 检查调用者是否为合约所有者
	if core.Sender() != core.ContractAddress() {
		core.Log("error", "message", "无权限重置计数器")
		return
	}

	// 获取默认Object
	defaultObj, err := core.GetObject(ObjectID{})
	if err != nil {
		core.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return
	}

	// 重置计数器值为0
	err = defaultObj.Set(CounterKey, uint64(0))
	if err != nil {
		core.Log("error", "message", fmt.Sprintf("重置计数器值失败: %v", err))
		return
	}

	// 记录事件
	core.Log("reset", "sender", core.Sender())
}

// 初始化计数器函数
func handleInitialize(params []byte) (any, error) {
	fmt.Println("handleInitialize")

	out := Initialize()

	// 记录初始化事件
	core.Log("CounterInitialized", "value", out)

	// 返回成功结果
	return out, nil
}

// 增加计数器函数
func handleIncrement(params []byte) (any, error) {
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

	newValue := Increment(uint64(incrParams.Amount))
	// 记录增加事件
	core.Log("CounterIncremented", "amount", incrParams.Amount, "new_value", newValue)

	// 返回成功结果
	return newValue, nil
}

// 获取计数器当前值
func handleGetCounter(params []byte) (any, error) {
	value := GetCounter()

	// 返回当前值
	return value, nil
}

// 重置计数器函数
func handleReset(params []byte) (any, error) {
	// 验证调用者权限
	Reset()
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
	registerContractFunction("Panic", func(params []byte) (any, error) {
		panic("test panic")
	})
}
