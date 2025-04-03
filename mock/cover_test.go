package mock

import (
	"strings"
	"testing"
)

func TestAddGasConsumption(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "simple function",
			input: `package test

func TestFunc(x int) int {
	y := x + 1
	return y
}`,
			expected: `package test

import "github.com/govm-net/vm/mock"

func TestFunc(x int) int {
	mock.ConsumeGas(2)
	y := x + 1
	return y
}
`,
		},
		{
			name: "function with multiple statements",
			input: `package test

func TestFunc(x int) int {
	a := x + 1
	b := a * 2
	return b
}`,
			expected: `package test

import "github.com/govm-net/vm/mock"

func TestFunc(x int) int {
	mock.ConsumeGas(3)
	a := x + 1
	b := a * 2
	return b
}
`,
		},
		{
			name: "function with if statement",
			input: `package test

func TestFunc(x int) int {
	if x > 0 {
		return x
	}
	return -x
}`,
			expected: `package test

import "github.com/govm-net/vm/mock"

func TestFunc(x int) int {
	mock.ConsumeGas(1)
	if x > 0 {
		mock.ConsumeGas(1)
		return x
	}
	mock.ConsumeGas(1)
	return -x
}
`,
		},
		{
			name: "function with existing imports",
			input: `package test

import "fmt"

func TestFunc(x int) int {
	fmt.Println(x)
	return x
}`,
			expected: `package test

import "github.com/govm-net/vm/mock"
import "fmt"

func TestFunc(x int) int {
	mock.ConsumeGas(2) 
	fmt.Println(x)
	return x
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AddGasConsumption("test", []byte(tt.input))
			if err != nil {
				t.Fatalf("AddGasConsumption() error = %v", err)
				return
			}
			if !compareCode(string(got), tt.expected) {
				t.Errorf("AddGasConsumption() = \n%v\nwant\n%v", string(got), tt.expected)
			}
		})
	}
}

// 比较两个代码字符串，忽略格式差异
func compareCode(got, expected string) bool {
	// 移除所有空白字符
	got = strings.ReplaceAll(got, " ", "")
	got = strings.ReplaceAll(got, "\t", "")
	got = strings.ReplaceAll(got, "\n", "")
	got = strings.ReplaceAll(got, "\r", "")

	expected = strings.ReplaceAll(expected, " ", "")
	expected = strings.ReplaceAll(expected, "\t", "")
	expected = strings.ReplaceAll(expected, "\n", "")
	expected = strings.ReplaceAll(expected, "\r", "")

	return got == expected
}
