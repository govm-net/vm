package wasi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	api1 "github.com/govm-net/vm/api"
	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/types"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// WazeroVM implements a virtual machine using wazero
type WazeroVM struct {
	// Map of deployed contracts
	// contracts     map[types.Address][]byte
	contractsLock sync.RWMutex

	// Contract storage directory
	contractDir string

	// wazero runtime
	ctx context.Context

	// env module
	envModule api.Module
}

// NewWazeroVM creates a new wazero virtual machine instance
func NewWazeroVM(contractDir string) (*WazeroVM, error) {
	// Ensure contract directory exists
	if contractDir != "" {
		if err := os.MkdirAll(contractDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create contract directory: %w", err)
		}
	}

	// Create wazero runtime
	ctx := context.Background()

	vm := &WazeroVM{
		// contracts:   make(map[types.Address][]byte),
		contractDir: contractDir,
		ctx:         ctx,
	}

	return vm, nil
}

// DeployContract deploys a new WebAssembly contract
func (vm *WazeroVM) DeployContract(ctx types.BlockchainContext, wasmCode []byte, sender types.Address) (types.Address, error) {
	// Generate contract address
	contractAddr := api1.DefaultContractAddressGenerator(wasmCode, sender)
	return vm.DeployContractWithAddress(ctx, wasmCode, sender, contractAddr)
}

// DeployContractWithAddress deploys a new WebAssembly contract with specified address
func (vm *WazeroVM) DeployContractWithAddress(ctx types.BlockchainContext, wasmCode []byte, sender types.Address, contractAddr types.Address) (types.Address, error) {
	// Verify WASM code
	if len(wasmCode) == 0 {
		return types.Address{}, errors.New("contract code cannot be empty")
	}

	// Store contract code
	// vm.contractsLock.Lock()
	// vm.contracts[contractAddr] = wasmCode
	// vm.contractsLock.Unlock()
	var id core.ObjectID
	copy(id[:], contractAddr[:])
	ctx.CreateObjectWithID(contractAddr, id)

	// If contract directory is specified, save to file
	if vm.contractDir != "" {
		contractPath := filepath.Join(vm.contractDir, fmt.Sprintf("%x", contractAddr)+".wasm")
		if err := os.WriteFile(contractPath, wasmCode, 0644); err != nil {
			return types.Address{}, fmt.Errorf("failed to store contract code: %w", err)
		}
	}

	return contractAddr, nil
}

// DeleteContract deletes a WebAssembly contract
func (vm *WazeroVM) DeleteContract(ctx types.BlockchainContext, contractAddr types.Address) {
	vm.contractsLock.Lock()
	defer vm.contractsLock.Unlock()
	// Delete from contract map
	// delete(vm.contracts, contractAddr)
	os.RemoveAll(filepath.Join(vm.contractDir, fmt.Sprintf("%x", contractAddr)))
}

