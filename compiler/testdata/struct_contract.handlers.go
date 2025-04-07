package testdata

import (
	"encoding/json"
	"fmt"
)

type ProcessUserParams struct {
	User *User `json:"user,omitempty"`
}

func handleProcessUser(params []byte) (any, error) {
	var args ProcessUserParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &args); err != nil {
			return nil, fmt.Errorf("failed to unmarshal params: %w", err)
		}
	}

	// 调用原始函数
	result0 := ProcessUser(args.User)

	return result0, nil
}

type CreateOrderParams struct {
	Order *Order `json:"order,omitempty"`
}

func handleCreateOrder(params []byte) (any, error) {
	var args CreateOrderParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &args); err != nil {
			return nil, fmt.Errorf("failed to unmarshal params: %w", err)
		}
	}

	// 调用原始函数
	result0 := CreateOrder(args.Order)

	return result0, nil
}
