// SPDX-License-Identifier: BUSL-1.1
//
// Copyright (C) 2024, Berachain Foundation. All rights reserved.
// Use of this software is governed by the Business Source License included
// in the LICENSE file of this repository and at www.mariadb.com/bsl11.
//
// ANY USE OF THE LICENSED WORK IN VIOLATION OF THIS LICENSE WILL AUTOMATICALLY
// TERMINATE YOUR RIGHTS UNDER THIS LICENSE FOR THE CURRENT AND ALL OTHER
// VERSIONS OF THE LICENSED WORK.
//
// THIS LICENSE DOES NOT GRANT YOU ANY RIGHT IN ANY TRADEMARK OR LOGO OF
// LICENSOR OR ITS AFFILIATES (PROVIDED THAT YOU MAY USE A TRADEMARK OR LOGO OF
// LICENSOR AS EXPRESSLY REQUIRED BY THIS LICENSE).
//
// TO THE EXTENT PERMITTED BY APPLICABLE LAW, THE LICENSED WORK IS PROVIDED ON
// AN “AS IS” BASIS. LICENSOR HEREBY DISCLAIMS ALL WARRANTIES AND CONDITIONS,
// EXPRESS OR IMPLIED, INCLUDING (WITHOUT LIMITATION) WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, NON-INFRINGEMENT, AND
// TITLE.

package components

import (
	"cosmossdk.io/depinject"
	"github.com/berachain/beacon-kit/mod/beacon/blockchain"
	"github.com/berachain/beacon-kit/mod/beacon/validator"
	"github.com/berachain/beacon-kit/mod/consensus-types/pkg/types"
	dastore "github.com/berachain/beacon-kit/mod/da/pkg/store"
	datypes "github.com/berachain/beacon-kit/mod/da/pkg/types"
	"github.com/berachain/beacon-kit/mod/node-core/pkg/components/metrics"
	"github.com/berachain/beacon-kit/mod/primitives"
	"github.com/berachain/beacon-kit/mod/runtime/pkg/runtime/middleware"
	depositdb "github.com/berachain/beacon-kit/mod/storage/pkg/deposit"
)

// ValidatorMiddlewareInput is the input for the validator middleware provider.
type ValidatorMiddlewareInput struct {
	depinject.In
	ChainService *blockchain.Service[
		*dastore.Store[*types.BeaconBlockBody],
		*types.BeaconBlock,
		*types.BeaconBlockBody,
		BeaconState,
		*datypes.BlobSidecars,
		*types.Deposit,
		*depositdb.KVStore[*types.Deposit],
	]
	ChainSpec        primitives.ChainSpec
	StorageBackend   StorageBackend
	TelemetrySink    *metrics.TelemetrySink
	ValidatorService *validator.Service[
		*types.BeaconBlock,
		*types.BeaconBlockBody,
		BeaconState,
		*datypes.BlobSidecars,
		*depositdb.KVStore[*types.Deposit],
		*types.ForkData,
	]
}

// ProvideValidatorMiddleware is a depinject provider for the validator
// middleware.
func ProvideValidatorMiddleware(
	in ValidatorMiddlewareInput,
) *middleware.ValidatorMiddleware[
	*dastore.Store[*types.BeaconBlockBody],
	*types.BeaconBlock,
	*types.BeaconBlockBody,
	BeaconState,
	*datypes.BlobSidecars,
	StorageBackend,
] {
	return middleware.
		NewValidatorMiddleware[*dastore.Store[*types.BeaconBlockBody]](
		in.ChainSpec,
		in.ValidatorService,
		in.ChainService,
		in.TelemetrySink,
		in.StorageBackend,
	)
}

// FinalizeBlockMiddlewareInput is the input for the finalize block middleware.
type FinalizeBlockMiddlewareInput struct {
	depinject.In
	ChainService *blockchain.Service[
		*dastore.Store[*types.BeaconBlockBody],
		*types.BeaconBlock,
		*types.BeaconBlockBody,
		BeaconState,
		*datypes.BlobSidecars,
		*types.Deposit,
		*depositdb.KVStore[*types.Deposit],
	]
	ChainSpec     primitives.ChainSpec
	TelemetrySink *metrics.TelemetrySink
}

// ProvideFinalizeBlockMiddleware is a depinject provider for the finalize block
// middleware.
func ProvideFinalizeBlockMiddleware(
	in FinalizeBlockMiddlewareInput,
) *middleware.FinalizeBlockMiddleware[
	*types.BeaconBlock, BeaconState, *datypes.BlobSidecars,
] {
	return middleware.NewFinalizeBlockMiddleware[
		*types.BeaconBlock, BeaconState, *datypes.BlobSidecars,
	](
		in.ChainSpec,
		in.ChainService,
		in.TelemetrySink,
	)
}
