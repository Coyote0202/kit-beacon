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

package engine

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/log"
	eth "github.com/itsdevbear/bolaris/beacon/execution/engine/ethclient"
	"github.com/itsdevbear/bolaris/config"
	"github.com/itsdevbear/bolaris/third_party/go-ethereum/common"
	enginev1 "github.com/itsdevbear/bolaris/third_party/prysm/proto/engine/v1"
	"github.com/itsdevbear/bolaris/types/consensus/blocks/blocks"
	"github.com/itsdevbear/bolaris/types/consensus/interfaces"
	"github.com/itsdevbear/bolaris/types/consensus/primitives"
	"github.com/itsdevbear/bolaris/types/consensus/version"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/execution"
	payloadattribute "github.com/prysmaticlabs/prysm/v4/consensus-types/payload-attribute"
)

// Caller is implemented by engineCaller.
var _ Caller = (*engineCaller)(nil)

// engineCaller is a struct that holds a pointer to an Eth1Client.
type engineCaller struct {
	*eth.Eth1Client
	engineTimeout time.Duration
	beaconCfg     *config.Beacon
	logger        log.Logger
}

// NewCaller creates a new engine client engineCaller.
// It takes an Eth1Client as an argument and returns a pointer to an engineCaller.
func NewCaller(opts ...Option) Caller {
	ec := &engineCaller{}
	for _, opt := range opts {
		if err := opt(ec); err != nil {
			panic(err)
		}
	}

	return ec
}

// NewPayload calls the engine_newPayloadVX method via JSON-RPC.
func (s *engineCaller) NewPayload(
	ctx context.Context, payload interfaces.ExecutionData,
	versionedHashes []common.Hash, parentBlockRoot *common.Hash,
) ([]byte, error) {
	d := time.Now().Add(s.engineTimeout)
	ctx, cancel := context.WithDeadline(ctx, d)
	defer cancel()
	result := &enginev1.PayloadStatus{}

	switch payload.Proto().(type) {
	case *enginev1.ExecutionPayloadCapella:
		payloadPb, ok := payload.Proto().(*enginev1.ExecutionPayloadCapella)
		if !ok {
			return nil, errors.New("execution data must be a Capella execution payload")
		}
		err := s.Eth1Client.Client.Client().CallContext(ctx, result,
			execution.NewPayloadMethodV2, payloadPb)
		if err != nil {
			return nil, s.handleRPCError(err)
		}
	case *enginev1.ExecutionPayloadDeneb:
		payloadPb, ok := payload.Proto().(*enginev1.ExecutionPayloadDeneb)
		if !ok {
			return nil, errors.New("execution data must be a Deneb execution payload")
		}
		err := s.Eth1Client.Client.Client().CallContext(ctx,
			result, execution.NewPayloadMethodV3, payloadPb, versionedHashes, parentBlockRoot,
		)
		if err != nil {
			return nil, s.handleRPCError(err)
		}
	default:
		return nil, errors.New("unknown execution data type")
	}

	if result.GetValidationError() != "" {
		s.logger.Error("Got a validation error in newPayload", "err",
			errors.New(result.GetValidationError()))
	}

	switch result.GetStatus() {
	case enginev1.PayloadStatus_INVALID_BLOCK_HASH:
		return nil, execution.ErrInvalidBlockHashPayloadStatus
	case enginev1.PayloadStatus_ACCEPTED, enginev1.PayloadStatus_SYNCING:
		return nil, execution.ErrAcceptedSyncingPayloadStatus
	case enginev1.PayloadStatus_INVALID:
		return result.GetLatestValidHash(), execution.ErrInvalidPayloadStatus
	case enginev1.PayloadStatus_VALID:
		return result.GetLatestValidHash(), nil
	case enginev1.PayloadStatus_UNKNOWN:
		return nil, execution.ErrUnknownPayloadStatus
	default:
		return nil, execution.ErrUnknownPayloadStatus
	}
}

