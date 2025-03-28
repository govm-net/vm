// WebAssembly 合约通信包装层
// 为智能合约提供与主机环境通信的标准接口
package main

import (
	"encoding/json"
	"fmt"
	"unsafe"

	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/mock"
	"github.com/govm-net/vm/types"
)

// 导入函数ID常量 - 从types包导入以确保一致性
const (
	FuncGetSender          = int32(types.FuncGetSender)
	FuncGetContractAddress = int32(types.FuncGetContractAddress)
	FuncTransfer           = int32(types.FuncTransfer)
	FuncCreateObject       = int32(types.FuncCreateObject)
	FuncCall               = int32(types.FuncCall)
	FuncGetObject          = int32(types.FuncGetObject)
	FuncGetObjectWithOwner = int32(types.FuncGetObjectWithOwner)
	FuncDeleteObject       = int32(types.FuncDeleteObject)
	FuncLog                = int32(types.FuncLog)
	FuncGetObjectOwner     = int32(types.FuncGetObjectOwner)
	FuncSetObjectOwner     = int32(types.FuncSetObjectOwner)
	FuncGetObjectField     = int32(types.FuncGetObjectField)
	FuncSetObjectField     = int32(types.FuncSetObjectField)
	FuncGetObjectContract  = int32(types.FuncGetObjectContract)
)

// 定义全局接收数据缓冲区的大小
const HostBufferSize int32 = types.HostBufferSize

// 使用全局变量存储动态分配的主机缓冲区地址
var hostBufferPtr int32 = 0
var hostBuffer []byte

// 定义基本类型
type Address = core.Address
type ObjectID = core.ObjectID

// ZeroAddress 表示空地址
var ZeroAddress Address

// Context 实现智能合约执行上下文接口
type Context struct {
	sender          Address
	blockHeight     uint64
	blockTime       int64
	contractAddress Address
}

// Object 实现状态对象接口
type Object struct {
	id ObjectID
}

var _ core.Context = &Context{}
var _ core.Object = &Object{}

// 从主机环境导入的函数 - 这些函数由主机环境提供

func init() {
	hostBuffer = make([]byte, HostBufferSize)
	hostBufferPtr = int32(uintptr(unsafe.Pointer(&hostBuffer[0])))
}

//go:wasmimport env call_host_set
//export call_host_set
func call_host_set(funcID, argPtr, argLen, bufferPtr int32) int32

//go:wasmimport env call_host_get_buffer
//export call_host_get_buffer
func call_host_get_buffer(funcID, argPtr, argLen, bufferPtr int32) int32

// 一些常用的直接导出函数，提供性能优化

//go:wasmimport env get_block_height
//export get_block_height
func get_block_height() int64

//go:wasmimport env get_block_time
//export get_block_time
func get_block_time() int64

//go:wasmimport env get_balance
//export get_balance
func get_balance(addrPtr int32) uint64

//export get_buffer_address
func get_buffer_address() int32 {
	return hostBufferPtr
}

// 辅助函数 - 与主机环境通信的核心处理函数
func callHost(funcID int32, data []byte) (resultPtr int32, resultSize int32, errCode int32) {
	var argPtr int32 = 0
	var argLen int32 = 0

	fmt.Println("[wasm]--callHost", funcID, string(data))
	if len(data) > 0 {
		// 获取参数数据的指针和长度
		copy(hostBuffer[:len(data)], data)
		argPtr = hostBufferPtr
		argLen = int32(len(data))
	}

	// 根据函数ID选择合适的调用方式
	switch funcID {
	// 需要通过缓冲区返回复杂数据的函数
	case FuncGetSender, FuncGetContractAddress, FuncCall,
		FuncGetObject, FuncGetObjectWithOwner, FuncCreateObject,
		FuncGetObjectOwner, FuncGetObjectField:

		// 使用获取缓冲区数据的宿主函数（返回数据大小）
		resultSize = call_host_get_buffer(funcID, argPtr, argLen, hostBufferPtr)
		if resultSize > 0 {
			// 数据已存储在全局缓冲区
			resultPtr = hostBufferPtr
			errCode = 0
		} else if resultSize == 0 {
			errCode = -1
		} else {
			errCode = resultSize
		}

	// 不需要返回数据的函数或返回简单值的函数
	default:
		// 使用设置数据的宿主函数
		resultSize = call_host_set(funcID, argPtr, argLen, hostBufferPtr)
		if resultSize >= 0 {
			resultPtr = hostBufferPtr
			errCode = 0
		} else {
			errCode = resultSize
		}
	}

	return resultPtr, resultSize, errCode
}

