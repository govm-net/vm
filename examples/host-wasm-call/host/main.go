package main

import (
	"fmt"
	"log"
	"os"

	wasmer "github.com/wasmerio/wasmer-go/wasmer"
)

// 配置存储
type ConfigStore struct {
	values map[string]string
}

func NewConfigStore() *ConfigStore {
	return &ConfigStore{
		values: make(map[string]string),
	}
}

func (c *ConfigStore) Set(key, value string) {
	c.values[key] = value
}

func (c *ConfigStore) Get(key string) (string, bool) {
	value, exists := c.values[key]
	return value, exists
}

// 宿主环境
type HostEnvironment struct {
	config *ConfigStore
	memory *wasmer.Memory
}

// 从 WebAssembly 内存中读取字符串
func readString(memory *wasmer.Memory, ptr, len uint32) string {
	data := memory.Data()[ptr : ptr+len]
	return string(data)
}

// 将字符串写入 WebAssembly 内存
func writeString(memory *wasmer.Memory, ptr uint32, s string) {
	copy(memory.Data()[ptr:ptr+uint32(len(s))], s)
	// 添加 null 终止符
	memory.Data()[ptr+uint32(len(s))] = 0
}

// 获取配置值的导入函数
func getConfigValueFunction(env *HostEnvironment) func(keyPtr, keyLen, valuePtr, valueLen int32) int32 {
	return func(keyPtr, keyLen, valuePtr, valueLen int32) int32 {
		// 从 WebAssembly 内存中读取键
		key := readString(env.memory, uint32(keyPtr), uint32(keyLen))

		// 从配置存储中获取值
		value, exists := env.config.Get(key)
		if !exists {
			return 1 // 键不存在
		}

		// 检查缓冲区大小是否足够
		if int32(len(value)) >= valueLen {
			return 2 // 缓冲区太小
		}

		// 将值写入 WebAssembly 内存
		writeString(env.memory, uint32(valuePtr), value)

		return 0 // 成功
	}
}

// 日志记录导入函数
func logMessageFunction(env *HostEnvironment) func(msgPtr, msgLen int32) {
	return func(msgPtr, msgLen int32) {
		// 从 WebAssembly 内存中读取消息
		message := readString(env.memory, uint32(msgPtr), uint32(msgLen))

		// 打印消息
		fmt.Printf("[WASM] %s\n", message)
	}
}

// 宿主函数调用导入函数
func callHostFunctionFunction(env *HostEnvironment) func(funcID, argPtr, argLen int32) int64 {
	return func(funcID, argPtr, argLen int32) int64 {
		// 从 WebAssembly 内存中读取参数
		arg := readString(env.memory, uint32(argPtr), uint32(argLen))

		fmt.Printf("[HOST] 调用宿主函数 ID=%d, 参数=%s\n", funcID, arg)

		// 根据函数 ID 执行不同的操作
		switch funcID {
		case 1:
			return int64(len(arg))<<32 | 42 // 返回参数长度和 42
		case 2:
			return 0x1234567890ABCDEF // 返回一个特定的值
		default:
			return -1 // 未知函数 ID
		}
	}
}

