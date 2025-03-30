# Token2 合约

Token2 是一个基于 WASM 的简单代币合约实现，与 Token1 相比，它采用了更高效的状态存储方式，使用单一 Object 存储所有状态。

## 实现特点

1. **单一 Object 存储**
   - 使用默认 Object 存储所有状态
   - 不再为每个用户创建独立的余额 Object
   - 通过字段前缀区分不同用户的数据

2. **状态存储结构**
   - 基本信息直接存储在默认 Object 中
   - 用户余额使用 `balance_` + 用户地址作为 key 存储
   - 合约所有者信息存储在 `owner` 字段中

3. **安全性**
   - 使用自定义的 `owner` 字段管理合约所有权
   - 保持 Object 原生 owner 为合约本身
   - 严格的权限控制（mint/burn 仅限所有者）

## 主要功能

### 1. 初始化合约
```go
func InitializeToken(ctx core.Context, name string, symbol string, decimals uint8, totalSupply uint64) core.ObjectID
```
- 初始化代币基本信息
- 设置合约所有者
- 分配初始供应量给创建者

### 2. 查询功能
```go
func GetTokenInfo(ctx core.Context) (string, string, uint8, uint64)
func GetOwner(ctx core.Context) core.Address
func BalanceOf(ctx core.Context, owner core.Address) uint64
```
- 获取代币基本信息
- 查询合约所有者
- 查询账户余额

### 3. 转账功能
```go
func Transfer(ctx core.Context, to core.Address, amount uint64) bool
```
- 在同一个 Object 中更新发送者和接收者的余额
- 自动处理接收者首次接收代币的情况
- 包含余额检查和错误处理

### 4. 铸造功能
```go
func Mint(ctx core.Context, to core.Address, amount uint64) bool
```
- 仅限合约所有者调用
- 增加总供应量
- 铸造新代币给指定地址

### 5. 销毁功能
```go
func Burn(ctx core.Context, amount uint64) bool
```
- 仅限合约所有者调用
- 减少总供应量
- 从所有者账户销毁代币

## 与 Token1 的主要区别

1. **存储方式**
   - Token1: 使用多个 Object 存储状态
   - Token2: 使用单一 Object 存储所有状态

2. **所有权管理**
   - Token1: 依赖 Object 的原生 owner
   - Token2: 使用自定义 owner 字段

3. **性能特点**
   - Token1: 支持并行转账（不同用户的转账可以并行执行）
   - Token2: 所有操作都需要串行执行（因为共享同一个 Object）

4. **功能扩展**
   - Token1: 无法实现 approve 功能（因为用户无法操作他人的 Object）
   - Token2: 可以实现 approve 功能（因为所有状态在同一个 Object 中）

5. **状态访问**
   - Token1: 需要获取多个 Object
   - Token2: 只需访问单一 Object

## 使用示例

```go
// 初始化代币
InitializeToken(ctx, "MyToken", "MTK", 18, 1000000)

// 查询代币信息
name, symbol, decimals, totalSupply := GetTokenInfo(ctx)

// 查询余额
balance := BalanceOf(ctx, userAddress)

// 转账
Transfer(ctx, recipientAddress, 100)

// 铸造新代币（仅限所有者）
Mint(ctx, recipientAddress, 1000)

// 销毁代币（仅限所有者）
Burn(ctx, 500)
```

## 注意事项

1. 合约所有者通过 `owner` 字段管理，而不是 Object 的原生 owner
2. 首次接收代币的地址会自动创建余额记录
3. 所有操作都包含适当的错误处理和状态回滚机制
4. 合约事件记录所有重要操作
5. 所有操作都需要串行执行，可能影响性能
6. 支持实现 approve 功能，适合需要授权场景 