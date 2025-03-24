// Package testing provides utilities for testing WebAssembly contracts
package testing

import (
	"fmt"
)

// MockHostFunctionHook 是用于测试在 contract.go 中声明的宿主函数的接口
// 这个接口允许测试代码提供模拟实现
type MockHostFunctionHook interface {
	// 常用宿主函数
	CallHostSet(funcID, argPtr, argLen int32) int64
	CallHostGetBuffer(funcID, argPtr, argLen int32) int32
	GetBlockHeight() int64
	GetBlockTime() int64
	GetBalance(addrPtr int32) uint64
}

// defaultMockHook 是默认的模拟宿主函数实现
type defaultMockHook struct{}

func (h *defaultMockHook) CallHostSet(funcID, argPtr, argLen int32) int64 {
	return CallHostSet(funcID, argPtr, argLen)
}

func (h *defaultMockHook) CallHostGetBuffer(funcID, argPtr, argLen int32) int32 {
	return CallHostGetBuffer(funcID, argPtr, argLen)
}

func (h *defaultMockHook) GetBlockHeight() int64 {
	return GetBlockHeight()
}

func (h *defaultMockHook) GetBlockTime() int64 {
	return GetBlockTime()
}

func (h *defaultMockHook) GetBalance(addrPtr int32) uint64 {
	return GetBalance(addrPtr)
}

// 全局模拟钩子实例
var currentMockHook MockHostFunctionHook = &defaultMockHook{}

// SetMockHook 设置用于测试的自定义宿主函数实现
func SetMockHook(hook MockHostFunctionHook) {
	if hook == nil {
		currentMockHook = &defaultMockHook{}
	} else {
		currentMockHook = hook
	}
}

// ResetMockHook 重置为默认实现
func ResetMockHook() {
	currentMockHook = &defaultMockHook{}
}

// GetMockHook 获取当前使用的模拟钩子
func GetMockHook() MockHostFunctionHook {
	return currentMockHook
}

// 以下是提供给主包使用的导出函数 - 这些函数应该在主包的测试文件中实现
// 并在测试前将实现连接到此包中提供的模拟函数

/*
// 在main包的测试文件中需要实现的函数：

//export call_host_set
func call_host_set(funcID, argPtr, argLen int32) int64 {
	return testing.GetMockHook().CallHostSet(funcID, argPtr, argLen)
}

//export call_host_get_buffer
	return testing.GetMockHook().CallHostGetBuffer(funcID, argPtr, argLen)
}

//export get_block_height
func get_block_height() int64 {
	return testing.GetMockHook().GetBlockHeight()
}

//export get_block_time
func get_block_time() int64 {
	return testing.GetMockHook().GetBlockTime()
}
func call_host_get_buffer(funcID, argPtr, argLen int32) int32 {

//export get_balance
func get_balance(addrPtr int32) uint64 {
	return testing.GetMockHook().GetBalance(addrPtr)
}
*/

// PrintHowToUse 打印使用说明
func PrintHowToUse() {
	fmt.Println(``)
}

/*
如何测试WebAssembly导出的宿主函数:

1. 在您的测试文件中导入这个包:
   import wasmtest "github.com/govm-net/vm/wasm/testing"

2. 在测试文件中实现以下桥接函数:

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

3. 在测试初始化时设置:

   func init() {
       wasmtest.Init(int32(types.HostBufferSize))
   }

4. 编写测试函数，通过调用合约的API进行测试，例如:

   func TestSender(t *testing.T) {
       ctx := &Context{}
       sender := ctx.Sender()

       // 验证结果
       if !bytes.Equal(sender[:], wasmtest.MockSender[:]) {
           t.Fatalf("Expected sender %x but got %x", wasmtest.MockSender, sender)
       }
   }

*/
