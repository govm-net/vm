// Package vm provides the implementation of the virtual machine
// that executes blockchain smart contracts.
package vm

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"sync"

	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/vm/api"
	"github.com/govm-net/vm/vm/runtime"
)

// Engine implements the api.VM interface and provides
// the execution environment for smart contracts.
type Engine struct {
	maker         *runtime.Maker
	config        api.ContractConfig
	contracts     map[core.Address][]byte
	contractsLock sync.RWMutex
	objects       map[core.ObjectID]core.Object // Map of objectID string to object
	objectsLock   sync.RWMutex
	db            Database
	keyGenerator  KeyGenerator
}

// Database defines the interface for persistent storage backends.
type Database interface {
	// Get retrieves a value from the database.
	Get(key []byte) ([]byte, error)

	// Put stores a key-value pair in the database.
	Put(key []byte, value []byte) error

	// Delete removes a key-value pair from the database.
	Delete(key []byte) error

	// Has checks if a key exists in the database.
	Has(key []byte) (bool, error)

	// Iterator returns an iterator over a specific key range.
	Iterator(start, end []byte) (Iterator, error)

	// Close releases any resources used by the database.
	Close() error
}

// Iterator defines an iterator for navigating through database entries.
type Iterator interface {
	// Next moves the iterator to the next key-value pair.
	// Returns false if the iterator is exhausted.
	Next() bool

	// Error returns any accumulated error during iteration.
	Error() error

	// Key returns the key of the current key-value pair.
	Key() []byte

	// Value returns the value of the current key-value pair.
	Value() []byte

	// Close releases any resources used by the iterator.
	Close() error
}

// KeyGenerator creates storage keys based on contract address, object ID, owner, and field.
type KeyGenerator interface {
	// ObjectKey generates a key for storing an object's metadata.
	// Format: contract_address + object_id
	ObjectKey(contractAddr core.Address, objectID core.ObjectID) []byte

	// FieldKey generates a key for storing a field value.
	// Format: contract_address + object_id + field_name
	FieldKey(contractAddr core.Address, objectID core.ObjectID, field string) []byte

	// OwnerIndex generates a key for the owner index.
	// Format: contract_address + owner_address + object_id
	OwnerIndex(contractAddr core.Address, owner core.Address, objectID core.ObjectID) []byte

	// FindByOwner generates a prefix key for finding objects by owner.
	// Format: contract_address + owner_address
	FindByOwner(contractAddr core.Address, owner core.Address) []byte

	// ExtractObjectIDFromKey extracts the object ID from a given key.
	ExtractObjectIDFromKey(key []byte) core.ObjectID
}

// NewEngine creates a new VM engine with the given configuration.
func NewEngine(config api.ContractConfig) *Engine {
	return &Engine{
		maker:     runtime.NewMaker(config),
		config:    config,
		contracts: make(map[core.Address][]byte),
		objects:   make(map[core.ObjectID]core.Object),
	}
}

// Deploy deploys a new smart contract to the blockchain.
func (e *Engine) Deploy(code []byte, args ...[]byte) (core.Address, error) {
	// Validate and compile the contract
	compiledCode, err := e.maker.CompileContract(code)
	if err != nil {
		return core.ZeroAddress(), fmt.Errorf("compilation failed: %w", err)
	}

	// Generate a unique address for the contract
	address := generateContractAddress(code)

	// Store the contract code
	e.contractsLock.Lock()
	e.contracts[address] = compiledCode
	e.contractsLock.Unlock()

	// No initialization of state at deployment time
	// State objects will be created during execution as needed

	return address, nil
}

// Execute executes a function on a deployed contract.
func (e *Engine) Execute(contract core.Address, function string, args ...[]byte) ([]byte, error) {
	// Get the contract code
	e.contractsLock.RLock()
	compiledCode, exists := e.contracts[contract]
	e.contractsLock.RUnlock()

	if !exists {
		return nil, core.ErrContractNotFound
	}

	// Create a new instance of the contract
	contractInstance, err := e.maker.InstantiateContract(compiledCode)
	if err != nil {
		return nil, fmt.Errorf("instantiation failed: %w", err)
	}

	// Create a context for execution
	ctx := NewExecutionContext(contract, e)

	// Find the function by name using reflection
	method := reflect.ValueOf(contractInstance).MethodByName(function)
	if !method.IsValid() {
		return nil, core.ErrFunctionNotFound
	}

	// Call the function
	result, err := callFunction(method, ctx, args)
	if err != nil {
		return nil, err
	}

	// Handle the return value
	if result == nil {
		return nil, nil
	}

	// 使用EncodeParam进行编码
	return EncodeParam(result)
}

