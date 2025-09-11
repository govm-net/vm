package interfaces

import (
	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/types"
)

// Config represents engine configuration
type Config struct {
	// Contract related configuration
	MaxContractSize  uint64         // Maximum contract size
	WASIContractsDir string         // WASI contract storage directory
	CodeManagerDir   string         // Code manager storage directory
	ContextType      string         // Blockchain context type
	ContextParams    map[string]any // Blockchain context parameters
}

// Engine 定义了虚拟机引擎的接口
type Engine interface {
	// WithContext 设置区块链上下文
	WithContext(ctx types.BlockchainContext) Engine

	// GetContext 获取当前区块链上下文
	GetContext() types.BlockchainContext

	// DeployContractWithAddress 使用指定地址部署合约
	DeployContractWithAddress(code []byte, contractAddr core.Address) error

	// DeployContract 部署合约
	DeployContract(code []byte) (core.Address, error)

	// DeleteContract 删除合约
	DeleteContract(contractAddr core.Address)

	// ExecuteContract 执行合约函数
	ExecuteContract(contractAddr core.Address, function string, args ...interface{}) (interface{}, error)

	// Execute 使用原始参数执行合约函数
	Execute(contractAddr core.Address, function string, args []byte) (interface{}, error)

	// Close 关闭引擎
	Close() error
}

// EngineConstructor 是引擎构造函数类型
type EngineConstructor func(config *Config) (Engine, error)

// EngineOptions 定义引擎选项
type EngineOptions struct {
	Compiler    Compiler
	WASIEngine  WASIEngine
	CodeManager CodeRepository
	Context     types.BlockchainContext
}

// EngineOption 定义引擎选项函数
type EngineOption func(*EngineOptions)
