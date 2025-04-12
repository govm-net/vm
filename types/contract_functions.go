// Package types contains shared type definitions and constants
// used by both the host environment and WebAssembly contracts
package types

import (
	"encoding/hex"
	"strings"
)

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

// Address 表示区块链上的地址
type Address [20]byte

// ObjectID 表示状态对象的唯一标识符
type ObjectID [32]byte

type Hash [32]byte

func (id ObjectID) String() string {
	return hex.EncodeToString(id[:])
}

func (addr Address) String() string {
	return hex.EncodeToString(addr[:])
}

func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

func HashFromString(str string) Hash {
	str = strings.TrimPrefix(str, "0x")
	h, err := hex.DecodeString(str)
	if err != nil {
		return Hash{}
	}
	var out Hash
	copy(out[:], h)
	return out
}

// Context 是合约与区块链环境交互的主要接口
type Context interface {
	// Blockchain information related
	BlockHeight() uint64      // Get current block height
	BlockTime() int64         // Get current block timestamp
	ContractAddress() Address // Get current contract address

	// Account operations related
	Sender() Address                                // Get transaction sender or contract caller
	Balance(addr Address) uint64                    // Get account balance
	Transfer(from, to Address, amount uint64) error // Transfer operation

	// Object storage related - Basic state operations use panic instead of returning error
	CreateObject() Object                             // Create new object, panic on failure
	GetObject(id ObjectID) (Object, error)            // Get specified object, may return error
	GetObjectWithOwner(owner Address) (Object, error) // Get object by owner, may return error
	DeleteObject(id ObjectID)                         // Delete object, panic on failure

	// Cross-contract calls
	Call(contract Address, function string, args ...any) ([]byte, error)

	// Logs and events
	Log(eventName string, keyValues ...any) // Log event
}

// Object 接口用于管理区块链状态对象
type Object interface {
	ID() ObjectID          // Get object ID
	Owner() Address        // Get object owner
	Contract() Address     // Get object's contract
	SetOwner(addr Address) // Set object owner, panic on failure

	// Field operations
	Get(field string, value any) error // Get field value
	Set(field string, value any) error // Set field value
}

type TransferParams struct {
	Contract Address `json:"contract,omitempty"`
	From     Address `json:"from,omitempty"`
	To       Address `json:"to,omitempty"`
	Amount   uint64  `json:"amount,omitempty"`
}

type CallParams struct {
	Caller   Address `json:"caller,omitempty"`
	Contract Address `json:"contract,omitempty"`
	Function string  `json:"function,omitempty"`
	Args     []any   `json:"args,omitempty"`
	GasLimit int64   `json:"gas_limit,omitempty"`
}

type CallResult struct {
	Data    []byte `json:"data,omitempty"`
	GasUsed int64  `json:"gas_used,omitempty"`
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
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

type LogParams struct {
	Contract  Address `json:"contract,omitempty"`
	Event     string  `json:"event,omitempty"`
	KeyValues []any   `json:"key_values,omitempty"`
}

type HandleContractCallParams struct {
	Contract Address `json:"contract,omitempty"`
	Sender   Address `json:"sender,omitempty"`
	Function string  `json:"function,omitempty"`
	Args     []byte  `json:"args,omitempty"`
	GasLimit int64   `json:"gas_limit,omitempty"`
}

// Context 是合约与区块链环境交互的主要接口
type BlockchainContext interface {
	// set block info and transaction info
	SetBlockInfo(height uint64, time int64, hash Hash) error
	SetTransactionInfo(hash Hash, from Address, to Address, value uint64) error
	// Blockchain information related
	BlockHeight() uint64      // Get current block height
	BlockTime() int64         // Get current block timestamp
	ContractAddress() Address // Get current contract address
	TransactionHash() Hash    // Get current transaction hash
	SetGasLimit(limit int64)  // Set gas limit
	GetGas() int64            // Get used gas
	// Account operations related
	Sender() Address                                          // Get transaction sender or contract caller
	Balance(addr Address) uint64                              // Get account balance
	Transfer(contract, from, to Address, amount uint64) error // Transfer operation

	// Object storage related - Basic state operations use panic instead of returning error
	CreateObject(contract Address) (VMObject, error)                      // Create new object
	CreateObjectWithID(contract Address, id ObjectID) (VMObject, error)   // Create new object
	GetObject(contract Address, id ObjectID) (VMObject, error)            // Get specified object
	GetObjectWithOwner(contract Address, owner Address) (VMObject, error) // Get object by owner
	DeleteObject(contract Address, id ObjectID) error                     // Delete object

	// Cross-contract calls
	Call(caller Address, contract Address, function string, args ...any) ([]byte, error)

	// Logs and events
	Log(contract Address, eventName string, keyValues ...any) // Log event
}

// Object 接口用于管理区块链状态对象
type VMObject interface {
	ID() ObjectID                                  // Get object ID
	Owner() Address                                // Get object owner
	Contract() Address                             // Get object's contract
	SetOwner(contract, sender, addr Address) error // Set object owner

	// Field operations
	Get(contract Address, field string) ([]byte, error)             // Get field value
	Set(contract, sender Address, field string, value []byte) error // Set field value
}