// callFunction uses reflection to call a function with the given context and arguments
func callFunction(method reflect.Value, ctx core.Context, args [][]byte) (interface{}, error) {
	// First, determine if the function takes a context as its first parameter
	methodType := method.Type()

	// Build the argument list
	var callArgs []reflect.Value

	// Check if the first argument should be the context
	argOffset := 0
	if methodType.NumIn() > 0 && methodType.In(0).Implements(reflect.TypeOf((*core.Context)(nil)).Elem()) {
		callArgs = append(callArgs, reflect.ValueOf(ctx))
		argOffset = 1
	}

	// Add the rest of the arguments
	for i := 0; i < methodType.NumIn()-argOffset && i < len(args); i++ {
		paramType := methodType.In(i + argOffset)

		// 使用DecodeParam解码参数
		param, err := DecodeParam(args[i], paramType)
		if err != nil {
			return nil, fmt.Errorf("failed to decode parameter %d: %w", i, err)
		}

		callArgs = append(callArgs, param)
	}

	// Call the method
	results := method.Call(callArgs)

	// Handle the return values
	if len(results) == 0 {
		return nil, nil
	}

	// Check for error as the last return value
	lastResult := results[len(results)-1]
	if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		if !lastResult.IsNil() {
			return nil, lastResult.Interface().(error)
		}

		// If there's only one other return value (besides the error), process it
		if len(results) > 1 {
			return results[0].Interface(), nil
		}
		return nil, nil
	}

	// If the function didn't return an error, return the first result
	return results[0].Interface(), nil
}

// ValidateContract checks if the contract code adheres to the restrictions.
func (e *Engine) ValidateContract(code []byte) error {
	return e.maker.ValidateContract(code)
}

// ExecutionContext is an implementation of the core.Context interface.
type ExecutionContext struct {
	contractAddress core.Address
	engine          *Engine
}

// NewExecutionContext creates a new execution context for a contract.
func NewExecutionContext(contractAddress core.Address, engine *Engine) *ExecutionContext {
	return &ExecutionContext{
		contractAddress: contractAddress,
		engine:          engine,
	}
}

// Sender returns the address of the account that called the contract.
func (ctx *ExecutionContext) Sender() core.Address {
	// In a real implementation, this would be the actual sender's address
	return core.ZeroAddress()
}

// BlockHeight returns the current block height.
func (ctx *ExecutionContext) BlockHeight() uint64 {
	// In a real implementation, this would be the actual block height
	return 0
}

// BlockTime returns the timestamp of the current block.
func (ctx *ExecutionContext) BlockTime() int64 {
	// In a real implementation, this would be the actual block timestamp
	return 0
}

// ContractAddress returns the address of the current contract.
func (ctx *ExecutionContext) ContractAddress() core.Address {
	return ctx.contractAddress
}

// Balance returns the balance of the given address.
func (ctx *ExecutionContext) Balance(addr core.Address) uint64 {
	// In a real implementation, this would query the actual balance
	return 0
}

// Transfer sends funds from the contract to the specified address.
func (ctx *ExecutionContext) Transfer(to core.Address, amount uint64) error {
	// In a real implementation, this would transfer actual funds
	return nil
}

// Call invokes a function on another contract.
func (ctx *ExecutionContext) Call(contract core.Address, function string, args ...interface{}) ([]byte, error) {
	// 使用EncodeArgs编码参数
	encodedArgs, err := EncodeArgs(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to encode arguments: %w", err)
	}

	// Execute the function on the other contract
	return ctx.engine.Execute(contract, function, encodedArgs...)
}

// Log emits an event to the blockchain.
func (ctx *ExecutionContext) Log(event string, data ...interface{}) {
	// In a real implementation, this would emit an actual blockchain event
}

