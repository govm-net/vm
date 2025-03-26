// Package types contains shared type definitions and constants
// used by both the host environment and WebAssembly contracts
package types

import "github.com/govm-net/vm/core"

// WasmFunctionID defines constants for function IDs used in host-contract communication
// These constants must be used on both sides (host and contract) to ensure compatibility
//
// IMPORTANT: These function IDs are critical for the communication between the host environment
// and WebAssembly contracts. Any mismatch in the function ID values between the host and contract
// will result in undefined behavior or system failures.
//
// Always import and use these constants in both host and contract code, rather than defining
// separate constants in different parts of the codebase. This ensures consistent communication
// and prevents hard-to-debug errors due to mismatched function IDs.
//
// Example usage in host code:
//
//	const FuncGetSender = int32(types.FuncGetSender)
//
// Example usage in contract code:
//
//	const FuncGetSender = int32(types.FuncGetSender)
type WasmFunctionID int32

const (
	// FuncGetSender returns the address of the sender (caller) of the current transaction
	FuncGetSender WasmFunctionID = iota + 1 // 1
	// FuncGetContractAddress returns the address of the current contract
	FuncGetContractAddress // 2
	// FuncTransfer transfers tokens from the contract to a recipient
	FuncTransfer // 3
	// FuncCreateObject creates a new state object
	FuncCreateObject // 4
	// FuncCall calls a function on another contract
	FuncCall // 5
	// FuncGetObject retrieves a state object by ID
	FuncGetObject // 6
	// FuncGetObjectWithOwner retrieves objects owned by a specific address
	FuncGetObjectWithOwner // 7
	// FuncDeleteObject removes a state object
	FuncDeleteObject // 8
	// FuncLog logs a message to the blockchain's event system
	FuncLog // 9
	// FuncGetObjectOwner gets the owner of a state object
	FuncGetObjectOwner // 10
	// FuncSetObjectOwner changes the owner of a state object
	FuncSetObjectOwner // 11
	// FuncGetObjectField retrieves a specific field from a state object
	FuncGetObjectField // 12
	// FuncSetObjectField updates a specific field in a state object
	FuncSetObjectField // 13
	// FuncGetObjectContract gets the contract of a state object
	FuncGetObjectContract // 14
)

// HostBufferSize defines the size of the buffer used for data exchange between host and contract
const HostBufferSize int32 = 2048

type Address = core.Address
type ObjectID = core.ObjectID

type TransferParams struct {
	From   Address `json:"from,omitempty"`
	To     Address `json:"to,omitempty"`
	Amount uint64  `json:"amount,omitempty"`
}

type CallParams struct {
	Caller   Address `json:"caller,omitempty"`
	Contract Address `json:"contract,omitempty"`
	Function string  `json:"function,omitempty"`
	Args     []any   `json:"args,omitempty"`
}

type GetObjectParams struct {
	Contract Address  `json:"contract,omitempty"`
	ID       ObjectID `json:"id,omitempty"`
}
type GetObjectWithOwnerParams struct {
	Contract Address `json:"contract,omitempty"`
	Owner    Address `json:"owner,omitempty"`
}

type GetObjectFieldParams struct {
	Contract Address  `json:"contract,omitempty"`
	ID       ObjectID `json:"id,omitempty"`
	Field    string   `json:"field,omitempty"`
}

type DeleteObjectParams struct {
	Contract Address  `json:"contract,omitempty"`
	ID       ObjectID `json:"id,omitempty"`
}

type SetOwnerParams struct {
	Contract Address  `json:"contract,omitempty"`
	Sender   Address  `json:"sender,omitempty"`
	ID       ObjectID `json:"id,omitempty"`
	Owner    Address  `json:"owner,omitempty"`
}

type SetObjectFieldParams struct {
	Contract Address  `json:"contract,omitempty"`
	Sender   Address  `json:"sender,omitempty"`
	ID       ObjectID `json:"id,omitempty"`
	Field    string   `json:"field,omitempty"`
	Value    any      `json:"value,omitempty"`
}

type ExecutionResult struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type LogParams struct {
	Contract  Address `json:"contract,omitempty"`
	Event     string  `json:"event,omitempty"`
	KeyValues []any   `json:"key_values,omitempty"`
}
