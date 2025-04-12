// Simple counter contract example based on wasm wrapper layer
package main

import (
	"encoding/json"
	"fmt"

	"github.com/govm-net/vm/core"
)

// Counter contract state key
const (
	CounterKey = "counter_value"
)

// Initialize contract
// This function is uppercase, so it will be automatically exported and called when the contract is deployed
func Initialize() int32 {
	// Get contract's default Object (empty ObjectID)
	defaultObj, err := core.GetObject(ObjectID{})
	if err != nil {
		core.Log("error", "message", fmt.Sprintf("Get default object failed: %v", err))
		return -1
	}

	// Initialize counter value to 0
	err = defaultObj.Set(CounterKey, uint64(0))
	if err != nil {
		core.Log("error", "message", fmt.Sprintf("Initialize failed: %v", err))
		return -1
	}

	core.Log("initialize", "contract_address", core.ContractAddress())
	return 0
}

// Increment counter
func Increment(value uint64) uint64 {
	// Get default Object
	defaultObj, err := core.GetObject(ObjectID{})
	if err != nil {
		core.Log("error", "message", fmt.Sprintf("Get default object failed: %v", err))
		return 0
	}

	// Get current counter value
	var currentValue uint64
	err = defaultObj.Get(CounterKey, &currentValue)
	if err != nil {
		core.Log("error", "message", fmt.Sprintf("Get counter value failed: %v", err))
		return 0
	}

	// Increment counter value
	newValue := currentValue + value

	// Update counter value
	err = defaultObj.Set(CounterKey, newValue)
	if err != nil {
		core.Log("error", "message", fmt.Sprintf("Update counter value failed: %v", err))
		return 0
	}

	// Record event
	core.Log("increment",
		"from", currentValue,
		"add", value,
		"to", newValue,
		"sender", core.Sender())

	return newValue
}

// Get current counter value
func GetCounter() uint64 {
	// Get default Object
	defaultObj, err := core.GetObject(ObjectID{})
	if err != nil {
		core.Log("error", "message", fmt.Sprintf("Get default object failed: %v", err))
		return 0
	}

	// Get current counter value
	var currentValue uint64
	err = defaultObj.Get(CounterKey, &currentValue)
	if err != nil {
		core.Log("error", "message", fmt.Sprintf("Get counter value failed: %v", err))
		return 0
	}

	return currentValue
}

// Reset counter value to 0
func Reset() {
	// Check if caller is contract owner
	if core.Sender() != core.ContractAddress() {
		core.Log("error", "message", "No permission to reset counter")
		return
	}

	// Get default Object
	defaultObj, err := core.GetObject(ObjectID{})
	if err != nil {
		core.Log("error", "message", fmt.Sprintf("Get default object failed: %v", err))
		return
	}

	// Reset counter value to 0
	err = defaultObj.Set(CounterKey, uint64(0))
	if err != nil {
		core.Log("error", "message", fmt.Sprintf("Reset counter value failed: %v", err))
		return
	}

	// Record event
	core.Log("reset", "sender", core.Sender())
}

// Initialize counter function
func handleInitialize(params []byte) (any, error) {
	fmt.Println("handleInitialize")

	out := Initialize()

	// Record initialization event
	core.Log("CounterInitialized", "value", out)

	// Return success result
	return out, nil
}

// Increment counter function
func handleIncrement(params []byte) (any, error) {
	// Parse parameters
	var incrParams struct {
		Amount int64 `json:"amount"`
	}
	fmt.Printf("handleIncrement params: %s\n", string(params))

	if len(params) > 0 {
		if err := json.Unmarshal(params, &incrParams); err != nil {
			return nil, fmt.Errorf("invalid increment parameters: %w", err)
		}
	} else {
		// Default increment by 1
		incrParams.Amount = 1
	}

	newValue := Increment(uint64(incrParams.Amount))
	// Record increment event
	core.Log("CounterIncremented", "amount", incrParams.Amount, "new_value", newValue)

	// Return success result
	return newValue, nil
}

// Get current counter value
func handleGetCounter(params []byte) (any, error) {
	value := GetCounter()

	// Return current value
	return value, nil
}

// Reset counter function
func handleReset(params []byte) (any, error) {
	// Verify caller permissions
	Reset()
	// Return success result
	return nil, nil
}

// Register contract functions
func init() {
	// Register counter contract function handlers
	registerContractFunction("Initialize", handleInitialize)
	registerContractFunction("Increment", handleIncrement)
	registerContractFunction("GetCounter", handleGetCounter)
	registerContractFunction("Reset", handleReset)
	registerContractFunction("Panic", func(params []byte) (any, error) {
		panic("test panic")
	})
}
