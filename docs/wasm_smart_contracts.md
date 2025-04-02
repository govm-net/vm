# WebAssembly 智能合约系统

本文档是 VM 项目 WebAssembly (WASM) 智能合约系统的主入口点，提供了对系统架构、核心组件和详细文档的整体导航。

## 1. 系统概述

VM 项目的 WebAssembly 智能合约系统允许开发者使用 Go 语言编写智能合约，并通过 TinyGo 编译为 WebAssembly 模块在沙箱环境中执行。该系统具有高安全性、接近原生的性能、良好的跨平台兼容性和精确的资源控制能力。

```mermaid
flowchart TD
    A[Go源码合约] --> B[TinyGo编译]
    B --> C[WASM模块]
    C --> D[wazero运行时]
    D <--> E[区块链状态]
    
    subgraph 开发者环境
        A
    end
    
    subgraph 编译链
        B
        C
    end
    
    subgraph 执行环境
        D
        E
    end
    
    style A fill:#d0f0c0
    style B fill:#d0f0c0
    style C fill:#c0d0f0
    style D fill:#f0d0c0
    style E fill:#f0d0c0
```

## 2. 文档导航

本系统文档按照从基础到高级的顺序组织，覆盖了合约开发、编译、部署和执行的完整生命周期：

1. **[基础接口系统](wasm_contract_interface.md)** - 介绍合约代码与区块链环境之间的通信桥梁
   * 合约侧接口 (Context, Object)
   * 主机侧接口
   * 内存管理机制
   * 系统调用分类

2. **[合约执行流程](wasm_contract_execution.md)** - 详细说明从编译到执行的完整生命周期
   * 编译流程
   * 部署流程
   * 执行流程
   * 参数传递
   * 资源控制

3. **[调用链追踪机制](wasm_contract_tracing.md)** - 解释自动插桩技术和调用链追踪
   * 自动插桩原理
   * Context接口增强
   * 合约间调用信息传递
   * 参数序列化与反序列化
   * 实际应用场景

4. **[WASI合约详解](wasi_contracts.md)** - 深入探讨基于WASI规范的合约模式
   * 合约示例
   * 编译到WebAssembly的内部流程
   * WebAssembly优势
   * 配置与使用
   * 最佳实践

## 3. 核心概念统一

为解决文档间的概念冲突和不一致，以下是核心概念的规范定义：

### 3.1 接口体系

系统提供了一套统一的接口体系，主要包括：

```go
// Context接口 - 提供访问区块链状态和功能的标准方法
type Context interface {
    // 区块链信息相关
    BlockHeight() uint64         // 获取当前区块高度
    BlockTime() int64            // 获取当前区块时间戳
    ContractAddress() Address    // 获取当前合约地址
    
    // 账户操作相关
    Sender() Address             // 获取交易发送者或调用合约
    Balance(addr Address) uint64 // 获取账户余额
    Transfer(to Address, amount uint64) error // 转账操作
    
    // 对象存储相关 - 基础状态操作使用panic而非返回error
    CreateObject() Object                    // 创建新对象，失败时panic
    GetObject(id ObjectID) (Object, error)   // 获取指定对象，可能返回error
    GetObjectWithOwner(owner Address) (Object, error) // 按所有者获取对象，可能返回error
    DeleteObject(id ObjectID)                // 删除对象，失败时panic
    
    // 跨合约调用
    Call(contract Address, function string, args ...any) ([]byte, error)
    
    // 日志与事件
    Log(eventName string, keyValues ...interface{}) // 记录事件
}

// Object接口 - 提供状态对象的操作方法
type Object interface {
    ID() ObjectID           // 获取对象ID
    Owner() Address         // 获取对象所有者
    SetOwner(addr Address)  // 设置对象所有者，失败时panic
    
    // 字段操作
    Get(field string, value any) error  // 获取字段值
    Set(field string, value any) error  // 设置字段值
}
```

### 3.2 统一的内存管理模型

本系统采用以下内存管理策略，在所有相关文档中保持一致：

- 为减少内存压力，合约代码应尽量重用缓冲区而非频繁分配内存
- WebAssembly 模块限制最大内存使用（默认上限为 128 MB）
- 内存管理包括 WebAssembly 线性内存和共享的主机缓冲区两部分

### 3.3 参数传递统一机制

为统一不同文档描述的参数传递机制，系统采用类似Go标准库RPC的工作流程：

1. **基于RPC模型的参数结构体**：
   - 自动为每个导出函数生成对应的参数结构体
   - 参数结构体包含Call Info用于传递调用链信息
   - 字段名称与原始参数名称一致

2. **自动化参数处理**：
   - 在合约编译阶段自动生成参数序列化/反序列化代码
   - 使用函数分发表实现高效调用路由
   - 支持丰富的错误处理和类型安全措施

