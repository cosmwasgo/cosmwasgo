package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"

	"github.com/CosmWasm/wasmd/wasmvm/v2/types"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

type Querier = types.Querier

func InitCache(config types.VMConfig) (Cache, error) {
	// Create the base directory
	err := os.MkdirAll(config.Cache.BaseDir, 0o755)
	if err != nil {
		return nil, fmt.Errorf("could not create base directory: %w", err)
	}

	// Acquire an exclusive lock on the cache directory
	lockfilePath := filepath.Join(config.Cache.BaseDir, "exclusive.lock")
	lockfile, err := os.OpenFile(lockfilePath, os.O_RDWR|os.O_CREATE, 0o666)
	if err != nil {
		return nil, fmt.Errorf("could not open exclusive.lock: %w", err)
	}

	if err := unix.Flock(int(lockfile.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return nil, fmt.Errorf("could not obtain exclusive lock: %w", err)
	}

	// Initialize wazero runtime with appropriate configuration
	wazeroRuntimeConfig := wazero.NewRuntimeConfig()
	runtime := wazero.NewRuntimeWithConfig(context.Background(), wazeroRuntimeConfig)

	// Create the cache object
	cache := &WazeroCache{
		runtime:       runtime,
		compiledCache: make(map[string]wazero.CompiledModule),
		baseDir:       config.Cache.BaseDir,
		lockfile:      lockfile,
	}

	return cache, nil
}

// Release frees up resources associated with the cache
func (c *WazeroCache) Release() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.lockfile != nil {
		if err := c.lockfile.Close(); err != nil {
			return fmt.Errorf("failed to close lockfile: %w", err)
		}
		c.lockfile = nil
	}
	if c.runtime != nil {
		if err := c.runtime.Close(context.Background()); err != nil {
			return fmt.Errorf("failed to close wazero runtime: %w", err)
		}
		c.runtime = nil
	}
	return nil
}

func (c *WazeroCache) GetCode(checksum []byte) ([]byte, error) {
	checksumHex := hex.EncodeToString(checksum)
	cacheFilepath := filepath.Join(c.baseDir, checksumHex+".wasm")

	wasm, err := os.ReadFile(cacheFilepath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read wasm file: %s", err)
	}

	return wasm, nil
}

func (c *WazeroCache) StoreCode(wasm []byte) ([]byte, error) {
	checksum := sha256.Sum256(wasm)
	checksumHex := hex.EncodeToString(checksum[:])
	cacheFilepath := filepath.Join(c.baseDir, checksumHex+".wasm")

	// Save the wasm file to disk
	err := os.WriteFile(cacheFilepath, wasm, 0644)
	if err != nil {
		return nil, fmt.Errorf("Failed to save wasm file: %s", err)
	}

	// Compile the module and store it in the cache
	ctx := context.Background()

	mod, err := c.runtime.CompileModule(ctx, wasm)
	if err != nil {
		return nil, fmt.Errorf("Failed to compile wasm module: %s", err)
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	c.compiledCache[checksumHex] = mod

	return checksum[:], nil
}

func (c *WazeroCache) StoreCodeUnchecked(wasm []byte) ([]byte, error) {
	// Similar to StoreCode but skips any checksum or validation checks
	return c.StoreCode(wasm)
}

func (c *WazeroCache) Pin(checksum []byte) error {
	checksumHex := hex.EncodeToString(checksum)
	c.lock.Lock()
	defer c.lock.Unlock()

	if _, ok := c.compiledCache[checksumHex]; ok {
		// Already pinned
		return nil
	}

	// Load the wasm code
	cacheFilepath := filepath.Join(c.baseDir, checksumHex+".wasm")
	wasm, err := os.ReadFile(cacheFilepath)
	if err != nil {
		return fmt.Errorf("Failed to read wasm file: %s", err)
	}

	// Compile and store the module
	ctx := context.Background()
	mod, err := c.runtime.CompileModule(ctx, wasm)
	if err != nil {
		return fmt.Errorf("Failed to compile wasm module: %s", err)
	}

	c.compiledCache[checksumHex] = mod

	return nil
}

func (c *WazeroCache) Unpin(checksum []byte) error {
	checksumHex := hex.EncodeToString(checksum)
	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.compiledCache, checksumHex)

	return nil
}

func (c *WazeroCache) GetMetrics() (*types.Metrics, error) {
	// Return dummy metrics or implement actual metrics collection
	metrics := &types.Metrics{
		HitsPinnedMemoryCache:     0,
		HitsMemoryCache:           0,
		HitsFsCache:               0,
		Misses:                    0,
		ElementsPinnedMemoryCache: uint64(len(c.compiledCache)),
		ElementsMemoryCache:       0,
		SizePinnedMemoryCache:     0,
		SizeMemoryCache:           0,
	}
	return metrics, nil
}

func (c *WazeroCache) GetPinnedMetrics() (*types.PinnedMetrics, error) {
	// Return dummy pinned metrics or implement actual metrics collection
	pinnedMetrics := &types.PinnedMetrics{
		ElementCount: 0,
	}
	return pinnedMetrics, nil
}

