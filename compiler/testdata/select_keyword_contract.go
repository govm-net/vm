package testdata

type RestrictedContract struct{}

func (c *RestrictedContract) DoSomething() chan string {
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
