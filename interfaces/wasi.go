package interfaces

import (
	"github.com/govm-net/vm/types"
)

// WASIEngine 定义了 WebAssembly 引擎的接口
type WASIEngine interface {
	// DeployContract 部署一个新的 WebAssembly 合约
	DeployContract(ctx types.BlockchainContext, wasmCode []byte, sender types.Address) (types.Address, error)

	// DeployContractWithAddress 使用指定地址部署一个新的 WebAssembly 合约
	DeployContractWithAddress(ctx types.BlockchainContext, wasmCode []byte, sender types.Address, contractAddr types.Address) (types.Address, error)

	// DeleteContract 删除一个 WebAssembly 合约
	DeleteContract(ctx types.BlockchainContext, contractAddr types.Address)

	// ExecuteContract 执行合约函数
	ExecuteContract(ctx types.BlockchainContext, contractAddr types.Address, functionName string, params []byte) (interface{}, error)

	// Close 关闭引擎
	Close() error
}

// WASIEngineConstructor 是 WASI 引擎构造函数类型
type WASIEngineConstructor func(contractDir string) (WASIEngine, error)
