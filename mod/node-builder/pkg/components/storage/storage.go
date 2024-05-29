// SPDX-License-Identifier: MIT
//
// Copyright (c) 2024 Berachain Foundation
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

package storage

import (
	"context"

	"github.com/berachain/beacon-kit/mod/consensus-types/pkg/types"
	datypes "github.com/berachain/beacon-kit/mod/da/pkg/types"
	engineprimitives "github.com/berachain/beacon-kit/mod/engine-primitives/pkg/engine-primitives"
	"github.com/berachain/beacon-kit/mod/primitives"
	"github.com/berachain/beacon-kit/mod/runtime/pkg/runtime"
	"github.com/berachain/beacon-kit/mod/state-transition/pkg/core"
	"github.com/berachain/beacon-kit/mod/state-transition/pkg/core/state"
	"github.com/berachain/beacon-kit/mod/storage/pkg/beacondb"
	"github.com/berachain/beacon-kit/mod/storage/pkg/deposit"
)

// KVStore is a type alias for the beacon store with
// the generics defined using primitives.
type KVStore = beacondb.KVStore[
	*types.Fork,
	*types.BeaconBlockHeader,
	engineprimitives.ExecutionPayloadHeader,
	*types.Eth1Data,
	*types.Validator,
]

// Backend is a struct that holds the storage backend. It
// provides a simply interface to access all types of storage
// required by the runtime.
type Backend[
	AvailabilityStoreT runtime.AvailabilityStore[
		types.BeaconBlockBody, *datypes.BlobSidecars,
	],
	BeaconStateT core.BeaconState[*types.Validator],
] struct {
	cs                primitives.ChainSpec
	availabilityStore AvailabilityStoreT
	beaconStore       *KVStore
	depositStore      *deposit.KVStore
}

func NewBackend[
	AvailabilityStoreT runtime.AvailabilityStore[
		types.BeaconBlockBody, *datypes.BlobSidecars,
	],
	BeaconStateT core.BeaconState[*types.Validator],
](
	cs primitives.ChainSpec,
	availabilityStore AvailabilityStoreT,
	beaconStore *KVStore,
	depositStore *deposit.KVStore,
) *Backend[AvailabilityStoreT, BeaconStateT] {
	return &Backend[AvailabilityStoreT, BeaconStateT]{
		cs:                cs,
		availabilityStore: availabilityStore,
		beaconStore:       beaconStore,
		depositStore:      depositStore,
	}
}

// AvailabilityStore returns the availability store struct initialized with a.
func (k *Backend[AvailabilityStoreT, BeaconStateT]) AvailabilityStore(
	_ context.Context,
) AvailabilityStoreT {
	return k.availabilityStore
}

// BeaconState returns the beacon state struct initialized with a given
// context and the store key.
func (k *Backend[AvailabilityStoreT, BeaconStateT]) StateFromContext(
	ctx context.Context,
) BeaconStateT {
	return state.NewBeaconStateFromDB[BeaconStateT](
		k.beaconStore.WithContext(ctx),
		k.cs,
	)
}

// BeaconStore returns the beacon store struct.
func (k *Backend[AvailabilityStoreT, BeaconStateT]) BeaconStore() *KVStore {
	return k.beaconStore
}

// DepositStore returns the deposit store struct initialized with a.
func (k *Backend[AvailabilityStoreT, BeaconStateT]) DepositStore(
	_ context.Context,
) *deposit.KVStore {
	return k.depositStore
}
