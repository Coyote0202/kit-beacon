// SPDX-License-Identifier: BUSL-1.1
//
// Copyright (C) 2024, Berachain Foundation. All rights reserved.
// Use of this software is govered by the Business Source License included
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

package blob

import (
	"time"

	"github.com/berachain/beacon-kit/mod/da/pkg/types"
	"github.com/berachain/beacon-kit/mod/log"
	"github.com/berachain/beacon-kit/mod/primitives"
	"github.com/berachain/beacon-kit/mod/primitives/pkg/math"
)

// Processor is the blob processor that handles the processing and verification
// of blob sidecars.
type Processor[
	AvailabilityStoreT AvailabilityStore[
		BeaconBlockBodyT, *types.BlobSidecars,
	],
	BeaconBlockBodyT any,
] struct {
	// logger is used to log information and errors.
	logger log.Logger[any]
	// chainSpec defines the specifications of the blockchain.
	chainSpec primitives.ChainSpec
	// verifier is responsible for verifying the blobs.
	verifier *Verifier
	// blockBodyOffsetFn is a function that calculates the block body offset
	// based on the slot and chain specifications.
	blockBodyOffsetFn func(math.Slot, primitives.ChainSpec) uint64
	// metrics is used to collect and report processor metrics.
	metrics *processorMetrics
}

// NewProcessor creates a new blob processor.
func NewProcessor[
	AvailabilityStoreT AvailabilityStore[
		BeaconBlockBodyT, *types.BlobSidecars,
	],
	BeaconBlockBodyT any,
](
	logger log.Logger[any],
	chainSpec primitives.ChainSpec,
	verifier *Verifier,
	blockBodyOffsetFn func(math.Slot, primitives.ChainSpec) uint64,
	telemetrySink TelemetrySink,
) *Processor[AvailabilityStoreT, BeaconBlockBodyT] {
	return &Processor[AvailabilityStoreT, BeaconBlockBodyT]{
		logger:            logger,
		chainSpec:         chainSpec,
		verifier:          verifier,
		blockBodyOffsetFn: blockBodyOffsetFn,
		metrics:           newProcessorMetrics(telemetrySink),
	}
}

// VerifyBlobs verifies the blobs and ensures they match the local state.
func (sp *Processor[AvailabilityStoreT, BeaconBlockBodyT]) VerifyBlobs(
	slot math.Slot,
	sidecars *types.BlobSidecars,
) error {
	var (
		numSidecars = math.U64(sidecars.Len())
		startTime   = time.Now()
	)

	defer sp.metrics.measureVerifySidecarsDuration(
		startTime, numSidecars,
	)

	// If there are no blobs to verify, return early.
	if numSidecars == 0 {
		sp.logger.Info(
			"no blob sidecars to verify, skipping verifier 🧢 ",
			"slot",
			slot,
		)
		return nil
	}

	// Otherwise, we run the verification checks on the blobs.
	if err := sp.verifier.VerifyBlobs(
		sidecars,
		sp.blockBodyOffsetFn(slot, sp.chainSpec),
	); err != nil {
		return err
	}

	return nil
}

// ProcessBlobs processes the blobs and ensures they match the local state.
func (sp *Processor[AvailabilityStoreT, BeaconBlockBodyT]) ProcessBlobs(
	slot math.Slot,
	avs AvailabilityStoreT,
	sidecars *types.BlobSidecars,
) error {
	startTime := time.Now()
	defer sp.metrics.measureProcessBlobsDuration(
		startTime, math.U64(sidecars.Len()),
	)
	return avs.Persist(slot, sidecars)
}