// 从内存读取数据
func readMemory(ptr, size int32) []byte {
	// 安全性检查
	if ptr == 0 || size <= 0 {
		return []byte{}
	}

	// 创建结果数组
	data := make([]byte, size)

	// 从指定位置读取数据
	src := unsafe.Pointer(uintptr(ptr))

	// 使用安全的复制方式
	for i := int32(0); i < size; i++ {
		data[i] = *(*byte)(unsafe.Pointer(uintptr(src) + uintptr(i)))
	}

	return data
}

// 将数据写入内存
func any2bytes(data interface{}) (bytes []byte, err error) {
	switch v := data.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		bytes, err = json.Marshal(v)
		if err != nil {
			return nil, err
		}
	}

	return bytes, nil
}

// 将数据写入内存
func writeToMemory(data interface{}) (ptr int32, size int32, err error) {
	bytes, err := any2bytes(data)
	if err != nil {
		return 0, 0, err
	}

	ptr = int32(uintptr(unsafe.Pointer(&bytes[0])))
	size = int32(len(bytes))
	return ptr, size, nil
}

// Context 接口实现

// Sender 返回调用合约的账户地址
func (c *Context) Sender() Address {
	addr := mock.GetCaller()
	if addr != ZeroAddress {
		return addr
	}
	if c.sender != ZeroAddress {
		return c.sender
	}
	// 调用宿主函数，使用常量FuncGetSender
	ptr, size, _ := callHost(FuncGetSender, nil)
	data := readMemory(ptr, size)
	copy(addr[:], data)
	c.sender = addr
	return addr
}

// BlockHeight 返回当前区块高度
func (c *Context) BlockHeight() uint64 {
	if c.blockHeight != 0 {
		return c.blockHeight
	}
	// 直接调用宿主函数，无需经过callHost中转
	value := get_block_height()
	c.blockHeight = uint64(value)
	return uint64(value)
}

// BlockTime 返回当前区块时间戳
func (c *Context) BlockTime() int64 {
	if c.blockTime != 0 {
		return c.blockTime
	}
	// 直接调用宿主函数，无需经过callHost中转
	value := get_block_time()
	c.blockTime = value
	return value
}

// ContractAddress 返回当前合约地址
func (c *Context) ContractAddress() Address {
	addr := mock.GetCurrentContract()
	if addr != ZeroAddress {
		return addr
	}
	if c.contractAddress != ZeroAddress {
		return c.contractAddress
	}
	// 调用宿主函数，使用常量FuncGetContractAddress
	ptr, size, _ := callHost(FuncGetContractAddress, nil)
	data := readMemory(ptr, size)

	copy(addr[:], data)
	c.contractAddress = addr
	return addr
}

// Balance 返回指定地址的余额
func (c *Context) Balance(addr Address) uint64 {
	// 直接调用宿主函数
	return get_balance(int32(uintptr(unsafe.Pointer(&addr[0]))))
}

// Transfer 从合约转账到指定地址
func (c *Context) Transfer(to Address, amount uint64) error {
	data := types.TransferParams{
		From:   c.ContractAddress(),
		To:     to,
		Amount: amount,
	}

	buff, err := any2bytes(data)
	if err != nil {
		return fmt.Errorf("failed to serialize transfer data: %w", err)
	}

	// 调用宿主函数，使用常量FuncTransfer
	_, _, result := callHost(FuncTransfer, buff)
	if result != 0 {
		return fmt.Errorf("transfer failed with code: %d", result)
	}
	return nil
}

