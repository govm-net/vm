// WebAssembly Contract Communication Wrapper Layer
// Provides standard interfaces for smart contracts to communicate with the host environment
package main

import (
	"encoding/json"
	"fmt"
	"runtime"
	"unsafe"

	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/mock"
	"github.com/govm-net/vm/types"
)

// Import function ID constants - imported from types package to ensure consistency
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

// Define the size of the global receive data buffer
const HostBufferSize int32 = types.HostBufferSize

// Use global variables to store dynamically allocated host buffer address
var hostBufferPtr int32 = 0
var hostBuffer []byte

// Define basic types
type Address = core.Address
type ObjectID = core.ObjectID

// ZeroAddress represents an empty address
var ZeroAddress Address

// Context implements the smart contract execution context interface
type Context struct {
	sender          Address
	blockHeight     uint64
	blockTime       int64
	contractAddress Address
}

// Object implements the state object interface
type Object struct {
	id ObjectID
}

var _ types.Context = &Context{}
var _ core.Object = &Object{}

// Functions imported from the host environment - these functions are provided by the host environment

func init() {
	hostBuffer = make([]byte, HostBufferSize)
	hostBufferPtr = int32(uintptr(unsafe.Pointer(&hostBuffer[0])))

	// Limit to using only one goroutine
	runtime.GOMAXPROCS(1)

	// Register contract functions
	// registerContractFunction("Transfer", handleTransfer)
	// Other function registrations can be added here
}

//go:wasmimport env call_host_set
//export call_host_set
func call_host_set(funcID, argPtr, argLen, bufferPtr int32) int32

//go:wasmimport env call_host_get_buffer
//export call_host_get_buffer
func call_host_get_buffer(funcID, argPtr, argLen, bufferPtr int32) int32

// Some commonly used direct export functions for performance optimization

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

