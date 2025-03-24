//go:build test
// +build test

package main

import (
	wasmtest "github.com/govm-net/vm/wasm/testing"
)

// 初始化测试环境
func init() {
	// 初始化测试环境，设置宿主缓冲区大小
	wasmtest.Init(HostBufferSize)
}

// 桥接宿主函数到测试实现

//export call_host_set
func call_host_set(funcID, argPtr, argLen int32) int64 {
	return wasmtest.GetMockHook().CallHostSet(funcID, argPtr, argLen)
}

//export call_host_get_buffer
func call_host_get_buffer(funcID, argPtr, argLen int32) int32 {
	return wasmtest.GetMockHook().CallHostGetBuffer(funcID, argPtr, argLen)
}

//export get_block_height
func get_block_height() int64 {
	return wasmtest.GetMockHook().GetBlockHeight()
}

//export get_block_time
func get_block_time() int64 {
	return wasmtest.GetMockHook().GetBlockTime()
}

//export get_balance
func get_balance(addrPtr int32) uint64 {
	return wasmtest.GetMockHook().GetBalance(addrPtr)
}
