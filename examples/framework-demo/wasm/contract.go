package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"unsafe"

	"github.com/govm-net/vm"
	"github.com/govm-net/vm/types"
)

// 函数ID常量定义 - 从types包导入以确保一致性
const (
	FuncGetSender          = int32(types.FuncGetSender)
	FuncGetBlockHeight     = int32(types.FuncGetBlockHeight)
	FuncGetBlockTime       = int32(types.FuncGetBlockTime)
	FuncGetContractAddress = int32(types.FuncGetContractAddress)
	FuncGetBalance         = int32(types.FuncGetBalance)
	FuncTransfer           = int32(types.FuncTransfer)
	FuncCreateObject       = int32(types.FuncCreateObject)
	FuncCall               = int32(types.FuncCall)
	FuncGetObject          = int32(types.FuncGetObject)
	FuncGetObjectWithOwner = int32(types.FuncGetObjectWithOwner)
	FuncDeleteObject       = int32(types.FuncDeleteObject)
	FuncLog                = int32(types.FuncLog)
	FuncGetObjectOwner     = int32(types.FuncGetObjectOwner)
	FuncSetObjectOwner     = int32(types.FuncSetObjectOwner)
	FuncDbRead             = int32(types.FuncDbRead)
	FuncDbWrite            = int32(types.FuncDbWrite)
	FuncDbDelete           = int32(types.FuncDbDelete)
	FuncSetHostBuffer      = int32(types.FuncSetHostBuffer)
)

// 定义全局接收数据缓冲区的大小
const HostBufferSize int32 = types.HostBufferSize

// 使用全局变量存储动态分配的主机缓冲区地址
var hostBufferPtr int32 = 0

// Address represents a blockchain address
type Address = vm.Address

// ObjectID represents a unique identifier for a state object
type ObjectID = vm.ObjectID

// Context implements the core.Context interface
type Context struct {
	contractAddr Address
}

// Object implements the core.Object interface
type Object struct {
	id ObjectID
}

var _ vm.Context = &Context{}
var _ vm.Object = &Object{}

// call_host_set - 用于向主机传递数据的操作，如set操作
//
//export call_host_set
func call_host_set(funcID, argPtr, argLen int32) int64

// call_host_get_buffer - 获取需要缓冲区的数据，如地址、对象、字节数组等
// 数据会写入到预先设置的全局缓冲区
//
//export call_host_get_buffer
func call_host_get_buffer(funcID, argPtr, argLen int32) int32

// 单独导出的简单数据类型获取函数
//
//export get_block_height
func get_block_height() int64

//export get_block_time
func get_block_time() int64

//export get_balance
func get_balance(addrPtr int32) float64

// 设置主机缓冲区地址的函数
//
//export set_host_buffer
func set_host_buffer(ptr int32) {
	hostBufferPtr = ptr
}

// 辅助函数 - 使用适当的导出函数调用主机功能
func callHost(funcID int32, data []byte) (resultPtr int32, resultSize int32, value int64) {
	var argPtr int32 = 0
	var argLen int32 = 0

	if len(data) > 0 {
		// 获取参数数据的指针和长度
		argPtr = int32(uintptr(unsafe.Pointer(&data[0])))
		argLen = int32(len(data))
	}

	// 根据函数ID选择合适的调用方式
	switch funcID {
	// 简单数据类型现在由Context方法直接调用相应的导出函数，这里不再处理
	// 需要通过缓冲区返回复杂数据的函数
	case FuncGetSender, FuncGetContractAddress, FuncCall,
		FuncGetObject, FuncGetObjectWithOwner, FuncCreateObject,
		FuncGetObjectOwner, FuncDbRead:
		// 检查是否已设置主机缓冲区
		if hostBufferPtr == 0 {
			// 如果主机缓冲区未设置，返回错误
			return 0, 0, 0
		}

		// 使用获取缓冲区数据的宿主函数（返回数据大小）
		resultSize = call_host_get_buffer(funcID, argPtr, argLen)
		if resultSize > 0 {
			// 数据已存储在全局缓冲区
			resultPtr = hostBufferPtr
			value = int64(resultSize)
		} else {
			value = 0
		}

	// 不需要返回数据的函数或返回简单值的函数
	default:
		// 使用设置数据的宿主函数
		value = call_host_set(funcID, argPtr, argLen)
		resultPtr = int32(value & 0xFFFFFFFF)
		resultSize = int32(value >> 32)
	}

	return resultPtr, resultSize, value
}

