# 合约执行模块详细设计文档

## 1. 模块概述

合约执行模块是 VM 系统中负责执行编译后的 WebAssembly 智能合约的核心组件。该模块基于 wazero 运行时环境，提供安全、高效的合约执行能力，同时实现了资源限制、调用链追踪和错误处理等重要功能。

## 2. 模块职责

1. **合约执行环境管理**：创建和维护 WebAssembly 执行环境
2. **合约函数调用**：执行指定的合约函数并处理参数
3. **资源限制控制**：实施内存、执行时间和指令计数等资源限制
4. **调用链管理**：维护合约间的调用关系和上下文信息
5. **错误处理与恢复**：处理执行过程中的各种异常情况

## 3. 核心接口设计

### 3.1 ContractExecutor 接口

```go
// ContractExecutor 定义了合约执行器的接口
type ContractExecutor interface {
    // Execute 执行合约函数
    Execute(ctx Context, contractAddr Address, functionName string, params []byte) (interface{}, error)
}
```

### 3.2 核心接口方法说明

#### 3.2.1 Execute 方法

```go
Execute(ctx Context, contractAddr Address, functionName string, params []byte) (interface{}, error)
```

**功能描述**：执行指定合约的函数

**参数说明**：
- `ctx`：区块链上下文环境
- `contractAddr`：合约地址
- `functionName`：要执行的函数名称
- `params`：函数参数数据

**返回值**：
- `interface{}`：函数执行结果
- `error`：成功返回 nil，失败返回错误信息

## 4. 执行环境设计

### 4.1 wazero 运行时集成

合约执行模块基于 wazero WebAssembly 运行时实现，具有以下优势：

1. **纯 Go 实现**：无需外部依赖，简化部署和维护
2. **高性能**：接近原生性能的执行速度
3. **安全性**：内置内存安全和沙箱执行机制
4. **跨平台**：支持多种操作系统和架构

### 4.2 执行环境初始化

```go
type WazeroVM struct {
    contractDir string
    ctx         context.Context
    envModule   api.Module
}
```

## 5. 合约执行流程

### 5.1 执行准备阶段

1. **加载合约模块**：从存储中加载指定地址的 WebAssembly 模块
2. **设置执行环境**：配置资源限制和主机函数
3. **准备参数数据**：序列化函数参数

### 5.2 执行阶段

1. **函数调用**：调用指定的合约函数
2. **主机函数处理**：处理合约对主机环境的调用
3. **资源监控**：实时监控资源使用情况

### 5.3 结果处理阶段

1. **结果解析**：反序列化执行结果
2. **错误处理**：处理执行过程中产生的错误
3. **资源统计**：统计资源使用情况

## 6. 资源限制机制

### 6.1 内存限制

通过 wazero 的内存限制功能控制合约可使用的最大内存量：

```go
// WebAssembly资源限制选项
type WASIOptions struct {
    MemoryLimit     uint64    // 内存限制 (字节)
    TableSize       uint64    // 函数表大小
    Timeout         uint64    // 执行超时 (毫秒)
    FuelLimit       uint64    // 指令计数限制
    StackSize       uint64    // 栈大小 (字节)
}
```

### 6.2 执行时间限制

通过上下文超时机制限制合约执行时间：

```go
ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
defer cancel()
```

### 6.3 指令计数限制

使用燃料机制限制合约可执行的指令数量：

```go
// 启用燃料限制
runtime := wazero.NewRuntimeWithConfig(ctx, config.WithFeatureBulkMemoryOperations(false))
```

## 7. 调用链管理

### 7.1 调用链追踪

系统自动追踪合约间的调用关系：

```go
// 记录函数入口
mock.Enter(contract.String(), functionName)

// 记录函数出口
mock.Exit(contract.String(), functionName)
```

### 7.2 上下文传递

在合约调用过程中正确传递上下文信息：

```go
// 设置当前合约地址
mock.SetCurrentContract(contractAddr)

// 设置调用者地址
mock.SetCaller(callerAddr)
```

## 8. 错误处理机制

### 8.1 执行错误分类

1. **编译错误**：合约代码编译失败
2. **运行时错误**：合约执行过程中出现的错误
3. **资源耗尽错误**：超出资源限制导致的错误
4. **业务逻辑错误**：合约业务逻辑产生的错误

### 8.2 异常恢复

通过 panic/recover 机制处理执行过程中的异常：

```go
defer func() {
    if r := recover(); r != nil {
        // 处理异常情况
        handleExecutionPanic(r)
    }
}()
```

## 9. 主机函数接口

### 9.1 主机函数注册

在 WebAssembly 环境中注册主机函数：

```go
// 注册主机函数
envBuilder.NewFunctionWithCallContext(
    "call_host_set",
    callHostSet,
    []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32},
    []api.ValueType{api.ValueTypeI32},
)
```

### 9.2 主机函数实现

主机函数的具体实现处理合约对区块链环境的访问请求：

```go
func callHostSet(ctx context.Context, module api.Module, funcID, argPtr, argLen, bufferPtr int32) int32 {
    // 获取参数数据
    argsData := readMemory(module, argPtr, argLen)
    
    // 根据函数ID处理不同请求
    switch funcID {
    case FuncGetSender:
        return handleGetSender(bufferPtr)
    case FuncTransfer:
        return handleTransfer(argsData, bufferPtr)
    // ... 其他函数处理
    }
    
    return ErrorCodeFunctionNotFound
}
```

## 10. 性能优化策略

### 10.1 模块缓存

缓存已加载的 WebAssembly 模块，避免重复加载：

```go
type ModuleCache struct {
    modules map[Address]api.Module
    mutex   sync.RWMutex
}
```

### 10.2 执行结果缓存

对频繁调用且结果不变的函数进行缓存：

```go
type ExecutionCache struct {
    results map[string]interface{}
    ttl     time.Duration
}
```

## 11. 安全机制

### 11.1 沙箱执行

通过 WebAssembly 的沙箱机制确保合约无法访问主机系统资源：

1. **内存隔离**：合约只能访问分配的线性内存
2. **函数限制**：只能调用允许的主机函数
3. **资源限制**：严格限制 CPU、内存等资源使用

### 11.2 权限控制

实施细粒度的权限控制：

1. **函数调用权限**：验证合约是否有权限调用特定函数
2. **状态访问权限**：验证合约是否有权限访问特定状态对象
3. **资金操作权限**：验证合约是否有权限执行资金转移操作

## 12. 监控与调试

### 12.1 执行统计

收集合约执行过程中的各种统计信息：

1. **执行时间**：合约函数的执行时间
2. **资源消耗**：内存、CPU 等资源使用情况
3. **调用次数**：函数调用频率统计

### 12.2 调试支持

提供调试信息输出和跟踪功能：

```go
// 启用调试模式
if debugMode {
    fmt.Printf("Executing contract %s function %s\n", contractAddr, functionName)
}
```

## 13. 总结

合约执行模块通过集成 wazero WebAssembly 运行时，提供了安全、高效的智能合约执行环境。模块实现了完善的资源限制、调用链追踪和错误处理机制，确保合约能够在受控环境中安全执行。通过多种性能优化策略，模块能够满足区块链系统对高并发、低延迟的要求。