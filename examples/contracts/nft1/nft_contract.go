// 基于wasm包装层的简单NFT合约示例
package nft

import (
	"fmt"

	"github.com/govm-net/vm/core"
)

// 常量定义 - 对象中的字段名
const (
	// 默认Object中的字段
	NFTNameKey        = "name"         // NFT名称
	NFTSymbolKey      = "symbol"       // NFT符号
	NFTTotalSupplyKey = "total_supply" // 总供应量
	NFTTokenURIKey    = "token_uri"    // Token URI
	NFTTokenIDKey     = "token_id"     // Token ID
	NFTTokenOwnerKey  = "owner"        // Token所有者
)

// 初始化NFT合约
func InitializeNFT(ctx core.Context, name string, symbol string) core.ObjectID {
	// 获取默认Object（空ObjectID）
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return core.ZeroObjectID
	}

	// 存储NFT基本信息
	err = defaultObj.Set(NFTNameKey, name)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("存储NFT名称失败: %v", err))
		return core.ZeroObjectID
	}

	err = defaultObj.Set(NFTSymbolKey, symbol)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("存储NFT符号失败: %v", err))
		return core.ZeroObjectID
	}

	// 初始化总供应量为0
	err = defaultObj.Set(NFTTotalSupplyKey, uint64(0))
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("初始化总供应量失败: %v", err))
		return core.ZeroObjectID
	}

	defaultObj.SetOwner(ctx.Sender())

	// 记录初始化事件
	ctx.Log("initialize",
		"id", defaultObj.ID(),
		"name", name,
		"symbol", symbol,
		"owner", ctx.Sender())

	return defaultObj.ID()
}

// 获取NFT信息
func GetNFTInfo(ctx core.Context) (string, string, uint64) {
	// 获取默认Object
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return "", "", 0
	}

	// 读取NFT基本信息
	var name string
	err = defaultObj.Get(NFTNameKey, &name)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取NFT名称失败: %v", err))
		return "", "", 0
	}

	var symbol string
	err = defaultObj.Get(NFTSymbolKey, &symbol)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取NFT符号失败: %v", err))
		return "", "", 0
	}

	var totalSupply uint64
	err = defaultObj.Get(NFTTotalSupplyKey, &totalSupply)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取总供应量失败: %v", err))
		return "", "", 0
	}

	return name, symbol, totalSupply
}

// 获取所有者
func GetOwner(ctx core.Context) core.Address {
	// 获取默认Object
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return core.ZeroAddress
	}

	return defaultObj.Owner()
}

// 获取NFT所有者
func OwnerOf(ctx core.Context, tokenId core.ObjectID) core.Address {

	// 获取NFT对象
	nftObj, err := ctx.GetObject(tokenId)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取NFT对象失败: %v", err))
		return core.ZeroAddress
	}

	return nftObj.Owner()
}

// 获取NFT URI
func TokenURI(ctx core.Context, tokenId core.ObjectID) string {
	// 获取NFT对象
	nftObj, err := ctx.GetObject(tokenId)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取NFT对象失败: %v", err))
		return ""
	}

	var uri string
	err = nftObj.Get(NFTTokenURIKey, &uri)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取NFT URI失败: %v", err))
		return ""
	}

	return uri
}

// 铸造新NFT（仅限所有者）
func Mint(ctx core.Context, to core.Address, tokenURI string) (uint64, bool) {
	sender := ctx.Sender()

	// 获取默认Object
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return 0, false
	}

	// 检查是否为合约所有者
	if sender != defaultObj.Owner() {
		ctx.Log("error", "message", "只有合约所有者才能铸造新NFT")
		return 0, false
	}

	// 获取当前总供应量
	var totalSupply uint64
	err = defaultObj.Get(NFTTotalSupplyKey, &totalSupply)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取总供应量失败: %v", err))
		return 0, false
	}

	// 创建新的NFT对象
	nftObj := ctx.CreateObject()
	err = nftObj.Set(NFTTokenURIKey, tokenURI)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("设置NFT URI失败: %v", err))
		return 0, false
	}

	err = nftObj.Set(NFTTokenIDKey, totalSupply+1)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("设置NFT ID失败: %v", err))
		return 0, false
	}

	// 设置NFT所有者
	nftObj.SetOwner(to)

	// 更新总供应量
	err = defaultObj.Set(NFTTotalSupplyKey, totalSupply+1)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("更新总供应量失败: %v", err))
		return 0, false
	}

	// 记录铸造事件
	ctx.Log("mint",
		"to", to,
		"token_id", totalSupply+1,
		"token_uri", tokenURI)

	return totalSupply + 1, true
}

// 转移NFT
func Transfer(ctx core.Context, from core.Address, to core.Address, tokenId core.ObjectID) bool {
	sender := ctx.Sender()

	// 获取NFT对象
	nftObj, err := ctx.GetObject(tokenId)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取NFT对象失败: %v", err))
		return false
	}

	// 检查发送者是否为NFT所有者
	if sender != nftObj.Owner() {
		ctx.Log("error", "message", "只有NFT所有者才能转移NFT")
		return false
	}

	// 检查from地址
	if from != nftObj.Owner() {
		ctx.Log("error", "message", "from地址必须是NFT所有者")
		return false
	}

	// 转移NFT所有权
	nftObj.SetOwner(to)

	// 记录转移事件
	ctx.Log("transfer",
		"from", from,
		"to", to,
		"token_id", tokenId)

	return true
}

// 销毁NFT
func Burn(ctx core.Context, tokenId core.ObjectID) bool {
	sender := ctx.Sender()

	// 获取默认Object
	defaultObj, err := ctx.GetObject(core.ZeroObjectID)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取默认对象失败: %v", err))
		return false
	}

	// 获取NFT对象
	nftObj, err := ctx.GetObject(tokenId)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取NFT对象失败: %v", err))
		return false
	}

	// 检查是否为NFT所有者
	if sender != nftObj.Owner() {
		ctx.Log("error", "message", "只有NFT所有者才能销毁NFT")
		return false
	}

	// 获取当前总供应量
	var totalSupply uint64
	err = defaultObj.Get(NFTTotalSupplyKey, &totalSupply)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("获取总供应量失败: %v", err))
		return false
	}

	// 更新总供应量
	err = defaultObj.Set(NFTTotalSupplyKey, totalSupply-1)
	if err != nil {
		ctx.Log("error", "message", fmt.Sprintf("更新总供应量失败: %v", err))
		return false
	}

	// 销毁NFT对象
	ctx.DeleteObject(nftObj.ID())

	// 记录销毁事件
	ctx.Log("burn",
		"from", sender,
		"token_id", tokenId)

	return true
}
