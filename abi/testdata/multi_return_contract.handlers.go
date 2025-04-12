package testdata

import (
	"encoding/json"
	"fmt"
)

func init() {
	if false {
		fmt.Println("init")
		json.Marshal(nil)
	}
}

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

	result0, result1, result2 := GetUserInfo(args.Id)

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

	result0, result1, result2 := CreateOrderWithDetails(args.Userid, args.Amount)

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

	result0, result1, result2 := ProcessUserAndOrder(args.User)

	results := make([]any, 0)
	results = append(results, result0)
	results = append(results, result1)
	results = append(results, result2)
	return results, nil
}
