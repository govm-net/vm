// Package nft 实现一个简单的NFT合约示例
package nft

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/govm-net/vm/core"
)

// Initialize 初始化NFT合约，创建合约元数据对象
func Initialize(ctx core.Context, name string, symbol string) (core.ObjectID, error) {
	// 创建合约元数据对象
	metadataObj := ctx.CreateObject()

	// 设置合约基本信息
	metadataObj.Set("name", name)
	metadataObj.Set("symbol", symbol)
	metadataObj.Set("total_supply", uint64(0))
	metadataObj.Set("next_token_id", uint64(1)) // 开始的token ID

	// 记录初始化事件
	ctx.Log("NFTInitialized",
		"name", name,
		"symbol", symbol,
		"creator", ctx.Sender())

	return metadataObj.ID(), nil
}

// getMetadata 获取合约元数据对象
func getMetadata(ctx core.Context, metadataID core.ObjectID) (core.Object, error) {
	metadataObj, err := ctx.GetObject(metadataID)
	if err != nil {
		return nil, errors.New("nft contract not initialized")
	}

	// 确保元数据对象属于合约自身
	if metadataObj.Owner() != ctx.ContractAddress() {
		return nil, errors.New("invalid metadata object")
	}

	return metadataObj, nil
}

// 检查地址是否为零地址
func isZeroAddress(addr core.Address) bool {
	return bytes.Equal(addr[:], make([]byte, len(addr)))
}

// getTokenKey 构建token对象ID
func getTokenKey(tokenID uint64) core.ObjectID {
	// 创建一个新的ObjectID
	var id core.ObjectID

	// 将tokenID转换为字节并填充ObjectID的前8字节
	idBytes := []byte(fmt.Sprintf("%016d", tokenID))
	copy(id[:16], idBytes)

	// 填充后16字节为"token"标识
	copy(id[16:], []byte("token"))

	return id
}

// Mint 创建新的NFT，仅管理员可以调用
func Mint(ctx core.Context, metadataID core.ObjectID, to core.Address, tokenURI string) (uint64, error) {
	// 获取元数据对象
	metadataObj, err := getMetadata(ctx, metadataID)
	if err != nil {
		return 0, err
	}

	// 检查调用者是否为创建者
	if ctx.Sender() != metadataObj.Owner() {
		return 0, errors.New("only contract owner can mint new tokens")
	}

	// 获取下一个token ID
	var nextTokenID uint64
	if err := metadataObj.Get("next_token_id", &nextTokenID); err != nil {
		return 0, errors.New("failed to get next token id")
	}

	// 创建新的NFT对象
	tokenObj := ctx.CreateObject()

	// 设置NFT属性
	tokenObj.Set("token_id", nextTokenID)
	tokenObj.Set("token_uri", tokenURI)
	tokenObj.Set("created_at", ctx.BlockTime())

	// 设置NFT所有者
	tokenObj.SetOwner(to)

	// 更新元数据
	var totalSupply uint64
	if err := metadataObj.Get("total_supply", &totalSupply); err != nil {
		return 0, errors.New("failed to get total supply")
	}

	// 更新总供应量和下一个token ID
	metadataObj.Set("total_supply", totalSupply+1)
	metadataObj.Set("next_token_id", nextTokenID+1)

	// 创建token到所有者的映射对象
	tokenToOwnerObj := ctx.CreateObject()
	tokenToOwnerObj.Set("token_id", nextTokenID)
	tokenToOwnerObj.Set("owner", to)
	tokenToOwnerObj.SetOwner(ctx.ContractAddress()) // 合约管理此映射

	// 记录铸造事件
	ctx.Log("Mint",
		"token_id", nextTokenID,
		"to", to,
		"token_uri", tokenURI)

	return nextTokenID, nil
}

