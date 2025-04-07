package testdata

// User1 represents a user in the system
type User1 struct {
	Name  string
	Age   int
	Email string
}

// Order1 represents an order in the system
type Order1 struct {
	ID     string
	Amount float64
	Status string
}

// GetUserInfo returns user information with multiple values
func GetUserInfo(id string) (string, int, error) {
	// 模拟从数据库获取用户信息
	return "John Doe", 30, nil
}

// CreateOrderWithDetails creates an order and returns multiple values
func CreateOrderWithDetails(userID string, amount float64) (*Order1, string, error) {
	order := &Order1{
		ID:     "order123",
		Amount: amount,
		Status: "created",
	}
	return order, "Order1 created successfully", nil
}

// ProcessUserAndOrder processes a user and creates an order
func ProcessUserAndOrder(user *User1) (*User1, *Order1, error) {
	// 模拟处理用户和创建订单
	order := &Order1{
		ID:     "order456",
		Amount: 100.0,
		Status: "processed",
	}
	return user, order, nil
}
