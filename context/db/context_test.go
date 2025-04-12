package db

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/govm-net/vm/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *Context {
	// 使用临时文件作为测试数据库
	tmpFile := "./test.db"
	t.Cleanup(func() {
		os.Remove(tmpFile)
	})

	ctx := NewContext(map[string]any{
		"db_path": tmpFile,
	}).(*Context)

	return ctx
}

func TestBlockContext(t *testing.T) {
	ctx := setupTestDB(t)

	// 创建测试区块
	block := &DBBlock{
		Height: 100,
		Time:   1234567890,
		Hash:   "0x1234567890",
	}
	require.NoError(t, ctx.db.Create(block).Error)

	// 测试设置区块上下文
	err := ctx.WithBlock(100)
	require.NoError(t, err)

	// 验证区块信息
	assert.Equal(t, uint64(100), ctx.BlockHeight())
	assert.Equal(t, int64(1234567890), ctx.BlockTime())
}

func TestTransactionContext(t *testing.T) {
	ctx := setupTestDB(t)

	// 创建测试交易
	tx := &DBTransaction{
		Hash:        "0xabcdef",
		BlockHeight: 100,
		FromAddress: "0x1234",
		ToAddress:   "0x5678",
		Value:       1000,
		Data:        []byte("test data"),
	}
	require.NoError(t, ctx.db.Create(tx).Error)

	// 测试设置交易上下文
	err := ctx.WithTransaction("0xabcdef")
	require.NoError(t, err)

	// 验证交易信息
	assert.Equal(t, core.AddressFromString("0x1234"), ctx.Sender())
	assert.Equal(t, core.AddressFromString("0x5678"), ctx.ContractAddress())
	assert.Equal(t, core.HashFromString("0xabcdef"), ctx.TransactionHash())
}

func TestBalanceTransfer(t *testing.T) {
	ctx := setupTestDB(t)

	addr1 := core.AddressFromString("1111")
	addr2 := core.AddressFromString("2222")

	// 初始化余额
	balance1 := &DBBalance{
		Address: addr1.String(),
		Amount:  1000,
	}
	require.NoError(t, ctx.db.Create(balance1).Error)

	// 测试余额查询
	assert.Equal(t, uint64(1000), ctx.Balance(addr1))
	assert.Equal(t, uint64(0), ctx.Balance(addr2))

	// 测试转账
	err := ctx.Transfer(core.ZeroAddress, addr1, addr2, 500)
	require.NoError(t, err)

	// 验证转账结果
	assert.Equal(t, uint64(500), ctx.Balance(addr1))
	assert.Equal(t, uint64(500), ctx.Balance(addr2))

	// 测试余额不足
	err = ctx.Transfer(core.ZeroAddress, addr1, addr2, 1000)
	assert.Error(t, err)
}

func TestObjectOperations(t *testing.T) {
	ctx := setupTestDB(t)

	// 设置测试环境
	contract := core.AddressFromString("0xcontract")
	sender := core.AddressFromString("0xsender")
	ctx.sender = sender

	// 创建测试交易上下文
	tx := &DBTransaction{
		Hash:        "0xtx",
		FromAddress: sender.String(),
	}
	require.NoError(t, ctx.db.Create(tx).Error)
	ctx.currentTx = tx

	// 测试创建对象
	obj, err := ctx.CreateObject(contract)
	require.NoError(t, err)
	assert.NotNil(t, obj)

	// 测试设置字段
	err = obj.Set(contract, sender, "name", []byte("test"))
	require.NoError(t, err)

	// 测试获取字段
	value, err := obj.Get(contract, "name")
	require.NoError(t, err)
	assert.Equal(t, []byte("test"), value)

	// 测试按ID获取对象
	obj2, err := ctx.GetObject(contract, obj.ID())
	require.NoError(t, err)
	assert.Equal(t, obj.ID(), obj2.ID())
	assert.Equal(t, obj.Owner(), obj2.Owner())

	// 测试按所有者获取对象
	obj3, err := ctx.GetObjectWithOwner(contract, sender)
	require.NoError(t, err)
	assert.Equal(t, obj.ID(), obj3.ID())

	// 测试删除对象
	err = ctx.DeleteObject(contract, obj.ID())
	require.NoError(t, err)

	// 验证对象已删除
	_, err = ctx.GetObject(contract, obj.ID())
	assert.Error(t, err)
}

func TestEventLogging(t *testing.T) {
	ctx := setupTestDB(t)

	// 设置测试环境
	block := &DBBlock{
		Height: 100,
		Time:   1234567890,
		Hash:   "0xblock",
	}
	require.NoError(t, ctx.db.Create(block).Error)
	ctx.currentBlock = block

	tx := core.HashFromString("0x124578")
	ctx.SetTransactionInfo(tx, core.ZeroAddress, core.ZeroAddress, 1000)

	contract := core.AddressFromString("0xcontract")

	// 测试记录事件
	ctx.Log(contract, "TestEvent",
		"key1", "value1",
		"key2", 123,
	)

	// 验证事件记录
	var event DBEvent
	result := ctx.db.Where("contract_address = ? AND event_name = ?",
		contract.String(), "TestEvent").First(&event)
	require.NoError(t, result.Error)

	assert.Equal(t, uint64(100), event.BlockHeight)
	assert.Equal(t, tx.String(), event.TxHash)
	assert.Equal(t, contract.String(), event.Contract)
	assert.Equal(t, "TestEvent", event.EventName)

	// 验证事件参数
	var data []interface{}
	err := json.Unmarshal(event.KeyValues, &data)
	require.NoError(t, err)
	assert.Equal(t, "key1", data[0])
	assert.Equal(t, "value1", data[1])
	assert.Equal(t, "key2", data[2])
	assert.Equal(t, float64(123), data[3]) // JSON 将数字解码为 float64
}