// GetObject retrieves a state object by its ID.
func (ctx *ExecutionContext) GetObject(objectID core.ObjectID) (core.Object, error) {
	if objectID == (core.ObjectID{}) {
		copy(objectID[:], ctx.contractAddress[:])
	}

	ctx.engine.objectsLock.RLock()
	obj, exists := ctx.engine.objects[objectID]
	ctx.engine.objectsLock.RUnlock()

	if !exists {
		return nil, core.ErrObjectNotFound
	}

	return obj, nil
}

// CreateObject creates a new object based on context.
// If a database provider is set, it creates a DB-backed object;
// otherwise, it creates an in-memory object.
func (ctx *ExecutionContext) CreateObject() (core.Object, error) {
	// Check if DB provider is set
	if ctx.engine.db != nil && ctx.engine.keyGenerator != nil {
		// Create a DB-backed object
		return ctx.CreateDBObject()
	}

	// Create an in-memory state object
	object := NewStateObject(ctx.Sender(), ctx.ContractAddress())

	// Store the object
	ctx.engine.objectsLock.Lock()
	ctx.engine.objects[object.ID()] = object
	ctx.engine.objectsLock.Unlock()

	// Log object creation
	ctx.Log("ObjectCreated", "id", object.ID(), "owner", ctx.Sender().String(), "storage", "memory")

	return object, nil
}

// DeleteObject deletes an object from state.
func (ctx *ExecutionContext) DeleteObject(objectID core.ObjectID) error {
	ctx.engine.objectsLock.Lock()
	defer ctx.engine.objectsLock.Unlock()

	if _, exists := ctx.engine.objects[objectID]; !exists {
		return core.ErrObjectNotFound
	}

	delete(ctx.engine.objects, objectID)

	// Log object deletion
	ctx.Log("ObjectDeleted", "id", objectID)

	return nil
}

// StateObject is an implementation of the core.Object interface.
type StateObject struct {
	id      core.ObjectID
	objType string
	owner   core.Address
	fields  map[string]interface{}
	lock    sync.RWMutex
}

// NewStateObject creates a new state object.
func NewStateObject(owner core.Address, contractAddr core.Address) *StateObject {
	// Generate a unique ID for the object incorporating contract address
	id := generateObjectID(owner, contractAddr)

	return &StateObject{
		id:      id,
		objType: contractAddr.String(),
		owner:   owner,
		fields:  make(map[string]interface{}),
	}
}

// ID returns the unique identifier of the object.
func (o *StateObject) ID() core.ObjectID {
	return o.id
}

// Type returns the type of the object.
func (o *StateObject) Type() string {
	return o.objType
}

// Owner returns the owner address of the object.
func (o *StateObject) Owner() core.Address {
	return o.owner
}

// SetOwner changes the owner of the object.
func (o *StateObject) SetOwner(newOwner core.Address) error {
	o.owner = newOwner
	return nil
}

// Get retrieves a value from the object's fields.
func (o *StateObject) Get(field string) (interface{}, error) {
	o.lock.RLock()
	defer o.lock.RUnlock()

	value, exists := o.fields[field]
	if !exists {
		return nil, fmt.Errorf("field %s not found", field)
	}

	return value, nil
}

// Set stores a value in the object's fields.
func (o *StateObject) Set(field string, value interface{}) error {
	o.lock.Lock()
	defer o.lock.Unlock()

	o.fields[field] = value
	return nil
}

// Delete removes a field from the object.
func (o *StateObject) Delete(field string) error {
	o.lock.Lock()
	defer o.lock.Unlock()

	delete(o.fields, field)
	return nil
}

// Encode encodes the state object to bytes.
func (o *StateObject) Encode() ([]byte, error) {
	o.lock.RLock()
	defer o.lock.RUnlock()

	// Create a structure to represent the object metadata
	metadata := struct {
		ID     core.ObjectID          `json:"id"`
		Type   string                 `json:"type"`
		Owner  core.Address           `json:"owner"`
		Fields map[string]interface{} `json:"fields"`
	}{
		ID:     o.id,
		Type:   o.objType,
		Owner:  o.owner,
		Fields: o.fields,
	}

	return json.Marshal(metadata)
}

