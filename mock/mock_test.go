package mock

import (
	"testing"

	"github.com/govm-net/vm/core"
)

// Helper function to create a test Address
func createAddress(val byte) core.Address {
	var addr core.Address
	for i := 0; i < len(addr); i++ {
		addr[i] = val
	}
	return addr
}

// Test helper to reset the callStack between tests
func resetCallStack() {
	callStack = nil
}

func TestGetCurrentContract_EmptyStack(t *testing.T) {
	// Set up
	resetCallStack()

	// Test
	addr := GetCurrentContract()

	// Verify
	emptyAddr := core.Address{}
	if addr != emptyAddr {
		t.Errorf("Expected empty address, got %v", addr)
	}
}

func TestGetCurrentContract_NonEmptyStack(t *testing.T) {
	// Set up
	resetCallStack()
	addrA := createAddress(0xA)
	addrB := createAddress(0xB)
	callStack = append(callStack, addrA.String(), addrB.String())

	// Test
	addr := GetCurrentContract()

	// Verify
	if addr != addrB {
		t.Errorf("Expected address B (%v), got %v", addrB, addr)
	}
}

func TestGetCaller_EmptyStack(t *testing.T) {
	// Set up
	resetCallStack()

	// Test
	addr := GetCaller()

	// Verify
	emptyAddr := core.Address{}
	if addr != emptyAddr {
		t.Errorf("Expected empty address, got %v", addr)
	}
}

func TestGetCaller_SingleItemStack(t *testing.T) {
	// Set up
	resetCallStack()
	addrA := createAddress(0xA)
	callStack = append(callStack, addrA.String())

	// Test
	addr := GetCaller()

	// Verify
	emptyAddr := core.Address{}
	if addr != emptyAddr {
		t.Errorf("Expected empty address, got %v", addr)
	}
}

func TestGetCaller_SimpleTwoItemStack(t *testing.T) {
	// Set up
	resetCallStack()
	addrA := createAddress(0xA)
	addrB := createAddress(0xB)
	callStack = append(callStack, addrA.String(), addrB.String())

	// Test
	addr := GetCaller()

	// Verify
	if addr != addrA {
		t.Errorf("Expected address A (%v), got %v", addrA, addr)
	}
}

func TestGetCaller_SameContractCalls(t *testing.T) {
	// Set up
	resetCallStack()
	addrA := createAddress(0xA)
	callStack = append(callStack, addrA.String(), addrA.String(), addrA.String())

	// Test
	addr := GetCaller()

	// Verify
	emptyAddr := core.Address{}
	if addr != emptyAddr {
		t.Errorf("Expected empty address when all contracts are the same, got %v", addr)
	}
}

func TestGetCaller_MixedContractCalls(t *testing.T) {
	// Set up
	resetCallStack()
	addrA := createAddress(0xA)
	addrB := createAddress(0xB)
	callStack = append(callStack, addrA.String(), addrB.String(), addrB.String(), addrB.String())

	// Test
	addr := GetCaller()

	// Verify
	if addr != addrA {
		t.Errorf("Expected address A (%v) as the caller, got %v", addrA, addr)
	}
}

func TestEnter(t *testing.T) {
	// Set up
	resetCallStack()
	addrA := createAddress(0xA)

	// Test
	Enter(addrA.String(), "testFunction")

	// Verify
	if len(callStack) != 1 {
		t.Errorf("Expected callStack length of 1, got %d", len(callStack))
	}
	if callStack[0] != addrA.String() {
		t.Errorf("Expected address A (%v) on stack, got %v", addrA, callStack[0])
	}

	// Additional test: multiple Enter calls
	addrB := createAddress(0xB)
	Enter(addrB.String(), "anotherFunction")

	if len(callStack) != 2 {
		t.Errorf("Expected callStack length of 2, got %d", len(callStack))
	}
	if callStack[1] != addrB.String() {
		t.Errorf("Expected address B (%v) on top of stack, got %v", addrB, callStack[1])
	}
}

func TestExit(t *testing.T) {
	// Set up
	resetCallStack()
	addrA := createAddress(0xA)
	addrB := createAddress(0xB)
	callStack = append(callStack, addrA.String(), addrB.String())

	// Test
	Exit(addrB.String(), "testFunction")

	// Verify
	if len(callStack) != 1 {
		t.Errorf("Expected callStack length of 1, got %d", len(callStack))
	}
	if callStack[0] != addrA.String() {
		t.Errorf("Expected address A (%v) on stack, got %v", addrA, callStack[0])
	}

	// Test exit on empty stack (edge case)
	resetCallStack()
	Exit(addrA.String(), "someFunction") // This should not panic

	if len(callStack) != 0 {
		t.Errorf("Expected empty callStack, got length %d", len(callStack))
	}
}

func TestCompleteCallChain(t *testing.T) {
	// This test simulates a complete call chain scenario
	resetCallStack()

	addrA := createAddress(0xA)
	addrB := createAddress(0xB)
	addrC := createAddress(0xC)

	// External call to contract A
	Enter(addrA.String(), "mainFunction")

	// Contract A calls its own internal function
	Enter(addrA.String(), "helperFunction")

	// A's helper function calls contract B
	Enter(addrB.String(), "functionB")

	// B calls contract C
	Enter(addrC.String(), "functionC")

	// Check current contract is C
	if GetCurrentContract() != addrC {
		t.Errorf("Expected current contract to be C")
	}

	// Check C's caller is B
	if GetCaller() != addrB {
		t.Errorf("Expected C's caller to be B")
	}

	// C completes and returns to B
	Exit(addrC.String(), "functionC")

	// Check B's caller is A
	if GetCurrentContract() != addrB {
		t.Errorf("Expected current contract to be B")
	}
	if GetCaller() != addrA {
		t.Errorf("Expected B's caller to be A")
	}

	// B completes and returns to A's helper function
	Exit(addrB.String(), "functionB")

	// A's helper completes and returns to A's main function
	Exit(addrA.String(), "helperFunction")

	// Check A has no contract caller
	if GetCurrentContract() != addrA {
		t.Errorf("Expected current contract to be A")
	}
	if GetCaller() != (core.Address{}) {
		t.Errorf("Expected A's caller to be empty address")
	}

	// A completes
	Exit(addrA.String(), "mainFunction")

	// Check stack is empty
	if len(callStack) != 0 {
		t.Errorf("Expected empty call stack after all exits")
	}
}
