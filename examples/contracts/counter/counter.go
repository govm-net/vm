// A simple counter contract example based on WASM wrapper
package countercontract

import (
	"github.com/govm-net/vm/core"
)

// State key for the counter contract
const (
	CounterKey = "counter_value"
)

// Initialize the contract
// This function starts with a capital letter, so it will be automatically exported
// and called when the contract is deployed
func Initialize() int32 {
	// Get the contract's default Object (empty ObjectID)
	defaultObj, err := core.GetObject(core.ObjectID{})
	core.Assert(err)

	// Initialize counter value to 0
	err = defaultObj.Set(CounterKey, uint64(0))
	core.Assert(err)

	core.Log("initialize", "contract_address", core.ContractAddress())
	return 0
}

// Increment the counter
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

	// Log event
	core.Log("increment",
		"from", currentValue,
		"add", value,
		"to", newValue,
		"sender", core.Sender())

	return newValue
}

// GetCounter returns the current counter value
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

// Reset sets the counter value back to 0
func Reset() {
	// Get default Object
	defaultObj, err := core.GetObject(core.ObjectID{})
	core.Assert(err)

	// Reset counter value to 0
	err = defaultObj.Set(CounterKey, uint64(0))
	core.Assert(err)

	// Log event
	core.Log("reset", "sender", core.Sender())
}
