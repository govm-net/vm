package testing

import (
	"testing"
)

// 这是一个示例，说明如何在main package中实现测试文件

// 模拟实现一个简单的合约上下文，类似于main包中的 Context 结构
type ExampleContext struct{}

// 实现一个简单的方法，该方法将调用 GetBlockHeight 主机函数
func (c *ExampleContext) BlockHeight() uint64 {
	return uint64(GetBlockHeight())
}

// 测试 BlockHeight 方法
func TestBlockHeight(t *testing.T) {
	// 初始化测试环境
	Init(1024) // 假设宿主缓冲区大小为1024字节

	// 创建上下文
	ctx := &ExampleContext{}

	// 调用方法，该方法将通过我们的模拟函数返回预设值
	blockHeight := ctx.BlockHeight()

	// 验证结果
	if blockHeight != uint64(MockBlockHeight) {
		t.Fatalf("Expected block height %d but got %d", MockBlockHeight, blockHeight)
	}
}

// 自定义模拟实现 - 需要是具体类型
type customMockImpl struct {
	// 匿名嵌入默认实现
	defaultMockHook
}

// 重写 GetBlockHeight 方法
func (c *customMockImpl) GetBlockHeight() int64 {
	return 99999 // 自定义返回值
}

// 测试自定义模拟实现
func TestCustomMock(t *testing.T) {
	// 初始化测试环境
	Init(1024)

	// 保存当前钩子以便后续恢复
	originalHook := GetMockHook()
	defer SetMockHook(originalHook) // 测试完成后恢复

	// 创建并设置自定义模拟实现
	customMock := &customMockImpl{}
	SetMockHook(customMock)

	// 创建上下文并测试
	ctx := &ExampleContext{}
	blockHeight := ctx.BlockHeight()

	// 验证结果使用自定义值
	if blockHeight != 99999 {
		t.Fatalf("Expected custom block height 99999 but got %d", blockHeight)
	}
}
