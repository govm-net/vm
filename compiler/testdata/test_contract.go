package testdata

// Package-level function
func GetVersion() string {
	return "1.0.0"
}

// Another package-level function with parameters
func Calculate(a int, b int) (int, error) {
	return a + b, nil
}

type TestContract struct{}

func (c *TestContract) DoSomething() string {
	return "Something"
}
