package wasmtesting

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmvmtypes "github.com/CosmWasm/wasmd/wasmvm/v2/types"
)

type MockMsgDispatcher struct {
	DispatchSubmessagesFn func(ctx sdk.Context, contractAddr sdk.AccAddress, ibcPort string, msgs []wasmvmtypes.SubMsg) ([]byte, error)
}

func (m MockMsgDispatcher) DispatchSubmessages(ctx sdk.Context, contractAddr sdk.AccAddress, ibcPort string, msgs []wasmvmtypes.SubMsg) ([]byte, error) {
	if m.DispatchSubmessagesFn == nil {
		panic("not expected to be called")
	}
	return m.DispatchSubmessagesFn(ctx, contractAddr, ibcPort, msgs)
}
