package interfaces

import (
	"context"

	"github.com/govm-net/vm/core"
)

// ContractManager 定义了合约管理器的接口
type ContractManager interface {
	SaveSourceCode(address core.Address, code []byte) error
	SaveABI(address core.Address, abi *ABI) error
	SaveCompiledCode(address core.Address, code []byte) error

	// GetCompiled 获取编译后的合约代码
	GetCompiled(address core.Address) ([]byte, error)

	// GetABI 获取合约 ABI
	GetABI(address core.Address) (*ABI, error)

	// Delete 删除合约
	Delete(address core.Address) error
}

type ContractAddressGenerator interface {
	Generate(ctx context.Context, code []byte) core.Address
}
