// 基于wasm包装层的简单计数器合约示例
package testdata

import (
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
	defaultObj, err := core.GetObject(core.ObjectID{})
	core.Assert(err)

	// 初始化计数器值为0
	err = defaultObj.Set(CounterKey, uint64(0))
	core.Assert(err)

	core.Log("initialize", "contract_address", core.ContractAddress())
	return 0
}

// 增加计数器
func Increment(value uint64) uint64 {
	// 获取默认Object
	defaultObj, err := core.GetObject(core.ObjectID{})
	core.Assert(err)

	// 获取当前计数器值
	var currentValue uint64
	err = defaultObj.Get(CounterKey, &currentValue)
	core.Assert(err)

	// 增加计数器值
	newValue := currentValue + value

	// 更新计数器值
	err = defaultObj.Set(CounterKey, newValue)
	core.Assert(err)

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
	defaultObj, err := core.GetObject(core.ObjectID{})
	core.Assert(err)

	// 获取当前计数器值
	var currentValue uint64
	err = defaultObj.Get(CounterKey, &currentValue)
	core.Assert(err)

	return currentValue
}

// 重置计数器值为0
func Reset() {
	// 检查调用者是否为合约所有者
	if core.Sender() != core.ContractAddress() {
		return
	}

	// 获取默认Object
	defaultObj, err := core.GetObject(core.ObjectID{})
	core.Assert(err)

	// 重置计数器值为0
	err = defaultObj.Set(CounterKey, uint64(0))
	core.Assert(err)

	// 记录事件
	core.Log("reset", "sender", core.Sender())
}
