# Gas Billing Mechanism

## Overview
Gas is a unit of computational resources consumed during smart contract execution. Each contract execution requires a certain amount of gas, and when gas is exhausted, contract execution is terminated. This mechanism prevents infinite loops or overly complex operations in contracts.

## Gas Billing Rules

### Basic Billing
1. Code Line Billing
   - Each line of code consumes 1 gas point
   - Includes all statements such as variable declarations, assignments, and function calls
   - Excludes empty lines and comments
   - Code line billing is cumulative with interface call billing

2. Function Call Billing
   - Function entry consumes 1 gas point
   - Statements within function body are billed per line
   - When a statement block ends, all statements within the block are billed at once

### Context Interface Operation Billing
Based on actual consumption in wasm/contract.go:

1. Basic Information Queries
   - Sender(): 10 gas
   - BlockHeight(): 10 gas
   - BlockTime(): 10 gas
   - ContractAddress(): 10 gas

2. Balance and Transfers
   - Balance(addr): 50 gas
   - Transfer(to, amount): 500 gas

3. Contract Calls
   - Call(contract, function, args...): 10000 gas
   - Note: Call method reserves 10000 gas as basic call fee, remaining gas is allocated to the called contract
   - After called contract execution completes, billing is based on actual gas consumed by the called contract

4. Object Operations
   - CreateObject(): 500 gas
   - GetObject(id): 50 gas
   - GetObjectWithOwner(owner): 50 gas
   - DeleteObject(id): 500 gas

5. Logging
   - Log(event, keyValues...): 100 gas + data length gas

### Object Interface Operation Billing
Based on actual consumption in wasm/contract.go:

1. Basic Operations
   - ID(): 10 gas
   - Contract(): 100 gas

2. Ownership Operations
   - Owner(): 100 gas
   - SetOwner(owner): 500 gas

3. Field Operations
   - Get(field, value): 100 gas + result data size gas
   - Set(field, value): 1000 gas + data size * 100 gas

### Special Operation Billing
1. Contract Deployment
   - Basic deployment cost: 1000 gas
   - Code size billing: 1 gas per byte
   - Constructor parameters: 100 gas per parameter

2. Contract Calls
   - Basic call cost: 100 gas
   - Function parameters: 50 gas per parameter
   - Return values: 50 gas per return value

### Gas Refund Rules
1. Delete object, refund 300 gas

## Gas Limits
Gas limits are optional configuration items, with specific limits determined by the integration platform. Here are some common limit examples:

1. Block Limits
   - Maximum gas limit per block
   - Maximum gas limit per transaction, default 10000000

2. Contract Limits
   - Maximum gas for single contract call
   - Contract call depth limit
   - Contract code size limit

Note: Specific limit values need to be determined based on the actual configuration of the integration platform. The framework itself does not enforce these limits, but rather they are configured by the platform during implementation.

## Examples

### Basic Interface Calls
```go
func TestBasicGas(ctx core.Context) {
    addr := ctx.Sender()                 // Consumes 10 gas + 1 gas (code line)
    height := ctx.BlockHeight()          // Consumes 10 gas + 1 gas (code line)
    time := ctx.BlockTime()              // Consumes 10 gas + 1 gas (code line)
    contractAddr := ctx.ContractAddress() // Consumes 10 gas + 1 gas (code line)
    balance := ctx.Balance(addr)         // Consumes 50 gas + 1 gas (code line)
}
```
Total consumption: 95 gas (interface calls: 90 gas + code lines: 5 gas)

### Object Operations
```go
func TestObjectGas(ctx core.Context) {
    // Create and get object
    obj := ctx.CreateObject()             // Consumes 50 gas
    objID := obj.ID()                     // Consumes 10 gas
    
    // Set field
    data := map[string]string{"key": "value"}
    obj.Set("data", data)                 // Consumes 1000 gas + data size * 100 gas
    
    // Read field
    var result map[string]string
    obj.Get("data", &result)              // Consumes 100 gas + result size gas
    
    // Delete object
    ctx.DeleteObject(objID)               // Consumes 500 gas
}
```

### Contract Calls
```go
func TestCallGas(ctx core.Context) {
    targetContract := core.Address{0x01} // Target contract address
    
    // Assume current gas balance is 50000
    
    // Call other contract
    // Reserve 10000 gas as basic call fee
    // Allocate remaining 40000 gas to called contract
    result, _ := ctx.Call(targetContract, "TestFunc", 123, "abc")
    
    // If called contract actually consumed 25000 gas
    // Then total consumption is 10000 (basic fee) + 25000 (actual consumption) = 35000 gas
    
    // Log event
    ctx.Log("FunctionCalled", "contract", targetContract, "result", result) // Consumes 100 gas + data length
}
```

## Best Practices
1. Optimize Code Structure
   - Reduce unnecessary function calls, especially high gas consumption operations like Call()
   - Combine similar storage operations, avoid repeated Set() operations
   - Cache retrieved data, avoid repeated Get() calls

2. Optimize Storage Usage
   - Minimize storage of large data, break data into smaller chunks
   - Use batch operations instead of single operations in loops
   - Use caching appropriately to reduce storage access

3. Gas Optimization Techniques
   - Avoid high gas consumption operations in loops
   - Validate conditions at function start, return early to reduce unnecessary execution
   - Organize code efficiently, reduce number of code lines
   - Prefer simple data types, reduce serialization/deserialization overhead 