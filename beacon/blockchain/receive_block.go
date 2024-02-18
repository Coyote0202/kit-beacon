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

package blockchain

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	eth "github.com/itsdevbear/bolaris/execution/engine/ethclient"
	"github.com/itsdevbear/bolaris/types/consensus"
	"golang.org/x/sync/errgroup"
)

// ReceiveBeaconBlock receives an incoming beacon block, it first validates
// and then processes the block.
func (s *Service) ReceiveBeaconBlock(
	ctx context.Context,
	blk consensus.ReadOnlyBeaconKitBlock,
) error {
	// If we get any sort of error from the execution client, we bubble
	// it up and reject the proposal, as we do not want to write a block
	// finalization to the consensus layer that is invalid.
	var (
		eg, groupCtx   = errgroup.WithContext(ctx)
		isValidPayload bool
	)

	// TODO: Do we need to wait for the forkchoice to update?
	// TODO: move the error group to use GCD.

	// var postState state.BeaconState
	eg.Go(func() error {
		err := s.validateStateTransition(groupCtx, blk)
		if err != nil {
			s.Logger().Error("failed to validate state transition", "error", err)
			return err
		}
		return nil
	})

	eg.Go(func() error {
		var err error
		if isValidPayload, err = s.validateExecutionOnBlock(
			groupCtx, blk,
		); err != nil {
			s.Logger().Error("failed to notify engine of new payload", "error", err)
			return err
		}
		return nil
	})

	// Wait for the goroutines to finish.
	wgErr := eg.Wait()
	if wgErr != nil {
		// If we receive accepted or syncing status, we can ignore the error. This is required as
		// some execution clients will rely on a forkchoice update to forcibly set head.
		//
		//nolint:lll // hyperlink.
		// https://github.com/ethereum/go-ethereum/blob/95741b18448aaacacd0edd8f73a9364bd3df8c92/eth/catalyst/api.go#L660
		// TODO: we should probably handle this via the sync service and error here properly.
		if !errors.Is(wgErr, eth.ErrAcceptedSyncingPayloadStatus) {
			return wgErr
		}
	}

	// If the block is valid, we can process it.
	if err := s.postBlockProcess(
		ctx, blk, isValidPayload,
	); err != nil {
		return err
	}

	return wgErr
}

// validateStateTransition validates the state transition of a given block.
// TODO: add more rules and modularize, I am unsure if this is the best / correct place for this.
// It's also not very modular, its just hardcoded to single slot finality for now, which is fine,
// but maybe not the most extensible.
func (s *Service) validateStateTransition(
	ctx context.Context, blk consensus.ReadOnlyBeaconKitBlock,
) error {
	executionData, err := blk.ExecutionPayload()
	if err != nil {
		return err
	}

	finalizedHash := s.BeaconState(ctx).GetFinalizedEth1BlockHash()
	if !bytes.Equal(finalizedHash[:], executionData.GetParentHash()) {
		return fmt.Errorf(
			"parent block with hash %x is not finalized, expected finalized hash %x",
			executionData.GetParentHash(), finalizedHash,
		)
	}

	return nil
}

// validateExecutionOnBlock checks the validity of a proposed beacon block.
func (s *Service) validateExecutionOnBlock(
	ctx context.Context, blk consensus.ReadOnlyBeaconKitBlock,
) (bool, error) {
	payload, err := blk.ExecutionPayload()
	if err != nil {
		return false, err
	}

	return s.en.NotifyNewPayload(ctx, payload)
}
