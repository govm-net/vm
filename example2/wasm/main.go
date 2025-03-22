package main

import (
	"encoding/binary"
	"unsafe"
)

// 导入的函数声明 - 用于获取自定义数据
//
//export get_config_value
func get_config_value(keyPtr, keyLen, valuePtr, valueLen uint32) uint32

// 导入的函数声明 - 用于记录日志
//
//export log_message
func log_message(msgPtr, msgLen uint32)

// 导入的函数声明 - 用于调用宿主函数
//
//export call_host_function
func call_host_function(funcID, argPtr, argLen uint32) uint64

// 辅助函数 - 将字符串转换为内存指针和长度
func stringToPtr(s string) (uint32, uint32) {
	bytes := []byte(s)
	ptr := &bytes[0]
	return uint32(uintptr(unsafe.Pointer(ptr))), uint32(len(bytes))
}

// 辅助函数 - 记录日志
func logMessage(message string) {
	ptr, len := stringToPtr(message)
	log_message(ptr, len)
}

// 辅助函数 - 获取配置值
func getConfigValue(key string) (string, bool) {
	// 分配内存用于存储值
	valueBuffer := make([]byte, 1024)

	// 获取键的指针和长度
	keyPtr, keyLen := stringToPtr(key)

	// 调用导入函数获取值
	result := get_config_value(keyPtr, keyLen, uint32(uintptr(unsafe.Pointer(&valueBuffer[0]))), 1024)

	// 检查结果
	if result == 0 {
		// 找到值，确定实际长度
		var length int
		for i, b := range valueBuffer {
			if b == 0 {
				length = i
				break
			}
		}
		return string(valueBuffer[:length]), true
	}

	return "", false
}

// 导出的函数 - 处理自定义数据
//
//export process_config
func process_config() uint32 {
	// 获取配置值
	dbName, ok := getConfigValue("database_name")
	if !ok {
		logMessage("无法获取数据库名称")
		return 1
	}

	apiKey, ok := getConfigValue("api_key")
	if !ok {
		logMessage("无法获取 API 密钥")
		return 2
	}

	// 记录获取到的配置
	logMessage("已获取配置: 数据库=" + dbName + ", API密钥=" + apiKey)

	// 调用宿主函数
	funcID := uint32(1) // 假设 ID 1 是某个特定的宿主函数
	argData := "处理配置: " + dbName
	argPtr, argLen := stringToPtr(argData)
	result := call_host_function(funcID, argPtr, argLen)

	// 将 uint64 结果转换为两个 uint32 值
	resultHigh := uint32(result >> 32)
	resultLow := uint32(result)

	logMessage("宿主函数返回: " + string(binary.LittleEndian.AppendUint32(nil, resultHigh)) +
		"," + string(binary.LittleEndian.AppendUint32(nil, resultLow)))

	return 0
}

// 导出的函数 - 处理内存中的数据
//
//export process_memory_data
func process_memory_data(dataPtr, dataLen uint32) uint32 {
	// 从 WebAssembly 内存中读取数据
	data := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(dataPtr))), dataLen)

	// 简单处理：计算所有字节的总和
	var sum uint32
	for _, b := range data {
		sum += uint32(b)
	}

	logMessage("处理了 " + string(binary.LittleEndian.AppendUint32(nil, dataLen)) + " 字节的数据，总和为 " +
		string(binary.LittleEndian.AppendUint32(nil, sum)))

	return sum
}

// 导出的函数 - 内存分配
//
//export allocate
func allocate(size uint32) uint32 {
	// 分配指定大小的内存
	buffer := make([]byte, size)
	// 返回内存的指针
	return uint32(uintptr(unsafe.Pointer(&buffer[0])))
}

// 导出的函数 - 内存释放
//
//export deallocate
func deallocate(ptr uint32, size uint32) {
	// 在WebAssembly中，内存管理由Go的垃圾收集器处理
	// 这个函数主要是为了提供与主机环境匹配的接口
	// 实际上不需要做任何事情，Go的GC会自动处理
}

// 主函数 - TinyGo 需要但不会被调用
func main() {}
