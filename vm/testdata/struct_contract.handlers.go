package testcontract

import (
	"encoding/json"
	"fmt"

	"github.com/govm-net/vm/core"
)

type ProcessUserParams struct {
	User *User `json:"user"`
}

func handleProcessUser(ctx *core.Context, params []byte) (any, error) {
	var args ProcessUserParams
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, fmt.Errorf("failed to unmarshal params: %w", err)
	}

	// 调用原始函数
	result := ProcessUser(args.User)

	// 处理返回值
	return result, nil
}

type CreateOrderParams struct {
	Order *Order `json:"order"`
}

func handleCreateOrder(ctx *core.Context, params []byte) (any, error) {
	var args CreateOrderParams
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, fmt.Errorf("failed to unmarshal params: %w", err)
	}

	// 调用原始函数
	result := CreateOrder(args.Order)

	// 处理返回值
	return result, nil
}