3. **类型安全的序列化**：
   - 使用类型注册表确保类型信息保留
   - 避免JSON反序列化的数值类型问题（如将整数转为float64）
   - 支持复杂嵌套结构体的类型安全处理

```go
// 导出函数示例 - 使用大写字母开头即可，无需//export标记
func Transfer(ctx Context, to Address, amount uint64) error {
    // 函数实现...
}

// 自动生成的参数结构体
type TransferParams struct {
    CallInfo *CallInfo `json:"call_info"` // 自动注入的调用链信息
    To       Address   `json:"to"`        // 参数1
    Amount   uint64    `json:"amount"`    // 参数2
}

// 自动生成的方法处理器
func handleTransfer(ctx Context, paramsJSON []byte) int32 {
    var params TransferParams
    if err := json.Unmarshal(paramsJSON, &params); err != nil {
        // 错误处理...
        return ErrorCodeInvalidParams
    }
    
    // 设置调用上下文
    setCurrentCallInfo(params.CallInfo)
    
    // 调用实际函数
    err := Transfer(ctx, params.To, params.Amount)
    
    // 处理返回值...
}
```

这种基于Go RPC模型的参数处理机制提供了多项优势：
- 开发者只需使用标准Go函数命名规范（大写开头的函数自动导出），无需添加特殊注释
- 系统自动识别并导出所有大写开头的函数，简化开发流程
- 类型安全由系统保证，避免常见的JSON类型转换问题
- 参数验证和错误处理统一规范

### 3.4 函数导出规则简化

系统采用了Go语言规范的公共/私有标识方法，简化了合约函数的导出机制：

1. **自动导出规则**：
   - 大写字母开头的函数自动被视为导出函数，可被外部调用
   - 小写字母开头的函数为私有函数，仅合约内部可访问
   - 无需添加特殊的 `//export` 注释标记

2. **框架自动包装**：
   - 编译系统自动识别所有大写开头的函数
   - 为每个导出函数生成必要的包装代码
   - 自动注册函数，使其可被主机环境调用

3. **优势**：
   - 更符合Go语言习惯
   - 减少样板代码
   - 避免导出标记与实际导出不一致的问题
   - 简化合约开发流程

示例：
```go
// 公开函数 - 自动导出，外部可调用
func Transfer(to Address, amount uint64) error {
    return performTransfer(to, amount)
}

// 私有函数 - 不导出，仅内部使用
func performTransfer(to Address, amount uint64) error {
    // 实现转账逻辑...
}
```

### 3.5 错误处理统一框架

系统采用统一的错误处理模式：

- 合约内部使用 Go 的 error 类型返回错误
- 跨合约调用返回标准错误码和详细错误信息
- 所有错误信息可通过 Context.Log 记录
- 异常状态通过特定的返回值（通常为负数）表示

## 4. 版本与兼容性

当前文档适用于 VM 项目 v1.0.0 版本。系统保持向后兼容，但以下方面可能存在版本差异：

- TinyGo 版本：推荐使用 0.29.0 或更高版本
- wazero 运行时：推荐使用 2.3.0 或更高版本
- Go 语言：推荐使用 1.20 或更高版本

## 5. 文档更新计划

本文档将按照以下计划持续更新：

- **短期更新**：修复已知的文档不一致问题
- **中期增强**：添加完整的代码示例库和测试框架文档
- **长期规划**：增加合约升级机制、高级优化指南和与其他系统的互操作性文档

## 6. 扩展阅读

- [智能合约设计模式](/)（待完成）
- [合约升级与版本控制](/)（待完成）
- [合约测试与调试指南](/)（待完成）
- [安全最佳实践](/)（待完成）
- [合约性能优化](/)（待完成）

## 7. 附录：关键术语表

| 术语 | 定义 |
|------|------|
| WebAssembly | 可移植的二进制指令格式，作为智能合约执行的目标格式 |
| WASI | WebAssembly 系统接口，提供标准化的系统调用 |
| TinyGo | Go 语言的编译器，针对小内存环境优化，用于将 Go 代码编译为 WebAssembly |
| wazero | WebAssembly 运行时，用于执行 WebAssembly 模块 |
| 插桩 | 自动向合约代码中注入额外指令的过程，用于追踪调用链和增强功能 |
| 调用链 | 跨合约调用的执行路径，记录从起始调用到当前执行点的所有合约和函数 |
| 沙箱 | 隔离的执行环境，限制合约对外部资源的访问 |

## 8. 文档更新记录

### 2023年11月更新

本文档系统进行了全面的标准化和一致性更新，主要包括：

1. **统一Context和Object接口定义**：确保所有文档中接口签名一致，规范化核心接口的使用方式
2. **统一参数传递机制**：建立了基于Go RPC风格的统一参数传递模型，增强类型安全性
3. **调用链追踪规范**：完善了跨合约调用中的调用链信息传递机制，增强安全审计能力
4. **内存管理模型**：统一了内存分配和释放规范，明确了内存管理职责
5. **资源控制策略**：规范化了各种资源的使用限制和计费方式

