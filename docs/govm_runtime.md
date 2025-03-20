# GoVM Runtime Documentation

## Overview

GoVM (Go Virtual Machine) is a blockchain-based virtual machine that allows for the deployment and execution of smart contracts written in Go. Unlike other blockchain VMs like Ethereum's EVM, GoVM leverages the Go programming language's native capabilities, compilation, and execution model to run smart contracts.

The runtime component of GoVM is responsible for:
1. Smart contract compilation, deployment, and execution
2. Resource management (energy consumption)
3. State persistence through database interactions
4. Cross-chain communication
5. Security and isolation

## Core Components

### 1. Runtime Structure

The main runtime structure `TRuntime` includes:

```go
type TRuntime struct {
    Chain    uint64      // Chain identifier
    Flag     []byte      // Runtime flags
    addrType string      // Database address type
    address  string      // Database address
    mode     string      // Execution mode
    db       *client.Client // Database client
}
```

### 2. App Creation and Deployment

Apps in GoVM are deployed as Go packages that are compiled to native executables. The process involves:

1. **Code Submission**: Smart contract code is submitted along with metadata including dependencies.
2. **Code Verification**: The submitted code is verified to ensure it doesn't contain disallowed imports or operations.
3. **Compilation**: The code is compiled to a native Go executable.
4. **Line Counting**: The code lines are counted and annotated for energy consumption tracking.
5. **Deployment**: The compiled app is stored and made available for execution.

The `NewApp` function in `maker.go` handles this process:

```go
func NewApp(chain uint64, name []byte, code []byte) {
    // Process app metadata
    // Create source files and directories
    // Compile and validate code
    // Add instrumentation for energy tracking
    // Create executable if needed
}
```

### 3. App Execution

When an app is executed, GoVM:

1. Prepares the execution environment
2. Sets up resource limits
3. Loads the app as a Go package
4. Calls the entry point function
5. Monitors and accounts for resource usage

The execution happens through the native Go runtime with added instrumentation for resource tracking:

```go
func (r *TRuntime) RunApp(appName, user, data []byte, energy, cost uint64) {
    // Find app executable
    // Set up execution environment
    // Execute app with provided parameters
    // Monitor and validate resource usage
    // Return results
}
```

### 4. Energy Management

GoVM uses a concept of "energy" to limit resource usage by smart contracts:

```go
func ConsumeEnergy(n uint64) uint64 {
    if n == 0 {
        n = 10
    }
    mu.Lock()
    defer mu.Unlock()
    used += used / 100000
    used += n
    if used > energy {
        log.Panicf("energy.hope:%d,have:%d\n", used, energy)
    }
    return energy - used
}
```

Energy is consumed for:
- CPU operations
- Memory usage
- Storage operations
- External calls

### 5. State Persistence

GoVM provides database operations for contracts to persist state:

```go
func (r *TRuntime) DbSet(owner interface{}, key, value []byte, life uint64) {
    // Store key-value pair in database with expiration
}

func (r *TRuntime) DbGet(owner interface{}, key []byte) ([]byte, uint64) {
    // Retrieve value from database along with remaining life
}
```

### 6. Cross-Chain Communication

GoVM supports cross-chain communication through a log mechanism:

```go
func (r *TRuntime) LogWrite(owner interface{}, key, value []byte, life uint64) {
    // Write to log that can be read by other chains
}

func (r *TRuntime) LogRead(owner interface{}, chain uint64, key []byte) ([]byte, uint64) {
    // Read from log of another chain
}
```

## Smart Contract Interface

Smart contracts in GoVM interact with the VM through a standard runtime interface:

```go
type IRuntime interface {
    // Hash calculation
    GetHash(in []byte) []byte
    
    // Data encoding/decoding
    Encode(typ uint8, in interface{}) []byte
    Decode(typ uint8, in []byte, out interface{}) int
    JSONEncode(in interface{}) []byte
    JSONDecode(in []byte, out interface{})
    
    // Cryptographic operations
    Recover(address, sign, msg []byte) bool
    
    // Database operations
    DbSet(owner interface{}, key, value []byte, life uint64)
    DbGet(owner interface{}, key []byte) ([]byte, uint64)
    DbGetLife(owner interface{}, key []byte) uint64
    
    // Log operations for cross-chain communication
    LogWrite(owner interface{}, key, value []byte, life uint64)
    LogRead(owner interface{}, chain uint64, key []byte) ([]byte, uint64)
    LogReadLife(owner interface{}, key []byte) uint64
    
    // App management
    GetAppName(in interface{}) []byte
    NewApp(name []byte, code []byte)
    RunApp(name, user, data []byte, energy, cost uint64)
    
    // Resource management
    Event(user interface{}, event string, param ...[]byte)
    ConsumeEnergy(energy uint64)
}
```

## Execution Flow

1. **Deployment**:
   - Client submits contract code
   - GoVM validates, compiles, and instruments the code
   - Contract is stored with metadata

2. **Invocation**:
   - Client submits transaction to execute contract
   - GoVM loads contract and parameters
   - Runtime sets up execution environment
   - Contract executes with resource limitations
   - Results and state changes are recorded

## Security Considerations

1. **Code Validation**: GoVM validates that submitted code doesn't contain prohibited imports or operations.

2. **Resource Isolation**: Each contract execution is isolated with strict resource limits.

3. **Energy Limits**: The energy system prevents infinite loops and excessive resource consumption.

4. **Determinism**: GoVM ensures deterministic execution across different nodes for consensus.

## Implementation Details

### App Flags

Apps in GoVM can have different capabilities indicated by flags:

```go
const (
    // AppFlagRun the app can be called
    AppFlagRun = uint8(1 << iota)
    // AppFlagImport the app code can be included
    AppFlagImport
    // AppFlagPlublc App funds address uses the public address
    AppFlagPlublc
    // AppFlagGzipCompress gzip compress
    AppFlagGzipCompress
    // AppFlagEnd end of flag
    AppFlagEnd
)
```

### Templates

GoVM uses Go templates to generate execution wrappers:

1. **run.tmpl**: Creates a runner function for the app
2. **main.tmpl**: Creates the main executable for app invocation

### Environment Management

GoVM manages the execution environment for apps, including:
- Working directory
- Environment variables
- File system access
- Execution timeout

## Conclusion

GoVM provides a unique approach to blockchain virtual machines by leveraging the Go language's native compilation and execution model. This offers several advantages:

1. **Performance**: Native execution is more efficient than interpretation
2. **Familiarity**: Developers can use standard Go language features
3. **Tooling**: Access to Go's rich ecosystem of tools and libraries
4. **Type Safety**: Go's strong type system helps prevent common errors

However, this approach also introduces challenges in ensuring deterministic execution and security isolation that the GoVM runtime addresses through careful resource tracking and execution management. 