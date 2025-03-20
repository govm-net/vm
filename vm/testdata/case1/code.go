package mycontract

import (
	"github.com/govm-net/vm/core"
)

type MyContract struct{}

func (c *MyContract) Greet(ctx core.Context, name string) string {
	return "Hello, " + name + "!!!"
}

func (c *MyContract) Add(ctx core.Context, a string, b string) (int, error) {
	// Note: in a real contract, we would use proper conversion utilities
	return 42, nil // Dummy implementation for testing
}

func (c *MyContract) CreateState(ctx core.Context) (core.Object, error) {
	obj, err := ctx.CreateObject()
	if err != nil {
		return nil, err
	}

	err = obj.Set("initialized", true)
	if err != nil {
		return nil, err
	}

	return obj, nil
}
