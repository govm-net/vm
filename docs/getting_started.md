# 入门指南：使用 VM 构建你的第一个智能合约

本指南将帮助您开始使用 VM 编写、部署和测试基于 Go 语言的智能合约，这些合约将被编译为 WebAssembly 模块并通过 Wasmer 运行时执行。

## 安装要求

在开始之前，确保您的系统满足以下要求：

- 安装 Go 1.18 或更高版本
- 安装 TinyGo (用于将 Go 代码编译为 WebAssembly)
  ```bash
  go install tinygo.org/x/tinygo@latest
  ```
- 安装 Wasmer 运行时
  ```bash
  curl https://get.wasmer.io -sSfL | sh
  ```
- 安装 Go Wasmer 模块
  ```bash
  go get github.com/wasmerio/wasmer-go
  ```
- 熟悉基本的 Go 语言概念
- 了解基本的区块链和智能合约概念

## 1. 设置项目

首先，我们需要创建一个新的 Go 模块并添加虚拟机依赖：

```bash
# 创建项目目录
mkdir mycontract
cd mycontract

# 初始化 Go 模块
go mod init mycontract

# 添加 VM 依赖
go get -u github.com/govm-net/vm
```

## 2. 编写第一个智能合约

创建一个名为 `counter.go` 的文件，包含一个简单的计数器智能合约：

```go
package counter

import (
	"github.com/govm-net/vm/core"
)

// Counter 是一个简单的计数器合约
type Counter struct{}

// Initialize 创建一个新的计数器对象
func (c *Counter) Initialize(ctx core.Context) (core.ObjectID, error) {
	// 创建一个新的状态对象
	obj, err := ctx.CreateObject()
	if err != nil {
		return core.ObjectID{}, err
	}

	// 设置初始计数为 0
	err = obj.Set("count", uint64(0))
	if err != nil {
		return core.ObjectID{}, err
	}

	// 设置对象所有者为合约创建者
	err = obj.SetOwner(ctx.Sender())
	if err != nil {
		return core.ObjectID{}, err
	}

	// 记录创建事件
	ctx.Log("CounterCreated", obj.ID())

	return obj.ID(), nil
}

// Increment 增加计数器的值
func (c *Counter) Increment(ctx core.Context, counterID core.ObjectID) (uint64, error) {
	// 获取计数器对象
	obj, err := ctx.GetObject(counterID)
	if err != nil {
		return 0, err
	}

	// 只有所有者可以操作
	if obj.Owner() != ctx.Sender() {
		return 0, core.ErrUnauthorized
	}

	// 获取当前计数
	value, err := obj.Get("count")
	if err != nil {
		return 0, err
	}
	count := value.(uint64)

	// 增加计数
	count++

	// 更新计数
	err = obj.Set("count", count)
	if err != nil {
		return 0, err
	}

	// 记录更新事件
	ctx.Log("CounterIncremented", counterID, count)

	return count, nil
}

// GetCount 获取当前计数
func (c *Counter) GetCount(ctx core.Context, counterID core.ObjectID) (uint64, error) {
	// 获取计数器对象
	obj, err := ctx.GetObject(counterID)
	if err != nil {
		return 0, err
	}

	// 获取当前计数
	value, err := obj.Get("count")
	if err != nil {
		return 0, err
	}

	return value.(uint64), nil
}

// Reset 重置计数器为零
func (c *Counter) Reset(ctx core.Context, counterID core.ObjectID) (bool, error) {
	// 获取计数器对象
	obj, err := ctx.GetObject(counterID)
	if err != nil {
		return false, err
	}

	// 只有所有者可以操作
	if obj.Owner() != ctx.Sender() {
		return false, core.ErrUnauthorized
	}

	// 重置计数为 0
	err = obj.Set("count", uint64(0))
	if err != nil {
		return false, err
	}

	// 记录重置事件
	ctx.Log("CounterReset", counterID)

	return true, nil
}
```

### TinyGo 兼容性考虑

在编写智能合约时，需要注意 TinyGo 对标准库的支持有限，编写 WebAssembly 兼容的合约需要遵循以下规则：

1. **避免使用不支持的包**：
   - 避免大型标准库包
   - 不要使用反射 (`reflect` 包)
   - 避免 `unsafe` 和系统调用

2. **简化内存管理**：
   - 避免大量小内存分配
   - 合理设计对象生命周期

3. **使用纯数据类型**：
   - 优先使用基本类型
   - 谨慎使用接口和复杂结构

## 3. 测试智能合约

