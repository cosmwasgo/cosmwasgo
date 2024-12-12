package api

import (
	"os"
	"sync"

	"github.com/tetratelabs/wazero"
)

// Cache defines the interface for caching WASM code
type Cache interface {
	GetRuntime() wazero.Runtime
	GetCompiledCache() map[string]wazero.CompiledModule
	Lock()
	Unlock()
	RLock()
	RUnlock()
	Release() error
}

// WazeroCache implements the Cache interface
type WazeroCache struct {
	runtime       wazero.Runtime
	compiledCache map[string]wazero.CompiledModule
	lock          sync.RWMutex
	lockfile      *os.File
	baseDir       string
}
