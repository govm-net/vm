package vm

import (
	_ "embed"
	"testing"

	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/vm/api"
)

//go:embed testdata/case1/code.go
var case1Code []byte

//go:embed testdata/case2/code.go
var case2Code []byte

// TestEngineDeployAndExecute tests the basic functionality of deploying a contract and executing functions
func TestEngineDeployAndExecute(t *testing.T) {
	// Create a new VM engine
	config := api.DefaultContractConfig()
	engine := NewEngine(config)

	// Simple contract for testing
	contractCode := case1Code

	// Deploy the contract
	contractAddr, err := engine.Deploy(contractCode)
	if err != nil {
		t.Fatalf("Failed to deploy contract: %v", err)
	}

	// Test calling a function that returns a string
	result, err := engine.Execute(contractAddr, "Greet", []byte("World"))
	if err != nil {
		t.Fatalf("Failed to execute Greet function: %v", err)
	}
	expectedGreeting := "Hello, World!"
	if string(result) != expectedGreeting {
		t.Errorf("Expected greeting '%s', but got '%s'", expectedGreeting, string(result))
	}

	// Test calling a function that returns an integer
	result, err = engine.Execute(contractAddr, "Add", []byte("20"), []byte("22"))
	if err != nil {
		t.Fatalf("Failed to execute Add function: %v", err)
	}
	// Our dummy implementation returns 42 regardless of inputs
	expectedResult := "42"
	if string(result) != expectedResult {
		t.Errorf("Expected result '%s', but got '%s'", expectedResult, string(result))
	}
}

// TestEngineObjectOperations tests creating and working with state objects
func TestEngineObjectOperations(t *testing.T) {
	// Create a new VM engine
	config := api.DefaultContractConfig()
	engine := NewEngine(config)

	// Contract that creates and works with state objects
	contractCode := case2Code

	// Deploy the contract
	contractAddr, err := engine.Deploy(contractCode)
	if err != nil {
		t.Fatalf("Failed to deploy contract: %v", err)
	}

	// Create an object
	objectIDBytes, err := engine.Execute(contractAddr, "CreateObject")
	if err != nil {
		t.Fatalf("Failed to create object: %v", err)
	}

	// Convert objectIDBytes to string for later use
	objectIDStr := string(objectIDBytes)

	// Get the initial value
	// We use the string representation which will be converted to ObjectID via ObjectIDFromString
	result, err := engine.Execute(contractAddr, "GetValue", []byte(objectIDStr))
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}
	if string(result) != "initial" {
		t.Errorf("Expected initial value 'initial', but got '%s'", string(result))
	}

	// Set a new value
	_, err = engine.Execute(contractAddr, "SetValue", []byte(objectIDStr), []byte("updated"))
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// Get the updated value
	result, err = engine.Execute(contractAddr, "GetValue", []byte(objectIDStr))
	if err != nil {
		t.Fatalf("Failed to get updated value: %v", err)
	}
	if string(result) != "updated" {
		t.Errorf("Expected updated value 'updated', but got '%s'", string(result))
	}

	// Delete the object
	_, err = engine.Execute(contractAddr, "DeleteObject", []byte(objectIDStr))
	if err != nil {
		t.Fatalf("Failed to delete object: %v", err)
	}

	// Verify the object is deleted by trying to get its value
	_, err = engine.Execute(contractAddr, "GetValue", []byte(objectIDStr))
	if err == nil {
		t.Errorf("Expected error when getting value from deleted object, but got none")
	}
}

// TestEngineContractValidation tests the contract validation logic
func TestEngineContractValidation(t *testing.T) {
	// Create a new VM engine
	config := api.DefaultContractConfig()
	engine := NewEngine(config)

	// Valid contract
	validContract := []byte(`
package validcontract

import (
	"github.com/govm-net/vm/core"
)

type ValidContract struct{}

func (c *ValidContract) DoSomething(ctx core.Context) string {
	return "Something"
}
`)

	// Contract with disallowed import
	invalidImportContract := []byte(`
package invalidcontract

import (
	"github.com/govm-net/vm/core"
	"fmt" // This import is not allowed
)

type InvalidContract struct{}

func (c *InvalidContract) DoSomething(ctx core.Context) string {
	return fmt.Sprintf("Something")
}
`)

	// Contract with restricted keyword
	restrictedKeywordContract := []byte(`
package restrictedcontract

import (
	"github.com/govm-net/vm/core"
)

type RestrictedContract struct{}

func (c *RestrictedContract) DoSomething(ctx core.Context) string {
	go func() {} // Using the 'go' keyword is restricted
	return "Something"
}
`)

	// Contract with no exported functions
	noExportedFuncsContract := []byte(`
package noexportedfuncs

import (
	"github.com/govm-net/vm/core"
)

type NoExportedFuncsContract struct{}

func (c *NoExportedFuncsContract) doSomething(ctx core.Context) string {
	return "Something"
}
`)

	// Test valid contract
	_, err := engine.Deploy(validContract)
	if err != nil {
		t.Errorf("Valid contract should deploy without error, but got: %v", err)
	}

	// Test contract with disallowed import
	_, err = engine.Deploy(invalidImportContract)
	if err == nil {
		t.Errorf("Contract with disallowed import should fail to deploy")
	}

	// Test contract with restricted keyword
	_, err = engine.Deploy(restrictedKeywordContract)
	if err == nil {
		t.Errorf("Contract with restricted keyword should fail to deploy")
	}

	// Test contract with no exported functions
	_, err = engine.Deploy(noExportedFuncsContract)
	if err == nil {
		t.Errorf("Contract with no exported functions should fail to deploy")
	}
}

