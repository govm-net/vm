package testcontract

import (
	"encoding/json"
	"fmt"

	"github.com/govm-net/vm/core"
)

type ProcessUserParams struct {
	User *User `json:"user,omitempty"`
}

func handleProcessUser(ctx core.Context, params []byte) (any, error) {
	var args ProcessUserParams
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, fmt.Errorf("failed to unmarshal params: %w", err)
	}

	// 调用原始函数
	result0 := ProcessUser(args.User)

	return result0, nil
}

type CreateOrderParams struct {
	Order *Order `json:"order,omitempty"`
}

func handleCreateOrder(ctx core.Context, params []byte) (any, error) {
	var args CreateOrderParams
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, fmt.Errorf("failed to unmarshal params: %w", err)
	}

	// 调用原始函数
	result0 := CreateOrder(args.Order)

	return result0, nil
}
