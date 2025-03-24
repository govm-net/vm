//go:build test
// +build test

package main

import (
	"unsafe"
)

// 为测试提供主机函数的实现
// 这些函数在测试中会替换contract.go中声明但未实现的函数

// call_host_set 的测试实现
//
//export call_host_set
func call_host_set(funcID, argPtr, argLen int32) int64 {
	// 记录调用
	lastHostFuncID = funcID
	lastHostArgPtr = argPtr
	lastHostArgLen = argLen

	// 模拟不同函数的行为
	switch funcID {
	case FuncTransfer:
		// 模拟转账成功
		return 0
	case FuncDeleteObject:
		// 模拟删除对象成功
		return 0
	case FuncSetObjectOwner:
		// 模拟设置所有者成功
		return 0
	case FuncSetObjectField:
		// 模拟设置字段成功
		return 0
	case FuncLog:
		// 模拟日志记录成功
		return 0
	default:
		// 其他函数默认成功
		return 1
	}
}

// call_host_get_buffer 的测试实现
//
//export call_host_get_buffer
func call_host_get_buffer(funcID, argPtr, argLen int32) int32 {
	// 记录调用
	lastHostFuncID = funcID
	lastHostArgPtr = argPtr
	lastHostArgLen = argLen

	// 读取参数数据（如果有）
	var argData []byte
	if argPtr != 0 && argLen > 0 {
		argData = testReadMemory(argPtr, argLen)
	}

	// 模拟不同函数的行为
	switch funcID {
	case FuncGetSender:
		// 写入发送者地址到缓冲区
		copy(mockHostBuffer, mockSender[:])
		return int32(len(mockSender))

	case FuncGetContractAddress:
		// 写入合约地址到缓冲区
		copy(mockHostBuffer, mockContractAddr[:])
		return int32(len(mockContractAddr))

	case FuncGetObject:
		// 确保参数是对象ID
		if len(argData) == len(mockObjectID) {
			// 写入对象ID到缓冲区，表示找到对象
			copy(mockHostBuffer, mockObjectID[:])
			return int32(len(mockObjectID))
		}
		return 0 // 没有找到对象

	case FuncGetObjectWithOwner:
		// 确保参数是地址
		if len(argData) == len(mockSender) {
			// 写入对象ID到缓冲区，表示找到对象
			copy(mockHostBuffer, mockObjectID[:])
			return int32(len(mockObjectID))
		}
		return 0 // 没有找到对象

	case FuncCreateObject:
		// 写入新对象ID到缓冲区
		copy(mockHostBuffer, mockObjectID[:])
		return int32(len(mockObjectID))

	case FuncGetObjectOwner:
		// 确保参数是对象ID
		if len(argData) == len(mockObjectID) {
			// 写入所有者地址到缓冲区
			copy(mockHostBuffer, mockObjectOwner[:])
			return int32(len(mockObjectOwner))
		}
		return 0 // 没有找到所有者

	case FuncGetObjectField:
		// 模拟字段值
		fieldValue := []byte(`{"value": 123}`)
		copy(mockHostBuffer, fieldValue)
		return int32(len(fieldValue))

	case FuncCall:
		// 模拟合约调用结果
		copy(mockHostBuffer, mockContractResult)
		return int32(len(mockContractResult))

	default:
		// 其他函数返回空
		return 0
	}
}

// get_block_height 的测试实现
//
//export get_block_height
func get_block_height() int64 {
	return mockBlockHeight
}

// get_block_time 的测试实现
//
//export get_block_time
func get_block_time() int64 {
	return mockBlockTime
}

// get_balance 的测试实现
//
//export get_balance
func get_balance(addrPtr int32) uint64 {
	// 不检查地址，总是返回相同的余额
	return mockBalance
}

// 测试用读取内存的帮助函数
func testReadMemory(ptr int32, length int32) []byte {
	if ptr == 0 || length == 0 {
		return nil
	}

	data := make([]byte, length)
	src := unsafe.Pointer(uintptr(ptr))
	copy(data, (*[1 << 30]byte)(src)[:length:length])
	return data
}
