package api

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"runtime/debug"

	"github.com/CosmWasm/wasmd/wasmvm/v2/types"
	"github.com/tetratelabs/wazero"
	wazeroapi "github.com/tetratelabs/wazero/api"
)

// Define GoError as an integer type
type GoError int32

const (
	GoError_None            GoError = 0
	GoError_BadArgument     GoError = 1
	GoError_Panic           GoError = 2
	GoError_User            GoError = 3
	GoError_OutOfGas        GoError = 4
	GoError_CannotSerialize GoError = 5
)

// Helper functions to read and write memory from the Wasm instance
func readBytes(mem wazeroapi.Memory, offset uint32, length uint32) ([]byte, error) {
	buf, ok := mem.Read(offset, length)
	if !ok {
		return nil, errors.New("failed to read memory")
	}
	return buf, nil
}

func writeBytes(mem wazeroapi.Memory, offset uint32, data []byte) error {
	ok := mem.Write(offset, data)
	if !ok {
		return errors.New("failed to write memory")
	}
	return nil
}

func writeUint32(mem wazeroapi.Memory, offset uint32, value uint32) error {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, value)
	ok := mem.Write(offset, buf)
	if !ok {
		return fmt.Errorf("failed to write uint32 to memory at offset %d", offset)
	}
	return nil
}

func writeError(mod wazeroapi.Module, errPtrPtr uint32, errMsg string) {
	mem := mod.Memory()
	if errMsg == "" {
		// No error, so write null pointer
		_ = writeUint32(mem, errPtrPtr, 0)
		return
	}

	errBytes := []byte(errMsg)
	errPtr, err := allocateWasmMemory(context.Background(), mod, uint32(len(errBytes)))
	if err != nil {
		log.Printf("failed to allocate memory for error message: %v", err)
		_ = writeUint32(mem, errPtrPtr, 0)
		return
	}

	ok := mem.Write(errPtr, errBytes)
	if !ok {
		log.Printf("failed to write error message to memory")
		_ = writeUint32(mem, errPtrPtr, 0)
		return
	}

	_ = writeUint32(mem, errPtrPtr, errPtr)
}

// allocateWasmMemory allocates memory in the Wasm module using the exported "allocate" function.
func allocateWasmMemory(ctx context.Context, mod wazeroapi.Module, size uint32) (uint32, error) {
	allocFunc := mod.ExportedFunction("allocate")
	if allocFunc == nil {
		return 0, fmt.Errorf("allocate function not found")
	}
	results, err := allocFunc.Call(ctx, uint64(size))
	if err != nil || len(results) == 0 {
		return 0, fmt.Errorf("failed to allocate memory: %w", err)
	}
	ptr := uint32(results[0])
	return ptr, nil
}

// Define your wasmAPI struct to hold necessary context
type wasmAPI struct {
	store    types.KVStore
	api      *types.GoAPI
	querier  types.Querier
	gasMeter types.GasMeter
}

// Implement the recoverPanic function
func recoverPanic(ret *GoError) {
	if rec := recover(); rec != nil {
		// This is used to handle ErrorOutOfGas panics.
		name := reflect.TypeOf(rec).Name()
		switch name {
		case "ErrorOutOfGas":
			*ret = GoError_OutOfGas
		default:
			log.Printf("Panic in Go callback: %#v\n", rec)
			debug.PrintStack()
			*ret = GoError_Panic
		}
	}
}

/****** Host Functions ******/

// Implement the cGet function
func (w *wasmAPI) cGet(ctx context.Context, mod wazeroapi.Module, keyPtr uint32, keyLen uint32, valuePtrPtr uint32, valueLenPtr uint32, errPtrPtr uint32) {
	mem := mod.Memory()

	key, err := readBytes(mem, keyPtr, keyLen)
	if err != nil {
		writeError(mod, errPtrPtr, "failed to read key from memory")
		return
	}

	// Retrieve the value from the store
	value := w.store.Get(key)

	// If value is nil, return error
	if value == nil {
		writeError(mod, errPtrPtr, "key not found")
		return
	}

	// Allocate memory for the value
	valuePtr, err := allocateWasmMemory(ctx, mod, uint32(len(value)))
	if err != nil {
		writeError(mod, errPtrPtr, fmt.Sprintf("failed to allocate memory: %v", err))
		return
	}

	// Write the value to memory
	ok := mem.Write(valuePtr, value)
	if !ok {
		writeError(mod, errPtrPtr, "failed to write value to memory")
		return
	}

	// Write the pointers back to the WASM module
	_ = writeUint32(mem, valuePtrPtr, valuePtr)
	_ = writeUint32(mem, valueLenPtr, uint32(len(value)))
	writeError(mod, errPtrPtr, "")
}

