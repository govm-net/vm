// Package vm provides the implementation of the virtual machine.
package vm

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/govm-net/vm/core"
)

// DBStateObject is an implementation of the core.Object interface that
// persists data in a database instead of memory.
type DBStateObject struct {
	id           core.ObjectID
	contractAddr core.Address
	owner        core.Address
	objType      string
	db           Database
	keyGenerator KeyGenerator
	fieldCache   map[string]interface{} // Optional cache for field values
	cacheLock    sync.RWMutex
	cacheEnabled bool
}

// NewDBStateObject creates a new database-backed state object.
func NewDBStateObject(
	id core.ObjectID,
	contractAddr core.Address,
	owner core.Address,
	objType string,
	db Database,
	keyGenerator KeyGenerator,
	enableCache bool,
) *DBStateObject {
	obj := &DBStateObject{
		id:           id,
		contractAddr: contractAddr,
		owner:        owner,
		objType:      objType,
		db:           db,
		keyGenerator: keyGenerator,
		fieldCache:   make(map[string]interface{}),
		cacheEnabled: enableCache,
	}

	// Save the object metadata to the database
	obj.saveMetadata()

	// Create the owner index
	obj.updateOwnerIndex()

	return obj
}

// ID returns the unique identifier of the object.
func (o *DBStateObject) ID() core.ObjectID {
	return o.id
}

// Type returns the type of the object.
func (o *DBStateObject) Type() string {
	return o.objType
}

// Owner returns the owner address of the object.
func (o *DBStateObject) Owner() core.Address {
	return o.owner
}

// SetOwner changes the owner of the object.
func (o *DBStateObject) SetOwner(newOwner core.Address) error {
	// Remove the old owner index
	oldOwnerKey := o.keyGenerator.OwnerIndex(o.contractAddr, o.owner, o.id)
	if err := o.db.Delete(oldOwnerKey); err != nil {
		return err
	}

	// Update the owner
	o.owner = newOwner

	// Save metadata with the new owner
	if err := o.saveMetadata(); err != nil {
		return err
	}

	// Create new owner index
	return o.updateOwnerIndex()
}

// Get retrieves a value from the object's fields.
func (o *DBStateObject) Get(field string) (interface{}, error) {
	// Check cache first if enabled
	if o.cacheEnabled {
		o.cacheLock.RLock()
		if value, exists := o.fieldCache[field]; exists {
			o.cacheLock.RUnlock()
			return value, nil
		}
		o.cacheLock.RUnlock()
	}

	// Generate the field key
	fieldKey := o.keyGenerator.FieldKey(o.contractAddr, o.id, field)

	// Get the value from the database
	data, err := o.db.Get(fieldKey)
	if err != nil {
		return nil, err
	}

	if data == nil {
		return nil, fmt.Errorf("field %s not found", field)
	}

	// Deserialize the value
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}

	// Update cache if enabled
	if o.cacheEnabled {
		o.cacheLock.Lock()
		o.fieldCache[field] = value
		o.cacheLock.Unlock()
	}

	return value, nil
}

// Set stores a value in the object's fields.
func (o *DBStateObject) Set(field string, value interface{}) error {
	// Serialize the value
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	// Generate the field key
	fieldKey := o.keyGenerator.FieldKey(o.contractAddr, o.id, field)

	// Store the value in the database
	if err := o.db.Put(fieldKey, data); err != nil {
		return err
	}

	// Update cache if enabled
	if o.cacheEnabled {
		o.cacheLock.Lock()
		o.fieldCache[field] = value
		o.cacheLock.Unlock()
	}

	return nil
}

// Delete removes a field from the object.
func (o *DBStateObject) Delete(field string) error {
	// Generate the field key
	fieldKey := o.keyGenerator.FieldKey(o.contractAddr, o.id, field)

	// Delete the field from the database
	if err := o.db.Delete(fieldKey); err != nil {
		return err
	}

	// Update cache if enabled
	if o.cacheEnabled {
		o.cacheLock.Lock()
		delete(o.fieldCache, field)
		o.cacheLock.Unlock()
	}

	return nil
}

// Encode encodes the object to bytes.
func (o *DBStateObject) Encode() ([]byte, error) {
	// Create a structure to represent the object metadata
	metadata := struct {
		ID    core.ObjectID `json:"id"`
		Type  string        `json:"type"`
		Owner core.Address  `json:"owner"`
	}{
		ID:    o.id,
		Type:  o.objType,
		Owner: o.owner,
	}

	return json.Marshal(metadata)
}

// Decode decodes the object from bytes.
func (o *DBStateObject) Decode(data []byte) error {
	// Parse the metadata
	metadata := struct {
		ID    core.ObjectID `json:"id"`
		Type  string        `json:"type"`
		Owner core.Address  `json:"owner"`
	}{}

	if err := json.Unmarshal(data, &metadata); err != nil {
		return err
	}

	// Update object fields
	o.id = metadata.ID
	o.objType = metadata.Type
	o.owner = metadata.Owner

	// Update owner index if the owner changed
	return o.updateOwnerIndex()
}

// saveMetadata saves the object metadata to the database.
func (o *DBStateObject) saveMetadata() error {
	data, err := o.Encode()
	if err != nil {
		return err
	}

	objectKey := o.keyGenerator.ObjectKey(o.contractAddr, o.id)
	return o.db.Put(objectKey, data)
}

// updateOwnerIndex creates or updates the owner index in the database.
func (o *DBStateObject) updateOwnerIndex() error {
	ownerKey := o.keyGenerator.OwnerIndex(o.contractAddr, o.owner, o.id)

	// We just need to store an empty value as the key itself is the index
	return o.db.Put(ownerKey, []byte{})
}

// ClearCache clears the field cache.
func (o *DBStateObject) ClearCache() {
	if o.cacheEnabled {
		o.cacheLock.Lock()
		o.fieldCache = make(map[string]interface{})
		o.cacheLock.Unlock()
	}
}

// LoadAllFields loads all fields into the cache.
func (o *DBStateObject) LoadAllFields() error {
	if !o.cacheEnabled {
		return nil
	}

	// Clear the current cache
	o.ClearCache()

	// Get the prefix for all fields of this object
	prefix := o.keyGenerator.FieldKey(o.contractAddr, o.id, "")
	start, end := PrefixRange(prefix)

	// Iterate over all fields
	iter, err := o.db.Iterator(start, end)
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		key := iter.Key()
		value := iter.Value()

		// Extract the field name from the key
		fieldName := string(key[len(prefix):])

		// Deserialize the value
		var fieldValue interface{}
		if err := json.Unmarshal(value, &fieldValue); err != nil {
			return err
		}

		// Add to cache
		o.cacheLock.Lock()
		o.fieldCache[fieldName] = fieldValue
		o.cacheLock.Unlock()
	}

	return iter.Error()
}
