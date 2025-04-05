package db

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/govm-net/vm/context"
	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	defaultDBPath = "./sqlite.db"
)

type DBBlock struct {
	gorm.Model
	Height uint64 `gorm:"column:height;not null;unique;index"`
	Time   int64  `gorm:"column:block_time;not null"`
	Hash   string `gorm:"column:block_hash;not null;unique;index;size:66"`
}

func (DBBlock) TableName() string {
	return "blocks"
}

type DBTransaction struct {
	gorm.Model
	Hash        string `gorm:"column:tx_hash;not null;unique;index;size:66"`
	BlockHeight uint64 `gorm:"column:block_height;not null;index"`
	FromAddress string `gorm:"column:from_address;not null;index;size:42"`
	ToAddress   string `gorm:"column:to_address;not null;index;size:42"`
	Value       uint64 `gorm:"column:value;not null"`
	Data        []byte `gorm:"column:tx_data;type:blob;default:''"`
}

func (DBTransaction) TableName() string {
	return "transactions"
}

// DBObject represents the object in database
type DBObject struct {
	gorm.Model
	ObjectID string `gorm:"column:object_id;not null;unique;index;size:66"`
	Owner    string `gorm:"column:owner_address;not null;index;size:42"`
	Contract string `gorm:"column:contract_address;not null;index;size:42"`
}

// TableName specifies the table name for DBObject
func (DBObject) TableName() string {
	return "objects"
}

// DBObjectField represents a field of an object
type DBObjectField struct {
	gorm.Model
	ObjectID string `gorm:"column:object_id;not null;index;size:66"`
	Key      string `gorm:"column:field_key;not null;index;size:255"`
	Value    []byte `gorm:"column:field_value;type:blob;not null"`
}

// TableName specifies the table name for DBObjectField
func (DBObjectField) TableName() string {
	return "object_fields"
}

// DBBalance represents the balance in database
type DBBalance struct {
	Address string `gorm:"column:address;primaryKey;size:42"`
	Amount  uint64 `gorm:"column:balance;not null;default:0"`
}

// TableName specifies the table name for DBBalance
func (DBBalance) TableName() string {
	return "balances"
}

// DBEvent represents an event in the database
type DBEvent struct {
	gorm.Model
	BlockHeight uint64 `gorm:"column:block_height;not null;index"`
	TxHash      string `gorm:"column:tx_hash;not null;index;size:66"`
	Contract    string `gorm:"column:contract_address;not null;index;size:42"`
	EventName   string `gorm:"column:event_name;not null;index;size:255"`
	KeyValues   []byte `gorm:"column:key_values;type:blob;not null"` // JSON encoded key-value pairs
}

// TableName specifies the table name for DBEvent
func (DBEvent) TableName() string {
	return "events"
}

// Context implements the BlockchainContext interface using SQLite with GORM
type Context struct {
	db *gorm.DB

	// Runtime state
	sender       core.Address
	gasLimit     int64
	gasUsed      int64
	currentTx    *DBTransaction
	currentBlock *DBBlock
	nonce        uint64
}

func init() {
	context.Register(context.DBContextType, NewContext)
}

// NewContext creates a new SQLite-backed blockchain context using GORM
func NewContext(params map[string]any) types.BlockchainContext {
	if params == nil {
		params = make(map[string]any)
	}
	dbPath := defaultDBPath
	if path, ok := params["db_path"].(string); ok && path != "" {
		dbPath = path
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		panic(fmt.Errorf("failed to create db directory: %v", err))
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		panic(fmt.Errorf("failed to open database: %v", err))
	}

	ctx := &Context{db: db}
	ctx.initDB()
	return ctx
}

func (c *Context) initDB() {
	// Auto migrate the schemas with indexes
	err := c.db.AutoMigrate(
		&DBBlock{},
		&DBTransaction{},
		&DBObject{},
		&DBObjectField{},
		&DBBalance{},
		&DBEvent{},
	)
	if err != nil {
		panic(fmt.Errorf("failed to migrate database: %v", err))
	}
}

func (c *Context) SetGasLimit(limit int64) {
	c.gasLimit = limit
}

// WithBlock sets the current block context
func (c *Context) WithBlock(height uint64) error {
	var block DBBlock
	result := c.db.Where("height = ?", height).First(&block)
	if result.Error != nil {
		return fmt.Errorf("failed to get block: %v", result.Error)
	}
	c.currentBlock = &block
	c.nonce = 0
	return nil
}

// WithTransaction sets the current transaction context
func (c *Context) WithTransaction(hash string) error {
	var tx DBTransaction
	result := c.db.Where("tx_hash = ?", hash).First(&tx)
	if result.Error != nil {
		return fmt.Errorf("failed to get transaction: %v", result.Error)
	}
	c.currentTx = &tx
	c.sender = core.AddressFromString(tx.FromAddress)
	c.nonce = 0
	return nil
}

