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

// Manager represents a code manager
type Manager struct {
	rootDir string // Root directory for code
}

// ContractCode represents contract code information
type ContractCode struct {
	Address      core.Address // Contract address
	OriginalCode []byte       // Original code
	InjectedCode []byte       // Code after gas information injection
	Dependencies []string     // Dependencies on other contract addresses
	UpdateTime   time.Time    // Last update time
	Hash         [32]byte     // Code hash
}

// ContractMetadata represents contract metadata
type ContractMetadata struct {
	Hash         string    `json:"hash"`         // Code hash
	UpdateTime   time.Time `json:"update_time"`  // Update time
	Dependencies []string  `json:"dependencies"` // Dependency list
}

// NewManager creates a new code manager
func NewManager(rootDir string) (*Manager, error) {
	if rootDir == "" {
		tempDir, err := os.MkdirTemp("", "code-manager-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temporary directory: %w", err)
		}
		rootDir = tempDir
	} else {
		if err := os.MkdirAll(rootDir, 0755); err != nil {
			slog.Error("failed to create root directory", "dir", rootDir, "error", err)
			return nil, fmt.Errorf("failed to create root directory: %w", err)
		}
	}

	return &Manager{
		rootDir: rootDir,
	}, nil
}

// RegisterCode registers new contract code
func (m *Manager) RegisterCode(address core.Address, code []byte) error {
	// Check if contract already exists
	contractDir := m.getContractDir(address)
	if _, err := os.Stat(contractDir); err == nil {
		return fmt.Errorf("contract already exists: %s", address)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check contract directory: %w", err)
	}

	// Calculate code hash
	hash := sha256.Sum256(code)

	// Create contract directory
	if err := os.MkdirAll(contractDir, 0755); err != nil {
		return fmt.Errorf("failed to create contract directory: %w", err)
	}

	// Inject gas consumption code
	injectedCode, err := mock.AddGasConsumption(address.String(), code)
	if err != nil {
		// Delete created directory
		os.RemoveAll(contractDir)
		return fmt.Errorf("failed to inject gas consumption: %w", err)
	}

	// Create ContractCode object
	contractCode := &ContractCode{
		Address:      address,
		OriginalCode: code,
		InjectedCode: injectedCode,
		UpdateTime:   time.Now(),
		Hash:         hash,
	}

	// Save code files
	if err := m.saveContractFiles(contractCode); err != nil {
		// Delete created directory
		os.RemoveAll(contractDir)
		return fmt.Errorf("failed to save contract files: %w", err)
	}

	return nil
}

// GetCode retrieves contract code
func (m *Manager) GetCode(address core.Address) (*ContractCode, error) {
	return m.loadContractCode(address)
}

// GetInjectedCode retrieves code after gas information injection
func (m *Manager) GetInjectedCode(address core.Address) ([]byte, error) {
	code, err := m.GetCode(address)
	if err != nil {
		return nil, err
	}
	return code.InjectedCode, nil
}

// getContractDir gets the contract directory path
func (m *Manager) getContractDir(address core.Address) string {
	return filepath.Join(m.rootDir, address.String())
}

// saveContractFiles saves contract related files
func (m *Manager) saveContractFiles(code *ContractCode) error {
	dir := m.getContractDir(code.Address)

	// Save original code
	if err := os.WriteFile(filepath.Join(dir, "original.go.txt"), code.OriginalCode, 0644); err != nil {
		return fmt.Errorf("failed to save original code: %w", err)
	}

	// Save code after gas injection
	if err := os.WriteFile(filepath.Join(dir, "injected.go"), code.InjectedCode, 0644); err != nil {
		return fmt.Errorf("failed to save injected code: %w", err)
	}

	// Create metadata
	metadata := ContractMetadata{
		Hash:         hex.EncodeToString(code.Hash[:]),
		UpdateTime:   code.UpdateTime,
		Dependencies: code.Dependencies,
	}

	// Serialize metadata to JSON
	metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Save metadata
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), metadataBytes, 0644); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	return nil
}

// loadContractCode loads contract code from filesystem
func (m *Manager) loadContractCode(address core.Address) (*ContractCode, error) {
	dir := m.getContractDir(address)

	// Read original code
	originalCode, err := os.ReadFile(filepath.Join(dir, "original.go.txt"))
	if err != nil {
		return nil, fmt.Errorf("failed to read original code: %w", err)
	}

	// Read code after gas injection
	injectedCode, err := os.ReadFile(filepath.Join(dir, "injected.go"))
	if err != nil {
		return nil, fmt.Errorf("failed to read injected code: %w", err)
	}

	// Read metadata
	metadataBytes, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	// Parse metadata
	var metadata ContractMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Parse hash
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
