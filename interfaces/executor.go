package interfaces

import (
	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/types"
)

// ContractExecutor 定义了合约执行器的接口
type ContractExecutor interface {
	// Execute 执行合约函数
	Execute(ctx types.BlockchainContext, contractAddr core.Address, functionName string, params []byte) (interface{}, error)
}
