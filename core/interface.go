// Package core defines the core interfaces required for smart contracts to interact with the VM system
// Contract developers only need to understand and use the interfaces in this file to write smart contracts
package core

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/govm-net/vm/types"
)

type Address = types.Address

type ObjectID = types.ObjectID

type Hash = types.Hash

type Object = types.Object

var ZeroAddress = Address{}
var ZeroObjectID = ObjectID{}
var ZeroHash = Hash{}

func IDFromString(str string) ObjectID {
	str = strings.TrimPrefix(str, "0x")
	id, err := hex.DecodeString(str)
	if err != nil {
		return ZeroObjectID
	}
	var out ObjectID
	copy(out[:], id)
	return out
}

func AddressFromString(str string) Address {
	str = strings.TrimPrefix(str, "0x")
	addr, err := hex.DecodeString(str)
	if err != nil {
		return ZeroAddress
	}
	var out Address
	copy(out[:], addr)
	return out
}

func HashFromString(str string) Hash {
	str = strings.TrimPrefix(str, "0x")
	h, err := hex.DecodeString(str)
	if err != nil {
		return ZeroHash
	}
	var out Hash
	copy(out[:], h)
	return out
}

func GetHash(data []byte) Hash {
	return Hash(sha256.Sum256(data))
}

func Assert(condition any) {
	switch v := condition.(type) {
	case bool:
		if !v {
			panic("assertion failed")
		}
	case error:
		if v != nil {
			panic(v)
		}
	}
}

func Error(msg string) error {
	return errors.New(msg)
}

var ctx types.Context

func SetContext(c types.Context) {
	if ctx != nil {
		panic("context already set")
	}
	ctx = c
}

func BlockHeight() uint64 {
	return ctx.BlockHeight()
}

func BlockTime() int64 {
	return ctx.BlockTime()
}

func ContractAddress() Address {
	return ctx.ContractAddress()
}

func Sender() Address {
	return ctx.Sender()
}

func Balance(addr Address) uint64 {
	return ctx.Balance(addr)
}

// Object storage related - Basic state operations use panic instead of returning error
func CreateObject() Object {
	return ctx.CreateObject()
}

func GetObject(id ObjectID) (Object, error) {
	return ctx.GetObject(id)
}

func GetObjectWithOwner(owner Address) (Object, error) {
	return ctx.GetObjectWithOwner(owner)
}

func DeleteObject(id ObjectID) {
	ctx.DeleteObject(id)
}

// Cross-contract calls
func Call(contract Address, function string, args ...any) ([]byte, error) {
	return ctx.Call(contract, function, args...)
}

// Logging and events
func Log(eventName string, keyValues ...interface{}) {
	ctx.Log(eventName, keyValues...)
}
