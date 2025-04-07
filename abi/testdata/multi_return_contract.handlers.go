package testdata

import (
	"encoding/json"
	"fmt"
)

type GetUserInfoParams struct {
	Id string `json:"id,omitempty"`
}

func handleGetUserInfo(params []byte) (any, error) {
	var args GetUserInfoParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &args); err != nil {
			return nil, fmt.Errorf("failed to unmarshal params: %w", err)
		}
	}

	// 调用原始函数
	result0, result1, result2 := GetUserInfo(args.Id)

	// 处理返回值
	results := make([]any, 0)
	results = append(results, result0)
	results = append(results, result1)
	results = append(results, result2)
	return results, nil
}

type CreateOrderWithDetailsParams struct {
	Userid string  `json:"userID,omitempty"`
	Amount float64 `json:"amount,omitempty"`
}

func handleCreateOrderWithDetails(params []byte) (any, error) {
	var args CreateOrderWithDetailsParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &args); err != nil {
			return nil, fmt.Errorf("failed to unmarshal params: %w", err)
		}
	}

	// 调用原始函数
	result0, result1, result2 := CreateOrderWithDetails(args.Userid, args.Amount)

	// 处理返回值
	results := make([]any, 0)
	results = append(results, result0)
	results = append(results, result1)
	results = append(results, result2)
	return results, nil
}

type ProcessUserAndOrderParams struct {
	User *User1 `json:"user,omitempty"`
}

func handleProcessUserAndOrder(params []byte) (any, error) {
	var args ProcessUserAndOrderParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &args); err != nil {
			return nil, fmt.Errorf("failed to unmarshal params: %w", err)
		}
	}

	// 调用原始函数
	result0, result1, result2 := ProcessUserAndOrder(args.User)

	// 处理返回值
	results := make([]any, 0)
	results = append(results, result0)
	results = append(results, result1)
	results = append(results, result2)
	return results, nil
}
