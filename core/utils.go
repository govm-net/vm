package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
)

// Hash calculates the SHA-256 hash of data
func Hash(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// AddressFromString converts a hex string to an Address
func AddressFromString(s string) (Address, error) {
	var addr Address

	// Remove 0x prefix if present
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
	}

	if len(s) != 40 {
		return addr, ErrInvalidArgument
	}

	bytes, err := hex.DecodeString(s)
	if err != nil {
		return addr, err
	}

	copy(addr[:], bytes)
	return addr, nil
}

// ZeroAddress returns an address with all bytes set to zero
func ZeroAddress() Address {
	return Address{}
}

// SafeMarshal marshals data to JSON without HTML escaping
func SafeMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// SafeUnmarshal unmarshals JSON data
func SafeUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// SafeParseInt safely parses a string to an int64
func SafeParseInt(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// SafeParseUint safely parses a string to a uint64
func SafeParseUint(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}

// SafeParseFloat safely parses a string to a float64
func SafeParseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}
