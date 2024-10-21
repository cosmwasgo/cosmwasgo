package app

import (
	wasmkeeper "github.com/cosmwasgo/cosmwasgo/x/wasm/keeper"
)

// Deprecated: Use BuiltInCapabilities from github.com/cosmwasgo/cosmwasgo/x/wasm/keeper
func AllCapabilities() []string {
	return wasmkeeper.BuiltInCapabilities()
}
