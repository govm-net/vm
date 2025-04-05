package context

import (
	"fmt"
	"sync"

	"github.com/govm-net/vm/types"
)

// ContextType represents the type of blockchain context
type ContextType string

const (
	// MemoryContextType represents in-memory context implementation
	MemoryContextType ContextType = "memory"
	// DBContextType represents database-backed context implementation
	DBContextType ContextType = "db"
)

// ContextConstructor is a function type that creates a new BlockchainContext instance
type ContextConstructor func(params map[string]any) types.BlockchainContext

// Registry defines the interface for managing BlockchainContext implementations
type Registry interface {
	// Register adds a new BlockchainContext implementation to the registry
	Register(ct ContextType, constructor ContextConstructor) error
	// SetDefault sets the default context type
	SetDefault(ct ContextType) error
	// Get returns a new instance of the specified context type
	Get(ct ContextType, params map[string]any) (types.BlockchainContext, error)
	// GetDefault returns a new instance of the default context type
	GetDefault(params map[string]any) (types.BlockchainContext, error)
	// DefaultContextType returns the current default context type
	DefaultContextType() ContextType
	// ListRegistered returns a list of all registered context types
	ListRegistered() []ContextType
}

// registry implements the Registry interface
type registry struct {
	mu        sync.RWMutex
	contexts  map[ContextType]ContextConstructor
	defaultCt ContextType
}

var (
	// defaultRegistry is the global singleton registry instance
	defaultRegistry Registry
)

func init() {
	defaultRegistry = &registry{
		contexts: make(map[ContextType]ContextConstructor),
	}
}

// GetRegistry returns the global Registry instance
func GetRegistry() Registry {
	return defaultRegistry
}

// Register adds a new BlockchainContext implementation to the registry
func (r *registry) Register(ct ContextType, constructor ContextConstructor) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.contexts[ct]; exists {
		return fmt.Errorf("context type %s already registered", ct)
	}

	r.contexts[ct] = constructor
	return nil
}

// SetDefault sets the default context type
func (r *registry) SetDefault(ct ContextType) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.contexts[ct]; !exists {
		return fmt.Errorf("context type %s not registered", ct)
	}

	r.defaultCt = ct
	return nil
}

// Get returns a new instance of the specified context type
func (r *registry) Get(ct ContextType, params map[string]any) (types.BlockchainContext, error) {
	r.mu.RLock()
	constructor, exists := r.contexts[ct]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("context type %s not found", ct)
	}

	return constructor(params), nil
}

// GetDefault returns a new instance of the default context type
func (r *registry) GetDefault(params map[string]any) (types.BlockchainContext, error) {
	r.mu.RLock()
	if r.defaultCt == "" {
		r.mu.RUnlock()
		return nil, fmt.Errorf("no default context type set")
	}

	return r.Get(r.defaultCt, params)
}

// DefaultContextType returns the current default context type
func (r *registry) DefaultContextType() ContextType {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.defaultCt == "" {
		return DBContextType
	}
	return r.defaultCt
}

// ListRegistered returns a list of all registered context types
func (r *registry) ListRegistered() []ContextType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]ContextType, 0, len(r.contexts))
	for ct := range r.contexts {
		types = append(types, ct)
	}
	return types
}

// Package level functions that delegate to defaultRegistry

// Register adds a new BlockchainContext implementation to the registry
func Register(ct ContextType, constructor ContextConstructor) error {
	return GetRegistry().Register(ct, constructor)
}

// SetDefault sets the default context type
func SetDefault(ct ContextType) error {
	return GetRegistry().SetDefault(ct)
}

// Get returns a new instance of the specified context type
func Get(ct ContextType, params map[string]any) (types.BlockchainContext, error) {
	if ct == "" {
		ct = GetRegistry().DefaultContextType()
	}
	return GetRegistry().Get(ct, params)
}

// GetDefault returns a new instance of the default context type
func GetDefault(params map[string]any) (types.BlockchainContext, error) {
	return GetRegistry().GetDefault(params)
}

// DefaultContextType returns the current default context type
func DefaultContextType() ContextType {
	return GetRegistry().DefaultContextType()
}

// ListRegistered returns a list of all registered context types
func ListRegistered() []ContextType {
	return GetRegistry().ListRegistered()
}
