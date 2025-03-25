// Package mock provides lightweight contract call tracing and logging
package mock

import (
	"github.com/govm-net/vm/core"
)

var (
	// callStack stores the contract call hierarchy
	callStack []core.Address
)

// GetCurrentContract returns the address of the currently executing contract
// Returns an empty address if the call stack is empty
func GetCurrentContract() core.Address {
	if len(callStack) == 0 {
		return core.Address{}
	}
	return callStack[len(callStack)-1]
}

// GetCaller returns the address of the contract that called the current contract
// This correctly handles the case where a contract calls its own functions
// Returns an empty address if there's no caller (e.g., top-level call)
func GetCaller() core.Address {
	if len(callStack) < 2 {
		return core.Address{} // No caller or top-level call
	}

	// Get the current contract address
	currentContract := callStack[len(callStack)-1]

	// Walk backwards through the call stack to find the first different address
	// This correctly handles cases where a contract calls its own functions
	for i := len(callStack) - 2; i >= 0; i-- {
		if callStack[i] != currentContract {
			return callStack[i]
		}
	}

	// If we reached here, all entries in the call stack are the same contract
	// This means the contract was called from outside (no contract caller)
	return core.Address{}
}

// Enter records function entry by pushing the contract address onto the call stack
func Enter(contract core.Address, function string) {
	callStack = append(callStack, contract)
}

// Exit records function exit by popping the top contract address from the call stack
func Exit(contract core.Address, function string) {
	if len(callStack) > 0 {
		callStack = callStack[:len(callStack)-1]
	}
}