// Call 调用另一个合约的函数
func (c *Context) Call(contract Address, function string, args ...interface{}) ([]byte, error) {
	// 构造调用参数
	callData := types.CallParams{
		Contract: contract,
		Function: function,
		Args:     args,
		Caller:   c.ContractAddress(), // 当前合约作为调用者
	}

	// 序列化调用参数
	bytes, err := any2bytes(callData)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize call data: %w", err)
	}

	// 调用主机函数
	resultPtr, resultSize, errCode := callHost(FuncCall, bytes)
	if errCode != 0 {
		return nil, fmt.Errorf("contract call failed with code: %d", errCode)
	}

	// 读取返回数据
	return readMemory(resultPtr, resultSize), nil
}

// CreateObject 创建一个新的状态对象
func (c *Context) CreateObject() core.Object {
	address := c.ContractAddress()
	// 调用宿主函数，创建对象并获取对象ID
	ptr, size, errCode := callHost(FuncCreateObject, address[:])
	if errCode != 0 {
		panic(fmt.Sprintf("failed to create object with code: %d", errCode))
	}

	// 解析对象ID
	idData := readMemory(ptr, size)
	var id ObjectID
	copy(id[:], idData)

	// 返回对象包装器
	return &Object{id: id}
}

// GetObject 获取指定ID的状态对象
func (c *Context) GetObject(id ObjectID) (core.Object, error) {
	var request types.GetObjectParams
	request.Contract = c.ContractAddress()
	request.ID = id

	// 将对象ID序列化
	bytes, err := any2bytes(request)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize object ID: %w", err)
	}

	// 调用宿主函数
	_, resultSize, errCode := callHost(FuncGetObject, bytes)
	if errCode != 0 {
		return nil, fmt.Errorf("failed to get object with code: %d", errCode)
	}

	// 解析对象数据
	if resultSize == 0 {
		return nil, fmt.Errorf("object not found")
	}

	return &Object{id: id}, nil
}

// GetObjectWithOwner 获取指定所有者的状态对象
func (c *Context) GetObjectWithOwner(owner Address) (core.Object, error) {
	var request types.GetObjectWithOwnerParams
	request.Contract = c.ContractAddress()
	request.Owner = owner

	// 将所有者地址序列化
	bytes, err := any2bytes(request)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize owner address: %w", err)
	}

	// 调用宿主函数
	resultPtr, resultSize, errCode := callHost(FuncGetObjectWithOwner, bytes)
	if errCode != 0 {
		return nil, fmt.Errorf("failed to get object with code: %d", errCode)
	}

	// 解析对象ID
	if resultSize == 0 {
		return nil, fmt.Errorf("object not found")
	}

	idData := readMemory(resultPtr, resultSize)
	var id ObjectID
	copy(id[:], idData)

	return &Object{id: id}, nil
}

// DeleteObject 删除指定ID的状态对象
func (c *Context) DeleteObject(id ObjectID) {
	var request types.DeleteObjectParams
	request.Contract = c.ContractAddress()
	request.ID = id

	// 将对象ID序列化
	bytes, err := any2bytes(request)
	if err != nil {
		panic(fmt.Sprintf("failed to serialize object ID: %v", err))
	}

	// 调用宿主函数
	_, _, errCode := callHost(FuncDeleteObject, bytes)
	if errCode != 0 {
		panic(fmt.Sprintf("failed to delete object with code: %d", errCode))
	}
}

// Log 记录事件
func (c *Context) Log(event string, keyValues ...any) {
	var request types.LogParams
	request.Contract = c.ContractAddress()
	request.Event = event
	request.KeyValues = keyValues

	// 将请求序列化
	bytes, err := any2bytes(request)
	if err != nil {
		panic(fmt.Sprintf("failed to serialize log request: %v", err))
	}

	// 调用宿主函数 - 忽略返回值，只关心发送日志操作
	_, _, _ = callHost(FuncLog, bytes)
}

// Object 接口实现

// ID 返回对象的唯一标识符
func (o *Object) ID() ObjectID {
	return o.id
}

// Owner 返回对象的所有者地址
func (o *Object) Owner() Address {
	// 将对象ID序列化
	bytes, err := any2bytes(o.id)
	if err != nil {
		return ZeroAddress
	}

	// 调用宿主函数
	resultPtr, resultSize, _ := callHost(FuncGetObjectOwner, bytes)
	if resultSize == 0 {
		return ZeroAddress
	}

	// 解析所有者地址
	ownerData := readMemory(resultPtr, resultSize)
	var owner Address
	copy(owner[:], ownerData)

	return owner
}

