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
func InitializeNFT(name string, symbol string) core.ObjectID {
	// Get default Object (empty ObjectID)
	defaultObj, err := core.GetObject(core.ZeroObjectID)
	core.Assert(err)

	// Store NFT basic information
	core.Assert(defaultObj.Set(NFTNameKey, name))
	core.Assert(defaultObj.Set(NFTSymbolKey, symbol))

	// Initialize total supply as 0
	core.Assert(defaultObj.Set(NFTTotalSupplyKey, uint64(0)))

	defaultObj.SetOwner(core.Sender())

	// Log initialization event
	core.Log("initialize",
		"id", defaultObj.ID(),
		"name", name,
		"symbol", symbol,
		"owner", core.Sender())

	return defaultObj.ID()
}

// Get NFT information
func GetNFTInfo() (string, string, uint64) {
	// Get default Object
	defaultObj, err := core.GetObject(core.ZeroObjectID)
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
func GetOwner() core.Address {
	// Get default Object
	defaultObj, err := core.GetObject(core.ZeroObjectID)
	core.Assert(err)

	return defaultObj.Owner()
}

// Get NFT owner
func OwnerOf(tokenId core.ObjectID) core.Address {
	// Get NFT object
	nftObj, err := core.GetObject(tokenId)
	core.Assert(err)

	return nftObj.Owner()
}

// Get NFT URI
func TokenURI(tokenId core.ObjectID) string {
	// Get NFT object
	nftObj, err := core.GetObject(tokenId)
	core.Assert(err)

	var uri string
	core.Assert(nftObj.Get(NFTTokenURIKey, &uri))

	return uri
}

// Mint new NFT (owner only)
func Mint(to core.Address, tokenURI string) core.ObjectID {
	sender := core.Sender()

	// Get default Object
	defaultObj, err := core.GetObject(core.ZeroObjectID)
	core.Assert(err)

	// Check if sender is contract owner
	core.Assert(sender == defaultObj.Owner())

	// Get current total supply
	var totalSupply uint64
	core.Assert(defaultObj.Get(NFTTotalSupplyKey, &totalSupply))
	totalSupply += 1

	// Create new NFT object
	nftObj := core.CreateObject()
	core.Assert(nftObj.Set(NFTTokenURIKey, tokenURI))
	core.Assert(nftObj.Set(NFTTokenIDKey, totalSupply))

	// Set NFT owner
	nftObj.SetOwner(to)

	// Update total supply
	core.Assert(defaultObj.Set(NFTTotalSupplyKey, totalSupply))

	// Log mint event
	core.Log("mint",
		"to", to,
		"token_id", totalSupply,
		"token_uri", tokenURI)

	return nftObj.ID()
}

// Transfer NFT
func Transfer(from core.Address, to core.Address, tokenId core.ObjectID) bool {
	sender := core.Sender()

	// Get NFT object
	nftObj, err := core.GetObject(tokenId)
	core.Assert(err)

	// Check if sender is NFT owner
	core.Assert(sender == nftObj.Owner())

	// Check from address
	core.Assert(from == nftObj.Owner())

	// Transfer NFT ownership
	nftObj.SetOwner(to)

	// Log transfer event
	core.Log("transfer",
		"from", from,
		"to", to,
		"token_id", tokenId)

	return true
}

// Burn NFT
func Burn(tokenId core.ObjectID) bool {
	sender := core.Sender()

	// Get default Object
	defaultObj, err := core.GetObject(core.ZeroObjectID)
	core.Assert(err)

	// Get NFT object
	nftObj, err := core.GetObject(tokenId)
	core.Assert(err)

	// Check if sender is NFT owner
	core.Assert(sender == nftObj.Owner())

	// Get current total supply
	var totalSupply uint64
	core.Assert(defaultObj.Get(NFTTotalSupplyKey, &totalSupply))

	// Update total supply
	core.Assert(defaultObj.Set(NFTTotalSupplyKey, totalSupply-1))

	// Delete NFT object
	core.DeleteObject(nftObj.ID())

	// Log burn event
	core.Log("burn",
		"from", sender,
		"token_id", tokenId)

	return true
}