// ForkchoiceUpdated calls the engine_forkchoiceUpdatedV1 method via JSON-RPC.
func (s *engineCaller) ForkchoiceUpdated(
	ctx context.Context, state *enginev1.ForkchoiceState, attrs payloadattribute.Attributer,
) (*enginev1.PayloadIDBytes, []byte, error) {
	d := time.Now().Add(s.engineTimeout)
	ctx, cancel := context.WithDeadline(ctx, d)
	defer cancel()
	result := &execution.ForkchoiceUpdatedResponse{}
	if attrs == nil {
		return nil, nil, errors.New("nil payload attributer")
	}
	switch attrs.Version() {
	case version.Deneb:
		a, err := attrs.PbV3()
		if err != nil {
			return nil, nil, err
		}
		err = s.Eth1Client.Client.Client().CallContext(ctx, result,
			execution.ForkchoiceUpdatedMethodV3, state, a)
		if err != nil {
			return nil, nil, s.handleRPCError(err)
		}
	case version.Capella:
		a, err := attrs.PbV2()
		if err != nil {
			return nil, nil, err
		}
		err = s.Eth1Client.Client.Client().CallContext(ctx, result,
			execution.ForkchoiceUpdatedMethodV2, state, a)
		if err != nil {
			return nil, nil, s.handleRPCError(err)
		}
	default:
		return nil, nil, fmt.Errorf("unknown payload attribute version: %v", attrs.Version())
	}

	if result.Status == nil {
		return nil, nil, execution.ErrNilResponse
	}
	if result.ValidationError != "" {
		s.logger.Error("Got validation error in forkChoiceUpdated", "err",
			errors.New(result.ValidationError))
	}
	resp := result.Status
	switch resp.GetStatus() {
	case enginev1.PayloadStatus_ACCEPTED, enginev1.PayloadStatus_SYNCING:
		return nil, nil, execution.ErrAcceptedSyncingPayloadStatus
	case enginev1.PayloadStatus_INVALID:
		return nil, resp.GetLatestValidHash(), execution.ErrInvalidPayloadStatus
	case enginev1.PayloadStatus_VALID:
		return result.PayloadId, resp.GetLatestValidHash(), nil
	case enginev1.PayloadStatus_UNKNOWN:
		return nil, nil, execution.ErrUnknownPayloadStatus
	case enginev1.PayloadStatus_INVALID_BLOCK_HASH:
		return nil, nil, execution.ErrInvalidBlockHashPayloadStatus
	}
	return nil, nil, execution.ErrUnknownPayloadStatus
}

// GetPayload calls the engine_getPayloadVX method via JSON-RPC.
// It returns the execution data as well as the blobs bundle.
func (s *engineCaller) GetPayload(
	ctx context.Context, payloadID [8]byte, slot primitives.Slot,
) (interfaces.ExecutionData, *enginev1.BlobsBundle, bool, error) {
	d := time.Now().Add(s.engineTimeout)
	ctx, cancel := context.WithDeadline(ctx, d)
	defer cancel()
	if primitives.Epoch(slot) >= s.beaconCfg.Forks.DenebForkEpoch {
		result := &enginev1.ExecutionPayloadDenebWithValueAndBlobsBundle{}
		err := s.Eth1Client.Client.Client().CallContext(ctx,
			result, execution.GetPayloadMethodV3, enginev1.PayloadIDBytes(payloadID))
		if err != nil {
			return nil, nil, false, s.handleRPCError(err)
		}
		ed, err := blocks.WrappedExecutionPayloadDeneb(result.GetPayload(),
			blocks.PayloadValueToWei(result.GetValue()))
		if err != nil {
			return nil, nil, false, err
		}
		return ed, result.GetBlobsBundle(), result.GetShouldOverrideBuilder(), nil
	}

	result := &enginev1.ExecutionPayloadCapellaWithValue{}
	err := s.Eth1Client.Client.Client().CallContext(ctx,
		result, execution.GetPayloadMethodV2, enginev1.PayloadIDBytes(payloadID))
	if err != nil {
		return nil, nil, false, s.handleRPCError(err)
	}
	ed, err := blocks.WrappedExecutionPayloadCapella(result.GetPayload(),
		blocks.PayloadValueToWei(result.GetValue()))
	if err != nil {
		return nil, nil, false, err
	}
	return ed, nil, false, nil
}

// ExecutionBlockByHash fetches an execution engine block by hash by calling
// eth_blockByHash via JSON-RPC.
func (s *engineCaller) ExecutionBlockByHash(ctx context.Context, hash common.Hash, withTxs bool,
) (*enginev1.ExecutionBlock, error) {
	result := &enginev1.ExecutionBlock{}
	err := s.Eth1Client.Client.Client().CallContext(
		ctx, result, "eth_getBlockByHash", hash, withTxs)
	return result, s.handleRPCError(err)
}