func (vm *WazeroVM) initContract(ctx types.BlockchainContext, wasmCode []byte) (api.Module, error) {
	ctx1 := context.Background()
	runtime := wazero.NewRuntime(ctx1)

	// Compile WASM module
	compiled, err := runtime.CompileModule(ctx1, wasmCode)
	if err != nil {
		return nil, fmt.Errorf("failed to compile WebAssembly module: %w", err)
	}

	// Create import object
	builder := runtime.NewHostModuleBuilder("env")

	// Add memory
	builder.NewFunctionBuilder().
		WithFunc(func() uint32 {
			return 0
		}).Export("memory")

	// Add host functions
	builder.NewFunctionBuilder().
		WithParameterNames("funcID", "argPtr", "argLen", "bufferPtr").
		WithResultNames("result").
		WithFunc(func(_ context.Context, m api.Module, funcID, argPtr, argLen, bufferPtr uint32) int32 {
			// fmt.Printf("call_host_set: %d, %d, %d, %d\n", funcID, argPtr, argLen, bufferPtr)
			// Read parameter data
			mem := m.Memory()
			if mem == nil {
				return 0
			}
			argData, ok := mem.Read(argPtr, argLen)
			if !ok || len(argData) != int(argLen) {
				return 0
			}

			return vm.handleHostSet(ctx, m, funcID, argData, bufferPtr)
		}).
		Export("call_host_set")

	builder.NewFunctionBuilder().
		WithParameterNames("funcID", "argPtr", "argLen", "buffer").
		WithResultNames("result").
		WithFunc(func(_ context.Context, m api.Module, funcID, argPtr, argLen, buffer uint32) int32 {
			// fmt.Printf("call_host_get_buffer: %d, %d, %d, %d\n", funcID, argPtr, argLen, buffer)
			// 读取参数数据
			mem := m.Memory()
			if mem == nil {
				return 0
			}
			argData, ok := mem.Read(argPtr, argLen)
			if !ok || len(argData) != int(argLen) {
				return 0
			}

			return vm.handleHostGetBuffer(ctx, m, funcID, argData, buffer)
		}).
		Export("call_host_get_buffer")

	builder.NewFunctionBuilder().
		WithResultNames("result").
		WithFunc(func(_ context.Context, _ api.Module) uint32 {
			return uint32(ctx.BlockHeight())
		}).
		Export("get_block_height")

	builder.NewFunctionBuilder().
		WithResultNames("result").
		WithFunc(func(_ context.Context, _ api.Module) uint32 {
			return uint32(ctx.BlockTime())
		}).
		Export("get_block_time")

	builder.NewFunctionBuilder().
		WithParameterNames("addrPtr").
		WithResultNames("result").
		WithFunc(func(_ context.Context, m api.Module, addrPtr uint32) uint32 {
			// 读取地址
			mem := m.Memory()
			if mem == nil {
				return 0
			}
			addrData, ok := mem.Read(addrPtr, 20)
			if !ok || len(addrData) != 20 {
				return 0
			}

			var addr types.Address
			copy(addr[:], addrData)

			// 获取余额
			return uint32(ctx.Balance(addr))
		}).
		Export("get_balance")

	// Initialize WASI
	envModule, err := builder.Instantiate(vm.ctx)
	if err != nil {
		return nil, fmt.Errorf("实例化导入对象失败: %w", err)
	}
	vm.envModule = envModule

	wasi_snapshot_preview1.MustInstantiate(vm.ctx, runtime)

	// Create module configuration
	config := wazero.NewModuleConfig().
		WithName("contract").WithStdout(os.Stdout).WithStderr(os.Stderr)

	// Instantiate module
	module, err := runtime.InstantiateModule(ctx1, compiled, config.WithStartFunctions("_initialize"))
	if err != nil {
		fmt.Printf("failed to instantiate module: %v\n", err)
		return nil, fmt.Errorf("failed to instantiate module: %w", err)
	}
	return module, nil
}

// ExecuteContract executes a deployed contract function
func (vm *WazeroVM) ExecuteContract(ctx types.BlockchainContext, contractAddr types.Address, functionName string, params []byte) (interface{}, error) {
	// Check if contract exists
	// vm.contractsLock.RLock()
	// wasmCode, exists := vm.contracts[contractAddr]
	// vm.contractsLock.RUnlock()

	// if !exists {
	// 	return nil, fmt.Errorf("contract does not exist: %x", contractAddr)
	// }
	wasmCode, err := os.ReadFile(filepath.Join(vm.contractDir, fmt.Sprintf("%x", contractAddr)+".wasm"))
	if err != nil {
		return nil, fmt.Errorf("failed to read contract code: %w", err)
	}

	module, err := vm.initContract(ctx, wasmCode)
	if err != nil {
		return types.Address{}, fmt.Errorf("failed to instantiate WebAssembly module: %w", err)
	}

	result, err := vm.callWasmFunction(ctx, module, functionName, params, contractAddr)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}
	var runResult types.ExecutionResult
	err = json.Unmarshal(result, &runResult)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize: %w", err)
	}
	return runResult.Data, nil
}

