package context

import (
	"encoding/json"

	"github.com/govm-net/vm/types"
)

// BlockchainContextAdapter adapts a BlockchainContext to the Context interface
type BlockchainContextAdapter struct {
	ctx      types.BlockchainContext
	contract types.Address
}

var _ types.Context = (*BlockchainContextAdapter)(nil)

// NewBlockchainContextAdapter creates a new adapter that implements the Context interface
// using a BlockchainContext as the underlying implementation
func NewBlockchainContextAdapter(ctx types.BlockchainContext, contract types.Address) *BlockchainContextAdapter {
	return &BlockchainContextAdapter{
		ctx:      ctx,
		contract: contract,
	}
}

// BlockHeight gets the current block height
func (a *BlockchainContextAdapter) BlockHeight() uint64 {
	return a.ctx.BlockHeight()
}

// BlockTime gets the current block timestamp
func (a *BlockchainContextAdapter) BlockTime() int64 {
	return a.ctx.BlockTime()
}

// ContractAddress gets the current contract address
func (a *BlockchainContextAdapter) ContractAddress() types.Address {
	return a.ctx.ContractAddress()
}

// Sender gets the transaction sender or contract caller
func (a *BlockchainContextAdapter) Sender() types.Address {
	return a.ctx.Sender()
}

// Balance gets the account balance
func (a *BlockchainContextAdapter) Balance(addr types.Address) uint64 {
	return a.ctx.Balance(addr)
}

// Transfer transfers funds between accounts
func (a *BlockchainContextAdapter) Transfer(from, to types.Address, amount uint64) error {
	return a.ctx.Transfer(a.contract, from, to, amount)
}

// CreateObject creates a new object
func (a *BlockchainContextAdapter) CreateObject() types.Object {
	obj, err := a.ctx.CreateObject(a.contract)
	if err != nil {
		panic(err)
	}
	return &ObjectAdapter{obj: obj, contract: a.contract}
}

// GetObject gets a specified object
func (a *BlockchainContextAdapter) GetObject(id types.ObjectID) (types.Object, error) {
	obj, err := a.ctx.GetObject(a.contract, id)
	if err != nil {
		return nil, err
	}
	return &ObjectAdapter{obj: obj, contract: a.contract}, nil
}

// GetObjectWithOwner gets an object by owner
func (a *BlockchainContextAdapter) GetObjectWithOwner(owner types.Address) (types.Object, error) {
	obj, err := a.ctx.GetObjectWithOwner(a.contract, owner)
	if err != nil {
		return nil, err
	}
	return &ObjectAdapter{obj: obj, contract: a.contract}, nil
}

// DeleteObject deletes an object
func (a *BlockchainContextAdapter) DeleteObject(id types.ObjectID) {
	err := a.ctx.DeleteObject(a.contract, id)
	if err != nil {
		panic(err)
	}
}

// Call calls a function on another contract
func (a *BlockchainContextAdapter) Call(contract types.Address, function string, args ...any) ([]byte, error) {
	return a.ctx.Call(a.contract, contract, function, args...)
}

// Log logs an event
func (a *BlockchainContextAdapter) Log(eventName string, keyValues ...any) {
	a.ctx.Log(a.contract, eventName, keyValues...)
}

// ObjectAdapter adapts a VMObject to the Object interface
type ObjectAdapter struct {
	obj      types.VMObject
	contract types.Address
}

// ID gets the object ID
func (o *ObjectAdapter) ID() types.ObjectID {
	return o.obj.ID()
}

// Owner gets the object owner
func (o *ObjectAdapter) Owner() types.Address {
	return o.obj.Owner()
}

// Contract gets the object's contract
func (o *ObjectAdapter) Contract() types.Address {
	return o.obj.Contract()
}

// SetOwner sets the object owner
func (o *ObjectAdapter) SetOwner(addr types.Address) {
	err := o.obj.SetOwner(o.contract, o.contract, addr)
	if err != nil {
		panic(err)
	}
}

// Get gets a field value
func (o *ObjectAdapter) Get(field string, value any) error {
	data, err := o.obj.Get(o.contract, field)
	if err != nil {
		return err
	}

	// Unmarshal the data into the provided value
	return json.Unmarshal(data, value)
}

// Set sets a field value
func (o *ObjectAdapter) Set(field string, value any) error {
	// Marshal the value to bytes
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return o.obj.Set(o.contract, o.contract, field, data)
}