创建一个名为 `counter_test.go` 的测试文件：

```go
package counter

import (
	"testing"

	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/vm"
	"github.com/govm-net/vm/vm/api"
)

func TestCounter(t *testing.T) {
	// 创建一个新的虚拟机实例，使用内存数据库
	config := api.DefaultContractConfig()
	config.EnableWASIContracts = true // 启用 WebAssembly 支持
	engine := vm.NewEngine(config)

	// 创建一个测试地址
	testAddress := core.Address{1, 2, 3, 4, 5}
	engine.SetSender(testAddress)

	// 加载合约代码
	code := []byte(`
package counter

import "github.com/govm-net/vm/core"

// Counter 是一个简单的计数器合约
type Counter struct{}

// Initialize 创建一个新的计数器对象
func (c *Counter) Initialize(ctx core.Context) (core.ObjectID, error) {
	// ... 合约代码 ...
}

// ... 其他方法 ...
`)

	// 部署为 WebAssembly 合约
	deployOptions := vm.DeployOptions{
		AsWASI: true,
		WASIOptions: vm.WASIOptions{
			MemoryLimit: 64 * 1024 * 1024, // 64MB 内存限制
		},
	}
	contractAddr, err := engine.DeployWithOptions(code, deployOptions)
	if err != nil {
		t.Fatalf("Failed to deploy WebAssembly contract: %v", err)
	}

	// 初始化计数器
	result, err := engine.ExecuteWithArgs(contractAddr, "Initialize")
	if err != nil {
		t.Fatalf("Failed to initialize counter: %v", err)
	}

	// 解析对象ID
	var objectID core.ObjectID
	if err := vm.DecodeResult(result, &objectID); err != nil {
		t.Fatalf("Failed to decode objectID: %v", err)
	}

	// 验证初始计数是 0
	countResult, err := engine.ExecuteWithArgs(contractAddr, "GetCount", objectID)
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}
	
	var count uint64
	if err := vm.DecodeResult(countResult, &count); err != nil {
		t.Fatalf("Failed to decode count: %v", err)
	}
	
	if count != 0 {
		t.Errorf("Expected initial count to be 0, got %d", count)
	}

	// 增加计数
	incrementResult, err := engine.ExecuteWithArgs(contractAddr, "Increment", objectID)
	if err != nil {
		t.Fatalf("Failed to increment counter: %v", err)
	}
	
	if err := vm.DecodeResult(incrementResult, &count); err != nil {
		t.Fatalf("Failed to decode incremented count: %v", err)
	}
	
	if count != 1 {
		t.Errorf("Expected count to be 1, got %d", count)
	}

	// ... 其他测试 ...
}
```

运行测试：

```bash
go test -v
```

## 4. 部署和执行 WebAssembly 合约

创建一个名为 `deploy.go` 的程序，用于部署和执行您的 WebAssembly 合约：

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/vm"
	"github.com/govm-net/vm/vm/api"
)

