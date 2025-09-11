package interfaces

import (
	"github.com/govm-net/vm/api"
)

// Compiler 定义了合约编译器的接口
type Compiler interface {
	// ValidateContract 验证合约代码是否符合规范
	ValidateContract(code []byte) error

	// CompileContract 将合约代码编译为 WebAssembly
	CompileContract(code []byte) ([]byte, error)
}

// CompilerConstructor 是编译器构造函数类型
type CompilerConstructor func(config api.ContractConfig) Compiler
