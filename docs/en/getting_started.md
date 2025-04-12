# Getting Started

This guide will help you get started with developing WebAssembly smart contracts using the VM project.

## 1. Prerequisites

### 1.1 Required Tools

- Go 1.20 or later
- TinyGo 0.28.0 or later
- Git

### 1.2 Installation

```bash
# Install Go
brew install go

# Install TinyGo
brew install tinygo

# Install VM SDK
go get github.com/govm-net/vm
```

## 2. Quick Start

### 2.1 Create a Simple Contract

Create a new directory for your contract:

```bash
mkdir counter
cd counter
```

Initialize a Go module:

```bash
go mod init counter
```

Create a simple counter contract:

```go
package counter

import (
    "github.com/govm-net/vm/core"
    "github.com/govm-net/vm/mock"
)

// Counter state
type Counter struct {
    Value int64
}

// Initialize counter
func Init() {
    counter := Counter{Value: 0}
    object := core.CreateObject()
    object.Set("counter", counter)
}

// Increment counter
func Increment() error {
    counter, err := getCounter()
    if err != nil {
        return err
    }
    
    counter.Value++
    object.Set("counter", counter)
    return nil
}

// Get counter value
func GetValue() (int64, error) {
    counter, err := getCounter()
    if err != nil {
        return 0, err
    }
    return counter.Value, nil
}
```

### 2.2 Compile the Contract

Compile the contract using TinyGo:

```bash
tinygo build -o counter.wasm -target=wasi -opt=z -no-debug .
```

### 2.3 Deploy the Contract

Deploy the contract using the VM CLI:

```bash
vm deploy counter.wasm
```

### 2.4 Interact with the Contract

Call contract functions using the VM CLI:

```bash
# Increment counter
vm call <contract_address> Increment

# Get counter value
vm call <contract_address> GetValue
```

## 3. Development Guide

### 3.1 Project Structure

A typical contract project has the following structure:

```
contract/
├── go.mod
├── contract.go
├── state.go
└── tests/
    └── contract_test.go
```

### 3.2 Contract Development

1. **State Definition**:
   ```go
   type State struct {
       // State fields
   }
   ```

2. **Initialization**:
   ```go
   func Init() {
       // Initialize state
   }
   ```

3. **Public Functions**:
   ```go
   func PublicFunction() error {
       // Function implementation
   }
   ```

4. **Private Functions**:
   ```go
   func privateFunction() error {
       // Function implementation
   }
   ```

### 3.3 Testing

1. **Unit Tests**:
   ```go
   func TestContract(t *testing.T) {
       // Test initialization
       Init()
       
       // Test functions
       err := PublicFunction()
       if err != nil {
           t.Fatal(err)
       }
   }
   ```

2. **Integration Tests**:
   ```go
   func TestContractIntegration(t *testing.T) {
       // Deploy contract
       contract, err := deployContract("contract.wasm")
       if err != nil {
           t.Fatal(err)
       }
       
       // Test contract functions
       result, err := contract.Call("PublicFunction", nil)
       if err != nil {
           t.Fatal(err)
       }
   }
   ```

## 4. Best Practices

### 4.1 Security

1. **Input Validation**:
   ```go
   func ProcessInput(input []byte) error {
       // Validate input size
       if len(input) > MaxInputSize {
           return errors.New("input too large")
       }
       
       // Validate input content
       if !isValidInput(input) {
           return errors.New("invalid input")
       }
       
       return nil
   }
   ```

2. **Access Control**:
   ```go
   func RestrictedFunction() error {
       // Check caller permission
       if !hasPermission(core.Sender()) {
           return errors.New("permission denied")
       }
       
       // Execute function
       return nil
   }
   ```

### 4.2 Performance

1. **State Management**:
   ```go
   // Cache state
   var cachedState *State
   
   func getState() (*State, error) {
       if cachedState != nil {
           return cachedState, nil
       }
       
       object := core.GetObjectWithOwner(core.ContractAddress())
       var state State
       err := object.Get("state", &state)
       if err != nil {
           return nil, err
       }
       
       cachedState = &state
       return cachedState, nil
   }
   ```

2. **Gas Optimization**:
   ```go
   func OptimizedFunction() error {
       // Batch operations
       updates := make([]StateUpdate, 0)
       
       // Collect updates
       for _, item := range items {
           updates = append(updates, StateUpdate{
               Field: item.Field,
               Value: item.Value,
           })
       }
       
       // Apply updates in batch
       return applyUpdates(updates)
   }
   ```

## 5. Examples

### 5.1 Simple Storage

```go
package storage

import (
    "github.com/govm-net/vm/core"
    "github.com/govm-net/vm/mock"
)

// Storage state
type Storage struct {
    Data map[string]string
}

// Initialize storage
func Init() {
    storage := Storage{
        Data: make(map[string]string),
    }
    object := core.CreateObject()
    object.Set("storage", storage)
}

// Store data
func Store(key, value string) error {
    storage, err := getStorage()
    if err != nil {
        return err
    }
    
    storage.Data[key] = value
    object.Set("storage", storage)
    return nil
}

// Retrieve data
func Retrieve(key string) (string, error) {
    storage, err := getStorage()
    if err != nil {
        return "", err
    }
    
    value, exists := storage.Data[key]
    if !exists {
        return "", errors.New("key not found")
    }
    
    return value, nil
}
```

### 5.2 Token Contract

```go
package token

import (
    "github.com/govm-net/vm/core"
    "github.com/govm-net/vm/mock"
)

// Token state
type Token struct {
    Name     string
    Symbol   string
    Supply   uint64
    Balances map[core.Address]uint64
}

// Initialize token
func Init(name, symbol string, initialSupply uint64) {
    token := Token{
        Name:     name,
        Symbol:   symbol,
        Supply:   initialSupply,
        Balances: make(map[core.Address]uint64),
    }
    
    // Mint initial supply to creator
    token.Balances[core.Sender()] = initialSupply
    
    object := core.CreateObject()
    object.Set("token", token)
}

// Transfer tokens
func Transfer(to core.Address, amount uint64) error {
    token, err := getToken()
    if err != nil {
        return err
    }
    
    from := core.Sender()
    if token.Balances[from] < amount {
        return errors.New("insufficient balance")
    }
    
    token.Balances[from] -= amount
    token.Balances[to] += amount
    
    object.Set("token", token)
    return nil
}

// Get balance
func BalanceOf(addr core.Address) (uint64, error) {
    token, err := getToken()
    if err != nil {
        return 0, err
    }
    return token.Balances[addr], nil
}
```

## 6. Next Steps

1. Read the [WASM Contract Interface](wasm_contract_interface.md) documentation
2. Explore the [VM Architecture](vm_architecture.md)
3. Learn about [Gas Billing](gas.md)
4. Check out more [Examples](../examples) 