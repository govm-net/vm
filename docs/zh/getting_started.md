# 快速入门

本指南将帮助您开始使用 VM 项目开发 WebAssembly 智能合约。

## 1. 环境要求

### 1.1 所需工具

- Go 1.20 或更高版本
- TinyGo 0.28.0 或更高版本
- Git

### 1.2 安装

```bash
# 安装 Go
brew install go

# 安装 TinyGo
brew install tinygo

# 安装 VM SDK
go get github.com/govm-net/vm
```

## 2. 快速开始

### 2.1 创建简单合约

创建新的合约目录：

```bash
mkdir counter
cd counter
```

初始化 Go 模块：

```bash
go mod init counter
```

创建简单的计数器合约：

```go
package counter

import (
    "github.com/govm-net/vm/core"
    "github.com/govm-net/vm/mock"
)

// 计数器状态
type Counter struct {
    Value int64
}

// 初始化计数器
func Init() {
    counter := Counter{Value: 0}
    object := core.CreateObject()
    object.Set("counter", counter)
}

// 增加计数器
func Increment() error {
    counter, err := getCounter()
    if err != nil {
        return err
    }
    
    counter.Value++
    object.Set("counter", counter)
    return nil
}

// 获取计数器值
func GetValue() (int64, error) {
    counter, err := getCounter()
    if err != nil {
        return 0, err
    }
    return counter.Value, nil
}
```

### 2.2 编译合约

使用 TinyGo 编译合约：

```bash
tinygo build -o counter.wasm -target=wasi -opt=z -no-debug .
```

### 2.3 部署合约

使用 VM CLI 部署合约：

```bash
vm deploy counter.wasm
```

### 2.4 与合约交互

使用 VM CLI 调用合约函数：

```bash
# 增加计数器
vm call <合约地址> Increment

# 获取计数器值
vm call <合约地址> GetValue
```

## 3. 开发指南

### 3.1 项目结构

典型的合约项目结构如下：

```
contract/
├── go.mod
├── contract.go
├── state.go
└── tests/
    └── contract_test.go
```

### 3.2 合约开发

1. **状态定义**：
   ```go
   type State struct {
       // 状态字段
   }
   ```

2. **初始化**：
   ```go
   func Init() {
       // 初始化状态
   }
   ```

3. **公共函数**：
   ```go
   func PublicFunction() error {
       // 函数实现
   }
   ```

4. **私有函数**：
   ```go
   func privateFunction() error {
       // 函数实现
   }
   ```

### 3.3 测试

1. **单元测试**：
   ```go
   func TestContract(t *testing.T) {
       // 测试初始化
       Init()
       
       // 测试函数
       err := PublicFunction()
       if err != nil {
           t.Fatal(err)
       }
   }
   ```

2. **集成测试**：
   ```go
   func TestContractIntegration(t *testing.T) {
       // 部署合约
       contract, err := deployContract("contract.wasm")
       if err != nil {
           t.Fatal(err)
       }
       
       // 测试合约函数
       result, err := contract.Call("PublicFunction", nil)
       if err != nil {
           t.Fatal(err)
       }
   }
   ```

## 4. 最佳实践

### 4.1 安全性

1. **输入验证**：
   ```go
   func ProcessInput(input []byte) error {
       // 验证输入大小
       if len(input) > MaxInputSize {
           return errors.New("输入过大")
       }
       
       // 验证输入内容
       if !isValidInput(input) {
           return errors.New("无效输入")
       }
       
       return nil
   }
   ```

2. **访问控制**：
   ```go
   func RestrictedFunction() error {
       // 检查调用者权限
       if !hasPermission(core.Sender()) {
           return errors.New("权限不足")
       }
       
       // 执行函数
       return nil
   }
   ```

### 4.2 性能

1. **状态管理**：
   ```go
   // 缓存状态
   var cachedState *State
   
   func getState() (*State, error) {
       if cachedState != nil {
           return cachedState, nil
       }
       
       object := core.GetObjectWithOwner(core.ContractAddress())
       var state State
       err := object.Get("state", &state)
       if err != nil {
           return nil, err
       }
       
       cachedState = &state
       return cachedState, nil
   }
   ```

2. **Gas 优化**：
   ```go
   func OptimizedFunction() error {
       // 批量操作
       updates := make([]StateUpdate, 0)
       
       // 收集更新
       for _, item := range items {
           updates = append(updates, StateUpdate{
               Field: item.Field,
               Value: item.Value,
           })
       }
       
       // 批量应用更新
       return applyUpdates(updates)
   }
   ```

## 5. 示例

### 5.1 简单存储

```go
package storage

import (
    "github.com/govm-net/vm/core"
    "github.com/govm-net/vm/mock"
)

// 存储状态
type Storage struct {
    Data map[string]string
}

// 初始化存储
func Init() {
    storage := Storage{
        Data: make(map[string]string),
    }
    object := core.CreateObject()
    object.Set("storage", storage)
}

// 存储数据
func Store(key, value string) error {
    storage, err := getStorage()
    if err != nil {
        return err
    }
    
    storage.Data[key] = value
    object.Set("storage", storage)
    return nil
}

// 检索数据
func Retrieve(key string) (string, error) {
    storage, err := getStorage()
    if err != nil {
        return "", err
    }
    
    value, exists := storage.Data[key]
    if !exists {
        return "", errors.New("键不存在")
    }
    
    return value, nil
}
```

### 5.2 代币合约

```go
package token

import (
    "github.com/govm-net/vm/core"
    "github.com/govm-net/vm/mock"
)

// 代币状态
type Token struct {
    Name     string
    Symbol   string
    Supply   uint64
    Balances map[core.Address]uint64
}

// 初始化代币
func Init(name, symbol string, initialSupply uint64) {
    token := Token{
        Name:     name,
        Symbol:   symbol,
        Supply:   initialSupply,
        Balances: make(map[core.Address]uint64),
    }
    
    // 铸造初始供应量给创建者
    token.Balances[core.Sender()] = initialSupply
    
    object := core.CreateObject()
    object.Set("token", token)
}

// 转移代币
func Transfer(to core.Address, amount uint64) error {
    token, err := getToken()
    if err != nil {
        return err
    }
    
    from := core.Sender()
    if token.Balances[from] < amount {
        return errors.New("余额不足")
    }
    
    token.Balances[from] -= amount
    token.Balances[to] += amount
    
    object.Set("token", token)
    return nil
}

// 获取余额
func BalanceOf(addr core.Address) (uint64, error) {
    token, err := getToken()
    if err != nil {
        return 0, err
    }
    return token.Balances[addr], nil
}
```

## 6. 后续步骤

1. 阅读 [WASM 合约接口](wasm_contract_interface.md) 文档
2. 探索 [VM 架构](vm_architecture.md)
3. 了解 [Gas 计费](gas.md)
4. 查看更多 [示例](../examples) 