// Implement the cSet function
func (w *wasmAPI) cSet(ctx context.Context, mod wazeroapi.Module, keyPtr uint32, keyLen uint32, valuePtr uint32, valueLen uint32, errPtrPtr uint32) {
	mem := mod.Memory()

	key, err := readBytes(mem, keyPtr, keyLen)
	if err != nil {
		writeError(mod, errPtrPtr, "failed to read key from memory")
		return
	}

	value, err := readBytes(mem, valuePtr, valueLen)
	if err != nil {
		writeError(mod, errPtrPtr, "failed to read value from memory")
		return
	}

	// Set the value in the KVStore
	w.store.Set(key, value)

	// Indicate no error
	writeError(mod, errPtrPtr, "")
}

// cCanonicalizeAddress implements the canonicalize_address function
func (w *wasmAPI) cCanonicalizeAddress(ctx context.Context, mod wazeroapi.Module, humanAddrPtr, humanAddrLen, resultPtr, errPtrPtr uint32) {
	mem := mod.Memory()

	humanAddrBytes, err := readBytes(mem, humanAddrPtr, humanAddrLen)
	if err != nil {
		writeError(mod, errPtrPtr, "failed to read human address from memory")
		return
	}
	humanAddr := string(humanAddrBytes)

	canonAddr, gasUsed, err := w.api.CanonicalizeAddress(humanAddr)
	if err != nil {
		writeError(mod, errPtrPtr, fmt.Sprintf("failed to canonicalize address: %v", err))
		return
	}

	canonPtr, err := allocateWasmMemory(ctx, mod, uint32(len(canonAddr)))
	if err != nil {
		writeError(mod, errPtrPtr, fmt.Sprintf("failed to allocate memory: %v", err))
		return
	}

	ok := mem.Write(canonPtr, canonAddr)
	if !ok {
		writeError(mod, errPtrPtr, "failed to write canonical address to memory")
		return
	}

	_ = writeUint32(mem, resultPtr, canonPtr)
	_ = writeUint32(mem, errPtrPtr, uint32(gasUsed))
	writeError(mod, errPtrPtr, "")
}

// cHumanizeAddress implements the humanize_address function
func (w *wasmAPI) cHumanizeAddress(ctx context.Context, mod wazeroapi.Module, canonAddrPtr, canonAddrLen, resultPtr, errPtrPtr uint32) {
	mem := mod.Memory()

	canonAddr, err := readBytes(mem, canonAddrPtr, canonAddrLen)
	if err != nil {
		writeError(mod, errPtrPtr, "failed to read canonical address from memory")
		return
	}

	humanAddr, gasUsed, err := w.api.HumanizeAddress(canonAddr)
	if err != nil {
		writeError(mod, errPtrPtr, fmt.Sprintf("failed to humanize address: %v", err))
		return
	}

	humanAddrBytes := []byte(humanAddr)

	humanPtr, err := allocateWasmMemory(ctx, mod, uint32(len(humanAddrBytes)))
	if err != nil {
		writeError(mod, errPtrPtr, fmt.Sprintf("failed to allocate memory: %v", err))
		return
	}

	ok := mem.Write(humanPtr, humanAddrBytes)
	if !ok {
		writeError(mod, errPtrPtr, "failed to write human address to memory")
		return
	}

	_ = writeUint32(mem, resultPtr, humanPtr)
	_ = writeUint32(mem, errPtrPtr, uint32(gasUsed))
	writeError(mod, errPtrPtr, "")
}

// cGas implements the gas function
func (w *wasmAPI) cGas(ctx context.Context, mod wazeroapi.Module, gas uint64) {
	w.gasMeter.ConsumeGas(gas, "wasm gas")
}

// cQueryChain implements the query_chain function
func (w *wasmAPI) cQueryChain(ctx context.Context, mod wazeroapi.Module, requestPtr, requestLen, resultPtr, errPtrPtr uint32) {
	mem := mod.Memory()

	requestData, err := readBytes(mem, requestPtr, requestLen)
	if err != nil {
		writeError(mod, errPtrPtr, "failed to read request data from memory")
		return
	}

	gasLimit := w.gasMeter.GasConsumed()
	var queryReq types.QueryRequest
	if err := json.Unmarshal(requestData, &queryReq); err != nil {
		writeError(mod, errPtrPtr, fmt.Sprintf("failed to parse query request: %v", err))
		return
	}
	responseData, err := w.querier.Query(queryReq, gasLimit)
	if err != nil {
		writeError(mod, errPtrPtr, fmt.Sprintf("query error: %v", err))
		return
	}

	var allocErr error
	resultPtr, allocErr = allocateWasmMemory(ctx, mod, uint32(len(responseData)))
	if allocErr != nil {
		writeError(mod, errPtrPtr, fmt.Sprintf("failed to allocate memory: %v", allocErr))
		return
	}

	ok := mem.Write(resultPtr, responseData)
	if !ok {
		writeError(mod, errPtrPtr, "failed to write response data to memory")
		return
	}

	_ = writeUint32(mem, resultPtr, resultPtr)
	_ = writeUint32(mem, errPtrPtr, uint32(len(responseData)))
	writeError(mod, errPtrPtr, "")
}

