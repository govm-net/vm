// A simple token contract example based on WASM wrapper
package token

import (
	"github.com/govm-net/vm/core"
)

// Constants - Field names in objects
const (
	// Fields in default Object
	TokenNameKey        = "name"         // Token name
	TokenSymbolKey      = "symbol"       // Token symbol
	TokenDecimalsKey    = "decimals"     // Token decimals
	TokenTotalSupplyKey = "total_supply" // Total supply
	TokenAmountKey      = "amount"       // Balance amount
)

// Initialize token contract
func InitializeToken(ctx core.Context, name string, symbol string, decimals uint8, totalSupply uint64) core.ObjectID {
	// Get default Object (empty ObjectID)
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	core.Assert(err)

	// Store token basic information
	core.Assert(defaultObj.Set(TokenNameKey, name))
	core.Assert(defaultObj.Set(TokenSymbolKey, symbol))
	core.Assert(defaultObj.Set(TokenDecimalsKey, decimals))
	core.Assert(defaultObj.Set(TokenTotalSupplyKey, totalSupply))

	defaultObj.SetOwner(ctx.Sender())

	obj := ctx.CreateObject()
	core.Assert(obj.Set(TokenAmountKey, totalSupply))
	obj.SetOwner(ctx.Sender())

	// Log initialization event
	ctx.Log("initialize",
		"id", defaultObj.ID(),
		"name", name,
		"symbol", symbol,
		"decimals", decimals,
		"total_supply", totalSupply,
		"owner", ctx.Sender())

	return defaultObj.ID()
}

// Get token information
func GetTokenInfo(ctx core.Context) (string, string, uint8, uint64) {
	// Get default Object
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	core.Assert(err)

	// Read token basic information
	var name string
	core.Assert(defaultObj.Get(TokenNameKey, &name))

	var symbol string
	core.Assert(defaultObj.Get(TokenSymbolKey, &symbol))

	var decimals uint8
	core.Assert(defaultObj.Get(TokenDecimalsKey, &decimals))

	var totalSupply uint64
	core.Assert(defaultObj.Get(TokenTotalSupplyKey, &totalSupply))

	return name, symbol, decimals, totalSupply
}

// Get contract owner
func GetOwner(ctx core.Context) core.Address {
	// Get default Object
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	core.Assert(err)

	return defaultObj.Owner()
}

// Get account balance
func BalanceOf(ctx core.Context, owner core.Address) uint64 {
	obj, err := ctx.GetObjectWithOwner(owner)
	core.Assert(err)

	var balance uint64
	core.Assert(obj.Get(TokenAmountKey, &balance))

	return balance
}

// Transfer tokens to another address
func Transfer(ctx core.Context, to core.Address, amount uint64) bool {
	from := ctx.Sender()

	// Check amount validity
	core.Assert(amount > 0)
	obj, err := ctx.GetObjectWithOwner(from)
	core.Assert(err)

	var fromBalance uint64
	core.Assert(obj.Get(TokenAmountKey, &fromBalance))

	// Check if balance is sufficient
	core.Assert(fromBalance >= amount)

	core.Assert(obj.Set(TokenAmountKey, fromBalance-amount))

	toObj := ctx.CreateObject()
	core.Assert(toObj.Set(TokenAmountKey, amount))
	toObj.SetOwner(to)

	// Log transfer event
	ctx.Log("transfer",
		"from", from,
		"to", to,
		"amount", amount)

	return true
}

// Collect balances from multiple objects into one
func Collect(ctx core.Context, ids []core.ObjectID) bool {
	sender := ctx.Sender()

	// Check if ids is not empty
	core.Assert(len(ids) > 1)
	var amount uint64
	// Migrate balances from other objects to the first object
	for i := 1; i < len(ids); i++ {
		id := ids[i]
		obj, err := ctx.GetObject(id)
		core.Assert(err)
		core.Assert(obj.Owner() == sender)
		var balance uint64
		core.Assert(obj.Get(TokenAmountKey, &balance))
		amount += balance
		ctx.DeleteObject(id)
	}
	obj, err := ctx.GetObject(ids[0])
	core.Assert(err)
	core.Assert(obj.Owner() == sender)
	core.Assert(obj.Set(TokenAmountKey, amount))

	return true
}

// Mint new tokens (owner only)
func Mint(ctx core.Context, to core.Address, amount uint64) bool {
	sender := ctx.Sender()

	// Check amount validity
	core.Assert(amount > 0)

	// Get default Object
	obj, err := ctx.GetObject(core.ZeroObjectID)
	core.Assert(err)
	core.Assert(obj.Owner() == sender)

	// Get current total supply
	var totalSupply uint64
	core.Assert(obj.Get(TokenTotalSupplyKey, &totalSupply))

	totalSupply += amount
	core.Assert(obj.Set(TokenTotalSupplyKey, totalSupply))

	toObj := ctx.CreateObject()
	core.Assert(toObj.Set(TokenAmountKey, amount))
	toObj.SetOwner(to)

	// Log mint event
	ctx.Log("mint",
		"to", to,
		"amount", amount,
		"total_supply", totalSupply)

	return true
}

// Burn tokens
func Burn(ctx core.Context, id core.ObjectID, amount uint64) bool {
	sender := ctx.Sender()

	core.Assert(amount > 0)

	// Get default Object
	obj, err := ctx.GetObject(core.ZeroObjectID)
	core.Assert(err)
	core.Assert(obj.Owner() == sender)

	// Get current total supply
	var totalSupply uint64
	core.Assert(obj.Get(TokenTotalSupplyKey, &totalSupply))

	totalSupply -= amount
	core.Assert(obj.Set(TokenTotalSupplyKey, totalSupply))

	var userObj core.Object
	if id == core.ZeroObjectID {
		userObj, err = ctx.GetObjectWithOwner(sender)
		core.Assert(err)
	} else {
		userObj, err = ctx.GetObject(id)
		core.Assert(err)
	}

	var userBalance uint64
	core.Assert(userObj.Get(TokenAmountKey, &userBalance))
	core.Assert(userBalance >= amount)

	userBalance -= amount
	if userBalance == 0 {
		ctx.DeleteObject(userObj.ID())
	} else {
		core.Assert(userObj.Set(TokenAmountKey, userBalance))
	}

	// Log burn event
	ctx.Log("burn",
		"from", sender,
		"amount", amount,
		"total_supply", totalSupply)

	return true
}
