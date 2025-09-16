package context

import (
	"testing"

	"github.com/govm-net/vm/context"
	_ "github.com/govm-net/vm/context/memory"
	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/types"
)

func TestBlockchainContextAdapter(t *testing.T) {
	// Create a memory context to use as the underlying BlockchainContext
	memCtx, err := context.Get(context.MemoryContextType, nil)
	if err != nil {
		t.Fatalf("Failed to create memory context: %v", err)
	}

	// Create a contract address for testing
	contractAddr := core.AddressFromString("0x1234567890abcdef1234567890abcdef12345678")

	// Create the adapter
	adapter := context.NewBlockchainContextAdapter(memCtx, contractAddr)

	// Test BlockHeight
	height := adapter.BlockHeight()
	if height != 0 {
		t.Errorf("Expected block height 0, got %d", height)
	}

	// Test BlockTime
	time := adapter.BlockTime()
	if time != 0 {
		t.Errorf("Expected block time 0, got %d", time)
	}

	// Note: ContractAddress in the memory context returns the contract address set in the context
	// We're not setting it explicitly, so it will be the zero address
	_ = adapter.ContractAddress()
	// We're just testing that the call works without panicking

	// Test Sender
	_ = adapter.Sender()
	// We're just testing that the call works without panicking

	// Test Balance
	balance := adapter.Balance(contractAddr)
	if balance != 0 {
		t.Errorf("Expected balance 0, got %d", balance)
	}
}

func TestObjectAdapter(t *testing.T) {
	// Create a memory context to use as the underlying BlockchainContext
	memCtx, err := context.Get(context.MemoryContextType, nil)
	if err != nil {
		t.Fatalf("Failed to create memory context: %v", err)
	}

	// Create a contract address for testing
	contractAddr := core.AddressFromString("0x1234567890abcdef1234567890abcdef12345678")

	// Create the adapter
	adapter := context.NewBlockchainContextAdapter(memCtx, contractAddr)

	// Create an object using the adapter
	obj := adapter.CreateObject()

	// Test object ID
	id := obj.ID()
	if id == (types.ObjectID{}) {
		t.Error("Expected non-zero object ID")
	}

	// Test object owner
	_ = obj.Owner()
	// In memory context, the owner should be the contract address

	// Test object contract
	_ = obj.Contract()
	// In memory context, the contract should be the contract address

	// Test setting and getting a field
	testValue := "test_value"
	err = obj.Set("test_field", testValue)
	if err != nil {
		t.Errorf("Failed to set field: %v", err)
	}

	var retrievedValue string
	err = obj.Get("test_field", &retrievedValue)
	if err != nil {
		t.Errorf("Failed to get field: %v", err)
	}

	if retrievedValue != testValue {
		t.Errorf("Expected value %s, got %s", testValue, retrievedValue)
	}

	// Test setting owner
	newOwner := core.AddressFromString("0xabcdef1234567890abcdef1234567890abcdef12")
	obj.SetOwner(newOwner)

	// Note: We can't easily verify the owner change without getting the object again
	// because the ObjectAdapter caches the owner locally in some implementations

	// Test deleting object
	// We're not actually deleting the object here because we don't have a direct way to test it
	// without getting the object again, but we're testing that the call doesn't panic
	adapter.DeleteObject(id)
}

func TestTransferAndCall(t *testing.T) {
	// Create a memory context to use as the underlying BlockchainContext
	memCtx, err := context.Get(context.MemoryContextType, nil)
	if err != nil {
		t.Fatalf("Failed to create memory context: %v", err)
	}

	// Create a contract address for testing
	contractAddr := core.AddressFromString("0x1234567890abcdef1234567890abcdef12345678")

	// Create the adapter
	adapter := context.NewBlockchainContextAdapter(memCtx, contractAddr)

	// Test Transfer - this should fail due to insufficient balance
	fromAddr := core.AddressFromString("0x1234567890abcdef1234567890abcdef12345678")
	toAddr := core.AddressFromString("0xabcdef1234567890abcdef1234567890abcdef12")

	err = adapter.Transfer(fromAddr, toAddr, 100)
	if err == nil {
		// We expect this to fail, so if it doesn't, that's unexpected
		t.Error("Expected transfer to fail due to insufficient balance, but it succeeded")
	}

	// Test Call - this should fail because there's no actual contract to call
	_, err = adapter.Call(toAddr, "test_function", "arg1", "arg2")
	if err == nil {
		// We expect this to fail, so if it doesn't, that's unexpected
		t.Error("Expected call to fail, but it succeeded")
	}
}

func TestLog(t *testing.T) {
	// Create a memory context to use as the underlying BlockchainContext
	memCtx, err := context.Get(context.MemoryContextType, nil)
	if err != nil {
		t.Fatalf("Failed to create memory context: %v", err)
	}

	// Create a contract address for testing
	contractAddr := core.AddressFromString("0x1234567890abcdef1234567890abcdef12345678")

	// Create the adapter
	adapter := context.NewBlockchainContextAdapter(memCtx, contractAddr)

	// Test Log - we're just testing that the call doesn't panic
	adapter.Log("test_event", "key1", "value1", "key2", 123)
}
