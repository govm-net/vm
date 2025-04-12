# WebAssembly Smart Contracts

This document provides an overview of WebAssembly smart contracts in the VM project, including their design principles, features, and implementation details.

## 1. Overview

WebAssembly smart contracts are programs written in Go that are compiled to WebAssembly bytecode and executed in a secure sandbox environment. They provide a deterministic and efficient way to implement blockchain business logic.

### 1.1 Key Features

- **Security**: Sandboxed execution environment prevents malicious behavior
- **Determinism**: Guaranteed consistent execution results
- **Efficiency**: Optimized compilation and execution
- **Interoperability**: Standard WebAssembly interface
- **Gas Billing**: Precise resource consumption tracking

### 1.2 Design Principles

1. **Security First**: All operations are sandboxed and validated
2. **Deterministic Execution**: No non-deterministic operations allowed
3. **Resource Control**: Gas-based resource consumption tracking
4. **State Isolation**: Contract state is isolated and controlled
5. **Standard Interface**: Consistent interface for all contracts

## 2. Contract Structure

A typical WebAssembly smart contract has the following structure:

```go
package contract

import (
    "github.com/govm-net/vm/core"
    "github.com/govm-net/vm/mock"
)

// Contract state structure
type State struct {
    Owner    core.Address
    Balance  uint64
    Counter  int64
}

// Initialize contract
func Init() {
    // Initialize contract state
    state := State{
        Owner:   core.Sender(),
        Balance: 0,
        Counter: 0,
    }
    
    // Save initial state
    object := core.CreateObject()
    object.Set("state", state)
}

// Public function - automatically exported
func Transfer(to core.Address, amount uint64) error {
    // Check sender is owner
    if core.Sender() != state.Owner {
        return errors.New("only owner can transfer")
    }
    
    // Check balance
    if state.Balance < amount {
        return errors.New("insufficient balance")
    }
    
    // Update state
    state.Balance -= amount
    object.Set("state", state)
    
    // Transfer tokens
    return core.TransferTo(to, amount)
}

// Public function - automatically exported
func GetBalance() uint64 {
    return state.Balance
}

// Private function - not exported
func verifyTransfer(amount uint64) bool {
    return amount > 0 && amount <= state.Balance
}
```

## 3. Contract Development

### 3.1 Development Environment

To develop WebAssembly smart contracts, you need:

1. **Go 1.21 or later**
2. **TinyGo 0.28 or later**
3. **VM SDK**:
   ```bash
   go get github.com/govm-net/vm
   ```

### 3.2 Contract Template

A basic contract template:

```go
package contract

import (
    "github.com/govm-net/vm/core"
    "github.com/govm-net/vm/mock"
)

// Contract state
type State struct {
    // Add state fields
}

// Initialize contract
func Init() {
    // Initialize state
}

// Public functions
func PublicFunction() error {
    // Function implementation
    return nil
}

// Private functions
func privateFunction() {
    // Function implementation
}
```

### 3.3 Compilation

Compile the contract using TinyGo:

```bash
tinygo build -o contract.wasm -target=wasi -opt=z -no-debug .
```

### 3.4 Deployment

Deploy the contract using the VM CLI:

```bash
vm deploy contract.wasm
```

## 4. Contract Features

### 4.1 State Management

Contracts can manage state using objects:

```go
// Create object
object := core.CreateObject()

// Set field
object.Set("field", value)

// Get field
value, err := object.Get("field")
```

### 4.2 Token Operations

Contracts can handle token transfers:

```go
// Receive tokens
func Receive(amount uint64) error {
    return core.Transfer(core.Sender(), core.ContractAddress(), amount)
}

// Transfer tokens
func TransferTo(to core.Address, amount uint64) error {
    return core.Transfer(core.ContractAddress(), to, amount)
}
```

### 4.3 Cross-contract Calls

Contracts can call other contracts:

```go
// Call another contract
func CallOtherContract(contract core.Address, function string, args ...any) ([]byte, error) {
    return core.Call(contract, function, args...)
}
```

### 4.4 Events and Logs

Contracts can emit events:

```go
// Log event
func LogEvent() {
    core.Log("event", "key", "value")
}
```

## 5. Best Practices

### 5.1 Security

1. **Validate all inputs**
2. **Check permissions**
3. **Handle errors properly**
4. **Use safe math operations**
5. **Avoid non-deterministic operations**

### 5.2 Gas Optimization

1. **Minimize state operations**
2. **Use efficient data structures**
3. **Cache frequently used values**
4. **Batch operations when possible**
5. **Avoid unnecessary computations**

### 5.3 Code Organization

1. **Clear state structure**
2. **Modular function design**
3. **Comprehensive error handling**
4. **Documentation and comments**
5. **Unit tests**

## 6. Examples

### 6.1 Simple Counter

```go
package contract

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

// Private helper
func getCounter() (*Counter, error) {
    object := core.GetObjectWithOwner(core.ContractAddress())
    var counter Counter
    err := object.Get("counter", &counter)
    return &counter, err
}
```

### 6.2 Token Contract

```go
package contract

import (
    "github.com/govm-net/vm/core"
    "github.com/govm-net/vm/mock"
)

// Token state
type Token struct {
    Name     string
    Symbol   string
    Decimals uint8
    Total    uint64
    Balances map[core.Address]uint64
}

// Initialize token
func Init(name, symbol string, decimals uint8, total uint64) {
    token := Token{
        Name:     name,
        Symbol:   symbol,
        Decimals: decimals,
        Total:    total,
        Balances: make(map[core.Address]uint64),
    }
    
    // Mint initial supply to creator
    token.Balances[core.Sender()] = total
    
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

// Private helper
func getToken() (*Token, error) {
    object := core.GetObjectWithOwner(core.ContractAddress())
    var token Token
    err := object.Get("token", &token)
    return &token, err
}
```

## 7. Summary

WebAssembly smart contracts provide a secure and efficient way to implement blockchain business logic. By following best practices and using the provided features, developers can create robust and maintainable contracts.

Key points to remember:
1. Security is paramount
2. Gas optimization is important
3. State management should be clear
4. Error handling must be comprehensive
5. Code should be well-documented 