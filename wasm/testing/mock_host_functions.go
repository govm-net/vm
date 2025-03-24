// Package testing provides mock implementations for WebAssembly host functions
package testing

import (
	"unsafe"
)

// 声明需要从主包导入的常量和类型
type Address [20]byte
type ObjectID [32]byte

// 函数ID常量
const (
	FuncGetSender          = int32(1)
	FuncGetBlockHeight     = int32(2)
	FuncGetBlockTime       = int32(3)
	FuncGetContractAddress = int32(4)
	FuncGetBalance         = int32(5)
	FuncTransfer           = int32(6)
	FuncCreateObject       = int32(7)
	FuncCall               = int32(8)
	FuncGetObject          = int32(9)
	FuncGetObjectWithOwner = int32(10)
	FuncDeleteObject       = int32(11)
	FuncLog                = int32(12)
	FuncGetObjectOwner     = int32(13)
	FuncSetObjectOwner     = int32(14)
	FuncGetObjectField     = int32(15)
	FuncSetObjectField     = int32(16)
	FuncDbRead             = int32(17)
	FuncDbWrite            = int32(18)
	FuncDbDelete           = int32(19)
	FuncSetHostBuffer      = int32(20)
)

// 模拟宿主函数的全局变量
var (
	// 模拟主机缓冲区
	MockHostBuffer []byte

	// 记录最后一次调用的宿主函数ID和参数
	LastHostFuncID   int32
	LastHostArgPtr   int32
	LastHostArgLen   int32
	LastHostRetValue int64

	// 主机缓冲区地址 - 需要从主包传入
	HostBufferPtr int32

	// 模拟状态数据
	MockSender         = Address{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	MockContractAddr   = Address{20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	MockBlockHeight    = int64(12345)
	MockBlockTime      = int64(1647312000)
	MockBalance        = uint64(1000000)
	MockObjectID       = ObjectID{1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 5, 6, 6, 6, 6, 7, 7, 7, 7, 8, 8, 8, 8}
	MockObjectOwner    = Address{5, 5, 5, 5, 6, 6, 6, 6, 7, 7, 7, 7, 8, 8, 8, 8, 9, 9, 9, 9}
	MockContractResult = []byte(`{"result": "success"}`)
)

// 初始化函数
func Init(hostBufferSize int32) {
	// 创建模拟主机缓冲区
	MockHostBuffer = make([]byte, hostBufferSize)
	// 设置主机缓冲区地址
	HostBufferPtr = int32(uintptr(unsafe.Pointer(&MockHostBuffer[0])))
}

// 测试用读取内存的帮助函数
func TestReadMemory(ptr int32, length int32) []byte {
	if ptr == 0 || length == 0 {
		return nil
	}

	data := make([]byte, length)
	src := unsafe.Pointer(uintptr(ptr))
	copy(data, (*[1 << 30]byte)(src)[:length:length])
	return data
}

// CallHostSet 的测试实现 - 可以从主包中调用
func CallHostSet(funcID, argPtr, argLen int32) int64 {
	// 记录调用
	LastHostFuncID = funcID
	LastHostArgPtr = argPtr
	LastHostArgLen = argLen

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

// CallHostGetBuffer 的测试实现 - 可以从主包中调用
func CallHostGetBuffer(funcID, argPtr, argLen int32) int32 {
	// 记录调用
	LastHostFuncID = funcID
	LastHostArgPtr = argPtr
	LastHostArgLen = argLen

	// 读取参数数据（如果有）
	var argData []byte
	if argPtr != 0 && argLen > 0 {
		argData = TestReadMemory(argPtr, argLen)
	}

	// 模拟不同函数的行为
	switch funcID {
	case FuncGetSender:
		// 写入发送者地址到缓冲区
		copy(MockHostBuffer, MockSender[:])
		return int32(len(MockSender))

	case FuncGetContractAddress:
		// 写入合约地址到缓冲区
		copy(MockHostBuffer, MockContractAddr[:])
		return int32(len(MockContractAddr))

	case FuncGetObject:
		// 确保参数是对象ID
		if len(argData) == len(MockObjectID) {
			// 写入对象ID到缓冲区，表示找到对象
			copy(MockHostBuffer, MockObjectID[:])
			return int32(len(MockObjectID))
		}
		return 0 // 没有找到对象

	case FuncGetObjectWithOwner:
		// 确保参数是地址
		if len(argData) == len(MockSender) {
			// 写入对象ID到缓冲区，表示找到对象
			copy(MockHostBuffer, MockObjectID[:])
			return int32(len(MockObjectID))
		}
		return 0 // 没有找到对象

	case FuncCreateObject:
		// 写入新对象ID到缓冲区
		copy(MockHostBuffer, MockObjectID[:])
		return int32(len(MockObjectID))

	case FuncGetObjectOwner:
		// 确保参数是对象ID
		if len(argData) == len(MockObjectID) {
			// 写入所有者地址到缓冲区
			copy(MockHostBuffer, MockObjectOwner[:])
			return int32(len(MockObjectOwner))
		}
		return 0 // 没有找到所有者

	case FuncGetObjectField:
		// 模拟字段值
		fieldValue := []byte(`{"value": 123}`)
		copy(MockHostBuffer, fieldValue)
		return int32(len(fieldValue))

	case FuncCall:
		// 模拟合约调用结果
		copy(MockHostBuffer, MockContractResult)
		return int32(len(MockContractResult))

	default:
		// 其他函数返回空
		return 0
	}
}

// GetBlockHeight 的测试实现 - 可以从主包中调用
func GetBlockHeight() int64 {
	return MockBlockHeight
}

// GetBlockTime 的测试实现 - 可以从主包中调用
func GetBlockTime() int64 {
	return MockBlockTime
}

// GetBalance 的测试实现 - 可以从主包中调用
func GetBalance(addrPtr int32) uint64 {
	// 不检查地址，总是返回相同的余额
	return MockBalance
}