func readMemory(ptr, size int32) []byte {
	// 安全性检查
	if ptr == 0 || size <= 0 {
		return []byte{}
	}

	// 创建结果数组
	data := make([]byte, size)

	// 从主机缓冲区读取数据
	if ptr == hostBufferPtr && size > 0 && hostBufferPtr != 0 {
		// 使用全局缓冲区
		if size > HostBufferSize {
			// 超出缓冲区大小，截断或返回空
			size = HostBufferSize
		}

		// 安全地从全局缓冲区复制数据
		globalBuffer := unsafe.Pointer(uintptr(hostBufferPtr))
		for i := int32(0); i < size; i++ {
			data[i] = *(*byte)(unsafe.Pointer(uintptr(globalBuffer) + uintptr(i)))
		}
		return data
	}

	// 从指定位置读取数据
	src := unsafe.Pointer(uintptr(ptr))

	// 使用安全的复制方式
	for i := int32(0); i < size; i++ {
		// 尝试读取字节，如果发生异常，返回已读取的部分
		defer func() {
			if r := recover(); r != nil {
				// 记录错误但继续执行
				fmt.Printf("内存读取错误: %v\n", r)
			}
		}()

		data[i] = *(*byte)(unsafe.Pointer(uintptr(src) + uintptr(i)))
	}

	return data
}

func writeToMemory(data interface{}) (ptr int32, size int32) {
	var bytes []byte
	switch v := data.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		var err error
		bytes, err = json.Marshal(v)
		if err != nil {
			return 0, 0
		}
	}

	size = int32(len(bytes))
	buffer := make([]byte, size)
	copy(buffer, bytes)

	ptr = int32(uintptr(unsafe.Pointer(&buffer[0])))
	return ptr, size
}

// Context interface implementation
func (c *Context) Sender() Address {
	// 调用宿主函数，使用常量FuncGetSender
	ptr, size, _ := callHost(FuncGetSender, nil)
	data := readMemory(ptr, size)
	var addr Address
	copy(addr[:], data)
	return addr
}

func (c *Context) BlockHeight() uint64 {
	// 直接调用宿主函数，无需经过callHost中转
	value := get_block_height()
	return uint64(value)
}

func (c *Context) BlockTime() int64 {
	// 直接调用宿主函数，无需经过callHost中转
	return get_block_time()
}

func (c *Context) ContractAddress() Address {
	// 调用宿主函数，使用常量FuncGetContractAddress
	ptr, size, _ := callHost(FuncGetContractAddress, nil)
	data := readMemory(ptr, size)
	var addr Address
	copy(addr[:], data)
	return addr
}

func (c *Context) Balance(addr Address) float64 {
	// 直接调用宿主函数，无需经过callHost中转
	return get_balance(int32(uintptr(unsafe.Pointer(&addr[0]))))
}

func (c *Context) Transfer(to Address, amount uint64) error {
	// 创建参数：20字节地址 + 8字节金额
	data := make([]byte, 28)
	copy(data[:20], to[:])

	// 将uint64转换为字节数组（小端序）
	binary.LittleEndian.PutUint64(data[20:], amount)

	// 调用宿主函数，使用常量FuncTransfer
	_, _, result := callHost(FuncTransfer, data)
	if result == 0 {
		return fmt.Errorf("transfer failed")
	}
	return nil
}

func (c *Context) Call(contract Address, function string, args ...any) ([]byte, error) {
	// 准备参数：合约地址 + 函数名 + 参数
	contractBytes := contract[:]
	functionBytes := []byte(function)
	argsBytes, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal args: %v", err)
	}

	// 创建完整参数数据
	data := make([]byte, len(contractBytes)+4+len(functionBytes)+4+len(argsBytes))

	// 追加合约地址
	copy(data[:20], contractBytes)

	// 追加函数名长度和函数名
	binary.LittleEndian.PutUint32(data[20:24], uint32(len(functionBytes)))
	copy(data[24:24+len(functionBytes)], functionBytes)

	// 追加参数长度和参数
	offset := 24 + len(functionBytes)
	binary.LittleEndian.PutUint32(data[offset:offset+4], uint32(len(argsBytes)))
	copy(data[offset+4:], argsBytes)

	// 调用宿主函数，使用常量FuncCall
	ptr, size, _ := callHost(FuncCall, data)
	if ptr == 0 || size == 0 {
		return nil, fmt.Errorf("call failed")
	}

	return readMemory(ptr, size), nil
}

func (c *Context) GetObject(objectID ObjectID) (vm.Object, error) {
	// 调用宿主函数，使用常量FuncGetObject
	ptr, size, _ := callHost(FuncGetObject, objectID[:])
	if ptr == 0 || size == 0 {
		return &Object{}, fmt.Errorf("object not found")
	}

	data := readMemory(ptr, size)
	var obj Object
	copy(obj.id[:], data)
	return &obj, nil
}

func (c *Context) GetObjectWithOwner(owner Address) (vm.Object, error) {
	// 调用宿主函数，使用常量FuncGetObjectWithOwner
	ptr, size, _ := callHost(FuncGetObjectWithOwner, owner[:])
	if ptr == 0 || size == 0 {
		return &Object{}, fmt.Errorf("object not found")
	}

	data := readMemory(ptr, size)
	var obj Object
	copy(obj.id[:], data)
	return &obj, nil
}

