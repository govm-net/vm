package wasi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/types"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// WazeroVM 使用wazero实现的虚拟机
type WazeroVM struct {
	// 存储已部署合约的映射表
	contracts     map[types.Address][]byte
	contractsLock sync.RWMutex

	// 合约存储目录
	contractDir string

	// wazero运行时
	ctx context.Context

	// env模块
	envModule api.Module
}

// NewWazeroVM 创建一个新的wazero虚拟机实例
func NewWazeroVM(contractDir string) (*WazeroVM, error) {
	// 确保合约目录存在
	if contractDir != "" {
		if err := os.MkdirAll(contractDir, 0755); err != nil {
			return nil, fmt.Errorf("创建合约目录失败: %w", err)
		}
	}

	// 创建wazero运行时
	ctx := context.Background()

	vm := &WazeroVM{
		contracts:   make(map[types.Address][]byte),
		contractDir: contractDir,
		ctx:         ctx,
	}

	return vm, nil
}

// DeployContract 部署新的WebAssembly合约
func (vm *WazeroVM) DeployContract(ctx types.BlockchainContext, wasmCode []byte, sender types.Address) (types.Address, error) {
	// 生成合约地址
	contractAddr := vm.generateContractAddress(wasmCode, sender)
	return vm.DeployContractWithAddress(ctx, wasmCode, sender, contractAddr)
}

// DeployContractWithAddress 部署新的WebAssembly合约
func (vm *WazeroVM) DeployContractWithAddress(ctx types.BlockchainContext, wasmCode []byte, sender types.Address, contractAddr types.Address) (types.Address, error) {
	// 验证WASM代码
	if len(wasmCode) == 0 {
		return types.Address{}, errors.New("合约代码不能为空")
	}

	// 存储合约代码
	vm.contractsLock.Lock()
	vm.contracts[contractAddr] = wasmCode
	vm.contractsLock.Unlock()
	var id core.ObjectID
	copy(id[:], contractAddr[:])
	ctx.CreateObjectWithID(contractAddr, id)

	// 如果指定了合约目录，则保存到文件
	if vm.contractDir != "" {
		contractPath := filepath.Join(vm.contractDir, fmt.Sprintf("%x", contractAddr)+".wasm")
		if err := os.WriteFile(contractPath, wasmCode, 0644); err != nil {
			return types.Address{}, fmt.Errorf("存储合约代码失败: %w", err)
		}
	}

	return contractAddr, nil
}

func (vm *WazeroVM) initContract(ctx types.BlockchainContext, wasmCode []byte, sender types.Address) (api.Module, error) {
	ctx1 := context.Background()
	runtime := wazero.NewRuntime(ctx1)

	// 编译WASM模块
	compiled, err := runtime.CompileModule(ctx1, wasmCode)
	if err != nil {
		return nil, fmt.Errorf("编译WebAssembly模块失败: %w", err)
	}

	// 创建导入对象
	builder := runtime.NewHostModuleBuilder("env")

	// 添加内存
	builder.NewFunctionBuilder().
		WithFunc(func() uint32 {
			return 0
		}).Export("memory")

	// 添加宿主函数
	builder.NewFunctionBuilder().
		WithParameterNames("funcID", "argPtr", "argLen", "bufferPtr").
		WithResultNames("result").
		WithFunc(func(_ context.Context, m api.Module, funcID, argPtr, argLen, bufferPtr uint32) int32 {
			fmt.Printf("call_host_set: %d, %d, %d, %d\n", funcID, argPtr, argLen, bufferPtr)
			// 读取参数数据
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
			fmt.Printf("call_host_get_buffer: %d, %d, %d, %d\n", funcID, argPtr, argLen, buffer)
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

	// 实例化导入对象
	envModule, err := builder.Instantiate(vm.ctx)
	if err != nil {
		return nil, fmt.Errorf("实例化导入对象失败: %w", err)
	}
	vm.envModule = envModule

	// 初始化WASI
	wasi_snapshot_preview1.MustInstantiate(vm.ctx, runtime)

	// 创建模块配置，使用合约地址作为模块名称的一部分
	moduleName := fmt.Sprintf("contract_%x", sender)
	config := wazero.NewModuleConfig().
		WithName(moduleName).WithStdout(os.Stdout).WithStderr(os.Stderr)

	// 实例化模块
	module, err := runtime.InstantiateModule(ctx1, compiled, config.WithStartFunctions("_initialize"))
	if err != nil {
		fmt.Printf("实例化模块失败: %v\n", err)
		return nil, fmt.Errorf("实例化模块失败: %w", err)
	}
	return module, nil
}

// ExecuteContract 执行已部署的合约函数
func (vm *WazeroVM) ExecuteContract(ctx types.BlockchainContext, contractAddr types.Address, sender types.Address, functionName string, params []byte) (interface{}, error) {
	// 检查合约是否存在
	vm.contractsLock.RLock()
	wasmCode, exists := vm.contracts[contractAddr]
	vm.contractsLock.RUnlock()

	if !exists {
		return nil, fmt.Errorf("合约不存在: %x", contractAddr)
	}

	module, err := vm.initContract(ctx, wasmCode, sender)
	if err != nil {
		return types.Address{}, fmt.Errorf("实例化WebAssembly模块失败: %w", err)
	}

	result, err := vm.callWasmFunction(ctx, module, functionName, params)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}
	var runResult types.ExecutionResult
	err = json.Unmarshal(result, &runResult)
	if err != nil {
		return nil, fmt.Errorf("反序列化失败: %w", err)
	}
	return runResult.Data, nil
}

// generateContractAddress 生成合约地址
func (vm *WazeroVM) generateContractAddress(code []byte, _ types.Address) types.Address {
	var addr types.Address
	hash := sha256.Sum256(code)
	copy(addr[:], hash[:])
	return addr
}

// callWasmFunction 调用WASM函数
func (vm *WazeroVM) callWasmFunction(ctx types.BlockchainContext, module api.Module, functionName string, params []byte) ([]byte, error) {
	fmt.Printf("调用合约函数:%s, %v\n", functionName, string(params))

	// 检查是否导出了allocate和deallocate函数
	allocate := module.ExportedFunction("allocate")
	if allocate == nil {
		return nil, fmt.Errorf("没有allocate函数")
	}

	processDataFunc := module.ExportedFunction("handle_contract_call")
	if processDataFunc == nil {
		return nil, fmt.Errorf("handle_contract_call没找到")
	}

	var input types.HandleContractCallParams
	input.Contract = ctx.ContractAddress()
	input.Function = functionName
	input.Args = params
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("handle_contract_call 序列化失败: %w", err)
	}

	// 分配内存并写入参数
	result, err := allocate.Call(vm.ctx, uint64(len(inputBytes)))
	if err != nil {
		return nil, fmt.Errorf("内存分配失败: %w", err)
	}
	inputAddr := uint32(result[0])

	// 写入参数数据
	if !module.Memory().Write(inputAddr, inputBytes) {
		return nil, fmt.Errorf("写入内存失败")
	}

	// 调用处理函数
	result, err = processDataFunc.Call(vm.ctx, uint64(inputAddr), uint64(len(inputBytes)))
	if err != nil {
		return nil, fmt.Errorf("执行%s失败: %w", functionName, err)
	}

	var out []byte
	resultLen := int32(result[0])
	// if resultLen > int32(types.HostBufferSize) {
	// 	resultLen = resultLen - int32(types.HostBufferSize)
	// }
	if resultLen > 0 {
		getBufferAddress := module.ExportedFunction("get_buffer_address")
		if getBufferAddress == nil {
			return nil, fmt.Errorf("没有get_buffer_address函数")
		}

		result, err = getBufferAddress.Call(vm.ctx)
		if err != nil {
			return nil, fmt.Errorf("get_buffer_address失败: %w", err)
		}
		bufferPtr := uint32(result[0])

		// 读取结果数据
		data, ok := module.Memory().Read(bufferPtr, uint32(resultLen))
		if !ok {
			return nil, fmt.Errorf("读取内存失败:%d, len:%d", bufferPtr, resultLen)
		}
		out = data
	}

	// 释放内存
	deallocate := module.ExportedFunction("deallocate")
	if deallocate == nil {
		return nil, fmt.Errorf("没有deallocate函数")
	}
	_, err = deallocate.Call(vm.ctx, uint64(inputAddr), uint64(len(inputBytes)))
	if err != nil {
		return nil, fmt.Errorf("释放内存失败: %w", err)
	}

	fmt.Printf("执行结束:%s, %v\n", functionName, resultLen)
	return out, nil
}

