package testdata

type NoExportedFuncsContract struct{}

func (c *NoExportedFuncsContract) doSomething() string {
	return "Something"
}
