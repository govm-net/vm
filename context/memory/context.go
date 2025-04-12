package memory

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/govm-net/vm/context"
	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/types"
)

// defaultBlockchainContext implements the default blockchain context
type defaultBlockchainContext struct {
	// Block information
	blockHeight uint64
	blockTime   int64

	// Account balances
	balances map[types.Address]uint64

	// Virtual machine object storage
	objects        map[core.ObjectID]map[string][]byte
	objectOwner    map[core.ObjectID]core.Address
	objectContract map[core.ObjectID]core.Address

	// Current execution context
	contractAddr types.Address
	sender       types.Address
	txHash       core.Hash
	nonce        uint64
	gasLimit     int64
	mu           sync.Mutex
}

func init() {
	context.Register(context.MemoryContextType, NewBlockchainContext)
}

// NewDefaultBlockchainContext creates a new simple blockchain context
func NewBlockchainContext(params map[string]any) types.BlockchainContext {
	return &defaultBlockchainContext{
		blockHeight:    0,
		blockTime:      0,
		balances:       make(map[core.Address]uint64),
		objects:        make(map[core.ObjectID]map[string][]byte),
		objectOwner:    make(map[core.ObjectID]core.Address),
		objectContract: make(map[core.ObjectID]core.Address),
		gasLimit:       10000000,
	}
}

func (ctx *defaultBlockchainContext) SetBlockInfo(height uint64, time int64, hash core.Hash) error {
	ctx.blockHeight = height
	ctx.blockTime = time
	ctx.txHash = hash
	return nil
}

func (ctx *defaultBlockchainContext) SetTransactionInfo(hash core.Hash, from types.Address, to types.Address, value uint64) error {
	ctx.txHash = hash
	ctx.sender = from
	ctx.contractAddr = to
	// ctx.value = value
	return nil
}

func (ctx *defaultBlockchainContext) WithTransaction(txHash core.Hash) types.BlockchainContext {
	ctx.txHash = txHash
	return ctx
}

func (ctx *defaultBlockchainContext) WithBlock(height uint64, time int64) types.BlockchainContext {
	ctx.blockHeight = height
	ctx.blockTime = time
	return ctx
}

// BlockHeight gets the current block height
func (ctx *defaultBlockchainContext) BlockHeight() uint64 {
	return ctx.blockHeight
}

// BlockTime gets the current block timestamp
func (ctx *defaultBlockchainContext) BlockTime() int64 {
	return ctx.blockTime
}

// ContractAddress gets the current contract address
func (ctx *defaultBlockchainContext) ContractAddress() types.Address {
	return ctx.contractAddr
}

// TransactionHash gets the current transaction hash
func (ctx *defaultBlockchainContext) TransactionHash() core.Hash {
	return core.Hash{} // Simplified implementation
}

// Sender gets the transaction sender
func (ctx *defaultBlockchainContext) Sender() types.Address {
	return ctx.sender
}

// Balance gets the account balance
func (ctx *defaultBlockchainContext) Balance(addr types.Address) uint64 {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	return ctx.balances[addr]
}

func (ctx *defaultBlockchainContext) SetGasLimit(limit int64) {
	ctx.gasLimit = limit
}

func (ctx *defaultBlockchainContext) GetGas() int64 {
	return ctx.gasLimit
}

// Transfer transfers funds
func (ctx *defaultBlockchainContext) Transfer(contract types.Address, from, to types.Address, amount uint64) error {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	// Check balance
	fromBalance := ctx.balances[from]
	if fromBalance < amount {
		return errors.New("insufficient balance")
	}

	// Execute transfer
	ctx.balances[from] -= amount
	ctx.balances[to] += amount
	return nil
}

// CreateObject creates a new object
func (ctx *defaultBlockchainContext) CreateObject(contract types.Address) (types.VMObject, error) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	// Create object ID, simplified version using random number
	id := ctx.generateObjectID(contract, ctx.sender)

	// Create object storage
	ctx.objects[id] = make(map[string][]byte)
	ctx.objectOwner[id] = contract
	ctx.objectContract[id] = contract

	// Return object wrapper
	return &vmObject{
		ctx:         ctx,
		objOwner:    contract,
		objContract: contract,
		id:          id,
	}, nil
}

// CreateObject creates a new object
func (ctx *defaultBlockchainContext) CreateObjectWithID(contract types.Address, id types.ObjectID) (types.VMObject, error) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	// Create object storage
	ctx.objects[id] = make(map[string][]byte)
	ctx.objectOwner[id] = contract
	ctx.objectContract[id] = contract

	// Return object wrapper
	return &vmObject{
		ctx:         ctx,
		objOwner:    contract,
		objContract: contract,
		id:          id,
	}, nil
}

