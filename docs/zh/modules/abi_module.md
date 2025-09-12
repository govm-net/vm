# ABI 生成模块设计文档

## 1. 概述

ABI（Application Binary Interface）生成模块负责从 Go 源码中提取合约的接口信息，并生成标准的 ABI 对象。该模块是智能合约系统的重要组成部分，为合约的部署、执行和交互提供必要的接口信息。

## 2. 模块职责

1. 解析 Go 源码并提取合约接口信息
2. 生成标准的 ABI 对象
3. 验证合约函数签名的合法性

## 3. 核心接口

```go
// ABIExtractor 定义了 ABI 提取器的接口
type ABIExtractor interface {
    // Extract 从源码中提取 ABI 信息
    Extract(code []byte) (*ABI, error)
}

// ABI 表示合约的应用程序二进制接口
type ABI struct {
    PackageName string     `json:"package_name,omitempty"`
    Functions   []Function `json:"functions,omitempty"`
    Events      []Event    `json:"events,omitempty"`
}

// Function 表示合约中的函数
type Function struct {
    Name    string      `json:"name,omitempty"`
    Inputs  []Parameter `json:"inputs,omitempty"`
    Outputs []Parameter `json:"outputs,omitempty"`
}

// Parameter 表示函数参数或事件字段
type Parameter struct {
    Name string `json:"name,omitempty"`
    Type string `json:"type,omitempty"`
}
```

## 4. 实现细节

### 4.1 AST 解析

使用 Go 标准库的 `go/parser` 和 `go/ast` 包来解析 Go 源码：

1. 解析源码为 AST（抽象语法树）
2. 遍历 AST 节点，提取函数声明
3. 过滤非导出函数（小写字母开头的函数）
4. 提取函数的输入和输出参数

### 4.2 函数信息提取

对于每个导出函数，提取以下信息：

1. 函数名称
2. 输入参数列表（参数名和类型）
3. 输出参数列表（参数名和类型）

### 4.3 事件信息提取

通过分析函数体中的 `core.Log` 调用来提取事件信息：

1. 查找 `core.Log` 函数调用
2. 提取事件名称（第一个参数）
3. 提取事件参数（后续的键值对参数）

## 5. 使用示例

```go
// 创建 ABI 提取器实例
extractor := NewABIExtractor()

// 合约源码
code := []byte(`package counter

import "github.com/govm-net/vm/core"

func Initialize() int32 {
    obj := core.CreateObject()
    obj.Set("value", uint64(0))
    core.Log("initialized", "value", 0)
    return 0
}

func Increment(delta uint64) uint64 {
    obj, _ := core.GetObject(core.ObjectID{})
    var value uint64
    obj.Get("value", &value)
    newValue := value + delta
    obj.Set("value", newValue)
    core.Log("incremented", "from", value, "to", newValue)
    return newValue
}`)

// 提取 ABI
abi, err := extractor.Extract(code)
if err != nil {
    log.Fatal(err)
}

// 输出 ABI 信息
fmt.Printf("Package: %s\n", abi.PackageName)
for _, fn := range abi.Functions {
    fmt.Printf("Function: %s\n", fn.Name)
    fmt.Printf("  Inputs: %v\n", fn.Inputs)
    fmt.Printf("  Outputs: %v\n", fn.Outputs)
}
```

## 6. 错误处理

ABI 提取过程中可能遇到的错误：

1. 源码语法错误：返回解析错误
2. 不支持的语法特性：返回不支持错误
3. 内部处理错误：返回相应错误信息

## 7. 性能优化

1. AST 解析缓存：对于相同的源码，缓存解析结果
2. 并发安全：支持多协程并发提取 ABI
3. 内存优化：及时释放不需要的 AST 节点

## 8. 扩展性

1. 支持自定义事件提取规则
2. 支持新的参数类型
3. 支持注解形式的 ABI 信息