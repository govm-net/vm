package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/mock"
)

// Manager 代码管理器
type Manager struct {
	rootDir string // 代码根目录
}

// ContractCode 合约代码信息
type ContractCode struct {
	Address      core.Address // 合约地址
	OriginalCode []byte       // 原始代码
	InjectedCode []byte       // 注入gas信息后的代码
	Dependencies []string     // 依赖的其他合约地址
	UpdateTime   time.Time    // 最后更新时间
	Hash         [32]byte     // 代码哈希
}

// ContractMetadata 合约元数据
type ContractMetadata struct {
	Hash         string    `json:"hash"`         // 代码哈希
	UpdateTime   time.Time `json:"update_time"`  // 更新时间
	Dependencies []string  `json:"dependencies"` // 依赖列表
}

// NewManager 创建代码管理器
func NewManager(rootDir string) (*Manager, error) {
	// 确保根目录存在
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		slog.Error("failed to create root directory", "dir", rootDir, "error", err)
		return nil, fmt.Errorf("failed to create root directory: %w", err)
	}

	return &Manager{
		rootDir: rootDir,
	}, nil
}

// RegisterCode 注册新的合约代码
func (m *Manager) RegisterCode(address core.Address, code []byte) error {

	// 检查合约是否已存在
	contractDir := m.getContractDir(address)
	if _, err := os.Stat(contractDir); err == nil {
		return fmt.Errorf("contract already exists: %s", address)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check contract directory: %w", err)
	}

	// 计算代码哈希
	hash := sha256.Sum256(code)

	// 创建合约目录
	if err := os.MkdirAll(contractDir, 0755); err != nil {
		return fmt.Errorf("failed to create contract directory: %w", err)
	}

	// 注入gas计费代码
	injectedCode, err := mock.AddGasConsumption(address.String(), code)
	if err != nil {
		// 删除已创建的目录
		os.RemoveAll(contractDir)
		return fmt.Errorf("failed to inject gas consumption: %w", err)
	}

	// 创建ContractCode对象
	contractCode := &ContractCode{
		Address:      address,
		OriginalCode: code,
		InjectedCode: injectedCode,
		UpdateTime:   time.Now(),
		Hash:         hash,
	}

	// 保存代码文件
	if err := m.saveContractFiles(contractCode); err != nil {
		// 删除已创建的目录
		os.RemoveAll(contractDir)
		return fmt.Errorf("failed to save contract files: %w", err)
	}

	return nil
}

// GetCode 获取合约代码
func (m *Manager) GetCode(address core.Address) (*ContractCode, error) {
	return m.loadContractCode(address)
}

// GetInjectedCode 获取注入gas信息后的代码
func (m *Manager) GetInjectedCode(address core.Address) ([]byte, error) {
	code, err := m.GetCode(address)
	if err != nil {
		return nil, err
	}
	return code.InjectedCode, nil
}

// getContractDir 获取合约目录路径
func (m *Manager) getContractDir(address core.Address) string {
	return filepath.Join(m.rootDir, address.String())
}

// saveContractFiles 保存合约相关文件
func (m *Manager) saveContractFiles(code *ContractCode) error {
	dir := m.getContractDir(code.Address)

	// 保存原始代码
	if err := os.WriteFile(filepath.Join(dir, "original.go.txt"), code.OriginalCode, 0644); err != nil {
		return fmt.Errorf("failed to save original code: %w", err)
	}

	// 保存注入gas后的代码
	if err := os.WriteFile(filepath.Join(dir, "injected.go"), code.InjectedCode, 0644); err != nil {
		return fmt.Errorf("failed to save injected code: %w", err)
	}

	// 创建元数据
	metadata := ContractMetadata{
		Hash:         hex.EncodeToString(code.Hash[:]),
		UpdateTime:   code.UpdateTime,
		Dependencies: code.Dependencies,
	}

	// 序列化元数据为JSON
	metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// 保存元数据
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), metadataBytes, 0644); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	return nil
}

// loadContractCode 从文件系统加载合约代码
func (m *Manager) loadContractCode(address core.Address) (*ContractCode, error) {
	dir := m.getContractDir(address)

	// 读取原始代码
	originalCode, err := os.ReadFile(filepath.Join(dir, "original.go.txt"))
	if err != nil {
		return nil, fmt.Errorf("failed to read original code: %w", err)
	}

	// 读取注入gas后的代码
	injectedCode, err := os.ReadFile(filepath.Join(dir, "injected.go"))
	if err != nil {
		return nil, fmt.Errorf("failed to read injected code: %w", err)
	}

	// 读取元数据
	metadataBytes, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	// 解析元数据
	var metadata ContractMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// 解析哈希
	hashBytes, err := hex.DecodeString(metadata.Hash)
	if err != nil {
		return nil, fmt.Errorf("invalid hash in metadata: %w", err)
	}
	var hash [32]byte
	copy(hash[:], hashBytes)

	return &ContractCode{
		Address:      address,
		OriginalCode: originalCode,
		InjectedCode: injectedCode,
		Dependencies: metadata.Dependencies,
		UpdateTime:   metadata.UpdateTime,
		Hash:         hash,
	}, nil
}
