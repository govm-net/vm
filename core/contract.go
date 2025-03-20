// Package core provides the fundamental interfaces and types for smart contracts
// that run on the blockchain virtual machine.
package core

import (
	"errors"
)

// Common errors that can be returned by smart contracts
var (
	ErrInvalidArgument   = errors.New("invalid argument")
	ErrUnauthorized      = errors.New("unauthorized operation")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrContractNotFound  = errors.New("contract not found")
	ErrFunctionNotFound  = errors.New("function not found")
	ErrExecutionReverted = errors.New("execution reverted")
	ErrObjectNotFound    = errors.New("object not found")
	ErrInvalidObjectType = errors.New("invalid object type")
)

// Encoder represents an object that can encode itself to bytes.
type Encoder interface {
	// Encode encodes the object to bytes
	Encode() ([]byte, error)
}

// Decoder represents an object that can decode itself from bytes.
type Decoder interface {
	// Decode decodes the object from bytes
	Decode(data []byte) error
}

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
	Balance(addr Address) uint64

	// Transfer sends funds from the contract to the specified address
	Transfer(to Address, amount uint64) error

	// Call invokes a function on another contract
	Call(contract Address, function string, args ...any) ([]byte, error)

	// Log emits an event to the blockchain
	Log(event string, data ...any)

	// GetObject retrieves a state object by its ID, if id is empty, it will be the default object
	GetObject(objectID ObjectID) (Object, error)

	// GetObjectWithOwner retrieves a state object by its type and owner
	GetObjectWithOwner(owner Address) (Object, error)

	// CreateObject creates a new object, using database backing if available
	// or in-memory storage if no database is configured
	CreateObject() (Object, error)

	// DeleteObject deletes an object from state
	DeleteObject(objectID ObjectID) error
}

// Address represents a blockchain address
type Address [20]byte

// String returns the hex string representation of the address
func (a Address) String() string {
	// Implementation will be provided by the VM
	return ""
}

// Encode encodes the address to bytes.
func (a Address) Encode() ([]byte, error) {
	return a[:], nil
}

// Decode decodes an address from bytes.
func (a *Address) Decode(data []byte) error {
	if len(data) != 20 {
		return ErrInvalidArgument
	}
	copy(a[:], data)
	return nil
}

// ObjectID is a unique identifier for a state object
type ObjectID [32]byte

// String returns the hex string representation of the object ID
func (id ObjectID) String() string {
	// Implementation will be provided by the VM
	return ""
}

// ObjectIDFromString converts a string to an ObjectID
func ObjectIDFromString(s string) ObjectID {
	var id ObjectID
	// 简单实现：复制字符串的字节到ID中
	// 在实际应用中，应该使用更复杂的哈希算法
	copy(id[:], []byte(s))
	return id
}

// Encode encodes the object ID to bytes.
func (id ObjectID) Encode() ([]byte, error) {
	return id[:], nil
}

// Decode decodes an object ID from bytes.
func (id *ObjectID) Decode(data []byte) error {
	if len(data) != 32 {
		return ErrInvalidArgument
	}
	copy(id[:], data)
	return nil
}

// Object represents a state object that can be manipulated by contracts
type Object interface {
	// ID returns the unique identifier of the object
	ID() ObjectID

	// Type returns the type of the object
	Type() string

	// Owner returns the owner address of the object
	Owner() Address

	// SetOwner changes the owner of the object
	SetOwner(owner Address) error

	// Get retrieves a value from the object's fields
	Get(field string) (any, error)

	// Set stores a value in the object's fields
	Set(field string, value any) error

	// Delete removes a field from the object
	Delete(field string) error

	// Encode encodes the object to bytes
	Encode() ([]byte, error)

	// Decode decodes the object from bytes
	Decode(data []byte) error
}
