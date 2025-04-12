# Custom Contracts Example

This directory contains examples of custom contracts using a simplified interface.

## Directory Structure

```
custom_contracts/
├── core/               # Contract interface definitions
│   └── contract.go     # Core contract interface
└── contract/           # Example contracts
    └── counter.go      # Counter contract example
```

## Simplified Interface

The contract interface provides a set of package-level functions for basic contract operations:

```go
// Storage operations
Get(key string, value interface{}) error
Set(key string, value interface{}) error
Delete(key string) error

// Utility functions
Log(event string, args ...interface{})
Sender() string
ContractAddress() string
```

## Example: Counter Contract

A simple counter contract that demonstrates the use of the interface:

```go
// Initialize the counter
Initialize(initialValue int64) error

// Increment the counter
Increment() error

// Decrement the counter
Decrement() error

// Get current value
GetValue() (int64, error)

// Reset counter to 0
Reset() error
```

## Notes

- The interface is designed to be simple and focused on essential operations
- Storage operations use string keys and interface{} values for flexibility
- Logging and address functions are provided for convenience 