// A simple NFT contract example based on WASM wrapper
package nft

import (
	"github.com/govm-net/vm/core"
)

// Constants - Field names in objects
const (
	// Fields in default Object
	NFTNameKey        = "name"         // NFT name
	NFTSymbolKey      = "symbol"       // NFT symbol
	NFTTotalSupplyKey = "total_supply" // Total supply
	NFTTokenURIKey    = "token_uri"    // Token URI
	NFTTokenIDKey     = "token_id"     // Token ID
	NFTTokenOwnerKey  = "owner"        // Token owner
)

// Initialize NFT contract
func InitializeNFT(ctx core.Context, name string, symbol string) core.ObjectID {
	// Get default Object (empty ObjectID)
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	core.Assert(err)

	// Store NFT basic information
	core.Assert(defaultObj.Set(NFTNameKey, name))
	core.Assert(defaultObj.Set(NFTSymbolKey, symbol))

	// Initialize total supply as 0
	core.Assert(defaultObj.Set(NFTTotalSupplyKey, uint64(0)))

	defaultObj.SetOwner(ctx.Sender())

	// Log initialization event
	ctx.Log("initialize",
		"id", defaultObj.ID(),
		"name", name,
		"symbol", symbol,
		"owner", ctx.Sender())

	return defaultObj.ID()
}

// Get NFT information
func GetNFTInfo(ctx core.Context) (string, string, uint64) {
	// Get default Object
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	core.Assert(err)

	// Read NFT basic information
	var name string
	core.Assert(defaultObj.Get(NFTNameKey, &name))

	var symbol string
	core.Assert(defaultObj.Get(NFTSymbolKey, &symbol))

	var totalSupply uint64
	core.Assert(defaultObj.Get(NFTTotalSupplyKey, &totalSupply))

	return name, symbol, totalSupply
}

// Get contract owner
func GetOwner(ctx core.Context) core.Address {
	// Get default Object
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	core.Assert(err)

	return defaultObj.Owner()
}

// Get NFT owner
func OwnerOf(ctx core.Context, tokenId core.ObjectID) core.Address {
	// Get NFT object
	nftObj, err := ctx.GetObject(tokenId)
	core.Assert(err)

	return nftObj.Owner()
}

// Get NFT URI
func TokenURI(ctx core.Context, tokenId core.ObjectID) string {
	// Get NFT object
	nftObj, err := ctx.GetObject(tokenId)
	core.Assert(err)

	var uri string
	core.Assert(nftObj.Get(NFTTokenURIKey, &uri))

	return uri
}

// Mint new NFT (owner only)
func Mint(ctx core.Context, to core.Address, tokenURI string) core.ObjectID {
	sender := ctx.Sender()

	// Get default Object
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	core.Assert(err)

	// Check if sender is contract owner
	core.Assert(sender == defaultObj.Owner())

	// Get current total supply
	var totalSupply uint64
	core.Assert(defaultObj.Get(NFTTotalSupplyKey, &totalSupply))
	totalSupply += 1

	// Create new NFT object
	nftObj := ctx.CreateObject()
	core.Assert(nftObj.Set(NFTTokenURIKey, tokenURI))
	core.Assert(nftObj.Set(NFTTokenIDKey, totalSupply))

	// Set NFT owner
	nftObj.SetOwner(to)

	// Update total supply
	core.Assert(defaultObj.Set(NFTTotalSupplyKey, totalSupply))

	// Log mint event
	ctx.Log("mint",
		"to", to,
		"token_id", totalSupply,
		"token_uri", tokenURI)

	return nftObj.ID()
}

// Transfer NFT
func Transfer(ctx core.Context, from core.Address, to core.Address, tokenId core.ObjectID) bool {
	sender := ctx.Sender()

	// Get NFT object
	nftObj, err := ctx.GetObject(tokenId)
	core.Assert(err)

	// Check if sender is NFT owner
	core.Assert(sender == nftObj.Owner())

	// Check from address
	core.Assert(from == nftObj.Owner())

	// Transfer NFT ownership
	nftObj.SetOwner(to)

	// Log transfer event
	ctx.Log("transfer",
		"from", from,
		"to", to,
		"token_id", tokenId)

	return true
}

// Burn NFT
func Burn(ctx core.Context, tokenId core.ObjectID) bool {
	sender := ctx.Sender()

	// Get default Object
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	core.Assert(err)

	// Get NFT object
	nftObj, err := ctx.GetObject(tokenId)
	core.Assert(err)

	// Check if sender is NFT owner
	core.Assert(sender == nftObj.Owner())

	// Get current total supply
	var totalSupply uint64
	core.Assert(defaultObj.Get(NFTTotalSupplyKey, &totalSupply))

	// Update total supply
	core.Assert(defaultObj.Set(NFTTotalSupplyKey, totalSupply-1))

	// Delete NFT object
	ctx.DeleteObject(nftObj.ID())

	// Log burn event
	ctx.Log("burn",
		"from", sender,
		"token_id", tokenId)

	return true
}
