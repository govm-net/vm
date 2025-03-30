package testcontract

import (
	"encoding/json"
	"fmt"

	"github.com/govm-net/vm/core"
)

type GetUserInfoParams struct {
	Ctx core.Context `json:"ctx,omitempty"`
	Id  string       `json:"id,omitempty"`
}

func handleGetUserInfo(ctx core.Context, params []byte) (any, error) {
	var args GetUserInfoParams
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, fmt.Errorf("failed to unmarshal params: %w", err)
	}

	// 调用原始函数
	result0, result1, result2 := GetUserInfo(ctx, args.Id)

	// 处理返回值
	return map[string]any{
		"result0": result0,
		"result1": result1,
		"result2": result2,
	}, nil
}

type CreateOrderWithDetailsParams struct {
	Ctx    core.Context `json:"ctx,omitempty"`
	Userid string       `json:"userID,omitempty"`
	Amount float64      `json:"amount,omitempty"`
}

func handleCreateOrderWithDetails(ctx core.Context, params []byte) (any, error) {
	var args CreateOrderWithDetailsParams
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, fmt.Errorf("failed to unmarshal params: %w", err)
	}

	// 调用原始函数
	result0, result1, result2 := CreateOrderWithDetails(ctx, args.Userid, args.Amount)

	// 处理返回值
	return map[string]any{
		"result0": result0,
		"result1": result1,
		"result2": result2,
	}, nil
}

type ProcessUserAndOrderParams struct {
	Ctx  core.Context `json:"ctx,omitempty"`
	User *User        `json:"user,omitempty"`
}

func handleProcessUserAndOrder(ctx core.Context, params []byte) (any, error) {
	var args ProcessUserAndOrderParams
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, fmt.Errorf("failed to unmarshal params: %w", err)
	}

	// 调用原始函数
	result0, result1, result2 := ProcessUserAndOrder(ctx, args.User)

	// 处理返回值
	return map[string]any{
		"result0": result0,
		"result1": result1,
		"result2": result2,
	}, nil
}
