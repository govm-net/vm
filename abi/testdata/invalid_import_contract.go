package testdata

import (
	"fmt"
)

type InvalidContract struct{}

func (c *InvalidContract) DoSomething() string {
	fmt.Println("DoSomething")
	return "Something"
}
