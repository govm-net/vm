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
	mock.Enter("0x12345678", "TestFunc")
	defer mock.Exit("0x12345678", "TestFunc")
	mock.ConsumeGas(int64(vm_cover_atomic_.NumStmt[0]))
	y := x + 1
	return y
}

var vm_cover_atomic_ = struct {
	Count   [1]uint32
	Pos     [3 * 1]uint32
	NumStmt [1]uint16
}{
	Pos: [3 * 1]uint32{
		3, 6, 0x2001a,
	},
	NumStmt: [1]uint16{
		2,
	},
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
	mock.Enter("0x12345678", "TestFunc")
	defer mock.Exit("0x12345678", "TestFunc")
	mock.ConsumeGas(int64(vm_cover_atomic_.NumStmt[0]))
	a := x + 1
	b := a * 2
	return b
}

var vm_cover_atomic_ = struct {
	Count   [1]uint32
	Pos     [3 * 1]uint32
	NumStmt [1]uint16
}{
	Pos: [3 * 1]uint32{
		3, 7, 0x4001c,
	},
	NumStmt: [1]uint16{
		3,
	},
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
            	mock.Enter("0x12345678", "TestFunc")
            	defer mock.Exit("0x12345678", "TestFunc")
            	mock.ConsumeGas(int64(vm_cover_atomic_.NumStmt[0]))
            	if x > 0 {
            		mock.ConsumeGas(int64(vm_cover_atomic_.NumStmt[2]))
            		return x
            	}
            	mock.ConsumeGas(int64(vm_cover_atomic_.NumStmt[1]))
            	return -x
            }
            
            var vm_cover_atomic_ = struct {
            	Count   [3]uint32
            	Pos     [3 * 3]uint32
            	NumStmt [3]uint16
            }{
            	Pos: [3 * 3]uint32{
            		3, 4, 0xd001c,
            		7, 7, 0xd0004,
            		4, 6, 0x5000d,
            	},
            	NumStmt: [3]uint16{
            		1,
            		1,
            		1,
            	},
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
            	mock.Enter("0x12345678", "TestFunc")
            	defer mock.Exit("0x12345678", "TestFunc")
            	mock.ConsumeGas(int64(vm_cover_atomic_.NumStmt[0]))
            	fmt.Println(x)
            	return x
            }
            
            var vm_cover_atomic_ = struct {
            	Count   [1]uint32
            	Pos     [3 * 1]uint32
            	NumStmt [1]uint16
            }{
            	Pos: [3 * 1]uint32{
            		5, 8, 0x4001c,
            	},
            	NumStmt: [1]uint16{
            		2,
            	},
            }
            
		`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AddGasConsumption("0x12345678", []byte(tt.input))
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
