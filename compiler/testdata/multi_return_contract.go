package testcontract

import (
	"github.com/govm-net/vm/core"
)

// User represents a user in the system
type User struct {
	Name  string
	Age   int
	Email string
}

// Order represents an order in the system
type Order struct {
	ID     string
	Amount float64
	Status string
}

// GetUserInfo returns user information with multiple values
func GetUserInfo(ctx core.Context, id string) (string, int, error) {
	// 模拟从数据库获取用户信息
	return "John Doe", 30, nil
}

// CreateOrderWithDetails creates an order and returns multiple values
func CreateOrderWithDetails(ctx core.Context, userID string, amount float64) (*Order, string, error) {
	order := &Order{
		ID:     "order123",
		Amount: amount,
		Status: "created",
	}
	return order, "Order created successfully", nil
}

// ProcessUserAndOrder processes a user and creates an order
func ProcessUserAndOrder(ctx core.Context, user *User) (*User, *Order, error) {
	// 模拟处理用户和创建订单
	order := &Order{
		ID:     "order456",
		Amount: 100.0,
		Status: "processed",
	}
	return user, order, nil
}
