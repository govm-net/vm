package vm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/govm-net/vm/api"
	"github.com/govm-net/vm/compiler"
	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/types"
	"github.com/govm-net/vm/wasi"
)

// Engine 合约引擎，负责合约的部署和执行
type Engine struct {
	config        *Config
	maker         *compiler.Maker
	wazero_engine *wasi.WazeroVM
	contracts     map[core.Address][]byte
	contractsLock sync.RWMutex
	ctx           types.BlockchainContext // 区块链上下文
}

// Config 引擎配置
type Config struct {
	// 合约相关配置
	MaxContractSize  int64  // 最大合约大小
	WASIContractsDir string // WASI合约存储目录

	// TinyGo相关配置
	TinyGoPath string // TinyGo编译器路径

	// WASI运行时配置
	WASIOptions WASIOptions
}

// WASIOptions WASI运行时选项
type WASIOptions struct {
	MemoryLimit      int64  // 内存限制
	TableSize        int    // 函数表大小
	Timeout          int64  // 执行超时(毫秒)
	FuelLimit        int64  // 指令限制
	StackSize        int    // 栈大小
	EnableSIMD       bool   // 是否启用SIMD
	EnableThreads    bool   // 是否启用线程
	EnableBulkMemory bool   // 是否启用批量内存操作
	PrecompiledCache bool   // 是否启用预编译缓存
	CacheDir         string // 缓存目录
	LogLevel         string // 日志级别
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

	// 创建缓存目录
	if config.WASIOptions.PrecompiledCache {
		if err := os.MkdirAll(config.WASIOptions.CacheDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create cache directory: %w", err)
		}
	}

	// 创建Maker实例
	contractConfig := api.ContractConfig{
		MaxCodeSize:    uint64(config.MaxContractSize),
		AllowedImports: []string{"github.com/govm-net/vm/core"},
	}
	maker := compiler.NewMaker(contractConfig)

	// 创建WazeroEngine实例
	wazero_engine, err := wasi.NewWazeroVM(config.WASIContractsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create wazero engine: %w", err)
	}

	return &Engine{
		config:        config,
		maker:         maker,
		wazero_engine: wazero_engine,
		contracts:     make(map[core.Address][]byte),
		ctx:           wasi.NewDefaultBlockchainContext(),
	}, nil
}

func (e *Engine) WithContext(ctx types.BlockchainContext) *Engine {
	e.ctx = ctx
	return e
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

	if config.TinyGoPath == "" {
		return fmt.Errorf("TinyGo path is empty")
	}

	if config.WASIOptions.MemoryLimit <= 0 {
		return fmt.Errorf("invalid memory limit: %d", config.WASIOptions.MemoryLimit)
	}

	if config.WASIOptions.Timeout <= 0 {
		return fmt.Errorf("invalid timeout: %d", config.WASIOptions.Timeout)
	}

	if config.WASIOptions.FuelLimit <= 0 {
		return fmt.Errorf("invalid fuel limit: %d", config.WASIOptions.FuelLimit)
	}

	return nil
}

// DeployContract 部署合约
func (e *Engine) DeployContract(code []byte) (core.Address, error) {
	// 验证合约代码
	if err := e.maker.ValidateContract(code); err != nil {
		return core.ZeroAddress, fmt.Errorf("contract validation failed: %w", err)
	}

	// 编译合约
	wasmCode, err := e.maker.CompileContract(code)
	if err != nil {
		return core.ZeroAddress, fmt.Errorf("contract compilation failed: %w", err)
	}

	// 部署合约
	contractAddr, err := e.wazero_engine.DeployContract(e.ctx, wasmCode, core.ZeroAddress)
	if err != nil {
		return core.ZeroAddress, fmt.Errorf("contract deployment failed: %w", err)
	}

	// 保存合约代码
	e.contractsLock.Lock()
	e.contracts[contractAddr] = wasmCode
	e.contractsLock.Unlock()

	// 解析合约代码获取ABI信息
	abi, err := e.maker.ParseABI(code)
	if err != nil {
		return contractAddr, fmt.Errorf("failed to parse contract ABI: %w", err)
	}

	// 将ABI转换为JSON
	abiJSON, err := json.MarshalIndent(abi, "", "  ")
	if err != nil {
		return contractAddr, fmt.Errorf("failed to marshal ABI: %w", err)
	}

	// 保存合约ABI文件
	abiPath := filepath.Join(e.config.WASIContractsDir, fmt.Sprintf("%x.abi", contractAddr))
	if err := os.WriteFile(abiPath, abiJSON, 0644); err != nil {
		return contractAddr, fmt.Errorf("failed to save contract ABI: %w", err)
	}

	return contractAddr, nil
}

// ExecuteContract 执行合约函数
func (e *Engine) ExecuteContract(contractAddr core.Address, function string, args ...interface{}) (interface{}, error) {
	// 获取合约代码
	e.contractsLock.RLock()
	_, exists := e.contracts[contractAddr]
	e.contractsLock.RUnlock()

	if !exists {
		return nil, fmt.Errorf("contract not found: %s", contractAddr)
	}

	// 读取合约ABI文件
	abiPath := filepath.Join(e.config.WASIContractsDir, fmt.Sprintf("%x.abi", contractAddr))
	abiData, err := os.ReadFile(abiPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read contract ABI: %w", err)
	}

	// 解析ABI JSON
	var abi map[string]compiler.FunctionInfo
	if err := json.Unmarshal(abiData, &abi); err != nil {
		return nil, fmt.Errorf("failed to parse ABI JSON: %w", err)
	}

	// 验证函数是否存在
	if _, ok := abi[function]; !ok {
		return nil, fmt.Errorf("function %s not found in contract", function)
	}

	// 将参数转换为map
	params := make(map[string]interface{})
	for i, arg := range args {
		params[abi[function].Args[i].Name] = arg
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
	// 获取合约代码
	e.contractsLock.RLock()
	_, exists := e.contracts[contractAddr]
	e.contractsLock.RUnlock()

	if !exists {
		return nil, fmt.Errorf("contract not found: %s", contractAddr)
	}

	// 执行合约函数
	return e.wazero_engine.ExecuteContract(e.ctx, contractAddr, core.ZeroAddress, function, args)
}

// Close 关闭引擎
func (e *Engine) Close() error {
	if err := e.wazero_engine.Close(); err != nil {
		return fmt.Errorf("failed to close wazero engine: %w", err)
	}
	return nil
}
