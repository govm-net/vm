package mock

import (
	"fmt"
	"sync"
)

var (
	mu   sync.RWMutex
	gas  int64
	used int64
)

// InitGas 初始化gas
func InitGas(initialGas int64) {
	mu.Lock()
	defer mu.Unlock()
	gas = initialGas
	used = 0
}

// GetGas 获取剩余gas
func GetGas() int64 {
	mu.RLock()
	defer mu.RUnlock()
	return gas
}

// GetUsedGas 获取已使用的gas
func GetUsedGas() int64 {
	mu.RLock()
	defer mu.RUnlock()
	return used
}

// ConsumeGas 消耗gas
func ConsumeGas(amount int64) {
	mu.Lock()
	defer mu.Unlock()

	if amount <= 0 {
		return
	}

	if gas < amount {
		panic(fmt.Sprintf("out of gas: gas=%d, need=%d", gas, amount))
	}

	gas -= amount
	used += amount
}

// RefundGas 退还gas
func RefundGas(amount int64) {
	mu.Lock()
	defer mu.Unlock()

	if amount <= 0 {
		return
	}

	if used < amount {
		panic(fmt.Sprintf("invalid refund: used=%d, refund=%d", used, amount))
	}

	gas += amount
	used -= amount
}

// ResetGas 重置gas
func ResetGas(initialGas int64) {
	mu.Lock()
	defer mu.Unlock()
	gas = initialGas
	used = 0
}
