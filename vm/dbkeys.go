// Package vm provides the implementation of the virtual machine.
package vm

import (
	"github.com/govm-net/vm/core"
)

// DefaultKeyGenerator implements the core.KeyGenerator interface.
type DefaultKeyGenerator struct{}

// NewDefaultKeyGenerator creates a new instance of DefaultKeyGenerator.
func NewDefaultKeyGenerator() *DefaultKeyGenerator {
	return &DefaultKeyGenerator{}
}

// ObjectKey generates a key for storing an object's metadata.
// Format: 'o' + contract_address + object_id
func (kg *DefaultKeyGenerator) ObjectKey(contractAddr core.Address, objectID core.ObjectID) []byte {
	key := append([]byte{'o'}, contractAddr[:]...)
	return append(key, objectID[:]...)
}

// FieldKey generates a key for storing a field value.
// Format: 'f' + contract_address + object_id + field_name
func (kg *DefaultKeyGenerator) FieldKey(contractAddr core.Address, objectID core.ObjectID, field string) []byte {
	key := append([]byte{'f'}, contractAddr[:]...)
	key = append(key, objectID[:]...)
	return append(key, []byte(field)...)
}

// OwnerIndex generates a key for the owner index.
// Format: 'i' + contract_address + owner_address + object_id
func (kg *DefaultKeyGenerator) OwnerIndex(contractAddr core.Address, owner core.Address, objectID core.ObjectID) []byte {
	key := append([]byte{'i'}, contractAddr[:]...)
	key = append(key, owner[:]...)
	return append(key, objectID[:]...)
}

// FindByOwner generates a prefix key for finding objects by owner.
// Format: 'i' + contract_address + owner_address
func (kg *DefaultKeyGenerator) FindByOwner(contractAddr core.Address, owner core.Address) []byte {
	key := append([]byte{'i'}, contractAddr[:]...)
	return append(key, owner[:]...)
}

// Helper functions for key manipulation

// ExtractObjectIDFromKey extracts the ObjectID from an object key.
func (kg *DefaultKeyGenerator) ExtractObjectIDFromKey(key []byte) core.ObjectID {
	// Skip the prefix byte and contract address
	idStart := 1 + 20
	if len(key) < idStart+32 {
		return core.ObjectID{}
	}

	var id core.ObjectID
	copy(id[:], key[idStart:idStart+32])
	return id
}

// ExtractOwnerFromKey extracts the owner address from an owner index key.
func (kg *DefaultKeyGenerator) ExtractOwnerFromKey(key []byte) core.Address {
	// Skip the prefix byte and contract address
	ownerStart := 1 + 20
	if len(key) < ownerStart+20 {
		return core.Address{}
	}

	var owner core.Address
	copy(owner[:], key[ownerStart:ownerStart+20])
	return owner
}

// ExtractContractAddressFromKey extracts the contract address from any key.
func (kg *DefaultKeyGenerator) ExtractContractAddressFromKey(key []byte) core.Address {
	// Skip the prefix byte
	if len(key) < 1+20 {
		return core.Address{}
	}

	var addr core.Address
	copy(addr[:], key[1:1+20])
	return addr
}

// PrefixRange returns key range that corresponds to the given prefix.
// It returns start (inclusive) and end (exclusive) keys for iteration.
func PrefixRange(prefix []byte) ([]byte, []byte) {
	if len(prefix) == 0 {
		return nil, nil
	}

	// Make a copy of the prefix to avoid modifying the original
	end := make([]byte, len(prefix))
	copy(end, prefix)

	// Increment the last byte in the prefix to get the end key
	for i := len(end) - 1; i >= 0; i-- {
		if end[i] < 0xff {
			end[i]++
			break
		}

		// If we reach here, the byte was 0xff, so set it to 0 and continue
		// to the next byte
		end[i] = 0

		// If we've reached the beginning and all bytes are 0xff,
		// then we can't increment any further
		if i == 0 {
			// In this case, return a nil end key, which means no upper bound
			return prefix, nil
		}
	}

	return prefix, end
}
