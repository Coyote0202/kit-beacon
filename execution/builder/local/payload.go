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

package local

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/ethereum/go-ethereum/common"

	"github.com/itsdevbear/bolaris/beacon/execution"
	"github.com/itsdevbear/bolaris/types/consensus/primitives"
	"github.com/itsdevbear/bolaris/types/engine"
	enginev1 "github.com/itsdevbear/bolaris/types/engine/v1"
)

func (s *Service) GetOrBuildLocalPayload(
	ctx context.Context,
	slot primitives.Slot,
) (engine.ExecutionPayload, *enginev1.BlobsBundle, bool, error) {
	// vIdx := blk.ProposerIndex()
	// headRoot := blk.ParentRoot()
	// logFields := logrus.Fields{
	// 	"validatorIndex": vIdx,
	// 	"slot":           slot,
	// 	"headRoot":       fmt.Sprintf("%#x", headRoot),
	// }

	// TODO: Proposer-Builder Seperation Improvements Later.
	// val, tracked := s.TrackedValidatorsCache.Validator(vIdx)
	// if !tracked {
	// 	logrus.WithFields(logFields).Warn("could not find tracked proposer index")
	// }

	parentEth1Hash, err := s.getParentEth1Hash(ctx)
	if err != nil {
		return nil, nil, false, err
	}

	// If we have a payload ID in the cache, we can return the payload from the cache.
	payloadID, ok := s.payloadCache.Get(slot, parentEth1Hash)
	if ok && (payloadID != primitives.PayloadID{}) {
		var (
			pidCpy          primitives.PayloadID
			payload         engine.ExecutionPayload
			overrideBuilder bool
			blobsBundle     *enginev1.BlobsBundle
		)

		// Payload ID is cache hit.
		telemetry.IncrCounter(1, MetricsPayloadIDCacheHit)
		copy(pidCpy[:], payloadID[:])
		if payload, blobsBundle, overrideBuilder, err = s.en.GetPayload(ctx, pidCpy, slot); err == nil {
			// bundleCache.add(slot, bundle)
			// warnIfFeeRecipientDiffers(payload, val.FeeRecipient)
			//  Return the cached payload ID.
			return payload, blobsBundle, overrideBuilder, nil
		}
		s.Logger().Warn("could not get cached payload from execution client", "error", err)
		telemetry.IncrCounter(1, MetricsPayloadIDCacheError)
	}

	// If we reach this point, we have a cache miss and must build a new payload.
	telemetry.IncrCounter(1, MetricsPayloadIDCacheMiss)
	//#nosec:G701 // won't overflow, time cannot be negative.
	return s.BuildLocalPayload(ctx, parentEth1Hash, slot, uint64(time.Now().Unix()))
}

// GetExecutionPayload retrieves the execution payload for the given slot.
func (s *Service) BuildLocalPayload(
	ctx context.Context,
	parentEth1Hash common.Hash,
	slot primitives.Slot,
	timestamp uint64,
) (engine.ExecutionPayload, *enginev1.BlobsBundle, bool, error) {
	var (
		st = s.BeaconState(ctx)
		// TODO: RANDAO
		prevRandao = make([]byte, 32) //nolint:gomnd // TODO: later
		// prevRandao, err := helpers.RandaoMix(st, time.CurrentEpoch(st))
		// TODO: Cancun
		headRoot = make([]byte, 32) //nolint:gomnd // TODO: Cancun
	)

	// Get the expected withdrawals to include in this payload.
	withdrawals, err := st.ExpectedWithdrawals()
	if err != nil {
		s.Logger().Error(
			"Could not get expected withdrawals to get payload attribute", "error", err)
		return nil, nil, false, err
	}

	// Build the payload attributes.
	attrs, err := engine.NewPayloadAttributesContainer(
		st.Version(),
		timestamp,
		prevRandao,
		s.BeaconCfg().Validator.SuggestedFeeRecipient[:],
		withdrawals,
		headRoot,
	)
	if err != nil {
		return nil, nil, false, errors.Wrap(err, "could not create payload attributes")
	}

	fcuConfig := &execution.FCUConfig{
		HeadEth1Hash:  parentEth1Hash,
		ProposingSlot: slot,
		Attributes:    attrs,
	}

	// Notify the execution client of the forkchoice update.
	var payloadID *enginev1.PayloadIDBytes
	payloadID, err = s.en.NotifyForkchoiceUpdate(ctx, fcuConfig)
	if err != nil {
		return nil, nil, false, errors.Wrap(err, "could not prepare payload")
	}

	// If the forkchoice update call has an attribute, update the payload ID cache.
	hasAttr := fcuConfig.Attributes != nil && !fcuConfig.Attributes.IsEmpty()
	if hasAttr && payloadID != nil {
		s.Logger().Info("forkchoice updated with payload attributes for proposal",
			"head_eth1_hash", fcuConfig.HeadEth1Hash,
			"proposing_slot", fcuConfig.ProposingSlot,
			"payload_id", fmt.Sprintf("%#x", payloadID),
		)
		s.payloadCache.Set(
			fcuConfig.ProposingSlot, fcuConfig.HeadEth1Hash, primitives.PayloadID(payloadID[:]))
	} else if hasAttr && payloadID == nil {
		s.Logger().Warn(
			"local block builder received nil payload ID on VALID engine response",
			"head_eth1_hash", fmt.Sprintf("%#x", fcuConfig.HeadEth1Hash),
			"proposing_slot", fcuConfig.ProposingSlot,
		)
		return nil, nil, false, fmt.Errorf("nil payload with block hash: %#x", parentEth1Hash)
	}

	// Get the payload from the execution client.
	payload, blobsBundle, overrideBuilder, err := s.en.GetPayload(
		ctx, primitives.PayloadID(*payloadID), slot,
	)
	if err != nil {
		return nil, nil, false, err
	}

	// TODO: Dencun
	_ = blobsBundle
	// bundleCache.add(slot, bundle)
	// warnIfFeeRecipientDiffers(payload, val.FeeRecipient)

	s.Logger().Debug("received execution payload from local engine", "value", payload.GetValue())
	return payload, blobsBundle, overrideBuilder, nil
}

// getParentEth1Hash retrieves the parent block hash for the given slot.
//
//nolint:unparam // todo: review this later.
func (s *Service) getParentEth1Hash(ctx context.Context) (common.Hash, error) {
	// The first slot should be proposed with the genesis block as parent.
	st := s.BeaconState(ctx)
	if st.Slot() == 1 {
		return st.GenesisEth1Hash(), nil
	}

	// We always want the parent block to be the last finalized block.
	return st.GetFinalizedEth1BlockHash(), nil
}
