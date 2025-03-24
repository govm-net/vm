// Package mock provides lightweight contract call tracing and logging
package mock

import (
	"bytes"
	"fmt"
	"sync"
	"time"
)

// ContractCallInfo stores information about a contract function call
type ContractCallInfo struct {
	Contract    string    // Contract address
	Function    string    // Function name
	EntryTime   time.Time // Time when function was entered
	ExitTime    time.Time // Time when function exited
	Error       string    // Error message if any
	Caller      string    // Caller contract address (for cross-contract calls)
	CallerFunc  string    // Caller function name
	IsCompleted bool      // Whether the call is completed
}

// CallStackEntry stores one entry in the call stack
type CallStackEntry struct {
	Contract  []byte // Contract address raw bytes
	Function  string // Function name
	Timestamp time.Time
}

var (
	// Mutex to protect shared data
	mu sync.Mutex

	// Call tree with parent-child relationships
	callTree = make(map[string][]string)

	// All call information by unique ID (contract:function:timestamp)
	callInfo = make(map[string]*ContractCallInfo)

	// Current call stack for each goroutine
	callStack = make(map[uint64][]CallStackEntry)
)

// getGoroutineID returns a unique ID for the current goroutine
// This is a simplified version - in a real implementation you might
// want to use runtime.Stack to get the real goroutine ID
func getGoroutineID() uint64 {
	return uint64(time.Now().UnixNano())
}

// genCallID generates a unique call ID
func genCallID(contract, function string, t time.Time) string {
	return fmt.Sprintf("%s:%s:%d", contract, function, t.UnixNano())
}

// Enter records function entry
func Enter(contract []byte, function string) string {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	callID := genCallID(string(contract), function, now)

	// Get current goroutine ID
	gid := getGoroutineID()

	// Get current call stack for this goroutine
	stack, exists := callStack[gid]
	if !exists {
		stack = []CallStackEntry{}
	}

	// Create new call info
	info := &ContractCallInfo{
		Contract:  fmt.Sprintf("%x", contract),
		Function:  function,
		EntryTime: now,
	}

	// If we have a call stack, set caller info
	if len(stack) > 0 {
		parent := stack[len(stack)-1]
		info.Caller = fmt.Sprintf("%x", parent.Contract)
		info.CallerFunc = parent.Function

		// Update call tree
		parentID := genCallID(fmt.Sprintf("%x", parent.Contract), parent.Function, parent.Timestamp)
		callTree[parentID] = append(callTree[parentID], callID)
	}

	// Store call info
	callInfo[callID] = info

	// Update call stack
	// Clone the contract address to avoid potential memory issues
	contractCopy := make([]byte, len(contract))
	copy(contractCopy, contract)

	callStack[gid] = append(stack, CallStackEntry{
		Contract:  contractCopy,
		Function:  function,
		Timestamp: now,
	})

	return callID
}

// Exit records function exit
func Exit(contract []byte, function string) {
	mu.Lock()
	defer mu.Unlock()

	// Get current goroutine ID
	gid := getGoroutineID()

	// Get current call stack
	stack, exists := callStack[gid]
	if !exists || len(stack) == 0 {
		return
	}

	// Pop the last entry from the stack
	lastEntry := stack[len(stack)-1]
	callStack[gid] = stack[:len(stack)-1]

	// Verify this is the expected function
	contractMatch := bytes.Equal(lastEntry.Contract, contract)
	if !contractMatch || lastEntry.Function != function {
		fmt.Printf("Warning: Mismatched Enter/Exit calls. Expected %x:%s, got %x:%s\n",
			lastEntry.Contract, lastEntry.Function, contract, function)
		return
	}

	// Update call info
	callID := genCallID(fmt.Sprintf("%x", lastEntry.Contract), lastEntry.Function, lastEntry.Timestamp)
	if info, ok := callInfo[callID]; ok {
		info.ExitTime = time.Now()
		info.IsCompleted = true
	}
}

// RecordError records an error that occurred during function execution
func RecordError(contract []byte, function, errMsg string) {
	mu.Lock()
	defer mu.Unlock()

	// Get current goroutine ID
	gid := getGoroutineID()

	// Get current call stack
	stack, exists := callStack[gid]
	if !exists || len(stack) == 0 {
		return
	}

	// Find the call info for the given contract and function
	for i := len(stack) - 1; i >= 0; i-- {
		entry := stack[i]
		if bytes.Equal(entry.Contract, contract) && entry.Function == function {
			callID := genCallID(fmt.Sprintf("%x", entry.Contract), entry.Function, entry.Timestamp)
			if info, ok := callInfo[callID]; ok {
				info.Error = errMsg
			}
			break
		}
	}
}

// RecordCrossContractCall records information about a cross-contract call
func RecordCrossContractCall(caller []byte, callerFunc string, target []byte, targetFunc string) {
	// In a real implementation, you might want to record more details
	// here we'll just log the cross-contract call
	fmt.Printf("Cross-contract call: %x.%s -> %x.%s\n", caller, callerFunc, target, targetFunc)
}

// GetCallTree returns the full call tree for debugging
func GetCallTree() map[string][]string {
	mu.Lock()
	defer mu.Unlock()

	// Make a copy to avoid concurrent map access
	treeCopy := make(map[string][]string, len(callTree))
	for k, v := range callTree {
		children := make([]string, len(v))
		copy(children, v)
		treeCopy[k] = children
	}

	return treeCopy
}

// GetCallInfo returns all call information for debugging
func GetCallInfo() map[string]*ContractCallInfo {
	mu.Lock()
	defer mu.Unlock()

	// Make a copy to avoid concurrent map access
	infoCopy := make(map[string]*ContractCallInfo, len(callInfo))
	for k, v := range callInfo {
		infoCopy[k] = &ContractCallInfo{
			Contract:    v.Contract,
			Function:    v.Function,
			EntryTime:   v.EntryTime,
			ExitTime:    v.ExitTime,
			Error:       v.Error,
			Caller:      v.Caller,
			CallerFunc:  v.CallerFunc,
			IsCompleted: v.IsCompleted,
		}
	}

	return infoCopy
}

// ClearAll clears all tracing data
func ClearAll() {
	mu.Lock()
	defer mu.Unlock()

	callTree = make(map[string][]string)
	callInfo = make(map[string]*ContractCallInfo)
	callStack = make(map[uint64][]CallStackEntry)
}