func AnalyzeCode(cache Cache, checksum []byte) (*types.AnalysisReport, error) {
	// For simplicity, return dummy data or implement actual analysis if needed
	report := &types.AnalysisReport{
		HasIBCEntryPoints:      false,
		RequiredCapabilities:   "",
		Entrypoints:            []string{},
		ContractMigrateVersion: nil,
	}
	return report, nil
}

func Instantiate(
	cache Cache,
	checksum []byte,
	envData, infoData, msgData []byte,
	gasMeter *types.GasMeter,
	store types.KVStore,
	api *types.GoAPI,
	querier *types.Querier,
	gasLimit uint64,
	printDebug bool,
) ([]byte, types.GasReport, error) {
	cache.Lock()
	defer cache.Unlock()

	checksumHex := hex.EncodeToString(checksum)
	compiledModule, ok := cache.GetCompiledCache()[checksumHex]
	if !ok {
		return nil, types.GasReport{}, fmt.Errorf("module with checksum %s not found in cache", checksumHex)
	}

	ctx := context.Background()

	// Configure the module with necessary environment
	moduleConfig := wazero.NewModuleConfig().
		WithName(checksumHex).
		WithStartFunctions("_start").
		WithMemoryPages(1, 32)

	// Instantiate the module
	instance, err := cache.GetRuntime().InstantiateModule(ctx, compiledModule, moduleConfig)
	if err != nil {
		return nil, types.GasReport{}, fmt.Errorf("failed to instantiate module: %w", err)
	}
	defer instance.Close(ctx)

	// Register host functions (imports)
	if err := registerImports(ctx, cache.GetRuntime(), instance, store, api, *querier, *gasMeter); err != nil {
		return nil, types.GasReport{}, fmt.Errorf("failed to register imports: %w", err)
	}

	// Prepare function arguments
	envPtr, err := writeDataToMemory(instance, envData)
	if err != nil {
		return nil, types.GasReport{}, fmt.Errorf("failed to write env to memory: %w", err)
	}
	infoPtr, err := writeDataToMemory(instance, infoData)
	if err != nil {
		return nil, types.GasReport{}, fmt.Errorf("failed to write info to memory: %w", err)
	}
	msgPtr, err := writeDataToMemory(instance, msgData)
	if err != nil {
		return nil, types.GasReport{}, fmt.Errorf("failed to write msg to memory: %w", err)
	}

	// Call the instantiate function
	initFunc := instance.ExportedFunction("instantiate")
	if initFunc == nil {
		return nil, types.GasReport{}, fmt.Errorf("function 'instantiate' not found in module")
	}

	results, err := initFunc.Call(ctx, uint64(envPtr), uint64(len(envData)), uint64(infoPtr), uint64(len(infoData)), uint64(msgPtr), uint64(len(msgData)))
	if err != nil {
		return nil, types.GasReport{}, fmt.Errorf("failed to call instantiate: %w", err)
	}

	// Retrieve the result
	resPtr := uint32(results[0])
	resLen := uint32(results[1])

	resData, err := readMemory(instance, resPtr, resLen)
	if err != nil {
		return nil, types.GasReport{}, fmt.Errorf("failed to read result from memory: %w", err)
	}

	// Generate gas report
	gasUsed := (*gasMeter).GasConsumed()
	gasReport := types.GasReport{
		UsedExternally: gasUsed,
	}

	return resData, gasReport, nil
}

// writeDataToMemory writes data to the module's memory and returns the pointer to the data.
func writeDataToMemory(mod api.Module, data []byte) (uint32, error) {
	allocFunc := mod.ExportedFunction("allocate")
	if allocFunc == nil {
		return 0, fmt.Errorf("allocate function not found")
	}
	ctx := context.Background()
	results, err := allocFunc.Call(ctx, uint64(len(data)))
	if err != nil || len(results) == 0 {
		return 0, fmt.Errorf("failed to allocate memory: %w", err)
	}
	ptr := uint32(results[0])
	mem := mod.Memory()
	success := mem.Write(ptr, data)
	if !success {
		return 0, fmt.Errorf("failed to write data to memory at address %d", ptr)
	}
	return ptr, nil
}

func readMemory(instance api.Module, offset uint32, length uint32) ([]byte, error) {
	data, ok := instance.Memory().Read(offset, length)
	if !ok {
		return nil, fmt.Errorf("failed to read memory")
	}
	return data, nil
}

func (c *WazeroCache) GetRuntime() wazero.Runtime {
	return c.runtime
}

func (c *WazeroCache) GetCompiledCache() map[string]wazero.CompiledModule {
	return c.compiledCache
}

func (c *WazeroCache) Lock() {
	c.lock.Lock()
}

func (c *WazeroCache) Unlock() {
	c.lock.Unlock()
}

func (c *WazeroCache) RLock() {
	c.lock.RLock()
}

func (c *WazeroCache) RUnlock() {
	c.lock.RUnlock()
}
