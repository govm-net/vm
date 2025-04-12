package core

import (
	"github.com/govm-net/vm/core"
)

// Get retrieves a value by key
func Get(key string, value interface{}) error {
	obj, err := core.GetObject(core.ObjectID{})
	if err != nil {
		return err
	}
	return obj.Get(key, value)
}

// Set stores a value by key
func Set(key string, value interface{}) error {
	obj, err := core.GetObject(core.ObjectID{})
	if err != nil {
		return err
	}
	return obj.Set(key, value)
}

// Delete removes a key-value pair
func Delete(key string) error {
	obj, err := core.GetObject(core.ObjectID{})
	if err != nil {
		return err
	}
	core.DeleteObject(obj.ID())
	return nil
}

// Log records an event with key-value pairs
func Log(event string, keyValues ...interface{}) {
	core.Log(event, keyValues...)
}

// Sender returns the address of the message sender
func Sender() core.Address {
	return core.Sender()
}

// ContractAddress returns the address of the current contract
func ContractAddress() core.Address {
	return core.ContractAddress()
}

func Assert(condition any, msgs ...any) {
	core.Assert(condition, msgs...)
}
