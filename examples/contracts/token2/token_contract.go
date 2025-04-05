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
	TokenBalancePrefix  = "balance_"     // Balance field prefix
	TokenOwnerKey       = "owner"        // Token owner
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

	// Store token owner
	core.Assert(defaultObj.Set(TokenOwnerKey, ctx.Sender()))

	// Initialize owner's balance
	core.Assert(defaultObj.Set(TokenBalancePrefix+ctx.Sender().String(), totalSupply))

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

// Get token owner
func GetOwner(ctx core.Context) core.Address {
	// Get default Object
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	core.Assert(err)

	// Get token owner
	var owner core.Address
	core.Assert(defaultObj.Get(TokenOwnerKey, &owner))

	return owner
}

// Get account balance
func BalanceOf(ctx core.Context, owner core.Address) uint64 {
	// Get default Object
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	core.Assert(err)

	// Get balance
	var balance uint64
	if err := defaultObj.Get(TokenBalancePrefix+owner.String(), &balance); err != nil {
		return 0 // If failed to get balance, return 0
	}

	return balance
}

// Transfer tokens to another address
func Transfer(ctx core.Context, to core.Address, amount uint64) bool {
	from := ctx.Sender()

	// Check amount validity
	core.Assert(amount > 0)

	// Get default Object
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	core.Assert(err)

	// Get sender balance
	var fromBalance uint64
	core.Assert(defaultObj.Get(TokenBalancePrefix+from.String(), &fromBalance))

	// Check if balance is sufficient
	core.Assert(fromBalance >= amount)

	// Get recipient balance
	var toBalance uint64
	if err := defaultObj.Get(TokenBalancePrefix+to.String(), &toBalance); err != nil {
		toBalance = 0 // If recipient has no balance record, start from 0
	}

	// Update balances
	core.Assert(defaultObj.Set(TokenBalancePrefix+from.String(), fromBalance-amount))
	core.Assert(defaultObj.Set(TokenBalancePrefix+to.String(), toBalance+amount))

	// Log transfer event
	ctx.Log("transfer",
		"from", from,
		"to", to,
		"amount", amount)

	return true
}

// Mint new tokens (owner only)
func Mint(ctx core.Context, to core.Address, amount uint64) bool {
	sender := ctx.Sender()

	// Get default Object
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	core.Assert(err)

	// Get token owner
	var owner core.Address
	core.Assert(defaultObj.Get(TokenOwnerKey, &owner))

	// Check if sender is token owner
	core.Assert(sender == owner)

	// Check amount validity
	core.Assert(amount > 0)

	// Get current total supply
	var totalSupply uint64
	core.Assert(defaultObj.Get(TokenTotalSupplyKey, &totalSupply))

	// Get recipient balance
	var toBalance uint64
	if err := defaultObj.Get(TokenBalancePrefix+to.String(), &toBalance); err != nil {
		toBalance = 0 // If recipient has no balance record, start from 0
	}

	// Update balance and total supply
	core.Assert(defaultObj.Set(TokenBalancePrefix+to.String(), toBalance+amount))

	newTotalSupply := totalSupply + amount
	if err := defaultObj.Set(TokenTotalSupplyKey, newTotalSupply); err != nil {
		// If failed to update total supply, restore recipient balance
		defaultObj.Set(TokenBalancePrefix+to.String(), toBalance)
		core.Assert(false)
	}

	// Log mint event
	ctx.Log("mint",
		"to", to,
		"amount", amount,
		"total_supply", newTotalSupply)

	return true
}

// Burn tokens
func Burn(ctx core.Context, amount uint64) bool {
	sender := ctx.Sender()

	// Get default Object
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	core.Assert(err)

	// Get token owner
	var owner core.Address
	core.Assert(defaultObj.Get(TokenOwnerKey, &owner))

	// Check if sender is token owner
	core.Assert(sender == owner)

	// Check amount validity
	core.Assert(amount > 0)

	// Get sender balance
	var senderBalance uint64
	core.Assert(defaultObj.Get(TokenBalancePrefix+sender.String(), &senderBalance))

	// Check if balance is sufficient
	core.Assert(senderBalance >= amount)

	// Get current total supply
	var totalSupply uint64
	core.Assert(defaultObj.Get(TokenTotalSupplyKey, &totalSupply))

	// Update balance and total supply
	core.Assert(defaultObj.Set(TokenBalancePrefix+sender.String(), senderBalance-amount))

	newTotalSupply := totalSupply - amount
	if err := defaultObj.Set(TokenTotalSupplyKey, newTotalSupply); err != nil {
		// If failed to update total supply, restore sender balance
		defaultObj.Set(TokenBalancePrefix+sender.String(), senderBalance)
		core.Assert(false)
	}

	// Log burn event
	ctx.Log("burn",
		"from", sender,
		"amount", amount,
		"total_supply", newTotalSupply)

	return true
}