// Helper functions - core processing functions for communication with the host environment
func callHost(funcID int32, data []byte) (resultPtr int32, resultSize int32, errCode int32) {
	var argPtr int32 = 0
	var argLen int32 = 0
	if len(data) > int(HostBufferSize) {
		panic(fmt.Sprintf("data length %d is too long, max is %d", len(data), HostBufferSize))
	}

	// fmt.Println("[wasm]--callHost", funcID, string(data))
	if len(data) > 0 {
		// Get pointer and length of parameter data
		copy(hostBuffer[:len(data)], data)
		argPtr = hostBufferPtr
		argLen = int32(len(data))
	}

	// Choose appropriate calling method based on function ID
	switch funcID {
	// Functions that need to return complex data through buffer
	case FuncGetSender, FuncGetContractAddress, FuncCall,
		FuncGetObject, FuncGetObjectWithOwner, FuncCreateObject,
		FuncGetObjectOwner, FuncGetObjectField:

		// Use host function to get buffer data (returns data size)
		resultSize = call_host_get_buffer(funcID, argPtr, argLen, hostBufferPtr)
		if resultSize > 0 {
			// Data is stored in global buffer
			resultPtr = hostBufferPtr
			errCode = 0
		} else if resultSize == 0 {
			errCode = -1
		} else {
			errCode = resultSize
		}

	// Functions that don't need to return data or return simple values
	default:
		// Use host function to set data
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

// Read data from memory
func readMemory(ptr, size int32) []byte {
	// Safety check
	if ptr == 0 || size <= 0 {
		return []byte{}
	}

	// Create result array
	data := make([]byte, size)

	// Read data from specified location
	for i := int32(0); i < size; i++ {
		data[i] = *(*byte)(unsafe.Pointer(uintptr(ptr) + uintptr(i)))
	}

	return data
}

// Convert data to bytes
func any2bytes(data any) (bytes []byte, err error) {
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

// Write data to memory
func writeToMemory(data any) (ptr int32, size int32, err error) {
	bytes, err := any2bytes(data)
	if err != nil {
		return 0, 0, err
	}

	ptr = int32(uintptr(unsafe.Pointer(&bytes[0])))
	size = int32(len(bytes))
	return ptr, size, nil
}

// Context interface implementation

// Sender returns the account address that called the contract
func (c *Context) Sender() Address {
	mock.ConsumeGas(10)
	addr := mock.GetCaller()
	if addr != ZeroAddress {
		return addr
	}
	if c.sender != ZeroAddress {
		return c.sender
	}
	// Call host function using FuncGetSender constant
	ptr, size, _ := callHost(FuncGetSender, nil)
	data := readMemory(ptr, size)
	copy(addr[:], data)
	c.sender = addr
	return addr
}

// BlockHeight returns the current block height
func (c *Context) BlockHeight() uint64 {
	mock.ConsumeGas(10)
	if c.blockHeight != 0 {
		return c.blockHeight
	}
	// Directly call host function without going through callHost
	value := get_block_height()
	c.blockHeight = uint64(value)
	return uint64(value)
}

// BlockTime returns the current block timestamp
func (c *Context) BlockTime() int64 {
	mock.ConsumeGas(10)
	if c.blockTime != 0 {
		return c.blockTime
	}
	// Directly call host function without going through callHost
	value := get_block_time()
	c.blockTime = value
	return value
}

// ContractAddress returns the current contract address
func (c *Context) ContractAddress() Address {
	mock.ConsumeGas(10)
	addr := mock.GetCurrentContract()
	if addr != ZeroAddress {
		return addr
	}
	if c.contractAddress != ZeroAddress {
		return c.contractAddress
	}
	// Call host function using FuncGetContractAddress constant
	ptr, size, _ := callHost(FuncGetContractAddress, nil)
	data := readMemory(ptr, size)

	copy(addr[:], data)
	c.contractAddress = addr
	return addr
}

// Balance returns the balance of the specified address
func (c *Context) Balance(addr Address) uint64 {
	mock.ConsumeGas(50)
	// Directly call host function
	return get_balance(int32(uintptr(unsafe.Pointer(&addr[0]))))
}

// Transfer transfers tokens from the contract to the specified address
func (c *Context) Transfer(from Address, to Address, amount uint64) error {
	mock.ConsumeGas(500)
	data := types.TransferParams{
		Contract: c.ContractAddress(),
		From:     from,
		To:       to,
		Amount:   amount,
	}

	buff, err := any2bytes(data)
	if err != nil {
		return fmt.Errorf("failed to serialize transfer data: %w", err)
	}

	// Call host function using FuncTransfer constant
	_, _, result := callHost(FuncTransfer, buff)
	if result != 0 {
		return fmt.Errorf("transfer failed with code: %d", result)
	}
	return nil
}

// Call calls a function on another contract
func (c *Context) Call(contract Address, function string, args ...any) ([]byte, error) {
	mock.ConsumeGas(10000)
	// Construct call parameters
	callData := types.CallParams{
		Contract: contract,
		Function: function,
		Args:     args,
		Caller:   c.ContractAddress(), // Current contract as caller
		GasLimit: mock.GetGas() - 10000,
	}

	// Serialize call parameters
	bytes, err := any2bytes(callData)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize call data: %w", err)
	}

	// Call host function
	resultPtr, resultSize, errCode := callHost(FuncCall, bytes)
	if errCode != 0 {
		return nil, fmt.Errorf("contract call failed with code: %d", errCode)
	}

	data := readMemory(resultPtr, resultSize)
	var callResult types.CallResult
	if err := json.Unmarshal(data, &callResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal call result: %w", err)
	}
	// Deduct actual gas used
	mock.ConsumeGas(callResult.GasUsed)

	// Read return data
	return callResult.Data, nil
}

// CreateObject creates a new state object
func (c *Context) CreateObject() core.Object {
	mock.ConsumeGas(500)
	address := c.ContractAddress()
	// Call host function to create object and get object ID
	ptr, size, errCode := callHost(FuncCreateObject, address[:])
	if errCode != 0 {
		panic(fmt.Sprintf("failed to create object with code: %d", errCode))
	}

	// Parse object ID
	idData := readMemory(ptr, size)
	var id ObjectID
	copy(id[:], idData)

	// Return object wrapper
	return &Object{id: id}
}

// GetObject retrieves a state object by ID
func (c *Context) GetObject(id ObjectID) (core.Object, error) {
	mock.ConsumeGas(50)
	var request types.GetObjectParams
	request.Contract = c.ContractAddress()
	request.ID = id

	// Serialize object ID
	bytes, err := any2bytes(request)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize object ID: %w", err)
	}

	// Call host function
	_, resultSize, errCode := callHost(FuncGetObject, bytes)
	if errCode != 0 {
		return nil, fmt.Errorf("failed to get object with code: %d", errCode)
	}

	// Parse object data
	if resultSize == 0 {
		return nil, fmt.Errorf("object not found")
	}

	return &Object{id: id}, nil
}

// GetObjectWithOwner retrieves a state object by owner
func (c *Context) GetObjectWithOwner(owner Address) (core.Object, error) {
	mock.ConsumeGas(50)
	var request types.GetObjectWithOwnerParams
	request.Contract = c.ContractAddress()
	request.Owner = owner

	// Serialize owner address
	bytes, err := any2bytes(request)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize owner address: %w", err)
	}

	// Call host function
	resultPtr, resultSize, errCode := callHost(FuncGetObjectWithOwner, bytes)
	if errCode != 0 {
		return nil, fmt.Errorf("failed to get object with code: %d", errCode)
	}

	// Parse object ID
	if resultSize == 0 {
		return nil, fmt.Errorf("object not found")
	}

	idData := readMemory(resultPtr, resultSize)
	var id ObjectID
	copy(id[:], idData)

	return &Object{id: id}, nil
}