// BlockHeight implements types.BlockchainContext
func (c *Context) BlockHeight() uint64 {
	if c.currentBlock != nil {
		return c.currentBlock.Height
	}
	var height uint64
	c.db.Model(&DBBlock{}).Select("COALESCE(MAX(height), 0)").Scan(&height)
	return height
}

// BlockTime implements types.BlockchainContext
func (c *Context) BlockTime() int64 {
	if c.currentBlock != nil {
		return c.currentBlock.Time
	}
	return 0
}

// ContractAddress implements types.BlockchainContext
func (c *Context) ContractAddress() core.Address {
	if c.currentTx != nil {
		return core.AddressFromString(c.currentTx.ToAddress)
	}
	return core.Address{}
}

// TransactionHash implements types.BlockchainContext
func (c *Context) TransactionHash() core.Hash {
	if c.currentTx != nil {
		return core.HashFromString(c.currentTx.Hash)
	}
	return core.Hash{}
}

// Sender implements types.BlockchainContext
func (c *Context) Sender() core.Address {
	if c.currentTx != nil {
		return core.AddressFromString(c.currentTx.FromAddress)
	}
	return c.sender
}

// GetGas implements types.BlockchainContext
func (c *Context) GetGas() int64 {
	return c.gasUsed
}

// Balance implements types.BlockchainContext
func (c *Context) Balance(addr core.Address) uint64 {
	var balance DBBalance
	result := c.db.Where("address = ?", addr.String()).First(&balance)
	if result.Error == gorm.ErrRecordNotFound {
		return 0
	}
	if result.Error != nil {
		panic(fmt.Errorf("failed to get balance: %v", result.Error))
	}
	return balance.Amount
}

// Transfer implements types.BlockchainContext
func (c *Context) Transfer(from, to core.Address, amount uint64) error {
	return c.db.Transaction(func(tx *gorm.DB) error {
		// Get sender balance
		var fromBalance DBBalance
		result := tx.Where("address = ?", from.String()).First(&fromBalance)
		if result.Error == gorm.ErrRecordNotFound {
			// If sender doesn't exist, they have 0 balance
			return fmt.Errorf("insufficient balance")
		} else if result.Error != nil {
			return fmt.Errorf("failed to get sender balance: %v", result.Error)
		}

		// Check if sender has sufficient balance
		if fromBalance.Amount < amount {
			return fmt.Errorf("insufficient balance")
		}

		// Update sender balance
		if err := tx.Model(&DBBalance{}).Where("address = ?", from.String()).
			Update("balance", fromBalance.Amount-amount).Error; err != nil {
			return fmt.Errorf("failed to update sender balance: %v", err)
		}

		// Get and update recipient balance
		var toBalance DBBalance
		result = tx.Where("address = ?", to.String()).First(&toBalance)
		if result.Error == gorm.ErrRecordNotFound {
			// Create recipient balance
			toBalance = DBBalance{
				Address: to.String(),
				Amount:  amount,
			}
			if err := tx.Create(&toBalance).Error; err != nil {
				return fmt.Errorf("failed to create recipient balance: %v", err)
			}
		} else if result.Error != nil {
			return fmt.Errorf("failed to get recipient balance: %v", result.Error)
		} else {
			// Update recipient balance
			if err := tx.Model(&DBBalance{}).Where("address = ?", to.String()).
				Update("balance", toBalance.Amount+amount).Error; err != nil {
				return fmt.Errorf("failed to update recipient balance: %v", err)
			}
		}

		return nil
	})
}

// CreateObject implements types.BlockchainContext
func (c *Context) CreateObject(contract core.Address) (types.VMObject, error) {
	c.nonce++
	str := fmt.Sprintf("%x:%x:%d", c.currentTx.Hash, c.sender.String(), c.nonce)
	hash := core.GetHash([]byte(str))
	return c.CreateObjectWithID(contract, core.ObjectID(hash))
}

// CreateObjectWithID implements types.BlockchainContext
func (c *Context) CreateObjectWithID(contract core.Address, id core.ObjectID) (types.VMObject, error) {
	obj := &Object{
		ctx:      c,
		id:       id,
		owner:    c.sender,
		contract: contract,
	}

	dbObj := &DBObject{
		Owner:    obj.owner.String(),
		Contract: contract.String(),
		ObjectID: id.String(),
	}

	if err := c.db.Create(dbObj).Error; err != nil {
		return nil, fmt.Errorf("failed to create object: %v", err)
	}

	return obj, nil
}

