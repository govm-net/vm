package noexportedfuncs

import (
	"github.com/govm-net/vm/core"
)

type NoExportedFuncsContract struct{}

func (c *NoExportedFuncsContract) doSomething(ctx core.Context) string {
	return "Something"
}
