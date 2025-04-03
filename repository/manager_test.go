package repository

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/govm-net/vm/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "code_manager_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// 创建代码管理器
	manager, err := NewManager(tmpDir)
	require.NoError(t, err)

	// 测试合约地址
	addr := core.AddressFromString("1234567890abcdef1234567890abcdef12345678")

	// 测试代码
	code := []byte(`package main

import (
	"contract/abcdef1234567890abcdef1234567890abcdef12"
	"contract/2222567890abcdef1234567890abcdef12345678"
)

func main() {
	// 测试代码
}`)

	// 注册代码
	err = manager.RegisterCode(addr, code)
	require.NoError(t, err)

	// 验证文件是否创建
	contractDir := filepath.Join(tmpDir, addr.String())
	assert.DirExists(t, contractDir)
	assert.FileExists(t, filepath.Join(contractDir, "original.go.txt"))
	assert.FileExists(t, filepath.Join(contractDir, "injected.go"))
	assert.FileExists(t, filepath.Join(contractDir, "metadata.json"))

	// 获取代码并验证内容
	contractCode, err := manager.GetCode(addr)
	require.NoError(t, err)
	assert.Equal(t, code, contractCode.OriginalCode)
}

func TestDependencyGraphWithMissingContracts(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "code_manager_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// 创建代码管理器
	manager, err := NewManager(tmpDir)
	require.NoError(t, err)

	// 测试合约地址
	addr := core.AddressFromString("1234567890abcdef1234567890abcdef12345678")

	// 测试代码，包含对不存在合约的依赖
	code := []byte(`package main

import (
	"contract/abcdef1234567890abcdef1234567890abcdef12"
	"contract/9876543210fedcba9876543210fedcba98765432"
)

func main() {
	// 测试代码
}`)

	// 注册代码
	err = manager.RegisterCode(addr, code)
	require.NoError(t, err)
}

func TestContractImmutability(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "code_manager_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// 创建代码管理器
	manager, err := NewManager(tmpDir)
	require.NoError(t, err)

	// 测试合约地址
	addr := core.AddressFromString("1234567890abcdef1234567890abcdef12345678")

	// 原始代码
	code := []byte(`package main

func main() {
	// 测试代码
}`)

	// 首次注册代码应该成功
	err = manager.RegisterCode(addr, code)
	require.NoError(t, err)

	// 尝试更新代码应该失败
	newCode := []byte(`package main

func main() {
	// 新的测试代码
}`)

	err = manager.RegisterCode(addr, newCode)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "contract already exists")

	// 验证代码没有被更新
	contractCode, err := manager.GetCode(addr)
	require.NoError(t, err)
	assert.Equal(t, code, contractCode.OriginalCode)

	// 验证文件内容没有被更新
	originalCode, err := os.ReadFile(filepath.Join(tmpDir, addr.String(), "original.go.txt"))
	require.NoError(t, err)
	assert.Equal(t, code, originalCode)
}
