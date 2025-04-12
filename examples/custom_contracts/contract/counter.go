package contract

import (
	"github.com/govm-net/vm/examples/custom_contracts/core"
)

// Initialize initializes the counter with an initial value
func Initialize(initialValue int64) error {
	return core.Set("counter", initialValue)
}

// Increment increases the counter by 1
func Increment() error {
	var value int64
	core.Assert(core.Get("counter", &value))

	value++
	core.Assert(core.Set("counter", value))

	core.Log("Increment",
		"sender", core.Sender(),
		"new_value", value)

	return nil
}

// Decrement decreases the counter by 1
func Decrement() error {
	var value int64
	core.Assert(core.Get("counter", &value))

	value--
	core.Assert(core.Set("counter", value))

	core.Log("Decrement",
		"sender", core.Sender(),
		"new_value", value)

	return nil
}

// GetValue returns the current counter value
func GetValue() (int64, error) {
	var value int64
	core.Assert(core.Get("counter", &value))
	return value, nil
}

// Reset sets the counter back to 0
func Reset() error {
	core.Assert(core.Set("counter", int64(0)))

	core.Log("Reset", "sender", core.Sender())

	return nil
}
