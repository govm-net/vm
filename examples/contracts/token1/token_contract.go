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
func InitializeToken(name string, symbol string, decimals uint8, totalSupply uint64) core.ObjectID {
	// Get default Object (empty ObjectID)
	defaultObj, err := core.GetObject(core.ZeroObjectID)
	core.Assert(err)

	// Store token basic information
	core.Assert(defaultObj.Set(TokenNameKey, name))
	core.Assert(defaultObj.Set(TokenSymbolKey, symbol))
	core.Assert(defaultObj.Set(TokenDecimalsKey, decimals))
	core.Assert(defaultObj.Set(TokenTotalSupplyKey, totalSupply))

	defaultObj.SetOwner(core.Sender())

	obj := core.CreateObject()
	core.Assert(obj.Set(TokenAmountKey, totalSupply))
	obj.SetOwner(core.Sender())

	// Log initialization event
	core.Log("initialize",
		"id", defaultObj.ID(),
		"name", name,
		"symbol", symbol,
		"decimals", decimals,
		"total_supply", totalSupply,
		"owner", core.Sender())

	return defaultObj.ID()
}

// Get token information
func GetTokenInfo() (string, string, uint8, uint64) {
	// Get default Object
	defaultObj, err := core.GetObject(core.ZeroObjectID)
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
func GetOwner() core.Address {
	// Get default Object
	defaultObj, err := core.GetObject(core.ZeroObjectID)
	core.Assert(err)

	return defaultObj.Owner()
}

// Get account balance
func BalanceOf(owner core.Address) uint64 {
	obj, err := core.GetObjectWithOwner(owner)
	core.Assert(err)

	var balance uint64
	core.Assert(obj.Get(TokenAmountKey, &balance))

	return balance
}

// Transfer tokens to another address
func Transfer(to core.Address, amount uint64) bool {
	from := core.Sender()

	// Check amount validity
	core.Assert(amount > 0)
	obj, err := core.GetObjectWithOwner(from)
	core.Assert(err)

	var fromBalance uint64
	core.Assert(obj.Get(TokenAmountKey, &fromBalance))

	// Check if balance is sufficient
	core.Assert(fromBalance >= amount)

	core.Assert(obj.Set(TokenAmountKey, fromBalance-amount))

	toObj := core.CreateObject()
	core.Assert(toObj.Set(TokenAmountKey, amount))
	toObj.SetOwner(to)

	// Log transfer event
	core.Log("transfer",
		"from", from,
		"to", to,
		"amount", amount)

	return true
}

// Collect balances from multiple objects into one
func Collect(ids []core.ObjectID) bool {
	sender := core.Sender()

	// Check if ids is not empty
	core.Assert(len(ids) > 1)
	var amount uint64
	// Migrate balances from other objects to the first object
	for i := 1; i < len(ids); i++ {
		id := ids[i]
		obj, err := core.GetObject(id)
		core.Assert(err)
		core.Assert(obj.Owner() == sender)
		var balance uint64
		core.Assert(obj.Get(TokenAmountKey, &balance))
		amount += balance
		core.DeleteObject(id)
	}
	obj, err := core.GetObject(ids[0])
	core.Assert(err)
	core.Assert(obj.Owner() == sender)
	core.Assert(obj.Set(TokenAmountKey, amount))

	return true
}

// Mint new tokens (owner only)
func Mint(to core.Address, amount uint64) bool {
	sender := core.Sender()

	// Check amount validity
	core.Assert(amount > 0)

	// Get default Object
	obj, err := core.GetObject(core.ZeroObjectID)
	core.Assert(err)
	core.Assert(obj.Owner() == sender)

	// Get current total supply
	var totalSupply uint64
	core.Assert(obj.Get(TokenTotalSupplyKey, &totalSupply))

	totalSupply += amount
	core.Assert(obj.Set(TokenTotalSupplyKey, totalSupply))

	toObj := core.CreateObject()
	core.Assert(toObj.Set(TokenAmountKey, amount))
	toObj.SetOwner(to)

	// Log mint event
	core.Log("mint",
		"to", to,
		"amount", amount,
		"total_supply", totalSupply)

	return true
}

// Burn tokens
func Burn(id core.ObjectID, amount uint64) bool {
	sender := core.Sender()

	core.Assert(amount > 0)

	// Get default Object
	obj, err := core.GetObject(core.ZeroObjectID)
	core.Assert(err)
	core.Assert(obj.Owner() == sender)

	// Get current total supply
	var totalSupply uint64
	core.Assert(obj.Get(TokenTotalSupplyKey, &totalSupply))

	totalSupply -= amount
	core.Assert(obj.Set(TokenTotalSupplyKey, totalSupply))

	var userObj core.Object
	if id == core.ZeroObjectID {
		userObj, err = core.GetObjectWithOwner(sender)
		core.Assert(err)
	} else {
		userObj, err = core.GetObject(id)
		core.Assert(err)
	}

	var userBalance uint64
	core.Assert(userObj.Get(TokenAmountKey, &userBalance))
	core.Assert(userBalance >= amount)

	userBalance -= amount
	if userBalance == 0 {
		core.DeleteObject(userObj.ID())
	} else {
		core.Assert(userObj.Set(TokenAmountKey, userBalance))
	}

	// Log burn event
	core.Log("burn",
		"from", sender,
		"amount", amount,
		"total_supply", totalSupply)

	return true
}
