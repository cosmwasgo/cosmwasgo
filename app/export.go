package app

import (
	"encoding/json"
	"log"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	storetypes "cosmossdk.io/store/types"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// ExportAppStateAndValidators exports the state of the application for a genesis
// file.
func (app *WasmApp) ExportAppStateAndValidators(forZeroHeight bool, jailAllowedAddrs, modulesToExport []string) (servertypes.ExportedApp, error) {
	// as if they could withdraw from the start of the next block
	ctx := app.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})

	// We export at last height + 1, because that's the height at which
	// CometBFT will start InitChain.
	height := app.LastBlockHeight() + 1
	if forZeroHeight {
		height = 0
		app.prepForZeroHeightGenesis(ctx, jailAllowedAddrs)
	}

	genState, err := app.ModuleManager.ExportGenesisForModules(ctx, app.appCodec, modulesToExport)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	appState, err := json.MarshalIndent(genState, "", "  ")
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	validators, err := staking.WriteValidators(ctx, app.StakingKeeper)
	return servertypes.ExportedApp{
		AppState:        appState,
		Validators:      validators,
		Height:          height,
		ConsensusParams: app.BaseApp.GetConsensusParams(ctx),
	}, err
}

// prepare for fresh start at zero height
// NOTE zero height genesis is a temporary feature which will be deprecated
//
//	in favor of export at a block height
func (app *WasmApp) prepForZeroHeightGenesis(ctx sdk.Context, jailAllowedAddrs []string) {
	allowedAddrsMap := app.buildAllowedAddrsMap(jailAllowedAddrs)
	app.CrisisKeeper.AssertInvariants(ctx)

	height := ctx.BlockHeight()
	ctx = app.handleFeeDistribution(ctx)
	ctx = app.handleStakingState(ctx, allowedAddrsMap)
	ctx = app.handleSlashingState(ctx)
	ctx = ctx.WithBlockHeight(height)
}

func (app *WasmApp) buildAllowedAddrsMap(jailAllowedAddrs []string) map[string]bool {
	allowedAddrsMap := make(map[string]bool)
	for _, addr := range jailAllowedAddrs {
		_, err := sdk.ValAddressFromBech32(addr)
		if err != nil {
			log.Fatal(err)
		}
		allowedAddrsMap[addr] = true
	}
	return allowedAddrsMap
}

func (app *WasmApp) handleFeeDistribution(ctx sdk.Context) sdk.Context {
	// Handle withdrawals and fee distribution state
	app.withdrawValidatorCommissions(ctx)
	app.withdrawDelegatorRewards(ctx)
	app.clearValidatorData(ctx)
	return ctx
}

func (app *WasmApp) handleStakingState(ctx sdk.Context, allowedAddrsMap map[string]bool) sdk.Context {
	// Handle staking state updates
	app.resetRedelegations(ctx)
	app.resetUnbondingDelegations(ctx)
	app.updateValidators(ctx, allowedAddrsMap)
	return ctx
}

func (app *WasmApp) handleSlashingState(ctx sdk.Context) sdk.Context {
	// Reset slashing state
	if err := app.SlashingKeeper.IterateValidatorSigningInfos(
		ctx,
		func(addr sdk.ConsAddress, info slashingtypes.ValidatorSigningInfo) (stop bool) {
			info.StartHeight = 0
			return app.SlashingKeeper.SetValidatorSigningInfo(ctx, addr, info) != nil
		},
	); err != nil {
		panic(err)
	}
	return ctx
}

// withdraw all validator commission
func (app *WasmApp) withdrawValidatorCommissions(ctx sdk.Context) {
	err := app.StakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
		valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
		if err != nil {
			panic(err)
		}
		_, _ = app.DistrKeeper.WithdrawValidatorCommission(ctx, valBz)
		return false
	})
	if err != nil {
		panic(err)
	}
}

// withdraw all delegator rewards
func (app *WasmApp) withdrawDelegatorRewards(ctx sdk.Context) {
	dels, err := app.StakingKeeper.GetAllDelegations(ctx)
	if err != nil {
		panic(err)
	}

	for _, delegation := range dels {
		valAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
		if err != nil {
			panic(err)
		}

		delAddr := sdk.MustAccAddressFromBech32(delegation.DelegatorAddress)

		if _, err = app.DistrKeeper.WithdrawDelegationRewards(ctx, delAddr, valAddr); err != nil {
			panic(err)
		}
	}
}

// clear validator slash events
func (app *WasmApp) clearValidatorData(ctx sdk.Context) {
	app.DistrKeeper.DeleteAllValidatorSlashEvents(ctx)
	app.DistrKeeper.DeleteAllValidatorHistoricalRewards(ctx)
}

// reset redelegations
func (app *WasmApp) resetRedelegations(ctx sdk.Context) {
	var err error
	err = app.StakingKeeper.IterateRedelegations(ctx, func(_ int64, red stakingtypes.Redelegation) (stop bool) {
		for i := range red.Entries {
			red.Entries[i].CreationHeight = 0
		}
		err = app.StakingKeeper.SetRedelegation(ctx, red)
		if err != nil {
			panic(err)
		}
		return false
	})
	if err != nil {
		panic(err)
	}
}

// reset unbonding delegations
func (app *WasmApp) resetUnbondingDelegations(ctx sdk.Context) {
	var err error
	err = app.StakingKeeper.IterateUnbondingDelegations(ctx, func(_ int64, ubd stakingtypes.UnbondingDelegation) (stop bool) {
		for i := range ubd.Entries {
			ubd.Entries[i].CreationHeight = 0
		}
		err = app.StakingKeeper.SetUnbondingDelegation(ctx, ubd)
		if err != nil {
			panic(err)
		}
		return false
	})
	if err != nil {
		panic(err)
	}
}

// iterate through validators by power descending, reset bond heights, and
// update bond intra-tx counters.
func (app *WasmApp) updateValidators(ctx sdk.Context, allowedAddrsMap map[string]bool) {
	store := ctx.KVStore(app.GetKey(stakingtypes.StoreKey))
	iter := storetypes.KVStoreReversePrefixIterator(store, stakingtypes.ValidatorsKey)

	for ; iter.Valid(); iter.Next() {
		addr := sdk.ValAddress(stakingtypes.AddressFromValidatorsKey(iter.Key()))
		validator, err := app.StakingKeeper.GetValidator(ctx, addr)
		if err != nil {
			panic("expected validator, not found")
		}

		validator.UnbondingHeight = 0
		if !allowedAddrsMap[addr.String()] {
			validator.Jailed = true
		}

		err = app.StakingKeeper.SetValidator(ctx, validator)
		if err != nil {
			panic(err)
		}
	}

	if err := iter.Close(); err != nil {
		app.Logger().Error("error while closing the key-value store reverse prefix iterator: ", err)
		return
	}

	var err error
	_, err = app.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
