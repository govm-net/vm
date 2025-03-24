package main

import (
	"encoding/json"
	"fmt"
	"unsafe"
)

// 全局计数器
var counter uint64 = 0

// 合约所有者
var owner [20]byte

// 初始化函数参数
type InitializeParams struct {
	Value uint64 `json:"value"`
}

// 重置函数参数
type ResetParams struct {
	Value uint64 `json:"value"`
}

// 结果格式
type Result struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// 计数器数据
type CounterData struct {
	Value uint64 `json:"value"`
}

// 内存分配函数
//
//export allocate
func allocate(size int32) int32 {
	buffer := make([]byte, size)
	return int32(uintptr(unsafe.Pointer(&buffer[0])))
}

// 内存释放函数
//
//export deallocate
func deallocate(ptr int32, size int32) {
	// 在Go/TinyGo中，这个函数不需要实际实现，因为有GC
}

// 初始化合约
//
//export Initialize
func Initialize(paramsPtr, paramsLen int32) int32 {
	// 解析参数
	var params InitializeParams
	if paramsLen > 0 {
		paramBytes := getMemoryBytes(paramsPtr, paramsLen)
		if err := json.Unmarshal(paramBytes, &params); err != nil {
			return createErrorResult(fmt.Sprintf("Invalid parameters: %v", err))
		}
	}

	// 设置初始值
	counter = params.Value

	// 设置合约所有者为当前调用者
	getSender(unsafe.Pointer(&owner[0]), 20)

	// 记录日志
	log(fmt.Sprintf("Counter initialized to %d", counter))

	// 返回成功结果
	data, _ := json.Marshal(CounterData{Value: counter})
	return createSuccessResult(data)
}

// 增加计数器值
//
//export Increment
func Increment(paramsPtr, paramsLen int32) int32 {
	// 记录旧值
	oldValue := counter

	// 增加计数器
	counter++

	// 记录日志
	log(fmt.Sprintf("Counter incremented from %d to %d", oldValue, counter))

	// 返回成功结果
	data, _ := json.Marshal(CounterData{Value: counter})
	return createSuccessResult(data)
}

// 获取当前计数器值
//
//export GetCounter
func GetCounter(paramsPtr, paramsLen int32) int32 {
	data, _ := json.Marshal(CounterData{Value: counter})
	return createSuccessResult(data)
}

// 重置计数器
//
//export Reset
func Reset(paramsPtr, paramsLen int32) int32 {
	// 解析参数
	var params ResetParams
	if paramsLen > 0 {
		paramBytes := getMemoryBytes(paramsPtr, paramsLen)
		if err := json.Unmarshal(paramBytes, &params); err != nil {
			return createErrorResult(fmt.Sprintf("Invalid parameters: %v", err))
		}
	}

	// 检查调用者是否为合约所有者
	var sender [20]byte
	getSender(unsafe.Pointer(&sender[0]), 20)

	// 比较sender和owner
	if !bytesEqual(sender[:], owner[:]) {
		return createErrorResult("Only contract owner can reset the counter")
	}

	// 记录旧值
	oldValue := counter

	// 重置计数器
	counter = params.Value

	// 记录日志
	log(fmt.Sprintf("Counter reset from %d to %d", oldValue, counter))

	// 返回成功结果
	data, _ := json.Marshal(CounterData{Value: counter})
	return createSuccessResult(data)
}

// 字节数组比较
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// 合约调用处理函数
//
//export handle_contract_call
func handle_contract_call(funcNamePtr, funcNameLen, paramsPtr, paramsLen int32) int32 {
	// 获取函数名
	funcName := string(getMemoryBytes(funcNamePtr, funcNameLen))

	// 根据函数名调用对应的处理函数
	switch funcName {
	case "Initialize":
		return Initialize(paramsPtr, paramsLen)
	case "Increment":
		return Increment(paramsPtr, paramsLen)
	case "GetCounter":
		return GetCounter(paramsPtr, paramsLen)
	case "Reset":
		return Reset(paramsPtr, paramsLen)
	default:
		return createErrorResult(fmt.Sprintf("Unknown function: %s", funcName))
	}
}

// 从内存中读取字节数组
func getMemoryBytes(ptr int32, len int32) []byte {
	if ptr == 0 || len == 0 {
		return []byte{}
	}

	// 创建一个切片引用该内存
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), len)
}

// 主机函数: 获取调用者地址
//
//go:wasm-module env
//export call_host_get_buffer
func call_host_get_buffer(funcID int32, argPtr int32, argLen int32) int32

// 获取发送者地址
func getSender(dest unsafe.Pointer, size int32) {
	call_host_get_buffer(1, 0, 0) // 使用FuncGetSender=1

	// 从主机缓冲区复制数据到dest
	for i := int32(0); i < size; i++ {
		*(*byte)(unsafe.Add(dest, uintptr(i))) = getHostBufferByte(i)
	}
}

// 记录日志
func log(message string) {
	bytes := []byte(message)
	// 分配内存
	ptr := allocate(int32(len(bytes)))
	// 写入数据
	dest := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), len(bytes))
	copy(dest, bytes)
	// 调用主机日志函数
	call_host_set(12, ptr, int32(len(bytes))) // 使用FuncLog=12
	// 释放内存
	deallocate(ptr, int32(len(bytes)))
}

// 主机函数: 设置数据
//
//go:wasm-module env
//export call_host_set
func call_host_set(funcID int32, argPtr int32, argLen int32) int64

// 获取主机缓冲区中的字节
func getHostBufferByte(index int32) byte {
	// 直接访问主机缓冲区
	// 实际实现应该从主机缓冲区读取
	// 这里简化为返回一个伪值
	return byte(index % 255)
}

// 创建成功结果
func createSuccessResult(data []byte) int32 {
	result := Result{
		Success: true,
		Data:    data,
	}

	resultBytes, _ := json.Marshal(result)

	// 分配内存
	resultLen := int32(len(resultBytes))
	resultPtr := allocate(resultLen)

	// 写入数据
	dest := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(resultPtr))), resultLen)
	copy(dest, resultBytes)

	return resultPtr
}

// 创建错误结果
func createErrorResult(errorMsg string) int32 {
	result := Result{
		Success: false,
		Error:   errorMsg,
	}

	resultBytes, _ := json.Marshal(result)

	// 分配内存
	resultLen := int32(len(resultBytes))
	resultPtr := allocate(resultLen)

	// 写入数据
	dest := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(resultPtr))), resultLen)
	copy(dest, resultBytes)

	return resultPtr
}

func main() {
	// WebAssembly要求main函数但不会执行
}
