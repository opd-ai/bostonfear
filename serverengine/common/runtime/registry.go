package runtime

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/opd-ai/bostonfear/serverengine/common/contracts"
)

// Registry stores available game modules by stable key.
type Registry struct {
	mu      sync.RWMutex
	modules map[string]contracts.GameModule
}

// NewRegistry returns an empty module registry.
func NewRegistry() *Registry {
	return &Registry{modules: make(map[string]contracts.GameModule)}
}

// Register adds a module to the registry.
func (r *Registry) Register(module contracts.GameModule) error {
	if module == nil {
		return fmt.Errorf("module cannot be nil")
	}
	key := normalizeKey(module.Key())
	if key == "" {
		return fmt.Errorf("module key cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.modules[key]; exists {
		return fmt.Errorf("module %q already registered", key)
	}
	r.modules[key] = module
	return nil
}

// MustRegister adds a module and panics if it cannot be registered.
func (r *Registry) MustRegister(module contracts.GameModule) {
	if err := r.Register(module); err != nil {
		panic(err)
	}
}

// Get returns a module by key.
func (r *Registry) Get(key string) (contracts.GameModule, bool) {
	r.mu.RLock()
	module, ok := r.modules[normalizeKey(key)]
	r.mu.RUnlock()
	return module, ok
}

// Keys returns sorted registered module keys.
func (r *Registry) Keys() []string {
	r.mu.RLock()
	keys := make([]string, 0, len(r.modules))
	for k := range r.modules {
		keys = append(keys, k)
	}
	r.mu.RUnlock()
	sort.Strings(keys)
	return keys
}

func normalizeKey(k string) string {
	return strings.ToLower(strings.TrimSpace(k))
}
