package interfaces

import (
	"github.com/govm-net/vm/core"
)

// CodeRepository 定义了代码仓库管理器的接口
type CodeRepository interface {
	// RegisterCode 注册新的合约代码
	RegisterCode(address core.Address, code []byte) error

	// GetCode 获取合约代码
	GetCode(address core.Address) (*ContractCode, error)

	// GetInjectedCode 获取注入 Gas 信息后的合约代码
	GetInjectedCode(address core.Address) ([]byte, error)
}

// ContractCode 表示合约代码信息
type ContractCode struct {
	Address      core.Address // Contract address
	OriginalCode []byte       // Original code
	InjectedCode []byte       // Code after gas information injection
	Dependencies []string     // Dependencies on other contract addresses
	UpdateTime   string       // Last update time
	Hash         [32]byte     // Code hash
}

// RepositoryConstructor 是代码仓库构造函数类型
type RepositoryConstructor func(rootDir string) (CodeRepository, error)
