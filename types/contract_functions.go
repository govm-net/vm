// Package types contains shared type definitions and constants
// used by both the host environment and WebAssembly contracts
package types

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
	// FuncGetBlockHeight returns the current block height
	FuncGetBlockHeight // 2
	// FuncGetBlockTime returns the current block timestamp
	FuncGetBlockTime // 3
	// FuncGetContractAddress returns the address of the current contract
	FuncGetContractAddress // 4
	// FuncGetBalance returns the balance of a given address
	FuncGetBalance // 5
	// FuncTransfer transfers tokens from the contract to a recipient
	FuncTransfer // 6
	// FuncCreateObject creates a new state object
	FuncCreateObject // 7
	// FuncCall calls a function on another contract
	FuncCall // 8
	// FuncGetObject retrieves a state object by ID
	FuncGetObject // 9
	// FuncGetObjectWithOwner retrieves objects owned by a specific address
	FuncGetObjectWithOwner // 10
	// FuncDeleteObject removes a state object
	FuncDeleteObject // 11
	// FuncLog logs a message to the blockchain's event system
	FuncLog // 12
	// FuncGetObjectOwner gets the owner of a state object
	FuncGetObjectOwner // 13
	// FuncSetObjectOwner changes the owner of a state object
	FuncSetObjectOwner // 14
	// FuncGetObjectField retrieves a specific field from a state object
	FuncGetObjectField // 15
	// FuncSetObjectField updates a specific field in a state object
	FuncSetObjectField // 16
	// FuncDbRead reads data from the contract's storage
	FuncDbRead // 17
	// FuncDbWrite writes data to the contract's storage
	FuncDbWrite // 18
	// FuncDbDelete deletes data from the contract's storage
	FuncDbDelete // 19
	// FuncSetHostBuffer sets the host buffer for data exchange
	FuncSetHostBuffer // 20
)

// HostBufferSize defines the size of the buffer used for data exchange between host and contract
const HostBufferSize int32 = 2048