// generateObjectID generates a new object ID
func (ctx *defaultBlockchainContext) generateObjectID(contract types.Address, sender types.Address) core.ObjectID {
	ctx.nonce++
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%s-%d", contract, sender, ctx.txHash, ctx.nonce)))
	var id core.ObjectID
	copy(id[:], hash[:])
	return id
}

// GetObject gets a specified object
func (ctx *defaultBlockchainContext) GetObject(contract types.Address, id core.ObjectID) (types.VMObject, error) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	_, exists := ctx.objects[id]
	if !exists {
		return nil, errors.New("object does not exist")
	}

	return &vmObject{
		ctx:         ctx,
		objOwner:    ctx.objectOwner[id],
		objContract: ctx.objectContract[id],
		id:          id,
	}, nil
}

// GetObjectWithOwner gets objects by owner
func (ctx *defaultBlockchainContext) GetObjectWithOwner(contract, owner types.Address) (types.VMObject, error) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	for id, objOwner := range ctx.objectOwner {
		if objOwner == owner {
			return &vmObject{
				ctx:         ctx,
				objOwner:    objOwner,
				objContract: ctx.objectContract[id],
				id:          id,
			}, nil
		}
	}
	return nil, errors.New("object not found")
}

// DeleteObject deletes an object
func (ctx *defaultBlockchainContext) DeleteObject(contract types.Address, id core.ObjectID) error {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	delete(ctx.objects, id)
	delete(ctx.objectOwner, id)
	delete(ctx.objectContract, id)
	return nil
}

// Call cross-contract call
func (ctx *defaultBlockchainContext) Call(caller types.Address, contract types.Address, function string, args ...any) ([]byte, error) {
	return nil, errors.New("not implemented")
}

// Log records events
func (ctx *defaultBlockchainContext) Log(contract types.Address, eventName string, keyValues ...any) {
	params := []any{
		"contract", contract,
		"event", eventName,
	}
	params = append(params, keyValues...)
	slog.Info("Contract log", params...)
}

func (ctx *defaultBlockchainContext) setObjectField(id core.ObjectID, field string, value []byte) {
	obj, exists := ctx.objects[id]
	if !exists {
		obj = make(map[string][]byte)
	}
	obj[field] = value
	ctx.objects[id] = obj
}

func (ctx *defaultBlockchainContext) getObjectField(id core.ObjectID, field string) []byte {
	obj, exists := ctx.objects[id]
	if !exists {
		return nil
	}
	return obj[field]
}

// vmObject implements the object interface
type vmObject struct {
	ctx         *defaultBlockchainContext
	objOwner    types.Address
	objContract types.Address
	id          core.ObjectID
}

// ID gets the object ID
func (o *vmObject) ID() core.ObjectID {
	return o.id
}

// Owner gets the object owner
func (o *vmObject) Owner() types.Address {
	return o.objOwner
}

// Contract gets the object's contract
func (o *vmObject) Contract() types.Address {
	return o.objContract
}

// SetOwner sets the object owner
func (o *vmObject) SetOwner(contract, sender types.Address, addr types.Address) error {
	o.ctx.mu.Lock()
	defer o.ctx.mu.Unlock()
	if contract != o.objContract {
		return fmt.Errorf("invalid contract")
	}
	if sender != o.objOwner && contract != o.objOwner {
		return fmt.Errorf("not owner")
	}
	o.objOwner = addr
	o.ctx.objectOwner[o.id] = addr
	return nil
}

// Get gets the field value
func (o *vmObject) Get(contract types.Address, field string) ([]byte, error) {
	o.ctx.mu.Lock()
	defer o.ctx.mu.Unlock()
	if contract != o.objContract {
		return nil, fmt.Errorf("invalid contract")
	}
	fieldValue := o.ctx.getObjectField(o.id, field)
	if fieldValue == nil {
		return nil, errors.New("field does not exist")
	}

	return fieldValue, nil
}

// Set sets the field value
func (o *vmObject) Set(contract types.Address, sender types.Address, field string, value []byte) error {
	o.ctx.mu.Lock()
	defer o.ctx.mu.Unlock()
	if contract != o.objContract {
		return fmt.Errorf("invalid contract")
	}
	if sender != o.objOwner && contract != o.objOwner {
		return fmt.Errorf("not owner")
	}
	o.ctx.setObjectField(o.id, field, value)
	return nil
}
