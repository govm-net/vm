# Parameter Encoding and Decoding in Smart Contracts

The VM supports parameter encoding and decoding for smart contract interactions. This allows complex types like Object and Address to be properly serialized and deserialized when passing between contracts or external callers.

## Encoder and Decoder Interfaces

Two interfaces are defined for types that can be encoded to and decoded from byte arrays:

```go
// Encoder represents an object that can encode itself to bytes.
type Encoder interface {
    // Encode encodes the object to bytes
    Encode() ([]byte, error)
}

// Decoder represents an object that can decode itself from bytes.
type Decoder interface {
    // Decode decodes the object from bytes
    Decode(data []byte) error
}
```

## Built-in Types with Encoding Support

The core package provides implementations of these interfaces for common types:

### Address

```go
// Encode encodes the address to bytes.
func (a Address) Encode() ([]byte, error) {
    return a[:], nil
}

// Decode decodes an address from bytes.
func (a *Address) Decode(data []byte) error {
    if len(data) != 20 {
        return ErrInvalidArgument
    }
    copy(a[:], data)
    return nil
}
```

### ObjectID

```go
// Encode encodes the ObjectID to bytes.
func (id ObjectID) Encode() ([]byte, error) {
    return id[:], nil
}

// Decode decodes an ObjectID from bytes.
func (id *ObjectID) Decode(data []byte) error {
    if len(data) != 32 {
        return ErrInvalidArgument
    }
    copy(id[:], data)
    return nil
}
```

### Object Interface

The `Object` interface also requires implementations to support encoding and decoding:

```go
type Object interface {
    // ...other methods...
    
    // Encode encodes the object to bytes
    Encode() ([]byte, error)
    
    // Decode decodes the object from bytes
    Decode(data []byte) error
}
```

## Automatic Parameter Conversion

When executing smart contract functions, the VM automatically handles encoding and decoding:

1. **Input Parameters**: If a function parameter implements the `Decoder` interface, the VM will automatically call its `Decode` method to deserialize the input byte array.

2. **Return Values**: If a function return value implements the `Encoder` interface, the VM will automatically call its `Encode` method to serialize the result.

### Example

```go
// In a contract method
func (c *MyContract) TransferOwnership(ctx core.Context, objectID core.ObjectID, newOwner core.Address) (bool, error) {
    // objectID and newOwner are automatically decoded from byte arrays
    
    // ...implementation...
    
    return true, nil // Boolean value will be encoded to bytes
}
```

When this method is called through `engine.Execute`, the parameters and return values are automatically converted:

```go
// External call (parameters are sent as byte arrays)
objectIDBytes := []byte{...} // Serialized ObjectID
newOwnerBytes := []byte{...} // Serialized Address

result, err := engine.Execute(contractAddr, "TransferOwnership", objectIDBytes, newOwnerBytes)
// result is the encoded boolean (true)
```

## Creating Custom Encodable Types

You can create your own types that support encoding and decoding:

```go
type MyCustomType struct {
    Field1 string
    Field2 uint64
}

func (t MyCustomType) Encode() ([]byte, error) {
    // Implement serialization logic
    // ...
    return serialized, nil
}

func (t *MyCustomType) Decode(data []byte) error {
    // Implement deserialization logic
    // ...
    return nil
}
```

## Best Practices

1. **Use Standard Formats**: Consider using standard serialization formats like Protocol Buffers, JSON, or a custom binary format for complex types.

2. **Version Your Encodings**: Include version information in your encoding to allow for future changes.

3. **Error Handling**: Provide clear error messages when decoding fails.

4. **Size Constraints**: Be aware of the size of your encoded data, especially for on-chain storage.

5. **Performance**: Optimize encoding and decoding for frequently used types to reduce gas costs. 