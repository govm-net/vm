package validcontract

import (
	"github.com/govm-net/vm/core"
)

type ValidContract struct{}

func (c *ValidContract) DoSomething(ctx core.Context) string {
	return "Something"
}
