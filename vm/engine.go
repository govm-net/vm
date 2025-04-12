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

// Engine is responsible for contract deployment and execution
type Engine struct {
	config        *Config
	maker         *compiler.Maker
	wazero_engine *wasi.WazeroVM
	codeManager   *repository.Manager
	ctx           types.BlockchainContext // Blockchain context
}

// Config represents engine configuration
type Config struct {
	// Contract related configuration
	MaxContractSize  uint64         // Maximum contract size
	WASIContractsDir string         // WASI contract storage directory
	CodeManagerDir   string         // Code manager storage directory
	ContextType      string         // Blockchain context type
	ContextParams    map[string]any // Blockchain context parameters
}

// NewEngine creates a new contract engine
func NewEngine(config *Config) (*Engine, error) {
	// Ensure configuration is valid
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create contract storage directory
	if err := os.MkdirAll(config.WASIContractsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create contracts directory: %w", err)
	}

	// Create Maker instance
	contractConfig := api.DefaultContractConfig()
	contractConfig.MaxCodeSize = uint64(config.MaxContractSize)
	maker := compiler.NewMaker(contractConfig)

	// Create WazeroEngine instance
	wazero_engine, err := wasi.NewWazeroVM(config.WASIContractsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create wazero engine: %w", err)
	}

	// Create code manager
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

// validateConfig validates the configuration
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

// DeployContractWithAddress deploys a contract with specified address
func (e *Engine) DeployContractWithAddress(code []byte, contractAddr core.Address) error {
	// Validate contract code
	if err := e.maker.ValidateContract(code); err != nil {
		return fmt.Errorf("contract validation failed: %w", err)
	}

	// Save contract code, add gas consumption
	err := e.codeManager.RegisterCode(contractAddr, code)
	if err != nil {
		return fmt.Errorf("failed to save contract code: %w", err)
	}

	// Parse contract code to get ABI information
	abi, err := abi.ExtractABI(code)
	if err != nil {
		return fmt.Errorf("failed to parse contract ABI: %w", err)
	}
	// If there are no external functions in ABI, no need to compile to wasm, it might just be a public module
	if len(abi.Functions) == 0 {
		return nil
	}

	// Add gas consumption
	code, err = e.codeManager.GetInjectedCode(contractAddr)
	if err != nil {
		return fmt.Errorf("failed to get contract code: %w", err)
	}

	// Compile contract
	wasmCode, err := e.maker.CompileContract(code)
	if err != nil {
		return fmt.Errorf("contract compilation failed: %w", err)
	}

	// Deploy contract
	_, err = e.wazero_engine.DeployContractWithAddress(e.ctx, wasmCode, core.ZeroAddress, contractAddr)
	if err != nil {
		return fmt.Errorf("contract deployment failed: %w", err)
	}

	// Convert ABI to JSON
	abiJSON, err := json.MarshalIndent(abi, "", "  ")
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
	e.wazero_engine.DeleteContract(e.ctx, contractAddr)
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

// ExecuteContract executes a contract function with raw parameters, parameters are json.marshal(map[string]any)
func (e *Engine) Execute(contractAddr core.Address, function string, args []byte) (interface{}, error) {
	// Execute contract function
	return e.wazero_engine.ExecuteContract(e.ctx, contractAddr, function, args)
}

// Close closes the engine
func (e *Engine) Close() error {
	if err := e.wazero_engine.Close(); err != nil {
		return fmt.Errorf("failed to close wazero engine: %w", err)
	}
	return nil
}
