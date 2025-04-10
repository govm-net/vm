package vm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/govm-net/vm/abi"
	"github.com/govm-net/vm/api"
	"github.com/govm-net/vm/compiler"
	"github.com/govm-net/vm/context"
	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/repository"
	"github.com/govm-net/vm/types"
	"github.com/govm-net/vm/wasi"
)

// Engine 合约引擎，负责合约的部署和执行
type Engine struct {
	config        *Config
	maker         *compiler.Maker
	wazero_engine *wasi.WazeroVM
	codeManager   *repository.Manager
	ctx           types.BlockchainContext // 区块链上下文
}

// Config 引擎配置
type Config struct {
	// 合约相关配置
	MaxContractSize  uint64         // 最大合约大小
	WASIContractsDir string         // WASI合约存储目录
	CodeManagerDir   string         // 代码管理器存储目录
	ContextType      string         // 区块链上下文类型
	ContextParams    map[string]any // 区块链上下文参数
}

// NewEngine 创建新的合约引擎
func NewEngine(config *Config) (*Engine, error) {
	// 确保配置有效
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 创建合约存储目录
	if err := os.MkdirAll(config.WASIContractsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create contracts directory: %w", err)
	}

	// 创建Maker实例
	contractConfig := api.DefaultContractConfig()
	contractConfig.MaxCodeSize = uint64(config.MaxContractSize)
	maker := compiler.NewMaker(contractConfig)

	// 创建WazeroEngine实例
	wazero_engine, err := wasi.NewWazeroVM(config.WASIContractsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create wazero engine: %w", err)
	}

	// 创建代码管理器
	codeManager, err := repository.NewManager(config.CodeManagerDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create code manager: %w", err)
	}
	ctx, err := context.Get(context.ContextType(config.ContextType), config.ContextParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get default context: %w", err)
	}

	return &Engine{
		config:        config,
		maker:         maker,
		wazero_engine: wazero_engine,
		codeManager:   codeManager,
		ctx:           ctx,
	}, nil
}

func (e *Engine) WithContext(ctx types.BlockchainContext) *Engine {
	e.ctx = ctx
	return e
}

func (e *Engine) GetContext() types.BlockchainContext {
	return e.ctx
}

// validateConfig 验证配置
func validateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	if config.MaxContractSize <= 0 {
		return fmt.Errorf("invalid max contract size: %d", config.MaxContractSize)
	}

	if config.WASIContractsDir == "" {
		return fmt.Errorf("WASI contracts directory is empty")
	}

	return nil
}

// DeployContract 部署合约
func (e *Engine) DeployContractWithAddress(code []byte, contractAddr core.Address) error {
	// 验证合约代码
	if err := e.maker.ValidateContract(code); err != nil {
		return fmt.Errorf("contract validation failed: %w", err)
	}

	// 保存合约代码, 添加gas消耗
	err := e.codeManager.RegisterCode(contractAddr, code)
	if err != nil {
		return fmt.Errorf("failed to save contract code: %w", err)
	}

	// 解析合约代码获取ABI信息
	abi, err := abi.ExtractABI(code)
	if err != nil {
		return fmt.Errorf("failed to parse contract ABI: %w", err)
	}
	// 如果ABI中没有对外函数，不需要编译成wasm，可能只是公共模块
	if len(abi.Functions) == 0 {
		return nil
	}

	// 添加gas消耗
	code, err = e.codeManager.GetInjectedCode(contractAddr)
	if err != nil {
		return fmt.Errorf("failed to get contract code: %w", err)
	}

	// 编译合约
	wasmCode, err := e.maker.CompileContract(code)
	if err != nil {
		return fmt.Errorf("contract compilation failed: %w", err)
	}

	// 部署合约
	_, err = e.wazero_engine.DeployContractWithAddress(e.ctx, wasmCode, core.ZeroAddress, contractAddr)
	if err != nil {
		return fmt.Errorf("contract deployment failed: %w", err)
	}

	// 将ABI转换为JSON
	abiJSON, err := json.MarshalIndent(abi, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal ABI: %w", err)
	}

	// 保存合约ABI文件
	abiPath := filepath.Join(e.config.WASIContractsDir, fmt.Sprintf("%x.abi", contractAddr))
	if err := os.WriteFile(abiPath, abiJSON, 0644); err != nil {
		return fmt.Errorf("failed to save contract ABI: %w", err)
	}

	return nil
}

// DeployContract 部署合约
func (e *Engine) DeployContract(code []byte) (core.Address, error) {
	contractAddr := api.DefaultContractAddressGenerator(code)
	return contractAddr, e.DeployContractWithAddress(code, contractAddr)
}

func (e *Engine) DeleteContract(contractAddr core.Address) {
	e.wazero_engine.DeleteContract(e.ctx, contractAddr)
}

// ExecuteContract 执行合约函数
func (e *Engine) ExecuteContract(contractAddr core.Address, function string, args ...interface{}) (interface{}, error) {
	// 读取合约ABI文件
	abiPath := filepath.Join(e.config.WASIContractsDir, fmt.Sprintf("%x.abi", contractAddr))
	abiData, err := os.ReadFile(abiPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read contract ABI: %w", err)
	}

	// 解析ABI JSON
	var abiInfo abi.ABI
	if err := json.Unmarshal(abiData, &abiInfo); err != nil {
		return nil, fmt.Errorf("failed to parse ABI JSON: %w", err)
	}
	var funcInfo abi.Function
	for _, fn := range abiInfo.Functions {
		if fn.Name == function {
			funcInfo = fn
			break
		}
	}

	// 验证函数是否存在
	if funcInfo.Name == "" {
		return nil, fmt.Errorf("function %s not found in contract", function)
	}
	fArgs := funcInfo.Inputs

	params := make(map[string]interface{})
	// 如果函数第一个参数是core.Context，且入参长度少于需要的参数，则添加nil作为第一个参数，否则匹配错误
	if len(fArgs) > len(args) && fArgs[0].Type == "core.Context" {
		args = append([]any{nil}, args...)
	}
	// 将参数转换为map
	for i, arg := range args {
		params[fArgs[i].Name] = arg
	}

	// 将参数转换为JSON
	argsBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal function arguments: %w", err)
	}

	return e.Execute(contractAddr, function, argsBytes)
}

// ExecuteContract 执行合约函数带原始参数，参数是json.marshal(map[string]any)
func (e *Engine) Execute(contractAddr core.Address, function string, args []byte) (interface{}, error) {
	// 执行合约函数
	return e.wazero_engine.ExecuteContract(e.ctx, contractAddr, function, args)
}

// Close 关闭引擎
func (e *Engine) Close() error {
	if err := e.wazero_engine.Close(); err != nil {
		return fmt.Errorf("failed to close wazero engine: %w", err)
	}
	return nil
}
