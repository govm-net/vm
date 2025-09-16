package interfaces

// ContractCompiler 定义了合约编译器的接口
type ContractCompiler interface {
	// Compile 将源码编译为 WASM 二进制代码
	Compile(code []byte) ([]byte, error)
}
