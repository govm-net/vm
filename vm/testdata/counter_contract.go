package countercontract

import (
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
	defaultObj, err := ctx.GetObject(core.ObjectID{})
	core.Request(err)

	// 初始化计数器值为0
	err = defaultObj.Set(CounterKey, uint64(0))
	core.Request(err)

	ctx.Log("initialize", "contract_address", ctx.ContractAddress())
	return 0
}

// 增加计数器
func Increment(ctx core.Context, value uint64) uint64 {
	// 获取默认Object
	defaultObj, err := ctx.GetObject(core.ObjectID{})
	core.Request(err)

	// 获取当前计数器值
	var currentValue uint64
	err = defaultObj.Get(CounterKey, &currentValue)
	core.Request(err)

	// 增加计数器值
	newValue := currentValue + value

	// 更新计数器值
	err = defaultObj.Set(CounterKey, newValue)
	core.Request(err)

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
	defaultObj, err := ctx.GetObject(core.ObjectID{})
	core.Request(err)

	// 获取当前计数器值
	var currentValue uint64
	err = defaultObj.Get(CounterKey, &currentValue)
	core.Request(err)

	return currentValue
}

// 重置计数器值为0
func Reset(ctx core.Context) {
	// 检查调用者是否为合约所有者
	if ctx.Sender() != ctx.ContractAddress() {
		return
	}

	// 获取默认Object
	defaultObj, err := ctx.GetObject(core.ObjectID{})
	core.Request(err)

	// 重置计数器值为0
	err = defaultObj.Set(CounterKey, uint64(0))
	core.Request(err)

	// 记录事件
	ctx.Log("reset", "sender", ctx.Sender())
}