func main() {
	// 创建虚拟机实例，配置 WebAssembly 支持
	config := api.DefaultContractConfig()
	config.EnableWASIContracts = true
	config.WASIContractsDir = filepath.Join(".", "wasi_modules")
	config.TinyGoPath = "/usr/local/bin/tinygo" // TinyGo 可执行文件路径

	// 确保存储目录存在
	os.MkdirAll(config.WASIContractsDir, 0755)

	// 创建虚拟机实例
	engine := vm.NewEngine(config)

	// 设置发送方地址（在实际区块链环境中，这将从交易中获取）
	sender := core.Address{1, 2, 3, 4, 5}
	engine.SetSender(sender)

	// 读取合约代码
	code, err := os.ReadFile("counter.go")
	if err != nil {
		fmt.Printf("Failed to read contract code: %v\n", err)
		os.Exit(1)
	}

	// 部署为 WebAssembly 合约
	deployOptions := vm.DeployOptions{
		AsWASI: true,
		WASIOptions: vm.WASIOptions{
			MemoryLimit: 64 * 1024 * 1024, // 64MB 内存限制
			Timeout:     5000,              // 5秒超时
			TableSize:   1024,              // 函数表大小
			StackSize:   65536,             // 栈大小 (64KB)
		},
	}
	contractAddr, err := engine.DeployWithOptions(code, deployOptions)
	if err != nil {
		fmt.Printf("Failed to deploy WebAssembly contract: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("WebAssembly contract deployed at: %s\n", contractAddr.String())

	// 初始化计数器
	result, err := engine.ExecuteWithArgs(contractAddr, "Initialize")
	if err != nil {
		fmt.Printf("Failed to initialize counter: %v\n", err)
		os.Exit(1)
	}

	// 从结果中解析对象ID
	var objectID core.ObjectID
	if err := vm.DecodeResult(result, &objectID); err != nil {
		fmt.Printf("Failed to decode objectID: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Counter created with ID: %s\n", objectID.String())

	// 递增计数器
	incrementResult, err := engine.ExecuteWithArgs(contractAddr, "Increment", objectID)
	if err != nil {
		fmt.Printf("Failed to increment counter: %v\n", err)
		os.Exit(1)
	}
	
	var count uint64
	if err := vm.DecodeResult(incrementResult, &count); err != nil {
		fmt.Printf("Failed to decode incremented count: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Counter incremented, new value: %d\n", count)

	// 获取计数
	countResult, err := engine.ExecuteWithArgs(contractAddr, "GetCount", objectID)
	if err != nil {
		fmt.Printf("Failed to get counter value: %v\n", err)
		os.Exit(1)
	}
	
	if err := vm.DecodeResult(countResult, &count); err != nil {
		fmt.Printf("Failed to decode count: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Current counter value: %d\n", count)
}
```

运行部署程序：

```bash
go run deploy.go
```

## 5. WebAssembly 合约的优势

使用 WebAssembly 执行模式提供以下优势：

1. **更高的安全性**：WebAssembly 提供内置的内存安全和沙箱隔离
2. **跨平台兼容**：编译一次，可在任何支持 WebAssembly 的环境中运行
3. **更精细的资源控制**：可以精确限制内存使用和执行时间
4. **性能接近原生**：WebAssembly 执行速度接近原生代码
5. **轻量级部署**：WASI 模块通常比独立可执行文件小得多

## 6. 调试 WebAssembly 合约

调试 WebAssembly 合约时，可以使用以下工具和技术：

```bash
# 查看生成的 WASM 模块
ls -la wasi_modules/

# 使用 Wasmer 运行和调试 WASM 模块
wasmer run --mapdir=/tmp:/tmp wasi_modules/your_contract.wasm -- -debug

# 检查 WASM 模块的内部结构
wasm-objdump -x wasi_modules/your_contract.wasm
```

还可以在合约中添加详细的日志记录，用于调试：

```go
// 在合约代码中添加日志
ctx.Log("Debug", "Processing increment for object", counterID.String())
```

## 7. 添加持久化存储

要使用数据库支持的对象，您需要设置数据库提供程序。下面是一个使用内存数据库的示例：

```go
package main

import (
	"fmt"
	"path/filepath"
	"os"

	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/vm"
	"github.com/govm-net/vm/vm/api"
)

// MemoryDB 是一个简单的内存键值存储
type MemoryDB struct {
	data map[string][]byte
}

func NewMemoryDB() *MemoryDB {
	return &MemoryDB{
		data: make(map[string][]byte),
	}
}

func (db *MemoryDB) Get(key []byte) ([]byte, error) {
	value, ok := db.data[string(key)]
	if !ok {
		return nil, fmt.Errorf("key not found")
	}
	return value, nil
}

func (db *MemoryDB) Put(key []byte, value []byte) error {
	db.data[string(key)] = value
	return nil
}

func (db *MemoryDB) Delete(key []byte) error {
	delete(db.data, string(key))
	return nil
}

func (db *MemoryDB) Has(key []byte) (bool, error) {
	_, ok := db.data[string(key)]
	return ok, nil
}

// Iterator 实现（简化版本）
type MemoryIterator struct {
	keys   []string
	values [][]byte
	index  int
}

func (db *MemoryDB) Iterator(start, end []byte) (vm.Iterator, error) {
	// 简化实现
	return nil, fmt.Errorf("iterator not implemented")
}

func (db *MemoryDB) Close() error {
	return nil
}

func main() {
	// 创建虚拟机实例，配置 WebAssembly 支持
	config := api.DefaultContractConfig()
	config.EnableWASIContracts = true
	config.WASIContractsDir = filepath.Join(".", "wasi_modules")
	config.TinyGoPath = "/usr/local/bin/tinygo"

	// 确保存储目录存在
	os.MkdirAll(config.WASIContractsDir, 0755)

	// 创建虚拟机实例
	engine := vm.NewEngine(config)

	// 创建内存数据库
	db := NewMemoryDB()

	// 创建键生成器
	keyGen := vm.NewDefaultKeyGenerator()

	// 设置数据库提供程序
	engine.SetDBProvider(db, keyGen)

	// 现在可以部署和执行合约，数据将持久化到内存数据库中
	// ...（部署和执行代码）...
}
```

在实际生产环境中，您可能会使用 LevelDB、RocksDB 或其他持久化存储替代内存数据库。

## 8. 生产环境优化

在将合约部署到生产环境之前，可以考虑以下优化：

### 8.1 WebAssembly 资源控制

```go
// 详细的 WebAssembly 配置
config.WASIOptions = api.WASIOptions{
    MemoryLimit:   64 * 1024 * 1024, // 内存限制 (64MB)
    TableSize:     1024,             // 函数表大小
    Timeout:       5000,             // 执行超时 (毫秒)
    FuelLimit:     10000000,         // 指令计数限制
    StackSize:     65536,            // 栈大小 (64KB)
    EnableSIMD:    false,            // 禁用 SIMD 指令
    EnableThreads: false,            // 禁用线程
}
```

### 8.2 TinyGo 编译优化

使用优化参数编译 WebAssembly 模块：

```bash
tinygo build -o contract.wasm -target=wasi -opt=z -no-debug -gc=leaking contract.go
```

### 8.3 内存优化

在合约代码中优化内存使用：

```go
// 避免这样的代码（会创建大量临时对象）
for i := 0; i < 1000; i++ {
    str := fmt.Sprintf("Item %d", i)
    // 使用 str...
}

// 改为这样（重用缓冲区）
var buffer strings.Builder
for i := 0; i < 1000; i++ {
    buffer.Reset()
    fmt.Fprintf(&buffer, "Item %d", i)
    str := buffer.String()
    // 使用 str...
}
```

## 9. 创建更复杂的合约

一旦您掌握了基础知识，可以尝试编写更复杂的合约，如代币合约或 NFT 合约。查看 `contracts/` 目录中的示例获取灵感。

## 10. 智能合约编译为 WebAssembly 的详细流程

为了更深入地理解 VM 如何将 Go 源码转换为 WebAssembly 模块，本节详细介绍了 `engine.DeployWithOptions` 方法的内部实现流程。

### 10.1 编译流程概述

当调用 `DeployWithOptions` 方法部署合约时，以下步骤会依次执行：

```go
// 部署为 WebAssembly 合约
deployOptions := vm.DeployOptions{
    AsWASI: true,
    WASIOptions: vm.WASIOptions{
        MemoryLimit: 64 * 1024 * 1024, // 内存限制 
    },
}
contractAddr, err := engine.DeployWithOptions(code, deployOptions)
```

内部执行的流程如下：

1. **源码接收与解压**：处理可能被压缩的源码
2. **源码验证**：确保代码符合安全要求
3. **合约信息提取**：分析代码结构
4. **WASI 包装代码生成**：创建与 WebAssembly 接口通信的包装代码
5. **编译环境准备**：设置所需的编译环境
6. **TinyGo 编译**：将 Go 代码编译为 WASM 模块
7. **模块优化与验证**：确保生成的 WASM 模块有效
8. **存储与注册**：保存模块并记录合约信息

### 10.2 源码解压与验证

首先，系统会检查源码是否经过 GZIP 压缩，如果是，则进行解压：

```go
if isGzipCompressed(code) {
    code, err = decompressGzip(code)
    if err != nil {
        return core.ZeroAddress(), fmt.Errorf("解压合约代码失败: %w", err)
    }
}
```

随后，系统会对合约代码进行全面验证：

```go
if err := maker.ValidateContract(code); err != nil {
    return core.ZeroAddress(), fmt.Errorf("合约验证失败: %w", err)
}
```

验证过程包括：
- 检查包导入是否在允许列表中（如 `github.com/govm-net/vm/core`）
- 禁止使用会导致非确定性行为的关键字（如 `go`, `select`, `recover`）
- 确保合约大小在限制范围内
- 验证至少有一个公开（导出）函数
- 检查 Go 语法是否正确

### 10.3 合约信息提取与包装代码生成

系统使用 Go 语言的 AST（抽象语法树）处理库分析源码：

```go
packageName, contractName, err := extractContractInfo(code)
```

提取出包名和主要合约结构体名称后，系统会生成 WASI 包装代码，使合约能与 WebAssembly 系统接口通信：

```go
wrapperCode := generateWASIWrapper(packageName, contractName, code)
```

包装代码的框架如下：

```go
package main

import (
    "encoding/binary"
    "unsafe"
    
    "original_package" // 原始合约包
)

// 与 VM 通信的接口函数
//export vm_alloc
func vm_alloc(size uint32) uint32

//export vm_free
func vm_free(ptr uint32)

// 包装原始合约
var contract = &original_package.ContractName{}

// 主入口点
//export execute
func execute() int32 {
    // 参数解码和函数调用逻辑
    return 0
}

func main() {
    // WASI 模块需要 main 函数
}
```

### 10.4 编译环境准备

为了编译合约，系统会创建一个临时目录结构：

```
temp-dir/
├── main.go        # 包装代码
├── go.mod         # 模块定义
└── original/      # 原始合约代码
    └── contract.go
```

并创建 `go.mod` 文件声明依赖：

```
module wasm_contract

go 1.18

require github.com/govm-net/vm v0.0.0
```

### 10.5 TinyGo 编译过程

系统使用 TinyGo 编译器将 Go 代码转换为 WebAssembly 模块：

```go
tinygoPath := config.TinyGoPath
args := []string{
    "build",
    "-o", outputPath,
    "-target=wasi",
    "-opt=z",        // 优化大小
    "-no-debug",     // 移除调试信息
    "-gc=leaking",   // 简化垃圾收集
    "./main.go",
}

cmd := exec.Command(tinygoPath, args...)
output, err := cmd.CombinedOutput()
```

TinyGo 编译选项说明：

- `-target=wasi`: 编译目标为 WebAssembly 系统接口
- `-opt=z`: 优化生成的 WASM 大小
- `-no-debug`: 移除调试信息以减小文件大小
- `-gc=leaking`: 使用简化的垃圾收集器，提高性能

### 10.6 模块优化与验证

生成的 WebAssembly 模块会经过验证和可选的优化步骤：

```go
// 验证 WASM 格式
if !isValidWasmModule(wasmCode) {
    return nil, errors.New("无效的 WebAssembly 模块格式")
}

// 可选：使用外部工具进一步优化
if config.OptimizeWasm {
    wasmCode = optimizeWasmModule(wasmCode)
}
```

验证过程确保：
- 模块以正确的魔数开头 (`\0asm`)
- 模块结构完整有效
- 必要的导出函数存在

### 10.7 存储与注册

最后，系统将编译好的 WebAssembly 模块保存到指定位置：

```go
// 生成合约地址
contractAddr := generateContractAddress(wasmCode)

// 存储 WASM 文件
wasmPath := filepath.Join(config.WASIContractsDir, contractAddr.String()+".wasm")
os.WriteFile(wasmPath, wasmCode, 0644)

// 注册合约信息
contracts[contractAddr] = wasmPath
```

完成这些步骤后，系统返回生成的合约地址，供后续调用合约函数时使用。

### 10.8 常见编译问题与解决方案

在编译过程中可能遇到的常见问题及解决方案：

1. **TinyGo 兼容性问题**：
   - 症状：`不支持的功能：反射/并发/...`
   - 解决方案：避免使用 TinyGo 不支持的包和功能

2. **包依赖问题**：
   - 症状：`找不到包 xxx`
   - 解决方案：确保所有依赖都在允许列表中，并正确导入

3. **内存溢出**：
   - 症状：`编译过程中内存不足`
   - 解决方案：减小合约规模，拆分为多个小合约

4. **语法错误**：
   - 症状：`语法错误在 line:column`
   - 解决方案：修复源码中的语法错误

5. **禁用关键字使用**：
   - 症状：`禁止使用关键字: go`
   - 解决方案：避免使用会导致非确定性的 Go 关键字

## 下一步

- 探索 VM 的完整 API 文档
- 学习如何处理复杂的数据类型
- 了解如何优化合约以提高性能和降低资源消耗
- 探索如何将合约集成到区块链节点
- 阅读 [WebAssembly 合约文档](wasi_contracts.md) 获取更多详情

## 故障排除

### 常见问题

1. **"TinyGo not found"**：确保 TinyGo 已正确安装并且路径配置正确。

2. **"WebAssembly compilation failed"**：检查合约代码是否兼容 TinyGo，避免使用不支持的包和功能。

3. **"Memory limit exceeded"**：合约尝试使用超过限制的内存，优化内存使用或增加限制。

4. **"Execution timeout"**：合约执行时间超过限制，检查是否有无限循环或优化复杂操作。

5. **"unauthorized operation"**：检查对象所有权，只有所有者能执行某些操作。

6. **"DB provider not set"**：在使用数据库对象前设置数据库提供程序。

### 获取帮助

如果遇到问题，请查阅项目文档或提交 GitHub issue。 