// SetOwner 设置对象的所有者
func (o *Object) SetOwner(owner Address) {
	// 构造参数
	request := types.SetOwnerParams{
		Contract: mock.GetCurrentContract(),
		Sender:   mock.GetCaller(),
		ID:       o.id,
		Owner:    owner,
	}

	// 序列化参数
	bytes, err := any2bytes(request)
	if err != nil {
		panic(fmt.Sprintf("failed to serialize data: %v", err))
	}

	// 调用宿主函数
	_, _, errCode := callHost(FuncSetObjectOwner, bytes)
	if errCode != 0 {
		panic(fmt.Sprintf("set owner failed with code: %d", errCode))
	}
}

// Get 获取对象字段的值
func (o *Object) Get(field string, value interface{}) error {
	// 构造参数
	getData := types.GetObjectFieldParams{
		Contract: mock.GetCurrentContract(),
		ID:       o.id,
		Field:    field,
	}

	// 序列化参数
	bytes, err := any2bytes(getData)
	if err != nil {
		return fmt.Errorf("failed to serialize data: %w", err)
	}

	// 调用宿主函数
	resultPtr, resultSize, errCode := callHost(FuncGetObjectField, bytes)
	if errCode != 0 {
		return fmt.Errorf("get field failed with code: %d", errCode)
	}

	// 解析字段值
	if resultSize == 0 {
		return fmt.Errorf("field not found")
	}

	// 读取字段数据
	fieldData := readMemory(resultPtr, resultSize)
	fmt.Println("[wasm]--fieldData", field, string(fieldData))
	if err := json.Unmarshal(fieldData, value); err != nil {
		return fmt.Errorf("failed to unmarshal to target type: %w", err)
	}

	return nil
}

// Set 设置对象字段的值
func (o *Object) Set(field string, value interface{}) error {
	// 构造参数
	request := types.SetObjectFieldParams{
		Contract: mock.GetCurrentContract(),
		Sender:   mock.GetCaller(),
		ID:       o.id,
		Field:    field,
		Value:    value,
	}

	// 序列化参数
	bytes, err := any2bytes(request)
	if err != nil {
		return fmt.Errorf("failed to serialize data: %w", err)
	}

	// 调用宿主函数
	_, _, errCode := callHost(FuncSetObjectField, bytes)
	if errCode != 0 {
		return fmt.Errorf("set field failed with code: %d", errCode)
	}

	return nil
}

func (o *Object) Contract() Address {
	// 调用宿主函数
	resultPtr, resultSize, errCode := callHost(FuncGetObjectContract, o.id[:])
	if errCode != 0 {
		return ZeroAddress
	}
	// 解析字段值
	if resultSize == 0 {
		return ZeroAddress
	}

	// 读取字段数据
	fieldData := readMemory(resultPtr, resultSize)

	var contract Address
	copy(contract[:], fieldData)
	return contract
}

// 内存管理函数 - 供主机环境使用

//export allocate
func allocate(size int32) int32 {
	// 分配指定大小的内存
	buffer := make([]byte, size)
	// 返回内存地址
	return int32(uintptr(unsafe.Pointer(&buffer[0])))
}

//export deallocate
func deallocate(ptr int32, size int32) {
	// 在Go中，内存由GC管理，无需手动释放
	// 这个函数只是为了符合WASM接口要求
}

// 错误码常量
const (
	// 成功
	ErrorCodeSuccess int32 = 0
	// 函数未找到
	ErrorCodeFunctionNotFound int32 = -1
	// 参数解析错误
	ErrorCodeInvalidParams int32 = -2
	// 执行错误
	ErrorCodeExecutionError int32 = -3
)

// 全局错误消息
var lastErrorMessage string

// 合约函数处理器类型
type ContractFunctionHandler func(ctx *Context, params []byte) (interface{}, error)

// 合约函数处理表，将在初始化时填充
var contractFunctions = map[string]ContractFunctionHandler{}

// 合约函数注册器，用于将合约函数处理器添加到分发表中
func registerContractFunction(name string, handler ContractFunctionHandler) {
	contractFunctions[name] = handler
}