// DeleteObject deletes a state object by ID
func (c *Context) DeleteObject(id ObjectID) {
	mock.ConsumeGas(500)
	var request types.DeleteObjectParams
	request.Contract = c.ContractAddress()
	request.ID = id

	// Serialize
	bytes, err := any2bytes(request)
	if err != nil {
		panic(fmt.Sprintf("failed to serialize object ID: %v", err))
	}

	// Call host function
	_, _, errCode := callHost(FuncDeleteObject, bytes)
	if errCode != 0 {
		panic(fmt.Sprintf("failed to delete object with code: %d", errCode))
	}
	mock.RefundGas(800)
}

// Log records an event
func (c *Context) Log(event string, keyValues ...any) {
	mock.ConsumeGas(100)
	var request types.LogParams
	request.Contract = c.ContractAddress()
	request.Event = event
	request.KeyValues = keyValues

	// Serialize request
	bytes, err := any2bytes(request)
	if err != nil {
		panic(fmt.Sprintf("failed to serialize log request: %v", err))
	}
	mock.ConsumeGas(int64(len(bytes)))

	// Call host function - ignore return value, only care about sending log operation
	_, _, _ = callHost(FuncLog, bytes)
}

// Object interface implementation

// ID returns the unique identifier of the object
func (o *Object) ID() ObjectID {
	mock.ConsumeGas(10)
	return o.id
}

// Owner returns the owner address of the object
func (o *Object) Owner() Address {
	mock.ConsumeGas(100)
	// Serialize object ID
	bytes, err := any2bytes(o.id)
	if err != nil {
		return ZeroAddress
	}

	// Call host function
	resultPtr, resultSize, _ := callHost(FuncGetObjectOwner, bytes)
	if resultSize == 0 {
		return ZeroAddress
	}

	// Parse owner address
	ownerData := readMemory(resultPtr, resultSize)
	var owner Address
	copy(owner[:], ownerData)

	return owner
}

