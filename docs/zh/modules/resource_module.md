# 资源控制模块设计文档

## 1. 概述

资源控制模块负责管理智能合约执行过程中的资源消耗，包括计算资源、内存资源和时间资源。默认实现为 GAS 计费系统，通过精确计量合约执行过程中的资源使用，防止恶意合约消耗过多系统资源。

## 2. 模块职责

1. 资源消耗计量
2. 资源限制控制
3. 资源使用统计

## 3. 核心接口

```go
// ResourceManager 定义了资源管理器的接口
type ResourceManager interface {
    // Consume 消耗指定数量的资源
    Consume(amount int64) error
    
    // GetRemaining 获取剩余资源
    GetRemaining() int64
}

// GasManager 是 GAS 资源管理器的实现
type GasManager struct {
    limit  int64
    used   int64
    mu     sync.RWMutex
}
```

## 4. 实现细节

### 4.1 GAS 计费模型

GAS 计费分为两个维度：

1. **代码行计费**：
   - 每行代码执行消耗 1 点 GAS
   - 在编译阶段自动注入 GAS 消耗代码

2. **操作计费**：
   - 基础操作（查询信息）：10-50 GAS
   - 存储操作（创建、读取、写入）：100-1000 GAS
   - 网络操作（日志记录）：100+ GAS

### 4.2 资源消耗规则

| 操作类型 | GAS 消耗 | 说明 |
|---------|---------|------|
| BlockHeight() | 10 | 获取区块高度 |
| Sender() | 10 | 获取发送者地址 |
| Balance() | 50 | 查询余额 |
| CreateObject() | 500 | 创建对象 |
| GetObject() | 50 | 获取对象 |
| Object.Get() | 100 | 读取对象字段 |
| Object.Set() | 1000 | 写入对象字段 |
| Log() | 100 + 数据长度 | 记录日志 |

### 4.3 资源限制机制

1. **总量限制**：每个合约调用有最大 GAS 限制
2. **实时检查**：每次消耗 GAS 时检查是否超限
3. **异常处理**：GAS 耗尽时抛出异常，终止执行

## 5. 使用示例

```go
// 创建 GAS 管理器
gasManager := NewGasManager(1000000) // 100万 GAS 限制

// 消耗 GAS
err := gasManager.Consume(100)
if err != nil {
    log.Fatal("GAS 不足:", err)
}

// 检查剩余 GAS
remaining := gasManager.GetRemaining()
fmt.Printf("剩余 GAS: %d\n", remaining)

// 在合约执行中使用
func contractFunction() {
    // 消耗执行 GAS
    gasManager.Consume(50)
    
    // 执行业务逻辑
    // ...
    
    // 消耗存储 GAS
    gasManager.Consume(1000)
}
```

## 6. 并发安全

资源管理器支持多协程并发访问：

1. 使用读写锁保护共享数据
2. 原子操作更新资源计数
3. 避免死锁和竞态条件

## 7. 错误处理

资源消耗过程中可能遇到的错误：

1. **GAS 不足**：当消耗的 GAS 超过剩余量时，返回错误
2. **无效参数**：消耗负数或零 GAS 时的处理
3. **系统错误**：内部计算错误的处理

## 8. 性能优化

1. **锁优化**：使用读写锁减少锁竞争
2. **批量操作**：支持批量 GAS 消耗
3. **缓存机制**：缓存常用计算结果

## 9. 扩展性

1. **自定义计费模型**：支持不同的资源计费规则
2. **多资源类型**：支持内存、存储等其他资源类型
3. **动态调整**：支持运行时调整资源限制