func main() {
	// 读取 WASM 文件
	wasmBytes, err := os.ReadFile("../wasm/main.wasm")
	if err != nil {
		log.Fatalf("无法读取 WASM 文件: %v", err)
	}

	// 创建 Wasmer 引擎和存储
	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)

	// 编译模块
	module, err := wasmer.NewModule(store, wasmBytes)
	if err != nil {
		log.Fatalf("无法编译模块: %v", err)
	}

	// 创建 WASI 导入对象
	wasiEnv, err := wasmer.NewWasiStateBuilder("wasi-program").
		Argument("config_demo").
		Environment("ENV", "production").
		Finalize()
	if err != nil {
		log.Fatalf("无法创建 WASI 环境: %v", err)
	}

	// 获取 WASI 导入
	importObject, err := wasiEnv.GenerateImportObject(store, module)
	if err != nil {
		log.Fatalf("无法生成 WASI 导入对象: %v", err)
	}

	// 创建配置存储
	configStore := NewConfigStore()
	configStore.Set("database_name", "production_db")
	configStore.Set("api_key", "sk_live_12345abcdef")
	configStore.Set("max_connections", "100")

	// 创建宿主环境
	hostEnv := &HostEnvironment{
		config: configStore,
	}

	// 添加自定义导入
	importObject.Register("env", map[string]wasmer.IntoExtern{
		"get_config_value": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32),
				wasmer.NewValueTypes(wasmer.I32),
			),
			func(args []wasmer.Value) ([]wasmer.Value, error) {
				if hostEnv.memory == nil {
					return nil, fmt.Errorf("内存尚未初始化")
				}

				result := getConfigValueFunction(hostEnv)(
					args[0].I32(),
					args[1].I32(),
					args[2].I32(),
					args[3].I32(),
				)

				return []wasmer.Value{wasmer.NewI32(result)}, nil
			},
		),
		"log_message": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				wasmer.NewValueTypes(wasmer.I32, wasmer.I32),
				wasmer.NewValueTypes(),
			),
			func(args []wasmer.Value) ([]wasmer.Value, error) {
				if hostEnv.memory == nil {
					return nil, fmt.Errorf("内存尚未初始化")
				}

				logMessageFunction(hostEnv)(
					args[0].I32(),
					args[1].I32(),
				)

				return []wasmer.Value{}, nil
			},
		),
		"call_host_function": wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32),
				wasmer.NewValueTypes(wasmer.I64),
			),
			func(args []wasmer.Value) ([]wasmer.Value, error) {
				if hostEnv.memory == nil {
					return nil, fmt.Errorf("内存尚未初始化")
				}

				result := callHostFunctionFunction(hostEnv)(
					args[0].I32(),
					args[1].I32(),
					args[2].I32(),
				)

				return []wasmer.Value{wasmer.NewI64(result)}, nil
			},
		),
	})

	// 实例化模块
	instance, err := wasmer.NewInstance(module, importObject)
	if err != nil {
		log.Fatalf("无法实例化模块: %v", err)
	}

	// 获取内存并设置到环境中
	memory, err := instance.Exports.GetMemory("memory")
	if err != nil {
		log.Fatalf("无法获取内存: %v", err)
	}
	hostEnv.memory = memory

	// 获取导出的函数
	processConfig, err := instance.Exports.GetFunction("process_config")
	if err != nil {
		log.Fatalf("无法获取 process_config 函数: %v", err)
	}

	processMemoryData, err := instance.Exports.GetFunction("process_memory_data")
	if err != nil {
		log.Fatalf("无法获取 process_memory_data 函数: %v", err)
	}

	// 调用处理配置的函数
	fmt.Println("\n=== 调用 process_config ===")
	result, err := processConfig()
	if err != nil {
		log.Fatalf("调用 process_config 失败: %v", err)
	}
	fmt.Printf("process_config 返回: %v\n", result)

	// 准备内存数据
	fmt.Println("\n=== 调用 process_memory_data ===")
	data := []byte{1, 2, 3, 4, 5, 10, 20, 30, 40, 50}

	// 调用WebAssembly模块导出的内存分配函数
	allocate, err := instance.Exports.GetFunction("allocate")
	if err != nil {
		log.Fatalf("无法获取allocate函数: %v", err)
	}

	// 调用分配函数获取指针
	result, err = allocate(int32(len(data)))
	if err != nil {
		log.Fatalf("内存分配失败: %v", err)
	}

	// 获取返回的指针
	dataPtr := result.(int32)

	// 复制数据到WebAssembly内存
	copy(memory.Data()[int(dataPtr):int(dataPtr)+len(data)], data)

	// 使用完后释放内存
	deallocate, err := instance.Exports.GetFunction("deallocate")
	if err == nil {
		_, err = deallocate(dataPtr, int32(len(data)))
		if err != nil {
			log.Fatalf("内存释放失败: %v", err)
		}
	}

	// 调用处理内存数据的函数
	result, err = processMemoryData(int32(dataPtr), int32(len(data)))
	if err != nil {
		log.Fatalf("调用 process_memory_data 失败: %v", err)
	}
	fmt.Printf("process_memory_data 返回: %v\n", result)

	// 演示直接内存操作
	fmt.Println("\n=== 直接内存操作 ===")

	// 写入一个字符串到内存
	message := "这是从宿主应用写入的消息"
	messagePtr := 2048 // 另一个安全的内存位置
	copy(memory.Data()[messagePtr:messagePtr+len(message)], message)

	// 读取内存中的数据
	readData := memory.Data()[int(dataPtr) : int(dataPtr)+len(data)]
	fmt.Printf("从内存读取的数据: %v\n", readData)
}
