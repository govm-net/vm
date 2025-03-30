package restrictedcontract

import (
	"github.com/govm-net/vm/core"
)

type RestrictedContract struct{}

func (c *RestrictedContract) DoSomething(ctx core.Context) chan string {
	ch1 := make(chan string)
	ch2 := make(chan string)
	go func() {
		select { // Using 'select' keyword is restricted
		case <-ch1:
			return
		case <-ch2:
			return
		}
	}()
	return ch1
}