### 2023年12月更新

进一步优化和简化了WebAssembly智能合约系统，主要改进：

1. **函数导出机制简化**：采用Go语言的规范，所有大写字母开头的函数会被自动识别为导出函数，无需手动添加//export标记
2. **自动参数结构体生成**：系统现在能够自动分析导出函数的参数，生成对应的参数结构体，简化开发流程
3. **统一的RPC风格调用**：所有合约函数调用采用统一的RPC风格接口，增强了类型安全性和错误处理能力
4. **参数序列化与反序列化**：完善了参数处理流程，确保类型信息得到保留，避免常见的JSON类型转换问题
5. **编译时自动代码生成**：自动为导出函数生成包装代码、参数结构体和结果处理逻辑，减少重复编码工作

这些更新使WebAssembly智能合约系统更加易用、安全和高效，让开发者能够专注于业务逻辑实现，而不必过多关注底层调用细节。

## 4. 参数处理机制

通过统一的参数处理机制，VM系统确保了跨合约调用的类型安全和调用链信息传递。

### 4.1 参数序列化过程

在合约调用过程中，系统会自动处理参数序列化：

```go
// 调用示例
ctx.Call(tokenContractAddr, "Transfer", toAddress, amount)
```

系统自动执行以下步骤：

1. **为导出函数自动生成参数结构体**:
   - 系统在编译阶段为每个大写字母开头的函数生成对应的参数结构体
   - 结构体字段与函数参数一一对应，保持类型信息
   - 自动包含调用链信息字段

   ```go
   // 自动生成的参数结构体
   type TransferParams struct {
       CallInfo *CallInfo `json:"call_info"` // 自动注入的调用链信息
       To Address `json:"to"`                // 第一个参数
       Amount uint64 `json:"amount"`         // 第二个参数
   }
   ```

2. **构建调用信息**:
   - 记录调用者合约地址、函数名
   - 维护完整调用链，确保可追踪性

3. **参数包装与序列化**:
   - 将调用参数填充到生成的结构体中
   - 使用JSON进行序列化，保留完整类型信息
   - 支持各种复杂类型，包括自定义结构体和嵌套结构

### 4.2 参数反序列化过程

目标合约接收到请求后，系统自动完成参数反序列化：

```go
// 自动生成的函数处理器
func handleTransfer(paramsJSON []byte) int32 {
    // 自动反序列化
    var params TransferParams
    if err := json.Unmarshal(paramsJSON, &params); err != nil {
        return ErrorCodeInvalidParams
    }
    
    // 提取调用链信息
    callInfo := params.CallInfo
    
    // 调用实际函数
    err := Transfer(params.To, params.Amount)
    
    // 处理结果...
    return StatusSuccess
}
```

此过程的主要特点：

1. **直接反序列化到结构体**:
   - 无需手动解析JSON或处理类型转换
   - 系统自动将数据映射到正确的类型

2. **调用链信息提取**:
   - 自动从参数中提取调用链信息
   - 可用于权限检查和审计

3. **类型安全保证**:
   - 严格按照函数签名进行类型匹配
   - 避免JSON反序列化中常见的类型问题（如将数字默认转为float64）

### 4.3 自定义结构体参数处理

系统能够完整支持自定义结构体作为参数：

```go
// 合约中的自定义结构体
type TokenTransfer struct {
    From    Address `json:"from"`
    To      Address `json:"to"`
    TokenID uint64  `json:"token_id"`
}

// 导出函数使用自定义结构体
func TransferToken(transfer TokenTransfer) error {
    // 业务逻辑...
}
```

自定义结构体参数处理：

1. **保留完整类型信息**:
   - 结构体字段名称和类型在序列化过程中得到保留
   - 支持嵌套结构体和复杂字段类型

2. **自动递归序列化**:
   - 系统能够处理任意深度的嵌套结构
   - 保持引用和指针类型的正确性

3. **验证与默认值**:
   - 支持结构体标签中的验证规则
   - 可设置字段默认值和必填检查

### 4.4 性能优化

参数处理机制包含多项性能优化：

1. **缓冲区复用**:
   - 使用预分配的全局缓冲区减少内存分配
   - 动态调整缓冲区大小以适应不同规模的参数

2. **类型缓存**:
   - 缓存反射信息避免重复类型解析
   - 预编译的类型路径减少运行时开销

3. **延迟反序列化**:
   - 只有在实际需要时才执行完整反序列化
   - 支持按需加载大型数据结构

4. **二进制序列化选项**:
   - 可选支持高性能二进制序列化格式
   - 适用于高频调用的关键路径优化

这种自动化的参数处理机制显著简化了合约开发过程，使开发者能够专注于业务逻辑，同时确保类型安全和跨合约调用的一致性。
