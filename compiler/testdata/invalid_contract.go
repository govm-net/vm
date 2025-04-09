package testdata

// include ;
func Calculate(a int, b int) (int, error) {
	if a := a + b; a > 10 {
		return a, nil
	}
	return a + b, nil
}