// Decode decodes the state object from bytes.
func (o *StateObject) Decode(data []byte) error {
	o.lock.Lock()
	defer o.lock.Unlock()

	// Parse the metadata
	metadata := struct {
		ID     core.ObjectID          `json:"id"`
		Type   string                 `json:"type"`
		Owner  core.Address           `json:"owner"`
		Fields map[string]interface{} `json:"fields"`
	}{}

	if err := json.Unmarshal(data, &metadata); err != nil {
		return err
	}

	// Update object fields
	o.id = metadata.ID
	o.objType = metadata.Type
	o.owner = metadata.Owner
	o.fields = metadata.Fields

	return nil
}

// generateContractAddress creates a unique address for a contract.
// In a real implementation, this would use cryptographic methods
// based on the deployer's address and nonce.
func generateContractAddress(code []byte) core.Address {
	hash := core.Hash(code)
	var addr core.Address
	copy(addr[:], hash[:20])
	return addr
}

// generateObjectID creates a unique ID for an object.
func generateObjectID(owner core.Address, contractAddr core.Address) core.ObjectID {
	// Generate a deterministic object ID that incorporates:
	// 1. The contract address (to ensure different contracts have different object IDs)
	// 2. The owner address

	// For other objects, use a more general approach
	data := append([]byte(contractAddr.String()), owner[:]...)
	hash := core.Hash(data)

	var id core.ObjectID
	copy(id[:], hash[:])
	return id
}

// GetObjectWithOwner retrieves a state object by its type and owner.
func (ctx *ExecutionContext) GetObjectWithOwner(owner core.Address) (core.Object, error) {
	// Lock to prevent concurrent modifications
	ctx.engine.objectsLock.RLock()
	defer ctx.engine.objectsLock.RUnlock()

	// Fallback: Search through all objects for matching type and owner
	for _, obj := range ctx.engine.objects {
		if obj.Owner().String() == owner.String() {
			return obj, nil
		}
	}

	return nil, core.ErrObjectNotFound
}

// SetDBProvider sets the database provider for the engine.
// This enables persistence of state objects.
func (e *Engine) SetDBProvider(db Database, keyGen KeyGenerator) {
	e.db = db
	e.keyGenerator = keyGen
}

// CreateDBObject creates a new DB-backed state object.
func (ctx *ExecutionContext) CreateDBObject() (core.Object, error) {
	// Check if DB provider is set
	if ctx.engine.db == nil || ctx.engine.keyGenerator == nil {
		return nil, fmt.Errorf("DB provider not set, cannot create DB object")
	}

	// Generate the object ID
	id := generateObjectID(ctx.Sender(), ctx.ContractAddress())

	// Create the DB state object
	object := NewDBStateObject(
		id,
		ctx.ContractAddress(),
		ctx.Sender(),
		ctx.ContractAddress().String(), // Use contract address as type
		ctx.engine.db,
		ctx.engine.keyGenerator,
		true, // Enable caching by default
	)

	// Store the object
	ctx.engine.objectsLock.Lock()
	ctx.engine.objects[object.ID()] = object
	ctx.engine.objectsLock.Unlock()

	// Log object creation
	ctx.Log("ObjectCreated", "id", object.ID(), "owner", ctx.Sender().String(), "storage", "db")

	return object, nil
}

// LoadDBObject loads a DB-backed state object by its ID.
func (ctx *ExecutionContext) LoadDBObject(objectID core.ObjectID) (core.Object, error) {
	// Check if DB provider is set
	if ctx.engine.db == nil || ctx.engine.keyGenerator == nil {
		return nil, fmt.Errorf("DB provider not set, cannot load DB object")
	}

	// Check if the object is already loaded in memory
	ctx.engine.objectsLock.RLock()
	if obj, exists := ctx.engine.objects[objectID]; exists {
		ctx.engine.objectsLock.RUnlock()
		return obj, nil
	}
	ctx.engine.objectsLock.RUnlock()

	// Generate the object key
	objectKey := ctx.engine.keyGenerator.ObjectKey(ctx.ContractAddress(), objectID)

	// Retrieve the object metadata from DB
	data, err := ctx.engine.db.Get(objectKey)
	if err != nil {
		return nil, err
	}

	if data == nil {
		return nil, core.ErrObjectNotFound
	}

	// Parse the metadata
	metadata := struct {
		ID    core.ObjectID `json:"id"`
		Type  string        `json:"type"`
		Owner core.Address  `json:"owner"`
	}{}

	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	// Create the DB state object
	object := NewDBStateObject(
		metadata.ID,
		ctx.ContractAddress(),
		metadata.Owner,
		metadata.Type,
		ctx.engine.db,
		ctx.engine.keyGenerator,
		true, // Enable caching by default
	)

	// Store the object in memory cache
	ctx.engine.objectsLock.Lock()
	ctx.engine.objects[object.ID()] = object
	ctx.engine.objectsLock.Unlock()

	return object, nil
}

