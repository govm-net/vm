// Package counter 实现一个简单的计数器合约示例
package counter

import (
	"errors"

	"github.com/govm-net/vm/core"
)

// Initialize 初始化计数器合约，创建一个计数器对象并设置初始值
func Initialize(ctx core.Context, initialValue uint64) (core.ObjectID, error) {
	// 创建计数器对象
	counterObj := ctx.CreateObject()

	// 设置计数器初始值
	if err := counterObj.Set("value", initialValue); err != nil {
		panic(err)
	}

	// 设置对象所有者为合约创建者
	counterObj.SetOwner(ctx.Sender())

	// 记录初始化事件
	ctx.Log("CounterInitialized", "initial_value", initialValue, "creator", ctx.Sender())

	// 返回计数器对象ID
	return counterObj.ID(), nil
}

// GetValue 获取当前计数器的值
func GetValue(ctx core.Context, counterID core.ObjectID) (uint64, error) {
	// 获取计数器对象
	counterObj, err := ctx.GetObject(counterID)
	if err != nil {
		return 0, errors.New("counter not found")
	}

	// 获取计数器值
	var value uint64
	if err := counterObj.Get("value", &value); err != nil {
		return 0, errors.New("failed to get counter value")
	}

	return value, nil
}

// Increment 增加计数器的值
func Increment(ctx core.Context, counterID core.ObjectID, amount uint64) error {
	// 获取计数器对象
	counterObj, err := ctx.GetObject(counterID)
	if err != nil {
		return errors.New("counter not found")
	}

	// 检查调用者是否为对象所有者
	if counterObj.Owner() != ctx.Sender() {
		return errors.New("only the owner can increment the counter")
	}

	// 获取当前值
	var currentValue uint64
	if err := counterObj.Get("value", &currentValue); err != nil {
		return errors.New("failed to get counter value")
	}

	// 增加值
	newValue := currentValue + amount
	if err := counterObj.Set("value", newValue); err != nil {
		return errors.New("failed to update counter value")
	}

	// 记录事件
	ctx.Log("CounterIncremented", "counter_id", counterID, "old_value", currentValue, "new_value", newValue, "amount", amount)

	return nil
}

// Reset 重置计数器的值为0
func Reset(ctx core.Context, counterID core.ObjectID) error {
	// 获取计数器对象
	counterObj, err := ctx.GetObject(counterID)
	if err != nil {
		return errors.New("counter not found")
	}

	// 检查调用者是否为对象所有者
	if counterObj.Owner() != ctx.Sender() {
		return errors.New("only the owner can reset the counter")
	}

	// 重置计数器值为0
	if err := counterObj.Set("value", uint64(0)); err != nil {
		return errors.New("failed to reset counter value")
	}

	// 记录事件
	ctx.Log("CounterReset", "counter_id", counterID)

	return nil
}
