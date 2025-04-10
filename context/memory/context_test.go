package memory

import (
	"testing"

	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestContext() *defaultBlockchainContext {
	ctx := NewBlockchainContext(nil).(*defaultBlockchainContext)
	ctx.balances = make(map[types.Address]uint64)
	return ctx
}

func TestBlockContext(t *testing.T) {
	ctx := setupTestContext()

	// Test initial state
	assert.Equal(t, uint64(0), ctx.BlockHeight())
	assert.Equal(t, int64(0), ctx.BlockTime())

	// Test setting block context
	ctx.WithBlock(100, 1234567890)
	assert.Equal(t, uint64(100), ctx.BlockHeight())
	assert.Equal(t, int64(1234567890), ctx.BlockTime())
}

func TestTransactionContext(t *testing.T) {
	ctx := setupTestContext()

	// Set up test data
	sender := core.AddressFromString("0xsender")
	contract := core.AddressFromString("0xcontract")
	txHash := core.HashFromString("0xtx")

	// Test setting transaction context
	ctx.SetTransactionInfo(txHash, sender, contract, 1000)
	ctx.WithTransaction(txHash)

	// Verify transaction context
	assert.Equal(t, sender, ctx.Sender())
	assert.Equal(t, contract, ctx.ContractAddress())
	assert.Equal(t, txHash, ctx.TransactionHash())
}

func TestBalanceTransfer(t *testing.T) {
	ctx := setupTestContext()

	addr1 := core.AddressFromString("0x1111")
	addr2 := core.AddressFromString("0x2222")

	// Initialize balance
	ctx.balances[addr1] = 1000

	// Test balance query
	assert.Equal(t, uint64(1000), ctx.Balance(addr1))
	assert.Equal(t, uint64(0), ctx.Balance(addr2))

	// Test transfer
	err := ctx.Transfer(core.ZeroAddress, addr1, addr2, 500)
	require.NoError(t, err)

	// Verify transfer result
	assert.Equal(t, uint64(500), ctx.Balance(addr1))
	assert.Equal(t, uint64(500), ctx.Balance(addr2))

	// Test insufficient balance
	err = ctx.Transfer(core.ZeroAddress, addr1, addr2, 1000)
	assert.Error(t, err)
}

func TestObjectOperations(t *testing.T) {
	ctx := setupTestContext()

	// Set up test environment
	contract := core.AddressFromString("0xcontract")
	sender := core.AddressFromString("0xsender")
	txHash := core.HashFromString("0xtx")

	// Set transaction context
	ctx.SetTransactionInfo(txHash, sender, contract, 1000)
	ctx.WithTransaction(txHash)

	// Test object creation
	obj, err := ctx.CreateObject(contract)
	require.NoError(t, err)
	assert.NotNil(t, obj)

	// Test setting field
	err = obj.Set(contract, sender, "name", []byte("test"))
	require.NoError(t, err)

	// Test getting field
	value, err := obj.Get(contract, "name")
	require.NoError(t, err)
	assert.Equal(t, []byte("test"), value)

	// Test getting object by ID
	obj2, err := ctx.GetObject(contract, obj.ID())
	require.NoError(t, err)
	assert.Equal(t, obj.ID(), obj2.ID())
	assert.Equal(t, obj.Owner(), obj2.Owner())

	// Test getting object by owner
	obj3, err := ctx.GetObjectWithOwner(contract, sender)
	require.NoError(t, err)
	assert.Equal(t, obj.ID(), obj3.ID())

	// Test deleting object
	err = ctx.DeleteObject(contract, obj.ID())
	require.NoError(t, err)

	// Verify object is deleted
	_, err = ctx.GetObject(contract, obj.ID())
	assert.Error(t, err)
}

func TestGasOperations(t *testing.T) {
	ctx := setupTestContext()

	// Test initial gas state
	assert.Equal(t, int64(0), ctx.GetGas())

	// Test setting gas limit
	ctx.SetGasLimit(1000)
	assert.Equal(t, int64(1000), ctx.gasLimit)
}

func TestObjectOwnership(t *testing.T) {
	ctx := setupTestContext()

	// Set up test environment
	contract := core.AddressFromString("0xcontract")
	sender := core.AddressFromString("0xsender")
	newOwner := core.AddressFromString("0xnewowner")
	txHash := core.HashFromString("0xtx")

	// Set transaction context
	ctx.WithTransaction(txHash)

	// Create object
	obj, err := ctx.CreateObject(contract)
	require.NoError(t, err)

	// Test initial ownership
	assert.Equal(t, sender, obj.Owner())

	// Test setting new owner
	err = obj.SetOwner(contract, sender, newOwner)
	require.NoError(t, err)
	assert.Equal(t, newOwner, obj.Owner())

	// Test unauthorized ownership change
	unauthorized := core.AddressFromString("0xunauthorized")
	err = obj.SetOwner(contract, unauthorized, sender)
	assert.Error(t, err)
}