// callWasmFunction calls a WASM function
func (vm *WazeroVM) callWasmFunction(ctx types.BlockchainContext, module api.Module, functionName string, params []byte, contractAddr types.Address) ([]byte, error) {
	// fmt.Printf("calling contract function:%s, %v\n", functionName, string(params))

	// Check if allocate and deallocate functions are exported
	allocate := module.ExportedFunction("allocate")
	if allocate == nil {
		return nil, fmt.Errorf("allocate function not found")
	}

	processDataFunc := module.ExportedFunction("handle_contract_call")
	if processDataFunc == nil {
		return nil, fmt.Errorf("handle_contract_call not found")
	}

	var input types.HandleContractCallParams
	input.Contract = contractAddr
	input.Sender = ctx.Sender()
	input.Function = functionName
	input.Args = params
	input.GasLimit = ctx.GetGas()
	// fmt.Println("[host]handle_contract_call gasLimit", input.GasLimit)
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize handle_contract_call: %w", err)
	}

	// Allocate memory and write parameters
	result, err := allocate.Call(vm.ctx, uint64(len(inputBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to allocate memory: %w", err)
	}
	inputAddr := uint32(result[0])

	// Write parameter data
	if !module.Memory().Write(inputAddr, inputBytes) {
		return nil, fmt.Errorf("failed to write to memory")
	}

	// Call processing function
	result, err = processDataFunc.Call(vm.ctx, uint64(inputAddr), uint64(len(inputBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to execute %s: %w", functionName, err)
	}

	var out []byte
	resultLen := int32(result[0])
	// if resultLen > int32(types.HostBufferSize) {
	// 	resultLen = resultLen - int32(types.HostBufferSize)
	// }
	if resultLen > 0 {
		getBufferAddress := module.ExportedFunction("get_buffer_address")
		if getBufferAddress == nil {
			return nil, fmt.Errorf("get_buffer_address function not found")
		}

		result, err = getBufferAddress.Call(vm.ctx)
		if err != nil {
			return nil, fmt.Errorf("get_buffer_address failed: %w", err)
		}
		bufferPtr := uint32(result[0])

		// Read result data
		data, ok := module.Memory().Read(bufferPtr, uint32(resultLen))
		if !ok {
			return nil, fmt.Errorf("failed to read memory:%d, len:%d", bufferPtr, resultLen)
		}
		out = data
	}

	// Free memory
	deallocate := module.ExportedFunction("deallocate")
	if deallocate == nil {
		return nil, fmt.Errorf("deallocate function not found")
	}
	_, err = deallocate.Call(vm.ctx, uint64(inputAddr), uint64(len(inputBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to free memory: %w", err)
	}

	// fmt.Printf("execution completed:%s, %v\n", functionName, resultLen)
	return out, nil
}

// Host function handler
func (vm *WazeroVM) handleHostSet(ctx types.BlockchainContext, m api.Module, funcID uint32, argData []byte, bufferPtr uint32) int32 {
	// Process different operations based on function ID
	switch types.WasmFunctionID(funcID) {
	case types.FuncTransfer:
		var params types.TransferParams
		if err := json.Unmarshal(argData, &params); err != nil {
			return -1
		}
		err := ctx.Transfer(params.Contract, params.From, params.To, params.Amount)
		if err != nil {
			return -1
		}
		return 0
	case types.FuncCall:
		var params types.CallParams
		if err := json.Unmarshal(argData, &params); err != nil {
			return -1
		}
		ctx.SetGasLimit(params.GasLimit)
		result, err := ctx.Call(params.Caller, params.Contract, params.Function, params.Args...)
		if err != nil {
			return -1
		}
		currentGas := ctx.GetGas()
		if currentGas > params.GasLimit {
			return -1
		}
		var callResult types.CallResult
		callResult.Data = result
		callResult.GasUsed = params.GasLimit - currentGas
		resultBytes, err := json.Marshal(callResult)
		if err != nil {
			return -1
		}
		if !m.Memory().Write(bufferPtr, resultBytes) {
			return -1
		}
		return 0
	case types.FuncDeleteObject:
		var params types.DeleteObjectParams
		if err := json.Unmarshal(argData, &params); err != nil {
			return -1
		}
		if params.ID == (types.ObjectID{}) {
			copy(params.ID[:], params.Contract[:])
		}
		err := ctx.DeleteObject(params.Contract, params.ID)
		if err != nil {
			return -1
		}
		return 0

	case types.FuncLog:
		var params types.LogParams
		if err := json.Unmarshal(argData, &params); err != nil {
			return -1
		}
		ctx.Log(params.Contract, params.Event, params.KeyValues...)
		return 0

	case types.FuncSetObjectOwner:
		var params types.SetOwnerParams
		if err := json.Unmarshal(argData, &params); err != nil {
			return -1
		}
		if params.ID == (types.ObjectID{}) {
			copy(params.ID[:], params.Contract[:])
		}
		obj, err := ctx.GetObject(params.Contract, params.ID)
		if err != nil {
			return -1
		}
		if obj.Owner() != params.Contract {
			return -1
		}
		err = obj.SetOwner(params.Contract, params.Sender, params.Owner)
		if err != nil {
			return -1
		}
		return 0

	case types.FuncSetObjectField:
		var params types.SetObjectFieldParams
		if err := json.Unmarshal(argData, &params); err != nil {
			slog.Error("failed to deserialize set_object_field", "error", err)
			return -1
		}
		if params.ID == (types.ObjectID{}) {
			copy(params.ID[:], params.Contract[:])
		}
		obj, err := ctx.GetObject(params.Contract, params.ID)
		if err != nil {
			slog.Error("failed to get object in set_object_field", "error", err)
			return -1
		}
		if obj.Owner() != params.Contract {
			slog.Error("object owner mismatch in set_object_field", "contract", params.Contract, "owner", obj.Owner())
			return -1
		}
		// Convert value to []byte
		valueBytes, err := json.Marshal(params.Value)
		if err != nil {
			slog.Error("failed to serialize in set_object_field", "error", err)
			return -1
		}
		// slog.Info("---set_object_field", "field", params.Field, "value", params.Value, "string", valueBytes, "len", len(valueBytes))
		err = obj.Set(params.Contract, params.Sender, params.Field, valueBytes)
		if err != nil {
			slog.Error("failed to set field in set_object_field", "error", err)
			return -1
		}
		return 0

	default:
		return -1
	}
}

func (vm *WazeroVM) handleHostGetBuffer(ctx types.BlockchainContext, m api.Module, funcID uint32, argData []byte, offset uint32) int32 {
	mem := m.Memory()
	if mem == nil {
		return -1
	}
	// Process different operations based on function ID
	switch types.WasmFunctionID(funcID) {
	case types.FuncGetSender:
		sender := ctx.Sender()
		mem.Write(offset, sender[:])
		return int32(len(sender))

	case types.FuncGetContractAddress:
		contractAddr := ctx.ContractAddress()
		mem.Write(offset, contractAddr[:])
		return int32(len(contractAddr))

	case types.FuncCreateObject:
		obj, err := ctx.CreateObject(ctx.ContractAddress())
		if err != nil {
			return -1
		}
		id := obj.ID()
		mem.Write(offset, id[:])
		return int32(len(id))

	case types.FuncGetObjectField:
		var params types.GetObjectFieldParams
		if err := json.Unmarshal(argData, &params); err != nil {
			fmt.Printf("failed to deserialize obj.getfield: %v\n", err)
			return -1
		}
		if params.ID == (types.ObjectID{}) {
			copy(params.ID[:], params.Contract[:])
		}
		obj, err := ctx.GetObject(params.Contract, params.ID)
		if err != nil {
			fmt.Printf("failed to get object in obj.getfield:id:%x, %v\n", params.ID, err)
			return -1
		}
		data, err := obj.Get(params.Contract, params.Field)
		if err != nil {
			fmt.Printf("failed to get field in obj.getfield:id:%x, %v\n", params.ID, err)
			return -1
		}
		mem.Write(offset, data)
		// slog.Info("obj.getfield", "id", params.ID, "field", params.Field, "value", string(data))
		return int32(len(data))

	case types.FuncGetObject:
		var params types.GetObjectParams
		if err := json.Unmarshal(argData, &params); err != nil {
			return -1
		}
		if params.ID == (types.ObjectID{}) {
			copy(params.ID[:], params.Contract[:])
		}
		obj, err := ctx.GetObject(params.Contract, params.ID)
		if err != nil {
			return -1
		}
		if obj.Contract() != params.Contract {
			return -1
		}
		id := obj.ID()
		mem.Write(offset, id[:])
		return int32(len(id))

	case types.FuncGetObjectWithOwner:
		var params types.GetObjectWithOwnerParams
		if err := json.Unmarshal(argData, &params); err != nil {
			return -1
		}
		obj, err := ctx.GetObjectWithOwner(params.Contract, params.Owner)
		if err != nil {
			return -1
		}
		id := obj.ID()
		mem.Write(offset, id[:])
		return int32(len(id))

	case types.FuncGetObjectOwner:
		if len(argData) != 32 {
			return -1
		}
		var objID types.ObjectID
		copy(objID[:], argData)
		obj, err := ctx.GetObject(ctx.ContractAddress(), objID)
		if err != nil {
			return -1
		}
		owner := obj.Owner()
		mem.Write(offset, owner[:])
		return int32(len(owner))

	default:
		return -1
	}
}

// Close closes the virtual machine
func (vm *WazeroVM) Close() error {
	if vm.envModule != nil {
		if err := vm.envModule.Close(vm.ctx); err != nil {
			return fmt.Errorf("failed to close env module: %w", err)
		}
	}
	return nil
}