// FindDBObjectsByOwner finds all DB-backed objects owned by the given address.
func (ctx *ExecutionContext) FindDBObjectsByOwner(owner core.Address) ([]core.Object, error) {
	// Check if DB provider is set
	if ctx.engine.db == nil || ctx.engine.keyGenerator == nil {
		return nil, fmt.Errorf("DB provider not set, cannot find DB objects")
	}

	// Generate the owner lookup prefix
	prefix := ctx.engine.keyGenerator.FindByOwner(ctx.ContractAddress(), owner)
	start, end := PrefixRange(prefix)

	// Get an iterator over the owner index
	iter, err := ctx.engine.db.Iterator(start, end)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var objects []core.Object

	// Iterate through all owner index entries
	for iter.Next() {
		key := iter.Key()

		// Extract the object ID from the owner index key
		// Owner index format: 'i' + contract_address + owner_address + object_id
		objectID := ctx.engine.keyGenerator.ExtractObjectIDFromKey(key)

		// Load the object
		obj, err := ctx.LoadDBObject(objectID)
		if err != nil {
			continue // Skip if the object can't be loaded
		}

		objects = append(objects, obj)
	}

	if err := iter.Error(); err != nil {
		return nil, err
	}

	return objects, nil
}

// EncodeParam 将任意类型的参数编码为字节数组
func EncodeParam(param interface{}) ([]byte, error) {
	if param == nil {
		return nil, nil
	}

	// 检查参数类型
	switch v := param.(type) {
	case string:
		// 字符串直接返回 UTF-8 编码
		return []byte(v), nil
	case []byte:
		// 字节数组直接返回拷贝
		result := make([]byte, len(v))
		copy(result, v)
		return result, nil
	case core.ObjectID:
		// ObjectID 类型提取底层字节
		result := make([]byte, 32)
		copy(result, v[:])
		return result, nil
	case core.Address:
		// Address 类型提取底层字节
		result := make([]byte, 20)
		copy(result, v[:])
		return result, nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		// 基本数值类型转为字符串
		return []byte(fmt.Sprintf("%v", v)), nil
	default:
		// 如果参数实现了 Encoder 接口，调用其 Encode 方法
		if encoder, ok := param.(core.Encoder); ok {
			return encoder.Encode()
		}

		// 其他类型使用 JSON 序列化
		return json.Marshal(param)
	}
}

