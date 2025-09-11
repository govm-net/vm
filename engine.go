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
	"github.com/govm-net/vm/interfaces"
	"github.com/govm-net/vm/repository"
	"github.com/govm-net/vm/types"
	"github.com/govm-net/vm/wasi"
)

// Engine is responsible for contract deployment and execution
type Engine struct {
	config      *interfaces.Config
	compiler    interfaces.Compiler
	wasiEngine  interfaces.WASIEngine
	codeManager interfaces.CodeRepository
	ctx         types.BlockchainContext // Blockchain context
}

// EngineOptions 定义引擎选项
type EngineOptions struct {
	Compiler    interfaces.Compiler
	WASIEngine  interfaces.WASIEngine
	CodeManager interfaces.CodeRepository
	Context     types.BlockchainContext
}

// EngineOption 定义引擎选项函数
type EngineOption func(*EngineOptions)

// WithCompiler 设置编译器
func WithCompiler(compiler interfaces.Compiler) EngineOption {
	return func(opts *EngineOptions) {
		opts.Compiler = compiler
	}
}

// WithWASIEngine 设置WASI引擎
func WithWASIEngine(engine interfaces.WASIEngine) EngineOption {
	return func(opts *EngineOptions) {
		opts.WASIEngine = engine
	}
}

// WithCodeManager 设置代码管理器
func WithCodeManager(manager interfaces.CodeRepository) EngineOption {
	return func(opts *EngineOptions) {
		opts.CodeManager = manager
	}
}

// WithContext 设置区块链上下文
func WithContext(ctx types.BlockchainContext) EngineOption {
	return func(opts *EngineOptions) {
		opts.Context = ctx
	}
}

// NewEngine creates a new contract engine
func NewEngine(config *interfaces.Config, opts ...EngineOption) (interfaces.Engine, error) {
	// Ensure configuration is valid
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create contract storage directory
	if err := os.MkdirAll(config.WASIContractsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create contracts directory: %w", err)
	}

	// Apply options
	options := &EngineOptions{}
	for _, opt := range opts {
		opt(options)
	}

	if options.Compiler == nil {
		// Create Compiler instance if not provided
		contractConfig := api.DefaultContractConfig()
		contractConfig.MaxCodeSize = config.MaxContractSize
		options.Compiler = compiler.NewMaker(contractConfig)
	}

	// Create WASIEngine instance if not provided
	if options.WASIEngine == nil {
		var err error
		options.WASIEngine, err = wasi.NewWazeroVM(config.WASIContractsDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create wasi engine: %w", err)
		}
	}

	// Create code manager if not provided
	if options.CodeManager == nil {
		var err error
		options.CodeManager, err = repository.NewManager(config.CodeManagerDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create code manager: %w", err)
		}
	}

	// Create context if not provided
	if options.Context == nil {
		var err error
		options.Context, err = context.Get(context.ContextType(config.ContextType), config.ContextParams)
		if err != nil {
			return nil, fmt.Errorf("failed to get default context: %w", err)
		}
	}

	return &Engine{
		config:      config,
		compiler:    options.Compiler,
		wasiEngine:  options.WASIEngine,
		codeManager: options.CodeManager,
		ctx:         options.Context,
	}, nil
}

func (e *Engine) WithContext(ctx types.BlockchainContext) interfaces.Engine {
	e.ctx = ctx
	return e
}

func (e *Engine) GetContext() types.BlockchainContext {
	return e.ctx
}