// 初始化处理表
func init() {
	// 注册合约中的函数
	registerContractFunction("Transfer", handleTransfer)
	// 其他函数注册可以添加在这里
}

// 统一合约入口函数
// 参数:
// - funcNamePtr: 函数名的内存指针
// - funcNameLen: 函数名的长度
// - paramsPtr: 参数的内存指针
// - paramsLen: 参数的长度
// 返回值:
// - 结果指针或错误码
//
//export handle_contract_call
func handle_contract_call(inputPtr, inputLen int32) int32 {
	fmt.Println("handle_contract_call", inputPtr, inputLen)
	// 读取函数名
	inputBytes := readMemory(inputPtr, inputLen)
	var input types.HandleContractCallParams
	if err := json.Unmarshal(inputBytes, &input); err != nil {
		return ErrorCodeExecutionError
	}
	functionName := input.Function

	// 读取参数
	paramsBytes := input.Args
	mock.Enter(input.Sender, "handle_contract_call")
	mock.Enter(input.Contract, functionName)

	// 使用mock模块记录函数进入
	ctx := &Context{}

	// 获取处理函数
	handler, exists := contractFunctions[functionName]
	if !exists {
		// 函数未找到
		errMsg := fmt.Sprintf("Function not found: %s", functionName)
		fmt.Println(errMsg)

		// 返回错误结果
		result := types.ExecutionResult{
			Success: false,
			Error:   errMsg,
		}

		// 序列化结果
		resultBytes, err := any2bytes(result)
		if err != nil {
			return ErrorCodeExecutionError
		}

		// 写入全局缓冲区
		if hostBufferPtr != 0 && len(resultBytes) <= len(hostBuffer) {
			copy(hostBuffer[:len(resultBytes)], resultBytes)
			return int32(len(resultBytes))
		}

		return ErrorCodeFunctionNotFound
	}

	fmt.Println("handler, exists", handler, exists)

	// 执行处理函数
	data, err := handler(ctx, paramsBytes)
	if err != nil {
		// 执行出错
		errMsg := fmt.Sprintf("Execution error: %v", err)
		fmt.Println(errMsg)

		// 返回错误结果
		result := types.ExecutionResult{
			Success: false,
			Error:   errMsg,
		}

		// 序列化结果
		resultPtr, resultSize, err := writeToMemory(result)
		if err != nil {
			return ErrorCodeExecutionError
		}

		// 写入全局缓冲区
		if hostBufferPtr != 0 && resultSize <= HostBufferSize {
			copy(hostBuffer[:resultSize], readMemory(resultPtr, resultSize))
			return resultSize
		}

		return ErrorCodeExecutionError
	}

	// 成功执行
	result := types.ExecutionResult{
		Success: true,
		Data:    data,
	}
	fmt.Println("contract result", result)

	// 序列化结果
	resultPtr, resultSize, err := writeToMemory(result)
	if err != nil {
		fmt.Println("Failed to serialize result: ", err)
		return ErrorCodeExecutionError
	}

	// 写入全局缓冲区
	if hostBufferPtr != 0 && resultSize <= HostBufferSize {
		copy(hostBuffer[:resultSize], readMemory(resultPtr, resultSize))
		return resultSize
	}

	// 直接返回结果指针（如果缓冲区不可用）
	return resultPtr
}

// 示例合约函数处理器 - 实际项目中应根据需求实现
func handleTransfer(ctx *Context, params []byte) (interface{}, error) {
	// 解析参数
	var transferParams struct {
		To     Address `json:"to"`
		Amount uint64  `json:"amount"`
	}

	if err := json.Unmarshal(params, &transferParams); err != nil {
		return nil, fmt.Errorf("invalid transfer parameters: %w", err)
	}

	// 执行转账
	success := Transfer(ctx, transferParams.To, transferParams.Amount)
	if !success {
		return nil, fmt.Errorf("transfer failed")
	}

	// 返回成功结果
	return map[string]interface{}{
		"status": "success",
		"to":     transferParams.To,
		"amount": transferParams.Amount,
	}, nil
}

//export hello
func hello() int32 {
	fmt.Println("hello world")
	return 100
}

// WebAssembly 要求 main 函数
func main() {
	// 此函数在 WebAssembly 中不会被执行
}
