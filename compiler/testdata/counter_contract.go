// Simple counter contract example based on wasm wrapper layer
package testdata

import (
	"github.com/govm-net/vm/core"
)

// Counter contract state key
const (
	CounterKey = "counter_value"
)

// Initialize contract
// This function starts with uppercase, so it will be automatically exported and called when the contract is deployed
func Initialize() int32 {
	// Get contract's default Object (empty ObjectID)
	defaultObj, err := core.GetObject(core.ObjectID{})
	core.Assert(err)

	// Initialize counter value to 0
	err = defaultObj.Set(CounterKey, uint64(0))
	core.Assert(err)

	core.Log("initialize", "contract_address", core.ContractAddress())
	return 0
}

// Increment counter
func Increment(value uint64) uint64 {
	// Get default Object
	defaultObj, err := core.GetObject(core.ObjectID{})
	core.Assert(err)

	// Get current counter value
	var currentValue uint64
	err = defaultObj.Get(CounterKey, &currentValue)
	core.Assert(err)

	// Increment counter value
	newValue := currentValue + value

	// Update counter value
	err = defaultObj.Set(CounterKey, newValue)
	core.Assert(err)

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
	defaultObj, err := core.GetObject(core.ObjectID{})
	core.Assert(err)

	// Get current counter value
	var currentValue uint64
	err = defaultObj.Get(CounterKey, &currentValue)
	core.Assert(err)

	return currentValue
}

// Reset counter value to 0
func Reset() {
	// Check if caller is contract owner
	if core.Sender() != core.ContractAddress() {
		return
	}

	// Get default Object
	defaultObj, err := core.GetObject(core.ObjectID{})
	core.Assert(err)

	// Reset counter value to 0
	err = defaultObj.Set(CounterKey, uint64(0))
	core.Assert(err)

	// Record event
	core.Log("reset", "sender", core.Sender())
}
