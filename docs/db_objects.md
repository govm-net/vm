# Database-Backed Objects

This document describes the implementation of database-backed objects in the VM, which provide persistent storage for smart contract state.

## Overview

The VM supports two types of state objects:

1. **Memory-Backed Objects** (`StateObject`): These objects store their data in memory and are lost when the application shuts down. They are useful for temporary state or testing.

2. **Database-Backed Objects** (`DBStateObject`): These objects persist their data in a database, allowing for permanent storage of state. They follow the pattern: contractAddress -> ObjectID/Owner -> field -> value.

Both types implement the same `Object` interface, allowing them to be used interchangeably in smart contracts.

## Database Storage Schema

Database-backed objects use a key-value storage schema with the following key patterns:

- **Object Metadata**: `'o' + contract_address + object_id` → Object metadata (ID, type, owner)
- **Field Values**: `'f' + contract_address + object_id + field_name` → Field value
- **Owner Index**: `'i' + contract_address + owner_address + object_id` → Empty value (for lookup)

This schema allows for efficient lookups by:
- Contract address and object ID
- Contract address and owner

## Using Database-Backed Objects

Smart contracts can create and use DB-backed objects using specific context methods:

```go
// Create a new DB-backed object
dbObject, err := ctx.CreateDBObject()
if err != nil {
    return err
}

// Store data in the DB-backed object
err = dbObject.Set("key", "value")
if err != nil {
    return err
}

// Load an existing DB-backed object by ID
loadedObject, err := ctx.LoadDBObject(objectID)
if err != nil {
    return err
}

// Read data from the DB-backed object
value, err := loadedObject.Get("key")
if err != nil {
    return err
}
```

## Database Implementation

The VM provides interfaces for the database implementation:

```go
// Database defines the interface for persistent storage backends
type Database interface {
    Get(key []byte) ([]byte, error)
    Put(key []byte, value []byte) error
    Delete(key []byte) error
    Has(key []byte) (bool, error)
    Iterator(start, end []byte) (Iterator, error)
    Close() error
}

// KeyGenerator creates storage keys
type KeyGenerator interface {
    ObjectKey(contractAddr Address, objectID ObjectID) []byte
    FieldKey(contractAddr Address, objectID ObjectID, field string) []byte
    OwnerIndex(contractAddr Address, owner Address, objectID ObjectID) []byte
    FindByOwner(contractAddr Address, owner Address) []byte
}
```

The VM includes a default implementation of the KeyGenerator interface, but you can provide your own implementation if needed.

## Initializing the Database Provider

Before using database-backed objects, you must set the database provider on the VM engine:

```go
// Create a new database implementation
db := NewMyDatabaseImplementation()

// Create a key generator
keyGen := vm.NewDefaultKeyGenerator()

// Set the database provider on the engine
engine.SetDBProvider(db, keyGen)
```

## Memory Cache

For performance optimization, DBStateObject includes an optional in-memory cache for field values. This can be enabled or disabled when creating the object:

```go
// Enable the cache (default)
object := NewDBStateObject(..., true)

// Clear the cache
object.ClearCache()

// Load all fields into the cache
object.LoadAllFields()
```

## Error Handling

When working with database-backed objects, you may encounter these common errors:

- `DB provider not set`: You must call `SetDBProvider` on the Engine before creating or loading DB objects.
- `core.ErrObjectNotFound`: The requested object does not exist in the database.
- `core.ErrUnauthorized`: The sender is not authorized to modify the object (e.g., when changing the owner).

## Implementation Details

The `DBStateObject` implementation:

1. Stores metadata (ID, type, owner) in a single database entry
2. Stores each field value in a separate database entry
3. Maintains an owner index for looking up objects by owner
4. Optionally caches field values in memory for performance

When an object owner changes, the implementation automatically updates the owner index. 