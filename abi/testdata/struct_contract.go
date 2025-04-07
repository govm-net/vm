package testdata

type User struct {
	Name string
	Age  int
}

type Order struct {
	ID     string
	Amount int
}

type ExternalStruct struct {
	Data string
}

func ProcessUser(user *User) error {
	return nil
}

func CreateOrder(order *Order) error {
	return nil
}

func HandleExternal(data *ExternalStruct) error {
	return nil
}
