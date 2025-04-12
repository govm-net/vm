# Gas 计费机制

## 概述
Gas 是智能合约执行时消耗的计算资源单位。每个合约执行都需要消耗一定量的 gas，gas 消耗完时合约执行会被终止。这种机制可以防止合约无限循环或执行过于复杂的操作。

## Gas 计费规则

### 基础计费
1. 代码行计费
   - 每行代码消耗 1 点 gas
   - 包括变量声明、赋值、函数调用等所有语句
   - 不包括空行和注释
   - 代码行计费与接口调用计费是累加的

2. 函数调用计费
   - 函数入口消耗 1 点 gas
   - 函数体内的语句按行计费
   - 每个语句块结束时，一次性计费该块内所有语句的 gas

### Context 接口操作计费
根据 wasm/contract.go 实现的实际消耗：

1. 基础信息查询
   - Sender(): 10 gas
   - BlockHeight(): 10 gas
   - BlockTime(): 10 gas
   - ContractAddress(): 10 gas

2. 余额与转账
   - Balance(addr): 50 gas
   - Transfer(to, amount): 500 gas

3. 合约调用
   - Call(contract, function, args...): 10000 gas
   - 注意：Call 方法会预留 10000 gas 作为基本调用费用，剩余的 gas 会分配给被调用合约
   - 被调用合约执行完成后，会根据被调用合约实际消耗的 gas 进行计费

4. 对象操作
   - CreateObject(): 500 gas
   - GetObject(id): 50 gas
   - GetObjectWithOwner(owner): 50 gas
   - DeleteObject(id): 500 gas

5. 日志记录
   - Log(event, keyValues...): 100 gas + 数据长度 gas

### Object 接口操作计费
根据 wasm/contract.go 实现的实际消耗：

1. 基础操作
   - ID(): 10 gas
   - Contract(): 100 gas

2. 所有权操作
   - Owner(): 100 gas
   - SetOwner(owner): 500 gas

3. 字段操作
   - Get(field, value): 100 gas + 结果数据大小 gas
   - Set(field, value): 1000 gas + 数据大小 * 100 gas

### 特殊操作计费
1. 合约部署
   - 基础部署消耗：1000 gas
   - 代码大小计费：每字节 1 gas
   - 构造函数参数：每个参数 100 gas

2. 合约调用
   - 基础调用消耗：100 gas
   - 函数参数：每个参数 50 gas
   - 返回值：每个返回值 50 gas

### Gas 退还规则
1. 删除object，退还300

## Gas 限制
Gas 限制是可选的配置项，具体限制由对接平台决定。以下是一些常见的限制示例：

1. 区块限制
   - 每个区块最大 gas 限制
   - 每个交易最大 gas 限制，默认10000000

2. 合约限制
   - 单个合约调用最大 gas
   - 合约调用深度限制
   - 合约代码大小限制

注意：具体的限制值需要根据对接平台的实际配置来确定。框架本身不强制这些限制，而是由平台在实现时进行配置。

## 示例

### 基础接口调用
```go
func TestBasicGas(ctx core.Context) {
    addr := ctx.Sender()                 // 消耗 10 gas + 1 gas(代码行)
    height := ctx.BlockHeight()          // 消耗 10 gas + 1 gas(代码行)
    time := ctx.BlockTime()              // 消耗 10 gas + 1 gas(代码行)
    contractAddr := ctx.ContractAddress() // 消耗 10 gas + 1 gas(代码行)
    balance := ctx.Balance(addr)         // 消耗 50 gas + 1 gas(代码行)
}
```
总消耗：95 gas (接口调用: 90 gas + 代码行: 5 gas)

### 对象操作
```go
func TestObjectGas(ctx core.Context) {
    // 创建和获取对象
    obj := ctx.CreateObject()             // 消耗 50 gas
    objID := obj.ID()                     // 消耗 10 gas
    
    // 设置字段
    data := map[string]string{"key": "value"}
    obj.Set("data", data)                 // 消耗 1000 gas + 数据大小 * 100 gas
    
    // 读取字段
    var result map[string]string
    obj.Get("data", &result)              // 消耗 100 gas + 结果大小 gas
    
    // 删除对象
    ctx.DeleteObject(objID)               // 消耗 500 gas
}
```

### 合约调用
```go
func TestCallGas(ctx core.Context) {
    targetContract := core.Address{0x01} // 目标合约地址
    
    // 假设当前 gas 余额为 50000
    
    // 调用其他合约
    // 预留 10000 gas 作为基本调用费用
    // 分配剩余的 40000 gas 给被调用合约
    result, _ := ctx.Call(targetContract, "TestFunc", 123, "abc")
    
    // 如果被调用合约实际消耗了 25000 gas
    // 则总消耗为 10000(基本费用) + 25000(实际消耗) = 35000 gas
    
    // 记录日志
    ctx.Log("FunctionCalled", "contract", targetContract, "result", result) // 消耗 100 gas + 数据长度
}
```

## 最佳实践
1. 优化代码结构
   - 减少不必要的函数调用，特别是高 gas 消耗的操作如 Call()
   - 合并相似的存储操作，避免重复 Set() 操作
   - 缓存已获取的数据，避免重复调用 Get()

2. 优化存储使用
   - 尽量减少大数据的存储，将数据分解为较小的块
   - 使用批量操作替代循环单次操作
   - 适当使用缓存减少存储访问

3. Gas 优化技巧
   - 避免在循环中执行高 gas 消耗的操作
   - 在函数开始处验证条件，提前返回减少不必要的执行
   - 合理组织代码，减少代码行数
   - 优先使用简单数据类型，减少序列化/反序列化开销 