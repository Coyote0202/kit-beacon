// SPDX-License-Identifier: MIT
//
// Copyright (c) 2023 Berachain Foundation
//
// Permission is hereby granted, free of charge, to any person
// obtaining a copy of this software and associated documentation
// files (the "Software"), to deal in the Software without
// restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following
// conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package store

import (
	"context"

	"cosmossdk.io/store"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/itsdevbear/bolaris/config"
)

// BeaconStore is a wrapper around a KVStore sdk.Context
// that provides access to all beacon related data.
type BeaconStore struct {
	store.KVStore

	// sdkCtx is the context of the store.
	sdkCtx sdk.Context

	// cfg is the beacon configuration.
	cfg *config.Beacon

	// lastValidHash is the last valid head in the store.
	// TODO: we need to handle this in a better way.
	lastValidHash common.Hash
}

// NewBeaconStore creates a new instance of BeaconStore.
func NewBeaconStore(
	ctx context.Context,
	storeKey storetypes.StoreKey,
	// TODO: should this be stored in on-chain params?
	cfg *config.Beacon,
) *BeaconStore {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return &BeaconStore{
		sdkCtx:  sdkCtx,
		KVStore: sdkCtx.KVStore(storeKey),
		cfg:     cfg,
	}
}