// cSecp256k1Verify implements the secp256k1_verify function
func (w *wasmAPI) cSecp256k1Verify(ctx context.Context, mod wazeroapi.Module, msgPtr, msgLen, sigPtr, sigLen, pubkeyPtr, pubkeyLen, resultPtr uint32) {
	mem := mod.Memory()

	msg, err := readBytes(mem, msgPtr, msgLen)
	if err != nil {
		writeBool(mem, resultPtr, false)
		return
	}

	sig, err := readBytes(mem, sigPtr, sigLen)
	if err != nil {
		writeBool(mem, resultPtr, false)
		return
	}

	pubkey, err := readBytes(mem, pubkeyPtr, pubkeyLen)
	if err != nil {
		writeBool(mem, resultPtr, false)
		return
	}

	verified, err := w.api.VerifySecp256k1(msg, sig, pubkey)
	if err != nil {
		writeBool(mem, resultPtr, false)
		return
	}

	writeBool(mem, resultPtr, verified)
}

// Helper function to write a boolean value to memory
func writeBool(mem wazeroapi.Memory, offset uint32, value bool) {
	var val uint32
	if value {
		val = 1
	} else {
		val = 0
	}
	_ = writeUint32(mem, offset, val)
}

/****** Register Host Functions ******/

// Function to register imports
func registerImports(
	ctx context.Context,
	runtime wazero.Runtime,
	module wazeroapi.Module,
	store types.KVStore,
	goAPI *types.GoAPI,
	querier types.Querier,
	gasMeter types.GasMeter,
) error {
	// Create an instance of wasmAPI
	wasmAPI := &wasmAPI{
		store:    store,
		api:      goAPI,
		querier:  querier,
		gasMeter: gasMeter,
	}

	// Create env module builder
	envBuilder := runtime.NewHostModuleBuilder("env")

	// Register db_read
	envBuilder.NewFunctionBuilder().
		WithFunc(wasmAPI.cGet).
		Export("db_read")

	// Register db_write
	envBuilder.NewFunctionBuilder().
		WithFunc(wasmAPI.cSet).
		Export("db_write")

	// Register canonicalize_address
	envBuilder.NewFunctionBuilder().
		WithFunc(wasmAPI.cCanonicalizeAddress).
		Export("canonicalize_address")

	// Register humanize_address
	envBuilder.NewFunctionBuilder().
		WithFunc(wasmAPI.cHumanizeAddress).
		Export("humanize_address")

	// Register gas
	envBuilder.NewFunctionBuilder().
		WithFunc(wasmAPI.cGas).
		Export("gas")

	// Register query_chain
	envBuilder.NewFunctionBuilder().
		WithFunc(wasmAPI.cQueryChain).
		Export("query_chain")

	// Register secp256k1_verify
	envBuilder.NewFunctionBuilder().
		WithFunc(wasmAPI.cSecp256k1Verify).
		Export("secp256k1_verify")

	// Similarly, register other necessary functions

	// Instantiate the host module
	_, err := envBuilder.Instantiate(ctx)
	if err != nil {
		return fmt.Errorf("failed to instantiate host module: %w", err)
	}

	return nil
}

/****** Instantiate and Run Wasm Module ******/

// Function to instantiate and run the Wasm module
func instantiateAndRunWasmModule(wasmBytes []byte, store types.KVStore, api *types.GoAPI, querier types.Querier, gasMeter types.GasMeter) error {
	ctx := context.Background()

	// Create the wazero runtime
	runtime := wazero.NewRuntime(ctx)
	defer runtime.Close(ctx)

	// Compile the Wasm module
	compiledModule, err := runtime.CompileModule(ctx, wasmBytes)
	if err != nil {
		return err
	}

	// Instantiate the Wasm module
	mod, err := runtime.InstantiateModule(ctx, compiledModule, wazero.NewModuleConfig())
	if err != nil {
		return err
	}

	// Register imports with the new module
	if err := registerImports(ctx, runtime, mod, store, api, querier, gasMeter); err != nil {
		return err
	}

	// Create DBState
	dbState := &DBState{Store: store}

	// Set context values
	ctx = context.WithValue(ctx, "dbState", dbState)
	ctx = context.WithValue(ctx, "gasMeter", gasMeter)

	// Call the desired export function from the Wasm module
	// For example, "_start" or your custom function
	fn := mod.ExportedFunction("_start")
	if fn == nil {
		return fmt.Errorf("function '_start' not found in module")
	}

	// Call the function
	_, err = fn.Call(ctx)
	if err != nil {
		return err
	}

	return nil
}

// Add DBState type definition
type DBState struct {
	Store types.KVStore
}
