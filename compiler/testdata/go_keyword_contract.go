package testdata

type RestrictedContract1 struct{}

func (c *RestrictedContract1) DoSomething() string {
	go func() {}() // Using 'go' keyword is restricted
	return "Something"
}
