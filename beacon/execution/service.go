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

package execution

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/itsdevbear/bolaris/beacon/execution/engine"
	"github.com/itsdevbear/bolaris/runtime/service"
	"github.com/itsdevbear/bolaris/types/consensus/v1/interfaces"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
)

// Service is responsible for delivering beacon chain notifications to
// the execution client.
type Service struct {
	service.BaseService
	// engine gives the notifier access to the engine api of the execution client.
	engine engine.Caller
	// payloadCache is used to track currently building payload IDs for a given slot.
	payloadCache *cache.PayloadIDCache
}

// New creates a new Service with the provided options.
func New(
	base service.BaseService,
	opts ...Option,
) *Service {
	ec := &Service{
		BaseService: base,
	}
	for _, opt := range opts {
		if err := opt(ec); err != nil {
			ec.Logger().Error("Failed to apply option", "error", err)
		}
	}

	return ec
}

// Start spawns any goroutines required by the service.
func (s *Service) Start() {}

// Stop terminates all goroutines belonging to the service,
// blocking until they are all terminated.
func (s *Service) Stop() error {
	return nil
}

// Status returns error if the service is not considered healthy.
func (s *Service) Status() error {
	if !s.engine.ConnectedETH1() {
		return ErrExecutionClientDisconnected
	}
	return nil
}

// NotifyForkchoiceUpdate notifies the execution client of a forkchoice update.
// TODO: handle the bools better i.e attrs, retry, async.
func (s *Service) NotifyForkchoiceUpdate(
	ctx context.Context, fcuConfig *FCUConfig,
) error {
	var err error

	// Push the forkchoice request to the forkchoice dispatcher, we want to block until
	if e := s.GCD().GetQueue(forkchoiceDispatchQueue).Sync(func() {
		err = s.notifyForkchoiceUpdate(ctx, fcuConfig)
	}); e != nil {
		return e
	}

	return err
}

// GetBuiltPayload returns the payload and blobs bundle for the given slot.
func (s *Service) GetBuiltPayload(
	ctx context.Context, slot primitives.Slot, headHash common.Hash,
) (interfaces.ExecutionData, *enginev1.BlobsBundle, bool, error) {
	payloadID, found := s.payloadCache.PayloadID(
		slot, headHash,
	)
	if !found {
		return nil, nil, false, errors.New("payload not found")
	}

	return s.engine.GetPayload(ctx, payloadID, slot)
}

// NotifyNewPayload notifies the execution client of a new payload.
// It returns true if the EL has returned VALID for the block.
func (s *Service) NotifyNewPayload(ctx context.Context /*preStateVersion*/, _ int,
	preStateHeader interfaces.ExecutionData, /*, blk interfaces.ReadOnlySignedBeaconBlock*/
) (bool, error) {
	// var lastValidHash []byte
	// if blk.Version() >= version.Deneb {
	// 	var versionedHashes []common.Hash
	// 	versionedHashes, err = kzgCommitmentsToVersionedHashes(blk.Block().Body())
	// 	if err != nil {
	// 		return false, errors.Wrap(err, "could not get versioned hashes to feed the engine")
	// 	}
	// 	pr := common.Hash(blk.Block().ParentRoot())
	// 	lastValidHash, err = s.cfg.ExecutionEngineCaller.NewPayload
	//			(ctx, payload, versionedHashes, &pr)
	// } else {
	// 	lastValidHash, err = s.cfg.ExecutionEngineCaller.NewPayload(ctx, payload,
	// []common.Hash{}, &common.Hash{} /*empty version hashes and root before Deneb*/)
	// }

	lastValidHash, err := s.engine.NewPayload(ctx, preStateHeader,
		[]common.Hash{}, &common.Hash{} /* TODO: empty version hashes and root before Deneb*/)
	return lastValidHash != nil, err
}
