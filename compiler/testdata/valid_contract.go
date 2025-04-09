package testdata

type ValidContract struct{}

func (c *ValidContract) DoSomething() string {
	return "Something"
}

func DoSomething2() string {
	return "Something"
}
