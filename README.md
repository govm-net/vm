# VM

一个轻量级、高性能的虚拟机，专为执行 Go 语言编写的智能合约设计，将其编译为 WebAssembly 模块并通过 WASI 规范执行。

## 概述

VM 项目提供了一个专注于确定性执行的虚拟机环境，具有以下特点：

- **基于 WebAssembly 的执行**：将 Go 合约编译为 WebAssembly 模块，通过 Wasmer 运行时执行，提供高性能和强大的安全隔离
- **状态对象模型**：合约操作状态对象而非直接维护状态，增强代码重用性和可扩展性
- **对象所有权**：基于访问控制的所有权模型，保障数据安全和隐私
- **可序列化的参数和结果**：支持丰富的数据类型，包括基本类型、数组、结构体和自定义类型
- **确定性执行环境**：保证相同输入产生相同结果，适合区块链等分布式系统
- **持久化存储接口**：支持多种数据存储后端，保存合约状态
- **能源计量系统**：控制和限制合约执行资源消耗

## 项目架构

VM 项目分为几个关键部分：

### 核心层（core）

- **context.go**: 定义合约执行上下文接口
- **contract.go**: 包含核心数据类型和接口定义
- **object.go**: 实现状态对象模型
- **error.go**: 标准错误类型和错误处理

### VM 实现（vm）

- **engine.go**: 虚拟机的主要实现，包括合约部署、实例化和执行
- **params.go**: 参数编码和解码
- **state.go**: 状态管理和持久化
- **wasi.go**: WebAssembly 系统接口实现
- **api/**: 公共 API 和配置接口

### 示例合约（contracts）

- **counter/**: 简单计数器合约示例
- **token/**: 基本代币合约实现
- **nft/**: 非同质化代币合约示例

## 启发自 GoVM

本项目的设计受到了 GoVM 的启发，同时引入了多项增强功能：

- **WebAssembly 执行模型**：VM 通过将 Go 合约编译为 WebAssembly，结合了 Go 语言的易用性和 WebAssembly 的安全隔离性
- **增强的参数编解码**：支持更广泛的数据类型传递，包括嵌套结构和自定义类型
- **对象所有权模型**：实现细粒度访问控制，确保只有授权实体能修改对象状态
- **持久化存储机制**：为状态对象提供可靠的存储后端，支持不同的数据库实现

## 入门指南

要开始使用 VM：

1. 确保安装了 Go 1.18+、TinyGo 和 Wasmer 运行时
2. 克隆仓库并构建项目
3. 查看 `docs/getting_started.md` 指南
4. 尝试运行示例合约

示例代码：

```go
package main

import (
    "fmt"
    "github.com/govm-net/vm/core"
    "github.com/govm-net/vm/vm"
    "github.com/govm-net/vm/vm/api"
)

func main() {
    // 创建虚拟机配置
    config := api.DefaultContractConfig()
    config.EnableWASIContracts = true
    
    // 初始化虚拟机
    engine := vm.NewEngine(config)
    
    // 部署合约
    code := [] byte(`package counter

import "github.com/govm-net/vm/core"

// Counter 合约
type Counter struct{}

func (c *Counter) Initialize(ctx core.Context) (core.ObjectID, error) {
    // 创建并初始化计数器对象
    obj, err := ctx.CreateObject()
    if err != nil {
        return core.ObjectID{}, err
    }
    
    err = obj.Set("count", uint64(0))
    if err != nil {
        return core.ObjectID{}, err
    }
    
    return obj.ID(), nil
}

func (c *Counter) Increment(ctx core.Context, id core.ObjectID) (uint64, error) {
    // 获取对象
    obj, err := ctx.GetObject(id)
    if err != nil {
        return 0, err
    }
    
    // 读取当前值
    val, err := obj.Get("count")
    if err != nil {
        return 0, err
    }
    count := val.(uint64)
    
    // 递增并保存
    count++
    err = obj.Set("count", count)
    if err != nil {
        return 0, err
    }
    
    return count, nil
}`)

    // 部署为 WebAssembly 合约
    deployOptions := vm.DeployOptions{
        AsWASI: true,
        WASIOptions: vm.WASIOptions{
            MemoryLimit: 64 * 1024 * 1024, // 64MB
        },
    }
    addr, err := engine.DeployWithOptions(code, deployOptions)
    if err != nil {
        panic(err)
    }
    
    // 初始化计数器
    result, err := engine.ExecuteWithArgs(addr, "Initialize")
    if err != nil {
        panic(err)
    }
    
    // 解码结果获取对象ID
    var id core.ObjectID
    if err := vm.DecodeResult(result, &id); err != nil {
        panic(err)
    }
    
    // 递增计数器
    result, err = engine.ExecuteWithArgs(addr, "Increment", id)
    if err != nil {
        panic(err)
    }
    
    // 解码结果
    var count uint64
    if err := vm.DecodeResult(result, &count); err != nil {
        panic(err)
    }
    
    fmt.Printf("Counter value: %d\n", count)
}
```

## 文档

- [VM 架构文档](docs/vm_architecture.md) - 详细介绍 VM 的设计理念和架构
- [Getting Started Guide](docs/getting_started.md) - 逐步构建您的第一个智能合约
- [WebAssembly 合约](docs/wasi_contracts.md) - WebAssembly 合约的详细指南

## 使用场景

VM 适用于以下场景：

- 区块链智能合约执行环境
- 确定性业务规则引擎
- 安全沙箱中的第三方代码执行
- 分布式系统中的可验证计算
- 微服务架构中的可插拔业务逻辑

## 未来发展计划

- **资源计量增强**：完善能源消耗计算和限制机制
- **合约模板库**：提供常用合约模板
- **状态管理优化**：改进大型状态处理性能
- **跨合约调用**：增强合约间通信和调用机制
- **标准库扩展**：扩充确定性运行时标准库

## 贡献

欢迎贡献代码、报告问题或提出建议。请先查看我们的贡献指南。

## 许可证

本项目采用 MIT 许可证 - 详情参见 LICENSE 文件。
