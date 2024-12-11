package api

import (
	"crypto/sha256"
	"os"
	"testing"

	"github.com/CosmWasm/wasmd/wasmvm/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withCache(t testing.TB) (Cache, func()) {
	// Set up the cache configuration
	tmpdir, err := os.MkdirTemp("", "wasmvm-testing")
	require.NoError(t, err)
	config := types.VMConfig{
		Cache: types.CacheOptions{
			BaseDir:               tmpdir,
			AvailableCapabilities: []string{"staking", "stargate", "iterator", "cosmwasm_1_1", "cosmwasm_1_2", "cosmwasm_1_3"},
		},
	}

	// Initialize the cache
	cache, err := InitCache(config)
	require.NoError(t, err)

	cleanup := func() {
		cache.Release()
		os.RemoveAll(tmpdir)
	}

	return cache, cleanup
}

func TestInitAndReleaseCache(t *testing.T) {
	cache, cleanup := withCache(t)
	defer cleanup()
}

func TestStoreCodeAndGetCode(t *testing.T) {
	cache, cleanup := withCache(t)
	defer cleanup()

	// Load a WASM file
	wasm, err := os.ReadFile("../../testdata/hackatom.wasm")
	require.NoError(t, err)

	// Store the code
	checksum, err := cache.StoreCode(wasm)
	require.NoError(t, err)

	// Get the code
	retrievedWasm, err := cache.GetCode(checksum)
	require.NoError(t, err)

	// Verify checksum
	expectedChecksum := sha256.Sum256(wasm)
	require.Equal(t, expectedChecksum[:], checksum)

	// Verify the code content
	require.Equal(t, wasm, retrievedWasm)
}

func TestInstantiate(t *testing.T) {
	cache, cleanup := withCache(t)
	defer cleanup()

	// Load and store the contract
	wasm, err := os.ReadFile("../../testdata/hackatom.wasm")
	require.NoError(t, err)
	checksum, err := cache.StoreCode(wasm)
	require.NoError(t, err)

	// Prepare the environment and info
	env, err := os.ReadFile("../../testdata/mock_env.bin")
	require.NoError(t, err)
	info, err := os.ReadFile("../../testdata/mock_info.bin")
	require.NoError(t, err)
	msg := []byte(`{"verifier": "fred", "beneficiary": "bob"}`)

	// Set up the gas meter and other dependencies
	var gasMeter types.GasMeter = types.NewMockGasMeter(5_000_000)
	store := types.NewLookup(gasMeter)
	api := types.NewMockAPI()
	querier := types.DefaultQuerier(types.CanonicalAddress("mock_contract_addr"), nil)

	gasLimit := uint64(5_000_000)

	// Instantiate the contract
	res, gasReport, err := Instantiate(cache, checksum, env, info, msg, &gasMeter, store, api, &querier, gasLimit, false)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotZero(t, gasReport.UsedExternally)

	// Verify result (this depends on your contract's instantiate behavior)
	var result types.ContractResult
	err = types.Deserialize(res, &result)
	require.NoError(t, err)
	require.Empty(t, result.Err)
	require.Equal(t, 0, len(result.Ok.Messages))
}

func TestExecute(t *testing.T) {
	cache, cleanup := withCache(t)
	defer cleanup()

	// Load and store the contract
	wasm, err := os.ReadFile("../../testdata/hackatom.wasm")
	require.NoError(t, err)
	checksum, err := cache.StoreCode(wasm)
	require.NoError(t, err)

	// Prepare the environment and info
	env, err := os.ReadFile("../../testdata/mock_env.bin")
	require.NoError(t, err)
	info, err := os.ReadFile("../../testdata/mock_info.bin")
	require.NoError(t, err)
	msg := []byte(`{"verifier": "fred", "beneficiary": "bob"}`)

	// Set up the gas meter and other dependencies
	var gasMeter types.GasMeter = types.NewMockGasMeter(5_000_000)
	store := types.NewLookup(gasMeter)
	api := types.NewMockAPI()
	querier := types.DefaultQuerier(types.CanonicalAddress("mock_contract_addr"), nil)

	gasLimit := uint64(5_000_000)

	// Instantiate the contract
	_, _, err = Instantiate(cache, checksum, env, info, msg, &gasMeter, store, api, &querier, gasLimit, false)
	require.NoError(t, err)

	// Prepare execute message
	executeMsg := []byte(`{"release":{}}`)
	infoExec, err := os.ReadFile("../../testdata/mock_info_exec.bin")
	require.NoError(t, err)

	// Execute the contract
	res, gasReport, err := Execute(cache, checksum, env, infoExec, executeMsg, &gasMeter, store, api, &querier, gasLimit, false)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotZero(t, gasReport.UsedExternally)

	// Verify result
	var result types.ContractResult
	err = types.Deserialize(res, &result)
	require.NoError(t, err)
	require.Empty(t, result.Err)
	require.Equal(t, 1, len(result.Ok.Messages))
}

func TestPinAndUnpin(t *testing.T) {
	cache, cleanup := withCache(t)
	defer cleanup()

	// Load and store the contract
	wasm, err := os.ReadFile("../../testdata/hackatom.wasm")
	require.NoError(t, err)
	checksum, err := cache.StoreCode(wasm)
	require.NoError(t, err)

	// Pin the code
	err = cache.Pin(checksum)
	require.NoError(t, err)

	// Unpin the code
	err = cache.Unpin(checksum)
	require.NoError(t, err)
}

func TestGetMetrics(t *testing.T) {
	cache, cleanup := withCache(t)
	defer cleanup()

	// Get initial metrics
	metrics, err := cache.GetMetrics()
	require.NoError(t, err)
	assert.NotNil(t, metrics)

	// Load and store the contract
	wasm, err := os.ReadFile("../../testdata/hackatom.wasm")
	require.NoError(t, err)
	_, err = cache.StoreCode(wasm)
	require.NoError(t, err)

	// Get updated metrics
	metricsAfterStore, err := cache.GetMetrics()
	require.NoError(t, err)
	assert.NotNil(t, metricsAfterStore)

	// There should be one element in the cache
	assert.Equal(t, uint64(1), metricsAfterStore.ElementsPinnedMemoryCache)
}

func TestAnalyzeCode(t *testing.T) {
	cache, cleanup := withCache(t)
	defer cleanup()

	// Load and store the contract
	wasm, err := os.ReadFile("../../testdata/hackatom.wasm")
	require.NoError(t, err)
	checksum, err := cache.StoreCode(wasm)
	require.NoError(t, err)

	// Analyze the code
	report, err := AnalyzeCode(cache, checksum)
	require.NoError(t, err)
	require.NotNil(t, report)

	// Verify report values (depends on actual implementation)
	assert.False(t, report.HasIBCEntryPoints)
	assert.Empty(t, report.RequiredCapabilities)
	assert.NotEmpty(t, report.Entrypoints)
}