func (c *Context) CreateObject() vm.Object {
	// 调用宿主函数，使用常量FuncCreateObject
	ptr, size, _ := callHost(FuncCreateObject, nil)
	if ptr == 0 || size == 0 {
		panic("failed to create object")
	}

	data := readMemory(ptr, size)
	var obj Object
	copy(obj.id[:], data)
	return &obj
}

func (c *Context) DeleteObject(objectID ObjectID) {
	// 调用宿主函数，使用常量FuncDeleteObject
	_, _, result := callHost(FuncDeleteObject, objectID[:])
	if result == 0 {
		panic("failed to delete object")
	}
}

func (c *Context) Log(event string, data ...any) {
	// 序列化参数
	eventBytes := []byte(event)
	dataBytes, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	// 创建完整参数数据
	paramData := make([]byte, 4+len(eventBytes)+4+len(dataBytes))

	// 追加事件长度和事件
	binary.LittleEndian.PutUint32(paramData[:4], uint32(len(eventBytes)))
	copy(paramData[4:4+len(eventBytes)], eventBytes)

	// 追加数据长度和数据
	offset := 4 + len(eventBytes)
	binary.LittleEndian.PutUint32(paramData[offset:offset+4], uint32(len(dataBytes)))
	copy(paramData[offset+4:], dataBytes)

	// 调用宿主函数，使用常量FuncLog
	callHost(FuncLog, paramData)
}

// Object interface implementation
func (o *Object) ID() ObjectID {
	return o.id
}

func (o *Object) Owner() Address {
	// 调用宿主函数，使用常量FuncGetObjectOwner
	ptr, size, _ := callHost(FuncGetObjectOwner, o.id[:])
	if ptr == 0 || size == 0 {
		return Address{}
	}
	data := readMemory(ptr, size)
	var addr Address
	copy(addr[:], data)
	return addr
}

func (o *Object) SetOwner(owner Address) {
	// 创建参数：对象ID + 所有者地址
	data := make([]byte, 32+20)
	copy(data[:32], o.id[:])
	copy(data[32:], owner[:])

	// 调用宿主函数，使用常量FuncSetObjectOwner
	_, _, result := callHost(FuncSetObjectOwner, data)
	if result == 0 {
		panic("failed to set owner")
	}
}

func (o *Object) Get(field string, value any) error {
	// 构造存储键
	key := fmt.Sprintf("%x:%s", o.id, field)

	// 调用宿主函数，使用常量FuncDbRead
	ptr, size, _ := callHost(FuncDbRead, []byte(key))
	if ptr == 0 || size == 0 {
		return fmt.Errorf("field not found: %s", field)
	}

	// 读取并反序列化数据
	data := readMemory(ptr, size)
	return json.Unmarshal(data, value)
}

func (o *Object) Set(field string, value any) error {
	// 序列化数据
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %v", err)
	}

	// 构造存储键
	key := fmt.Sprintf("%x:%s", o.id, field)

	// 创建参数：键长度 + 键 + 值长度 + 值
	paramData := make([]byte, 4+len(key)+4+len(data))

	// 写入键长度和键
	binary.LittleEndian.PutUint32(paramData[:4], uint32(len(key)))
	copy(paramData[4:4+len(key)], key)

	// 写入值长度和值
	offset := 4 + len(key)
	binary.LittleEndian.PutUint32(paramData[offset:offset+4], uint32(len(data)))
	copy(paramData[offset+4:], data)

	// 调用宿主函数，使用常量FuncDbWrite
	_, _, result := callHost(FuncDbWrite, paramData)
	if result == 0 {
		return fmt.Errorf("failed to write value")
	}
	return nil
}

func main() {
	// TinyGo requires a main function, but we don't use it
}

//export hello
func hello() int32 {
	fmt.Println("hello")
	ctx := &Context{}
	ctx.Log("hello", "world")
	object := ctx.CreateObject()
	fmt.Println(object)
	return 1
}

//export process_data
func process_data(dataPtr int32, dataLen int32) int32 {
	// 读取传入的数据
	data := readMemory(dataPtr, dataLen)

	// 处理数据 - 这里简单地计算数据的总和作为示例
	var sum int32 = 0
	for _, b := range data {
		sum += int32(b)
	}

	fmt.Printf("处理数据: 指针=%d, 长度=%d, 内容=%v, 总和=%d\n",
		dataPtr, dataLen, data, sum)

	// 返回处理结果
	return sum
}

// 导出的函数 - 内存分配
//
//export allocate
func allocate(size int32) int32 {
	// 分配指定大小的内存
	buffer := make([]byte, size)
	// 返回内存的指针
	return int32(uintptr(unsafe.Pointer(&buffer[0])))
}

// 导出的函数 - 内存释放
//
//export deallocate
func deallocate(ptr int32, size int32) {
	// 在WebAssembly中，内存管理由Go的垃圾收集器处理
	// 这个函数主要是为了提供与主机环境匹配的接口
}
