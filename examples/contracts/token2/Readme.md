# Token2 Contract

Token2 is a WASM-based simple token contract implementation that uses a more efficient state storage method compared to Token1, storing all states in a single Object.

## Implementation Features

1. **Single Object Storage**
   - Uses default Object to store all states
   - No separate balance Objects for individual users
   - Distinguishes user data using field prefixes

2. **State Storage Structure**
   - Basic information stored directly in default Object
   - User balances stored using `balance_` + user address as key
   - Contract owner information stored in `owner` field

3. **Security**
   - Uses custom `owner` field for contract ownership management
   - Maintains Object's native owner as the contract itself
   - Strict permission control (mint/burn limited to owner)

## Main Functions

### 1. Initialize Contract
```go
func InitializeToken(ctx core.Context, name string, symbol string, decimals uint8, totalSupply uint64) core.ObjectID
```
- Initializes token basic information
- Sets contract owner
- Allocates initial supply to creator

### 2. Query Functions
```go
func GetTokenInfo(ctx core.Context) (string, string, uint8, uint64)
func GetOwner(ctx core.Context) core.Address
func BalanceOf(ctx core.Context, owner core.Address) uint64
```
- Get token basic information
- Query contract owner
- Query account balance

### 3. Transfer Function
```go
func Transfer(ctx core.Context, to core.Address, amount uint64) bool
```
- Updates sender and recipient balances in the same Object
- Automatically handles first-time token reception
- Includes balance checks and error handling

### 4. Minting Function
```go
func Mint(ctx core.Context, to core.Address, amount uint64) bool
```
- Only callable by contract owner
- Increases total supply
- Mints new tokens to specified address

### 5. Burning Function
```go
func Burn(ctx core.Context, amount uint64) bool
```
- Only callable by contract owner
- Decreases total supply
- Burns tokens from owner's account

## Key Differences from Token1

1. **Storage Method**
   - Token1: Uses multiple Objects for state storage
   - Token2: Uses a single Object for all states

2. **Ownership Management**
   - Token1: Relies on Object's native owner
   - Token2: Uses custom owner field

3. **Performance Characteristics**
   - Token1: Supports parallel transfers (transfers between different users can execute in parallel)
   - Token2: All operations must execute serially (due to shared Object)

4. **Functional Extensions**
   - Token1: Cannot implement approve functionality (users cannot operate others' Objects)
   - Token2: Can implement approve functionality (all states in single Object)

5. **State Access**
   - Token1: Requires accessing multiple Objects
   - Token2: Only needs to access single Object

## Usage Examples

```go
// Initialize token
InitializeToken(ctx, "MyToken", "MTK", 18, 1000000)

// Query token information
name, symbol, decimals, totalSupply := GetTokenInfo(ctx)

// Query balance
balance := BalanceOf(ctx, userAddress)

// Transfer tokens
Transfer(ctx, recipientAddress, 100)

// Mint new tokens (owner only)
Mint(ctx, recipientAddress, 1000)

// Burn tokens (owner only)
Burn(ctx, 500)
```

## Important Notes

1. Contract owner is managed through `owner` field, not Object's native owner
2. Balance records are automatically created for first-time token recipients
3. All operations include proper error handling and state rollback mechanisms
4. Contract events record all important operations
5. All operations must execute serially, which may impact performance
6. Supports approve functionality, suitable for authorization scenarios 