// validateConfig validates the configuration
func validateConfig(config *interfaces.Config) error {
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

// DeployContractWithAddress deploys a contract with specified address
func (e *Engine) DeployContractWithAddress(code []byte, contractAddr core.Address) error {
	// Validate contract code
	if err := e.compiler.ValidateContract(code); err != nil {
		return fmt.Errorf("contract validation failed: %w", err)
	}

	// Save contract code, add gas consumption
	err := e.codeManager.RegisterCode(contractAddr, code)
	if err != nil {
		return fmt.Errorf("failed to save contract code: %w", err)
	}

	// Parse contract code to get ABI information
	abiInfo, err := abi.ExtractABI(code)
	if err != nil {
		return fmt.Errorf("failed to parse contract ABI: %w", err)
	}
	// If there are no external functions in ABI, no need to compile to wasm, it might just be a public module
	if len(abiInfo.Functions) == 0 {
		return nil
	}

	// Get injected code
	injectedCode, err := e.codeManager.GetInjectedCode(contractAddr)
	if err != nil {
		return fmt.Errorf("failed to get contract code: %w", err)
	}

	// Compile contract
	wasmCode, err := e.compiler.CompileContract(injectedCode)
	if err != nil {
		return fmt.Errorf("contract compilation failed: %w", err)
	}

	// Deploy contract
	_, err = e.wasiEngine.DeployContractWithAddress(e.ctx, wasmCode, core.ZeroAddress, contractAddr)
	if err != nil {
		return fmt.Errorf("contract deployment failed: %w", err)
	}

	// Convert ABI to JSON
	abiJSON, err := json.MarshalIndent(abiInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal ABI: %w", err)
	}

	// Save contract ABI file
	abiPath := filepath.Join(e.config.WASIContractsDir, fmt.Sprintf("%x.abi", contractAddr))
	if err := os.WriteFile(abiPath, abiJSON, 0644); err != nil {
		return fmt.Errorf("failed to save contract ABI: %w", err)
	}

	return nil
}

// DeployContract deploys a contract
func (e *Engine) DeployContract(code []byte) (core.Address, error) {
	contractAddr := api.DefaultContractAddressGenerator(code, e.ctx.Sender())
	return contractAddr, e.DeployContractWithAddress(code, contractAddr)
}

func (e *Engine) DeleteContract(contractAddr core.Address) {
	e.wasiEngine.DeleteContract(e.ctx, contractAddr)
}

// ExecuteContract executes a contract function
func (e *Engine) ExecuteContract(contractAddr core.Address, function string, args ...interface{}) (interface{}, error) {
	// Read contract ABI file
	abiPath := filepath.Join(e.config.WASIContractsDir, fmt.Sprintf("%x.abi", contractAddr))
	abiData, err := os.ReadFile(abiPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read contract ABI: %w", err)
	}

	// Parse ABI JSON
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

	// Verify if function exists
	if funcInfo.Name == "" {
		return nil, fmt.Errorf("function %s not found in contract", function)
	}
	fArgs := funcInfo.Inputs

	params := make(map[string]interface{})
	// If the first parameter is core.Context and input length is less than required parameters, add nil as first parameter, otherwise match error
	if len(fArgs) > len(args) && fArgs[0].Type == "core.Context" {
		args = append([]any{nil}, args...)
	}
	// Convert parameters to map
	for i, arg := range args {
		params[fArgs[i].Name] = arg
	}

	// Convert parameters to JSON
	argsBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal function arguments: %w", err)
	}

	return e.Execute(contractAddr, function, argsBytes)
}

// Execute executes a contract function with raw parameters, parameters are json.marshal(map[string]any)
func (e *Engine) Execute(contractAddr core.Address, function string, args []byte) (interface{}, error) {
	// Execute contract function
	return e.wasiEngine.ExecuteContract(e.ctx, contractAddr, function, args)
}

// Close closes the engine
func (e *Engine) Close() error {
	if err := e.wasiEngine.Close(); err != nil {
		return fmt.Errorf("failed to close wasi engine: %w", err)
	}
	return nil
}

// Default constructors for built-in implementations
func newCompiler(config api.ContractConfig) interfaces.Compiler {
	// This would import the compiler package, but we'll use a factory pattern instead
	// For now, we'll return a placeholder
	return nil
}

func newWASIEngine(contractDir string) (interfaces.WASIEngine, error) {
	// This would import the wasi package, but we'll use a factory pattern instead
	// For now, we'll return a placeholder
	return nil, nil
}

func newCodeRepository(rootDir string) (interfaces.CodeRepository, error) {
	// This would import the repository package, but we'll use a factory pattern instead
	// For now, we'll return a placeholder
	return nil, nil
}
