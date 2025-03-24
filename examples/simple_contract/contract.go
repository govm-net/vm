// 简单计数器合约示例
package main

import (
	"fmt"
)

// 全局计数器
var counter uint64

// 入口函数：初始化合约
//
//export Initialize
func Initialize() {
	counter = 0
	fmt.Println("Counter initialized to 0")
}

// 增加计数器值
//
//export Increment
func Increment(value uint64) uint64 {
	counter += value
	fmt.Printf("Counter incremented by %d to %d\n", value, counter)
	return counter
}

// 获取当前计数器值
//
//export GetCounter
func GetCounter() uint64 {
	return counter
}

// 重置计数器
//
//export Reset
func Reset() {
	counter = 0
	fmt.Println("Counter reset to 0")
}

// WebAssembly要求main函数
func main() {
	// 此函数在WebAssembly中不会被执行
}
