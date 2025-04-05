// Package counter implements a simple counter contract example.
// Each counter is stored in a separate object, allowing multiple counters to exist simultaneously.
// The user who calls Initialize becomes the owner of the counter object.
// Only the owner can modify the counter value through Increment or Reset functions.
package counter

import (
	"github.com/govm-net/vm/core"
)

// Initialize creates a new counter object with the specified initial value.
// The caller becomes the owner of the counter object.
// Returns the ObjectID of the created counter.
func Initialize(ctx core.Context, initialValue uint64) core.ObjectID {
	// Create a new counter object
	counterObj := ctx.CreateObject()

	// Set initial counter value
	core.Assert(counterObj.Set("value", initialValue))

	// Set object owner to contract creator
	counterObj.SetOwner(ctx.Sender())

	// Log initialization event
	ctx.Log("CounterInitialized", "initial_value", initialValue, "creator", ctx.Sender())

	// Return counter object ID
	return counterObj.ID()
}

// GetValue retrieves the current value of the specified counter.
// This function can be called by any user.
func GetValue(ctx core.Context, counterID core.ObjectID) uint64 {
	// Get counter object
	counterObj, err := ctx.GetObject(counterID)
	core.Assert(err)

	// Get counter value
	var value uint64
	err = counterObj.Get("value", &value)
	core.Assert(err)

	return value
}

// Increment increases the counter value by the specified amount.
// Only the owner of the counter object can call this function.
func Increment(ctx core.Context, counterID core.ObjectID, amount uint64) error {
	// Get counter object
	counterObj, err := ctx.GetObject(counterID)
	core.Assert(err)

	// Check if caller is the object owner
	core.Assert(counterObj.Owner() != ctx.Sender())

	// Get current value
	var currentValue uint64
	err = counterObj.Get("value", &currentValue)
	core.Assert(err)

	// Increment value
	newValue := currentValue + amount
	core.Assert(counterObj.Set("value", newValue))

	// Log increment event
	ctx.Log("CounterIncremented", "counter_id", counterID, "old_value", currentValue, "new_value", newValue, "amount", amount)

	return nil
}

// Reset sets the counter value back to 0.
// Only the owner of the counter object can call this function.
func Reset(ctx core.Context, counterID core.ObjectID) error {
	// Get counter object
	counterObj, err := ctx.GetObject(counterID)
	core.Assert(err)

	// Check if caller is the object owner
	core.Assert(counterObj.Owner() != ctx.Sender())

	// Reset counter value to 0
	core.Assert(counterObj.Set("value", uint64(0)))

	// Log reset event
	ctx.Log("CounterReset", "counter_id", counterID)

	return nil
}