// TestEngineStateObject tests the StateObject implementation
func TestEngineStateObject(t *testing.T) {
	// Create a state object
	owner := core.Address{}
	contractAddr := core.Address{}
	obj := NewStateObject(owner, contractAddr)

	// Test ID, Type, and Owner
	if obj.ID() == (core.ObjectID{}) {
		t.Errorf("StateObject ID should not be empty")
	}
	if obj.Type() != contractAddr.String() {
		t.Errorf("StateObject Type should be contract address, got '%s'", obj.Type())
	}
	if obj.Owner() != owner {
		t.Errorf("StateObject Owner should match the provided value")
	}

	// Test Set and Get
	err := obj.Set("key1", "value1")
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}
	err = obj.Set("key2", 42)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	val1, err := obj.Get("key1")
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}
	if val1 != "value1" {
		t.Errorf("Expected value 'value1', got '%v'", val1)
	}

	val2, err := obj.Get("key2")
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}
	if val2 != 42 {
		t.Errorf("Expected value 42, got '%v'", val2)
	}

	// Test Delete
	err = obj.Delete("key1")
	if err != nil {
		t.Fatalf("Failed to delete value: %v", err)
	}
	_, err = obj.Get("key1")
	if err == nil {
		t.Errorf("Expected error when getting deleted field, but got none")
	}

	// Test SetOwner
	newOwner := core.Address{1, 2, 3}
	err = obj.SetOwner(newOwner)
	if err != nil {
		t.Fatalf("Failed to set owner: %v", err)
	}
	if obj.Owner() != newOwner {
		t.Errorf("Owner should be updated to the new value")
	}
}

// TestCreateObjectTypeSelection tests that CreateObject creates the appropriate
// type of object based on whether a database provider is set.
func TestCreateObjectTypeSelection(t *testing.T) {
	// Create a new VM engine
	config := api.DefaultContractConfig()
	engine := NewEngine(config)
	contractAddr := core.Address{}
	ctx := NewExecutionContext(contractAddr, engine)

	// 1. Test without DB provider (should create memory object)
	obj1, err := ctx.CreateObject()
	if err != nil {
		t.Fatalf("Failed to create memory object: %v", err)
	}

	// Verify it's a StateObject by type assertion
	_, isStateObject := obj1.(*StateObject)
	if !isStateObject {
		t.Errorf("Expected memory-backed StateObject, got different type: %T", obj1)
	}

	// 2. Set a DB provider and test again (should create DB object)
	memDB := newInMemoryDB()
	keyGen := NewDefaultKeyGenerator()
	engine.SetDBProvider(memDB, keyGen)

	obj2, err := ctx.CreateObject()
	if err != nil {
		t.Fatalf("Failed to create DB object: %v", err)
	}

	// Verify it's a DBStateObject by type assertion
	_, isDBStateObject := obj2.(*DBStateObject)
	if !isDBStateObject {
		t.Errorf("Expected database-backed DBStateObject, got different type: %T", obj2)
	}
}

// Simple in-memory database implementation for testing
type inMemoryDB struct {
	data map[string][]byte
}

func newInMemoryDB() *inMemoryDB {
	return &inMemoryDB{
		data: make(map[string][]byte),
	}
}

func (db *inMemoryDB) Get(key []byte) ([]byte, error) {
	value, exists := db.data[string(key)]
	if !exists {
		return nil, nil
	}
	return value, nil
}

func (db *inMemoryDB) Put(key []byte, value []byte) error {
	db.data[string(key)] = value
	return nil
}

func (db *inMemoryDB) Delete(key []byte) error {
	delete(db.data, string(key))
	return nil
}

func (db *inMemoryDB) Has(key []byte) (bool, error) {
	_, exists := db.data[string(key)]
	return exists, nil
}

func (db *inMemoryDB) Close() error {
	return nil
}

// Iterator for inMemoryDB
type inMemoryIterator struct {
	keys   []string
	values [][]byte
	pos    int
}

func (db *inMemoryDB) Iterator(start, end []byte) (Iterator, error) {
	var keys []string
	var values [][]byte

	startStr := string(start)
	endStr := ""
	if end != nil {
		endStr = string(end)
	}

	for k, v := range db.data {
		if (startStr == "" || k >= startStr) && (endStr == "" || k < endStr) {
			keys = append(keys, k)
			values = append(values, v)
		}
	}

	return &inMemoryIterator{
		keys:   keys,
		values: values,
		pos:    -1,
	}, nil
}

func (it *inMemoryIterator) Next() bool {
	it.pos++
	return it.pos < len(it.keys)
}

func (it *inMemoryIterator) Error() error {
	return nil
}

func (it *inMemoryIterator) Key() []byte {
	if it.pos >= len(it.keys) {
		return nil
	}
	return []byte(it.keys[it.pos])
}

func (it *inMemoryIterator) Value() []byte {
	if it.pos >= len(it.values) {
		return nil
	}
	return it.values[it.pos]
}

func (it *inMemoryIterator) Close() error {
	return nil
}