// OwnerOf 查询特定NFT的所有者
func OwnerOf(ctx core.Context, metadataID core.ObjectID, tokenID uint64) (core.Address, error) {
	// 获取元数据对象
	_, err := getMetadata(ctx, metadataID)
	if err != nil {
		return core.Address{}, err
	}

	// 获取token映射对象
	tokenKey := getTokenKey(tokenID)
	tokenToOwnerObj, err := ctx.GetObject(tokenKey)
	if err != nil {
		return core.Address{}, errors.New("token not found")
	}

	// 获取token所有者
	var owner core.Address
	if err := tokenToOwnerObj.Get("owner", &owner); err != nil {
		return core.Address{}, errors.New("failed to get token owner")
	}

	return owner, nil
}

// Transfer 转移NFT所有权
func Transfer(ctx core.Context, metadataID core.ObjectID, tokenID uint64, to core.Address) error {
	// 检查接收者地址
	if isZeroAddress(to) {
		return errors.New("invalid recipient address")
	}

	// 获取元数据对象
	_, err := getMetadata(ctx, metadataID)
	if err != nil {
		return err
	}

	// 获取token映射对象
	tokenKey := getTokenKey(tokenID)
	tokenToOwnerObj, err := ctx.GetObject(tokenKey)
	if err != nil {
		return errors.New("token not found")
	}

	// 获取token所有者
	var owner core.Address
	if err := tokenToOwnerObj.Get("owner", &owner); err != nil {
		return errors.New("failed to get token owner")
	}

	// 验证发送者是否为token所有者
	if owner != ctx.Sender() {
		return errors.New("not authorized to transfer this token")
	}

	// 获取token对象
	tokenObj, err := ctx.GetObjectWithOwner(owner)
	if err != nil {
		return errors.New("token object not found")
	}

	// 更新token映射
	tokenToOwnerObj.Set("owner", to)

	// 转移token所有权
	tokenObj.SetOwner(to)

	// 记录转移事件
	ctx.Log("Transfer",
		"token_id", tokenID,
		"from", owner,
		"to", to)

	return nil
}

// GetTokenURI 获取NFT的元数据URI
func GetTokenURI(ctx core.Context, metadataID core.ObjectID, tokenID uint64) (string, error) {
	// 获取元数据对象
	_, err := getMetadata(ctx, metadataID)
	if err != nil {
		return "", err
	}

	// 获取token所有者
	owner, err := OwnerOf(ctx, metadataID, tokenID)
	if err != nil {
		return "", err
	}

	// 获取token对象
	tokenObj, err := ctx.GetObjectWithOwner(owner)
	if err != nil {
		return "", errors.New("token object not found")
	}

	// 获取token URI
	var tokenURI string
	if err := tokenObj.Get("token_uri", &tokenURI); err != nil {
		return "", errors.New("failed to get token URI")
	}

	return tokenURI, nil
}

// Burn 销毁NFT
func Burn(ctx core.Context, metadataID core.ObjectID, tokenID uint64) error {
	// 获取元数据对象
	metadataObj, err := getMetadata(ctx, metadataID)
	if err != nil {
		return err
	}

	// 获取token所有者
	owner, err := OwnerOf(ctx, metadataID, tokenID)
	if err != nil {
		return err
	}

	// 验证发送者是否为token所有者
	if owner != ctx.Sender() {
		return errors.New("not authorized to burn this token")
	}

	// 获取token对象
	tokenObj, err := ctx.GetObjectWithOwner(owner)
	if err != nil {
		return errors.New("token object not found")
	}

	// 删除token对象
	ctx.DeleteObject(tokenObj.ID())

	// 获取并删除token映射对象
	tokenKey := getTokenKey(tokenID)
	tokenToOwnerObj, err := ctx.GetObject(tokenKey)
	if err != nil {
		return errors.New("token mapping not found")
	}

	ctx.DeleteObject(tokenToOwnerObj.ID())

	// 更新元数据
	var totalSupply uint64
	if err := metadataObj.Get("total_supply", &totalSupply); err != nil {
		return errors.New("failed to get total supply")
	}

	// 更新总供应量
	metadataObj.Set("total_supply", totalSupply-1)

	// 记录销毁事件
	ctx.Log("Burn",
		"token_id", tokenID,
		"from", owner)

	return nil
}
