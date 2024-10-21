//go:build cgo && !nolink_libwasmvm

package cosmwasm

import (
	"github.com/CosmWasm/wasmd/wasmvm/v2/internal/api"
)

func libwasmvmVersionImpl() (string, error) {
	return api.LibwasmvmVersion()
}
