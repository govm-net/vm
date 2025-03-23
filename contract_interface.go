// Package vm provides the fundamental interfaces and types for smart contracts
// that run on the blockchain virtual machine.
package vm

// Address represents a blockchain address
type Address [20]byte

// ObjectID is a unique identifier for a state object
type ObjectID [32]byte

// Context represents the execution context of a smart contract.
// It provides access to blockchain data and utility functions.
type Context interface {
	// Sender returns the address of the account that called the contract
	Sender() Address

	// BlockHeight returns the current block height
	BlockHeight() uint64

	// BlockTime returns the timestamp of the current block
	BlockTime() int64

	// ContractAddress returns the address of the current contract
	ContractAddress() Address

	// Balance returns the balance of the given address
	Balance(addr Address) float64

	// Transfer sends funds from the contract to the specified address
	Transfer(to Address, amount uint64) error

	// Call invokes a function on another contract
	Call(contract Address, function string, args ...any) ([]byte, error)

	// CreateObject creates a new object
	CreateObject() Object

	// GetObject retrieves a state object by its ID, if id is empty, it will be the default object
	GetObject(objectID ObjectID) (Object, error)

	// GetObjectWithOwner retrieves a state object by its type and owner
	GetObjectWithOwner(owner Address) (Object, error)

	// DeleteObject deletes an object from state
	DeleteObject(objectID ObjectID)

	// Log emits an event to the blockchain
	Log(event string, data ...any)
}

// Object represents a state object that can be manipulated by contracts
type Object interface {
	// ID returns the unique identifier of the object
	ID() ObjectID

	// Owner returns the owner address of the object
	Owner() Address

	// SetOwner changes the owner of the object
	SetOwner(owner Address)

	// Get retrieves a value from the object's fields
	Get(field string, value any) error

	// Set stores a value in the object's fields
	Set(field string, value any) error
}