// 宿主函数处理器
func (vm *WazeroVM) handleHostSet(ctx types.BlockchainContext, m api.Module, funcID uint32, argData []byte, bufferPtr uint32) int32 {
	// 根据函数ID处理不同的操作
	switch types.WasmFunctionID(funcID) {
	case types.FuncTransfer:
		var params types.TransferParams
		if err := json.Unmarshal(argData, &params); err != nil {
			return -1
		}
		err := ctx.Transfer(params.From, params.To, params.Amount)
		if err != nil {
			return -1
		}
		return 0
	case types.FuncCall:
		var params types.CallParams
		if err := json.Unmarshal(argData, &params); err != nil {
			return -1
		}
		result, err := ctx.Call(params.Caller, params.Contract, params.Function, params.Args...)
		if err != nil {
			return -1
		}
		if !m.Memory().Write(bufferPtr, result) {
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
		if obj.Contract() != params.Contract {
			return -1
		}
		if obj.Owner() != ctx.Sender() && obj.Owner() != params.Contract && obj.Owner() != params.Sender {
			return -1
		}
		err = obj.SetOwner(params.Owner)
		if err != nil {
			return -1
		}
		return 0

	case types.FuncSetObjectField:
		var params types.SetObjectFieldParams
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
		if obj.Owner() != ctx.Sender() && obj.Owner() != params.Contract && obj.Owner() != params.Sender {
			return -1
		}
		// 将value转换为[]byte
		valueBytes, err := json.Marshal(params.Value)
		if err != nil {
			return -1
		}
		err = obj.Set(params.Field, valueBytes)
		if err != nil {
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
	// 根据函数ID处理不同的操作
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
			fmt.Printf("obj.getfield 反序列化失败: %v\n", err)
			return -1
		}
		if params.ID == (types.ObjectID{}) {
			copy(params.ID[:], params.Contract[:])
		}
		obj, err := ctx.GetObject(params.Contract, params.ID)
		if err != nil {
			fmt.Printf("obj.getfield 获取对象失败:id:%x, %v\n", params.ID, err)
			return -1
		}
		data, err := obj.Get(params.Field)
		if err != nil {
			fmt.Printf("obj.getfield 获取字段失败:id:%x, %v\n", params.ID, err)
			return -1
		}
		mem.Write(offset, data)
		fmt.Printf("obj.getfield 获取字段成功:id:%x, %s\n", params.ID, string(data))
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

// Close 关闭虚拟机
func (vm *WazeroVM) Close() error {
	if vm.envModule != nil {
		if err := vm.envModule.Close(vm.ctx); err != nil {
			return fmt.Errorf("failed to close env module: %w", err)
		}
	}
	return nil
}
