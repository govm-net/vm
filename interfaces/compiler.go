package interfaces

import (
	"context"

	"github.com/govm-net/vm/core"
)

// Compiler 定义了合约编译器的接口
type Compiler interface {
	// CompileContract 将合约代码编译为可执行程序
	CompileContract(ctx context.Context, code []byte) (core.Address, error)
}