// SetOwner sets the owner of the object
func (o *Object) SetOwner(owner Address) {
	mock.ConsumeGas(500)
	// Construct parameters
	request := types.SetOwnerParams{
		Contract: mock.GetCurrentContract(),
		Sender:   mock.GetCaller(),
		ID:       o.id,
		Owner:    owner,
	}

	// Serialize parameters
	bytes, err := any2bytes(request)
	if err != nil {
		panic(fmt.Sprintf("failed to serialize data: %v", err))
	}

	// Call host function
	_, _, errCode := callHost(FuncSetObjectOwner, bytes)
	if errCode != 0 {
		panic(fmt.Sprintf("set owner failed with code: %d", errCode))
	}
}

// Get retrieves the value of an object field
func (o *Object) Get(field string, value any) error {
	mock.ConsumeGas(100)
	// Construct parameters
	getData := types.GetObjectFieldParams{
		Contract: mock.GetCurrentContract(),
		ID:       o.id,
		Field:    field,
	}

	// Serialize parameters
	bytes, err := any2bytes(getData)
	if err != nil {
		return fmt.Errorf("failed to serialize data: %w", err)
	}

	// Call host function
	resultPtr, resultSize, errCode := callHost(FuncGetObjectField, bytes)
	if errCode != 0 {
		return fmt.Errorf("get field failed with code: %d", errCode)
	}

	// Parse field value
	if resultSize == 0 {
		return fmt.Errorf("field not found")
	}

	// Read field data
	fieldData := readMemory(resultPtr, resultSize)
	mock.ConsumeGas(int64(resultSize))
	if err := json.Unmarshal(fieldData, value); err != nil {
		return fmt.Errorf("failed to unmarshal to target type, field: %s, value: %s, err: %w", field, fieldData, err)
	}

	return nil
}

// Set sets the value of an object field
func (o *Object) Set(field string, value any) error {
	mock.ConsumeGas(1000)
	// Construct parameters
	request := types.SetObjectFieldParams{
		Contract: mock.GetCurrentContract(),
		Sender:   mock.GetCaller(),
		ID:       o.id,
		Field:    field,
		Value:    value,
	}

	// Serialize parameters
	bytes, err := any2bytes(request)
	if err != nil {
		return fmt.Errorf("failed to serialize data: %w", err)
	}
	mock.ConsumeGas(int64(len(bytes)) * 100)

	// Call host function
	_, _, errCode := callHost(FuncSetObjectField, bytes)
	if errCode != 0 {
		return fmt.Errorf("set field failed with code: %d", errCode)
	}

	return nil
}

func (o *Object) Contract() Address {
	mock.ConsumeGas(100)
	// Call host function
	resultPtr, resultSize, errCode := callHost(FuncGetObjectContract, o.id[:])
	if errCode != 0 {
		return ZeroAddress
	}
	// Parse field value
	if resultSize == 0 {
		return ZeroAddress
	}

	// Read field data
	fieldData := readMemory(resultPtr, resultSize)

	var contract Address
	copy(contract[:], fieldData)
	return contract
}

// Memory management functions - for host environment use

//export allocate
func allocate(size int32) int32 {
	// Allocate memory of specified size
	buffer := make([]byte, size)
	// Return memory address
	return int32(uintptr(unsafe.Pointer(&buffer[0])))
}

//export deallocate
func deallocate(ptr int32, size int32) {
	// In Go, memory is managed by GC, no need for manual deallocation
	// This function is only to comply with WASM interface requirements
}

// Error code constants
const (
	// Success
	ErrorCodeSuccess int32 = 0
	// Function not found
	ErrorCodeFunctionNotFound int32 = -1
	// Parameter parsing error
	ErrorCodeInvalidParams int32 = -2
	// Execution error
	ErrorCodeExecutionError int32 = -3
	// Execution error
	ErrorCodeExecutionPanic int32 = -4
)

// Global error message
var lastErrorMessage string

