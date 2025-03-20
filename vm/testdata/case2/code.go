package statecontract

import (
	"github.com/govm-net/vm/core"
)

type StateContract struct{}

// CreateObject creates a new state object
func (c *StateContract) CreateObject(ctx core.Context) (core.ObjectID, error) {
	obj, err := ctx.CreateObject()
	if err != nil {
		return core.ObjectID{}, err
	}

	err = obj.Set("value", "initial")
	if err != nil {
		return core.ObjectID{}, err
	}

	return obj.ID(), nil
}

// GetValue gets a value from a state object
func (c *StateContract) GetValue(ctx core.Context, id core.ObjectID) (string, error) {
	// Get the object
	obj, err := ctx.GetObject(id)
	if err != nil {
		return "", err
	}

	// Get the value
	val, err := obj.Get("value")
	if err != nil {
		return "", err
	}

	return val.(string), nil
}

// SetValue sets a value in a state object
func (c *StateContract) SetValue(ctx core.Context, objectID string, newValue string) (bool, error) {
	// Convert string to ObjectID
	id := core.ObjectIDFromString(objectID)

	// Get the object
	obj, err := ctx.GetObject(id)
	if err != nil {
		return false, err
	}

	// Set the value
	err = obj.Set("value", newValue)
	if err != nil {
		return false, err
	}

	return true, nil
}

// DeleteObject deletes a state object
func (c *StateContract) DeleteObject(ctx core.Context, id core.ObjectID) (bool, error) {
	// Delete the object
	err := ctx.DeleteObject(id)
	if err != nil {
		return false, err
	}

	return true, nil
}
