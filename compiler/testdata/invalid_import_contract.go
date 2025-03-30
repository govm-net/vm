package invalidcontract

import (
	"fmt"

	"github.com/govm-net/vm/core"
)

type InvalidContract struct{}

func (c *InvalidContract) DoSomething(ctx core.Context) string {
	fmt.Println("DoSomething")
	return "Something"
}