// Contract function handler type
type ContractFunctionHandler func(params []byte) (any, error)

// Contract function dispatch table, will be populated during initialization
var contractFunctions = map[string]ContractFunctionHandler{}

// Contract function registrar, used to add contract function handlers to the dispatch table
func registerContractFunction(name string, handler ContractFunctionHandler) {
	contractFunctions[name] = handler
}

// Unified contract entry function
// Parameters:
// - funcNamePtr: Memory pointer to function name
// - funcNameLen: Length of function name
// - paramsPtr: Memory pointer to parameters
// - paramsLen: Length of parameters
// Return value:
// - Result pointer or error code
//
//export handle_contract_call
func handle_contract_call(inputPtr, inputLen int32) (code int32) {
	// fmt.Println("handle_contract_call", inputPtr, inputLen)
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("handle_contract_call panic")
			code = ErrorCodeExecutionPanic
		}
	}()
	// Read function name
	inputBytes := readMemory(inputPtr, inputLen)
	var input types.HandleContractCallParams
	if err := json.Unmarshal(inputBytes, &input); err != nil {
		return ErrorCodeInvalidParams
	}
	functionName := input.Function
	// fmt.Println("handle_contract_call gasLimit", input.GasLimit)
	if input.GasLimit > 0 {
		mock.ResetGas(input.GasLimit)
	}

	// fmt.Println("handle_contract_call functionName", functionName, string(input.Args))

	// Read parameters
	paramsBytes := input.Args
	mock.Enter(input.Sender.String(), "handle_contract_call")
	mock.Enter(input.Contract.String(), functionName)

	// Use mock module to record function entry
	ctx := &Context{}

	// Get handler function
	handler, exists := contractFunctions[functionName]
	if !exists {
		// Function not found
		errMsg := fmt.Sprintf("Function not found: %s", functionName)
		fmt.Println(errMsg)

		// Return error result
		result := types.ExecutionResult{
			Success: false,
			Error:   errMsg,
		}

		// Serialize result
		resultBytes, err := any2bytes(result)
		if err != nil {
			return ErrorCodeExecutionError
		}

		// Write to global buffer
		if hostBufferPtr != 0 && len(resultBytes) <= len(hostBuffer) {
			copy(hostBuffer[:len(resultBytes)], resultBytes)
			return int32(len(resultBytes))
		}

		return ErrorCodeFunctionNotFound
	}

	// fmt.Println("handler, exists", handler, exists)
	core.SetContext(ctx)

	// Execute handler function
	data, err := handler(paramsBytes)
	if err != nil {
		// Execution error
		errMsg := fmt.Sprintf("Execution error: %v", err)
		fmt.Println(errMsg)

		// Return error result
		result := types.ExecutionResult{
			Success: false,
			Error:   errMsg,
		}

		// Serialize result
		resultPtr, resultSize, err := writeToMemory(result)
		if err != nil {
			return ErrorCodeExecutionError
		}

		// Write to global buffer
		if hostBufferPtr != 0 && resultSize <= HostBufferSize {
			copy(hostBuffer[:resultSize], readMemory(resultPtr, resultSize))
			return resultSize
		}

		return ErrorCodeExecutionError
	}

	// Successful execution
	result := types.ExecutionResult{
		Success: true,
		Data:    data,
	}
	// fmt.Println("contract result", result)

	// Serialize result
	resultPtr, resultSize, err := writeToMemory(result)
	if err != nil {
		fmt.Println("Failed to serialize result: ", err)
		return ErrorCodeExecutionError
	}

	// Write to global buffer
	if hostBufferPtr != 0 && resultSize <= HostBufferSize {
		copy(hostBuffer[:resultSize], readMemory(resultPtr, resultSize))
		return resultSize
	}

	// Directly return result pointer (if buffer is not available)
	return resultPtr
}

// WebAssembly requires a main function
func main() {
	// This function will not be executed in WebAssembly
}