// GetObject implements types.BlockchainContext
func (c *Context) GetObject(contract core.Address, id core.ObjectID) (types.VMObject, error) {
	var dbObj DBObject
	result := c.db.Where("object_id = ? AND contract_address = ?", id.String(), contract.String()).First(&dbObj)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("object not found")
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get object: %v", result.Error)
	}

	owner := core.AddressFromString(dbObj.Owner)
	return &Object{
		ctx:      c,
		id:       id,
		owner:    owner,
		contract: contract,
	}, nil
}

// GetObjectWithOwner implements types.BlockchainContext
func (c *Context) GetObjectWithOwner(contract core.Address, owner core.Address) (types.VMObject, error) {
	var dbObj DBObject
	result := c.db.Where("owner_address = ? AND contract_address = ?", owner.String(), contract.String()).First(&dbObj)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("object not found")
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get object: %v", result.Error)
	}

	id := core.HashFromString(dbObj.ObjectID)
	return &Object{
		ctx:      c,
		id:       core.ObjectID(id),
		owner:    owner,
		contract: contract,
	}, nil
}

// DeleteObject implements types.BlockchainContext
func (c *Context) DeleteObject(contract core.Address, id core.ObjectID) error {
	result := c.db.Where("object_id = ? AND contract_address = ?", id.String(), contract.String()).Delete(&DBObject{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete object: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("object not found")
	}
	return nil
}

// Call implements types.BlockchainContext
func (c *Context) Call(caller core.Address, contract core.Address, function string, args ...any) ([]byte, error) {
	// TODO: Implement contract call logic
	return nil, fmt.Errorf("not implemented")
}

// Log implements types.BlockchainContext
func (c *Context) Log(contract core.Address, eventName string, keyValues ...any) {
	// 确保当前有区块和交易上下文
	if c.currentBlock == nil || c.currentTx == nil {
		slog.Error("Cannot log event without block and transaction context")
		return
	}

	// 将 keyValues 编码为 JSON
	data, err := json.Marshal(keyValues)
	if err != nil {
		slog.Error("Failed to marshal event data", "error", err)
		return
	}

	// 创建事件记录
	event := &DBEvent{
		BlockHeight: c.currentBlock.Height,
		TxHash:      c.currentTx.Hash,
		Contract:    contract.String(),
		EventName:   eventName,
		KeyValues:   data,
	}

	// 保存到数据库
	if err := c.db.Create(event).Error; err != nil {
		slog.Error("Failed to save event", "error", err)
		return
	}

	// 同时输出到日志
	params := []any{
		"block", c.currentBlock.Height,
		"tx", c.currentTx.Hash,
		"contract", contract,
		"event", eventName,
	}
	params = append(params, keyValues...)
	slog.Info("Contract event", params...)
}

// Object implements the VMObject interface
type Object struct {
	ctx      *Context
	id       core.ObjectID
	owner    core.Address
	contract core.Address
}

func (o *Object) ID() core.ObjectID {
	return o.id
}

func (o *Object) Owner() core.Address {
	return o.owner
}

func (o *Object) Contract() core.Address {
	return o.contract
}

func (o *Object) SetOwner(contract, sender, addr core.Address) error {
	if contract != o.contract {
		return fmt.Errorf("invalid contract")
	}
	if sender != o.owner && contract != o.owner {
		return fmt.Errorf("not owner")
	}

	result := o.ctx.db.Model(&DBObject{}).Where("id = ? AND contract = ?", o.id.String(), o.contract.String()).
		Update("owner", addr.String())
	if result.Error != nil {
		return fmt.Errorf("failed to update owner: %v", result.Error)
	}

	o.owner = addr
	return nil
}

func (o *Object) Get(contract core.Address, field string) ([]byte, error) {
	if contract != o.contract {
		return nil, fmt.Errorf("invalid contract")
	}

	var dbField DBObjectField
	result := o.ctx.db.Where("object_id = ? AND field_key = ?", o.id.String(), field).First(&dbField)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get field: %v", result.Error)
	}

	return dbField.Value, nil
}

func (o *Object) Set(contract, sender core.Address, field string, value []byte) error {
	if contract != o.contract {
		return fmt.Errorf("invalid contract")
	}
	if sender != o.owner && contract != o.owner {
		return fmt.Errorf("not owner")
	}

	// Update or create field
	result := o.ctx.db.Where("object_id = ? AND field_key = ?", o.id.String(), field).
		Assign(DBObjectField{Value: value}).
		FirstOrCreate(&DBObjectField{
			ObjectID: o.id.String(),
			Key:      field,
			Value:    value,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update field: %v", result.Error)
	}
	return nil
}