// DecodeParam 将字节数组解码为指定类型的参数值
func DecodeParam(data []byte, targetType reflect.Type) (reflect.Value, error) {
	// 检查目标类型
	switch targetType.Kind() {
	case reflect.String:
		// 将字节数组解码为字符串
		return reflect.ValueOf(string(data)), nil
	case reflect.Slice:
		// 目标是字节切片类型
		if targetType.Elem().Kind() == reflect.Uint8 {
			// 复制一份数据
			result := make([]byte, len(data))
			copy(result, data)
			return reflect.ValueOf(result), nil
		}
		// 其他切片类型，尝试使用 JSON 解码
		slicePtr := reflect.New(targetType)
		if err := json.Unmarshal(data, slicePtr.Interface()); err != nil {
			return reflect.Value{}, fmt.Errorf("failed to unmarshal slice: %w", err)
		}
		return slicePtr.Elem(), nil
	case reflect.Array:
		// 处理固定长度数组，如[32]byte类型
		arrayLen := targetType.Len()
		arrayValue := reflect.New(targetType).Elem()

		// 将数据复制到数组中
		copyLen := len(data)
		if copyLen > arrayLen {
			copyLen = arrayLen
		}

		// 逐个设置数组元素
		for i := 0; i < copyLen; i++ {
			arrayValue.Index(i).Set(reflect.ValueOf(data[i]))
		}

		return arrayValue, nil
	case reflect.Struct:
		// 处理结构体类型
		// 检查是否为 ObjectID 类型
		if targetType == reflect.TypeOf(core.ObjectID{}) {
			var id core.ObjectID
			// 尝试直接解码
			if len(data) == 32 {
				copy(id[:], data)
				return reflect.ValueOf(id), nil
			} else if len(data) > 0 {
				// 尝试从字符串转换
				id = core.ObjectIDFromString(string(data))
				return reflect.ValueOf(id), nil
			}
			return reflect.Value{}, fmt.Errorf("invalid data for ObjectID")
		}

		// 检查是否为 Address 类型
		if targetType == reflect.TypeOf(core.Address{}) {
			var addr core.Address
			// 尝试直接解码
			if len(data) == 20 {
				copy(addr[:], data)
				return reflect.ValueOf(addr), nil
			}
			return reflect.Value{}, fmt.Errorf("invalid data for Address")
		}

		// 检查是否实现了 Decoder 接口
		if targetType.Implements(reflect.TypeOf((*core.Decoder)(nil)).Elem()) {
			// 创建新实例
			ptrValue := reflect.New(targetType)

			// 调用 Decode 方法
			method := ptrValue.MethodByName("Decode")
			results := method.Call([]reflect.Value{reflect.ValueOf(data)})

			// 检查错误
			if !results[0].IsNil() {
				return reflect.Value{}, results[0].Interface().(error)
			}

			return ptrValue.Elem(), nil
		}

		// 其他结构体类型使用 JSON 解码
		structPtr := reflect.New(targetType)
		if err := json.Unmarshal(data, structPtr.Interface()); err != nil {
			return reflect.Value{}, fmt.Errorf("failed to unmarshal struct: %w", err)
		}
		return structPtr.Elem(), nil
	case reflect.Ptr:
		// 处理指针类型
		// 创建目标类型的元素类型的值
		elemType := targetType.Elem()
		elemVal, err := DecodeParam(data, elemType)
		if err != nil {
			return reflect.Value{}, err
		}

		// 创建指向该值的指针
		ptrVal := reflect.New(elemType)
		ptrVal.Elem().Set(elemVal)
		return ptrVal, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// 数值类型从字符串解析
		val, err := strconv.ParseInt(string(data), 10, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("failed to parse int: %w", err)
		}
		return reflect.ValueOf(val).Convert(targetType), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// 数值类型从字符串解析
		val, err := strconv.ParseUint(string(data), 10, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("failed to parse uint: %w", err)
		}
		return reflect.ValueOf(val).Convert(targetType), nil
	case reflect.Float32, reflect.Float64:
		// 浮点类型从字符串解析
		val, err := strconv.ParseFloat(string(data), 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("failed to parse float: %w", err)
		}
		return reflect.ValueOf(val).Convert(targetType), nil
	case reflect.Bool:
		// 布尔类型从字符串解析
		val, err := strconv.ParseBool(string(data))
		if err != nil {
			return reflect.Value{}, fmt.Errorf("failed to parse bool: %w", err)
		}
		return reflect.ValueOf(val), nil
	default:
		return reflect.Value{}, fmt.Errorf("unsupported parameter type: %v", targetType)
	}
}

// EncodeArgs 将任意类型的参数列表编码为字节数组切片
func EncodeArgs(args ...interface{}) ([][]byte, error) {
	encodedArgs := make([][]byte, len(args))

	for i, arg := range args {
		encodedArg, err := EncodeParam(arg)
		if err != nil {
			return nil, fmt.Errorf("failed to encode argument %d: %w", i, err)
		}
		encodedArgs[i] = encodedArg
	}

	return encodedArgs, nil
}

// ExecuteWithArgs 执行合约函数，并自动编码任意类型的参数
func (e *Engine) ExecuteWithArgs(contract core.Address, function string, args ...interface{}) ([]byte, error) {
	// 编码参数
	encodedArgs, err := EncodeArgs(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to encode arguments: %w", err)
	}

	// 执行合约函数
	return e.Execute(contract, function, encodedArgs...)
}
