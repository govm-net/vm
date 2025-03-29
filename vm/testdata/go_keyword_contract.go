package restrictedcontract

import (
	"github.com/govm-net/vm/core"
)

type RestrictedContract struct{}

func (c *RestrictedContract) DoSomething(ctx core.Context) string {
	go func() {} // Using 'go' keyword is restricted
	return "Something"
} 