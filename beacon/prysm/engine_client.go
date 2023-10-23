// SPDX-License-Identifier: BUSL-1.1
//
// Copyright (C) 2023, Berachain Foundation. All rights reserved.
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

package prysm

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/execution"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/execution/types"
	"github.com/prysmaticlabs/prysm/v4/config/features"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	payloadattribute "github.com/prysmaticlabs/prysm/v4/consensus-types/payload-attribute"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	pb "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	gethRPC "github.com/ethereum/go-ethereum/rpc"
)

const (
	// Defines the seconds before timing out engine endpoints with non-block execution semantics.
	defaultEngineTimeout = time.Second
)

var ErrEmptyBlockHash = errors.New("block hash is empty 0x0000...000")

type Service struct {
	*ethclient.Client
	rpcClient RPCClient
}

func NewEngineClientService(ethclient *ethclient.Client) *Service {
	return &Service{
		Client:    ethclient,
		rpcClient: ethclient.Client(),
	}
}

// NewPayload calls the engine_newPayloadVX method via JSON-RPC.
func (s *Service) NewPayload(ctx context.Context, payload interfaces.ExecutionData,
	versionedHashes []common.Hash, parentBlockRoot *common.Hash) ([]byte, error) {
	// ctx, _ := trace.StartSpan(ctx, "powchain.engine-api-client.NewPayload")
	// defer span.End()
	// start := time.Now()
	// defer func() {
	// 	newPayloadLatency.Observe(float64(time.Since(start).Milliseconds()))
	// }()

	d := time.Now().Add(
		time.Duration(
			params.BeaconConfig().ExecutionEngineTimeoutValue,
		) * time.Second)
	ctx, cancel := context.WithDeadline(ctx, d)
	defer cancel()
	result := &pb.PayloadStatus{}

	switch payload.Proto().(type) {
	case *pb.ExecutionPayload:
		payloadPb, ok := payload.Proto().(*pb.ExecutionPayload)
		if !ok {
			return nil, errors.New("execution data must be a Bellatrix or Capella execution payload")
		}
		err := s.rpcClient.CallContext(ctx, result, execution.NewPayloadMethod, payloadPb)
		if err != nil {
			return nil, handleRPCError(err)
		}
	case *pb.ExecutionPayloadCapella:
		payloadPb, ok := payload.Proto().(*pb.ExecutionPayloadCapella)
		if !ok {
			return nil, errors.New("execution data must be a Capella execution payload")
		}
		err := s.rpcClient.CallContext(ctx, result, execution.NewPayloadMethodV2, payloadPb)
		if err != nil {
			return nil, handleRPCError(err)
		}
	case *pb.ExecutionPayloadDeneb:
		payloadPb, ok := payload.Proto().(*pb.ExecutionPayloadDeneb)
		if !ok {
			return nil, errors.New("execution data must be a Deneb execution payload")
		}
		err := s.rpcClient.CallContext(ctx,
			result, execution.NewPayloadMethodV3, payloadPb, versionedHashes, parentBlockRoot,
		)
		if err != nil {
			return nil, handleRPCError(err)
		}
	default:
		return nil, errors.New("unknown execution data type")
	}
	// if result.ValidationError != "" {
	// 	// log.(errors.New(result.ValidationError)).Error("Got a validation error in newPayload")
	// }
	switch result.Status {
	case pb.PayloadStatus_INVALID_BLOCK_HASH:
		return nil, execution.ErrInvalidBlockHashPayloadStatus
	case pb.PayloadStatus_ACCEPTED, pb.PayloadStatus_SYNCING:
		return nil, execution.ErrAcceptedSyncingPayloadStatus
	case pb.PayloadStatus_INVALID:
		return result.LatestValidHash, execution.ErrInvalidPayloadStatus
	case pb.PayloadStatus_VALID:
		return result.LatestValidHash, nil
	case pb.PayloadStatus_UNKNOWN:
		return nil, execution.ErrUnknownPayloadStatus
	default:
		return nil, execution.ErrUnknownPayloadStatus
	}
}

// ForkchoiceUpdated calls the engine_forkchoiceUpdatedV1 method via JSON-RPC.
func (s *Service) ForkchoiceUpdated(
	ctx context.Context, state *pb.ForkchoiceState, attrs payloadattribute.Attributer,
) (*pb.PayloadIDBytes, []byte, error) {
	// ctx, span := trace.StartSpan(ctx, "powchain.engine-api-client.ForkchoiceUpdated")
	// defer span.End()
	// start := time.Now()
	// // defer func() {
	// // 	forkchoiceUpdatedLatency.Observe(float64(time.Since(start).Milliseconds()))
	// }()

	d := time.Now().Add(time.Duration(
		params.BeaconConfig().ExecutionEngineTimeoutValue) * time.Second)
	ctx, cancel := context.WithDeadline(ctx, d)
	defer cancel()
	result := &execution.ForkchoiceUpdatedResponse{}

	if attrs == nil {
		return nil, nil, errors.New("nil payload attributer")
	}
	switch attrs.Version() {
	case version.Bellatrix:
		a, err := attrs.PbV1()
		if err != nil {
			return nil, nil, err
		}
		err = s.rpcClient.CallContext(ctx, result, execution.ForkchoiceUpdatedMethod, state, a)
		if err != nil {
			return nil, nil, handleRPCError(err)
		}
	case version.Capella:
		a, err := attrs.PbV2()
		if err != nil {
			return nil, nil, err
		}
		err = s.rpcClient.CallContext(ctx, result, execution.ForkchoiceUpdatedMethodV2, state, a)
		if err != nil {
			return nil, nil, handleRPCError(err)
		}
	case version.Deneb:
		a, err := attrs.PbV3()
		if err != nil {
			return nil, nil, err
		}
		err = s.rpcClient.CallContext(ctx, result, execution.ForkchoiceUpdatedMethodV3, state, a)
		if err != nil {
			return nil, nil, handleRPCError(err)
		}
	default:
		return nil, nil, fmt.Errorf("unknown payload attribute version: %d", attrs.Version())
	}

	if result.Status == nil {
		return nil, nil, execution.ErrNilResponse
	}
	// if result.ValidationError != "" {
	// 	// log.WithError(errors.New(result.ValidationError)).Error("Got a validation error in
	// 	//  forkChoiceUpdated")
	// }

	resp := result.Status
	switch resp.Status {
	case pb.PayloadStatus_SYNCING:
		return nil, nil, execution.ErrAcceptedSyncingPayloadStatus
	case pb.PayloadStatus_INVALID:
		return nil, resp.LatestValidHash, execution.ErrInvalidPayloadStatus
	case pb.PayloadStatus_ACCEPTED: // handle something with how this accepted means reorg?
		return result.PayloadId, resp.LatestValidHash, nil
	case pb.PayloadStatus_VALID:
		return result.PayloadId, resp.LatestValidHash, nil
	case pb.PayloadStatus_INVALID_BLOCK_HASH:
		return nil, nil, execution.ErrInvalidBlockHashPayloadStatus
	case pb.PayloadStatus_UNKNOWN:
		return nil, nil, execution.ErrUnknownPayloadStatus
	default:
		return nil, nil, execution.ErrUnknownPayloadStatus
	}
}

// GetPayload calls the engine_getPayloadVX method via JSON-RPC.
// It returns the execution data as well as the blobs bundle.
func (s *Service) GetPayload(ctx context.Context, payloadID [8]byte, slot primitives.Slot) (
	interfaces.ExecutionData, *pb.BlobsBundle, bool, error) {
	// ctx, span := trace.StartSpan(ctx, "powchain.engine-api-client.GetPayload")
	// defer span.End()
	// start := time.Now()
	// defer func() {
	// 	getPayloadLatency.Observe(float64(time.Since(start).Milliseconds()))
	// }()

	_ = slot
	d := time.Now().Add(defaultEngineTimeout)
	ctx, cancel := context.WithDeadline(ctx, d)
	defer cancel()

	// if slots.ToEpoch(slot) >= params.BeaconConfig().DenebForkEpoch {
	// 	result := &pb.ExecutionPayloadDenebWithValueAndBlobsBundle{}
	// 	err := s.rpcClient.CallContext(ctx, result, execution.GetPayloadMethodV3,
	// 		pb.PayloadIDBytes(payloadID))
	// 	if err != nil {
	// 		return nil, nil, false, handleRPCError(err)
	// 	}
	// 	ed, err := blocks.WrappedExecutionPayloadDeneb(result.Payload,
	// 		blocks.PayloadValueToGwei(result.Value))
	// 	if err != nil {
	// 		return nil, nil, false, err
	// 	}
	// 	return ed, result.BlobsBundle, result.ShouldOverrideBuilder, nil
	// }

	// if slots.ToEpoch(slot) >= params.BeaconConfig().CapellaForkEpoch {
	result := &pb.ExecutionPayloadCapellaWithValue{}
	err := s.rpcClient.CallContext(ctx, result, execution.GetPayloadMethodV2,
		pb.PayloadIDBytes(payloadID))
	if err != nil {
		return nil, nil, false, handleRPCError(err)
	}
	ed, err := blocks.WrappedExecutionPayloadCapella(result.Payload,
		blocks.PayloadValueToGwei(result.Value))
	if err != nil {
		return nil, nil, false, err
	}
	// return ed, nil, false, nil
	// }

	// result := &pb.ExecutionPayload{}
	// err := s.rpcClient.CallContext(ctx, result, execution.GetPayloadMethod,
	// 	pb.PayloadIDBytes(payloadID),
	// )
	// if err != nil {
	// 	return nil, nil, false, handleRPCError(err)
	// }
	// ed, err := blocks.WrappedExecutionPayload(result)
	// if err != nil {
	// 	return nil, nil, false, err
	// }
	return ed, nil, false, nil
}

// ExchangeTransitionConfiguration calls the engine_exchangeTransitionConfigurationV1
// method via JSON-RPC.
func (s *Service) ExchangeTransitionConfiguration(
	ctx context.Context, cfg *pb.TransitionConfiguration,
) error {
	ctx, span := trace.StartSpan(ctx, "powchain.engine-api-client.ExchangeTransitionConfiguration")
	defer span.End()

	// We set terminal block number to 0 as the parameter is not set on the consensus layer.
	zeroBigNum := big.NewInt(0)
	cfg.TerminalBlockNumber = zeroBigNum.Bytes()
	d := time.Now().Add(defaultEngineTimeout)
	ctx, cancel := context.WithDeadline(ctx, d)
	defer cancel()
	result := &pb.TransitionConfiguration{}
	if err := s.rpcClient.CallContext(ctx, result, execution.ExchangeTransitionConfigurationMethod,
		cfg); err != nil {
		return handleRPCError(err)
	}

	// We surface an error to the user if local configuration settings mismatch
	// according to the response from the execution node.
	cfgTerminalHash := params.BeaconConfig().TerminalBlockHash[:]
	if !bytes.Equal(cfgTerminalHash, result.TerminalBlockHash) {
		return errors.Wrapf(
			execution.ErrConfigMismatch,
			"got %#x from execution node, wanted %#x",
			result.TerminalBlockHash,
			cfgTerminalHash,
		)
	}
	ttdCfg := params.BeaconConfig().TerminalTotalDifficulty
	ttdResult, err := hexutil.DecodeBig(result.TerminalTotalDifficulty)
	if err != nil {
		return errors.Wrap(err, "could not decode received terminal total difficulty")
	}
	if ttdResult.String() != ttdCfg {
		return errors.Wrapf(
			execution.ErrConfigMismatch,
			"got %s from execution node, wanted %s",
			ttdResult.String(),
			ttdCfg,
		)
	}
	return nil
}

// GetTerminalBlockHash returns the valid terminal block hash based on total difficulty.
//
// Spec code:
// def get_pow_block_at_terminal_total_difficulty(pow_chain: Dict[Hash32, PowBlock])
// -> Optional[PowBlock]:
//
//	# `pow_chain` abstractly represents all blocks in the PoW chain
//	for block in pow_chain:
//	    parent = pow_chain[block.parent_hash]
//	    block_reached_ttd = block.total_difficulty >= TERMINAL_TOTAL_DIFFICULTY
//	    parent_reached_ttd = parent.total_difficulty >= TERMINAL_TOTAL_DIFFICULTY
//	    if block_reached_ttd and not parent_reached_ttd:
//	        return block
//
//	return None
//
//nolint:gocognit // from prysm.
func (s *Service) GetTerminalBlockHash(ctx context.Context, transitionTime uint64,
) ([]byte, bool, error) {
	ttd := new(big.Int)
	ttd.SetString(params.BeaconConfig().TerminalTotalDifficulty, 10) //nolint:gomnd // from prysm.
	terminalTotalDifficulty, overflows := uint256.FromBig(ttd)
	if overflows {
		return nil, false, errors.New("could not convert terminal total difficulty to uint256")
	}
	blk, err := s.LatestExecutionBlock(ctx)
	if err != nil {
		return nil, false, errors.Wrap(err, "could not get latest execution block")
	}
	if blk == nil {
		return nil, false, errors.New("latest execution block is nil")
	}

	for {
		if ctx.Err() != nil {
			return nil, false, ctx.Err()
		}
		var currentTotalDifficulty *uint256.Int
		currentTotalDifficulty, err = tDStringToUint256(blk.TotalDifficulty)
		if err != nil {
			return nil, false, errors.Wrap(err, "could not convert total difficulty to uint256")
		}
		blockReachedTTD := currentTotalDifficulty.Cmp(terminalTotalDifficulty) >= 0

		parentHash := blk.ParentHash
		if parentHash == params.BeaconConfig().ZeroHash {
			return nil, false, nil
		}
		var parentBlk *pb.ExecutionBlock
		parentBlk, err = s.ExecutionBlockByHash(ctx, parentHash, false /* no txs */)
		if err != nil {
			return nil, false, errors.Wrap(err, "could not get parent execution block")
		}
		if parentBlk == nil {
			return nil, false, errors.New("parent execution block is nil")
		}

		//nolint:nestif // from prysm.
		if blockReachedTTD {
			var parentTotalDifficulty *uint256.Int
			parentTotalDifficulty, err = tDStringToUint256(parentBlk.TotalDifficulty)
			if err != nil {
				return nil, false, errors.Wrap(err,
					"could not convert total difficulty to uint256")
			}

			// If terminal block has time same timestamp or greater than transition time,
			// then the node violates the invariant that a block's timestamp must be
			// greater than its parent's timestamp. Execution layer will reject
			// a fcu call with such payload attributes. It's best that we return `None` in this a case.
			parentReachedTTD := parentTotalDifficulty.Cmp(terminalTotalDifficulty) >= 0
			if !parentReachedTTD {
				if blk.Time >= transitionTime {
					return nil, false, nil
				}

				// log.WithFields(logrus.Fields{
				// 	"number":   blk.Number,
				// 	"hash":     fmt.Sprintf("%#x", bytesutil.Trunc(blk.Hash[:])),
				// 	"td":       blk.TotalDifficulty,
				// 	"parentTd": parentBlk.TotalDifficulty,
				// 	"ttd":      terminalTotalDifficulty,
				// }).Info("Retrieved terminal block hash")
				return blk.Hash[:], true, nil
			}
		} else {
			return nil, false, nil
		}
		blk = parentBlk
	}
}

// LatestExecutionBlock fetches the latest execution engine block by calling
// eth_blockByNumber via JSON-RPC.
func (s *Service) LatestExecutionBlock(ctx context.Context) (*pb.ExecutionBlock, error) {
	ctx, span := trace.StartSpan(ctx, "powchain.engine-api-client.LatestExecutionBlock")
	defer span.End()

	result := &pb.ExecutionBlock{}
	err := s.rpcClient.CallContext(
		ctx,
		result,
		execution.ExecutionBlockByNumberMethod,
		"latest",
		false, /* no full transaction objects */
	)
	return result, handleRPCError(err)
}

// ExecutionBlockByHash fetches an execution engine block by hash by calling
// eth_blockByHash via JSON-RPC.
func (s *Service) ExecutionBlockByHash(ctx context.Context, hash common.Hash, withTxs bool,
) (*pb.ExecutionBlock, error) {
	ctx, span := trace.StartSpan(ctx, "powchain.engine-api-client.ExecutionBlockByHash")
	defer span.End()
	result := &pb.ExecutionBlock{}
	err := s.rpcClient.CallContext(ctx, result, execution.ExecutionBlockByHashMethod, hash, withTxs)
	return result, handleRPCError(err)
}

// ExecutionBlocksByHashes fetches a batch of execution engine blocks by hash by calling
// eth_blockByHash via JSON-RPC.
func (s *Service) ExecutionBlocksByHashes(ctx context.Context, hashes []common.Hash, withTxs bool,
) ([]*pb.ExecutionBlock, error) {
	_, span := trace.StartSpan(ctx, "powchain.engine-api-client.ExecutionBlocksByHashes")
	defer span.End()
	numOfHashes := len(hashes)
	elems := make([]gethRPC.BatchElem, 0, numOfHashes)
	execBlks := make([]*pb.ExecutionBlock, 0, numOfHashes)
	if numOfHashes == 0 {
		return execBlks, nil
	}
	for _, h := range hashes {
		blk := &pb.ExecutionBlock{}
		newH := h
		elems = append(elems, gethRPC.BatchElem{
			Method: execution.ExecutionBlockByHashMethod,
			Args:   []interface{}{newH, withTxs},
			Result: blk,
			Error:  error(nil),
		})
		execBlks = append(execBlks, blk)
	}
	ioErr := s.rpcClient.BatchCall(elems)
	if ioErr != nil {
		return nil, ioErr
	}
	for _, e := range elems {
		if e.Error != nil {
			return nil, handleRPCError(e.Error)
		}
	}
	return execBlks, nil
}

// HeaderByHash returns the relevant header details for the provided block hash.
func (s *Service) HeaderByHash(ctx context.Context, hash common.Hash,
) (*types.HeaderInfo, error) {
	var hdr *types.HeaderInfo
	err := s.rpcClient.CallContext(ctx, &hdr,
		execution.ExecutionBlockByHashMethod, hash, false /* no transactions */)
	if err == nil && hdr == nil {
		err = ethereum.NotFound
	}
	return hdr, err
}

// HeaderByNumber returns the relevant header details for the provided block number.
func (s *Service) HeaderByNumber(ctx context.Context, number *big.Int,
) (*types.HeaderInfo, error) {
	var hdr *types.HeaderInfo
	err := s.rpcClient.CallContext(ctx, &hdr,
		execution.ExecutionBlockByNumberMethod, toBlockNumArg(number), false /* no transactions */)
	if err == nil && hdr == nil {
		err = ethereum.NotFound
	}
	return hdr, err
}

// GetPayloadBodiesByHash returns the relevant payload bodies for the provided block hash.
func (s *Service) GetPayloadBodiesByHash(
	ctx context.Context, executionBlockHashes []common.Hash,
) ([]*pb.ExecutionPayloadBodyV1, error) {
	ctx, span := trace.StartSpan(ctx, "powchain.engine-api-client.GetPayloadBodiesByHashV1")
	defer span.End()

	result := make([]*pb.ExecutionPayloadBodyV1, 0)
	err := s.rpcClient.CallContext(ctx, &result,
		execution.GetPayloadBodiesByHashV1, executionBlockHashes)

	for i, item := range result {
		if item == nil {
			result[i] = &pb.ExecutionPayloadBodyV1{
				Transactions: make([][]byte, 0),
				Withdrawals:  make([]*pb.Withdrawal, 0),
			}
		}
	}
	return result, handleRPCError(err)
}

// GetPayloadBodiesByRange returns the relevant payload bodies for the provided range.
func (s *Service) GetPayloadBodiesByRange(
	ctx context.Context, start, count uint64,
) ([]*pb.ExecutionPayloadBodyV1, error) {
	ctx, span := trace.StartSpan(ctx, "powchain.engine-api-client.GetPayloadBodiesByRangeV1")
	defer span.End()

	result := make([]*pb.ExecutionPayloadBodyV1, 0)
	err := s.rpcClient.CallContext(ctx, &result,
		execution.GetPayloadBodiesByRangeV1, start, count)

	for i, item := range result {
		if item == nil {
			result[i] = &pb.ExecutionPayloadBodyV1{
				Transactions: make([][]byte, 0),
				Withdrawals:  make([]*pb.Withdrawal, 0),
			}
		}
	}
	return result, handleRPCError(err)
}

// ReconstructFullBlock takes in a blinded beacon block and reconstructs
// a beacon block with a full execution payload via the engine API.
func (s *Service) ReconstructFullBlock(
	ctx context.Context, blindedBlock interfaces.ReadOnlySignedBeaconBlock,
) (interfaces.SignedBeaconBlock, error) {
	if err := blocks.BeaconBlockIsNil(blindedBlock); err != nil {
		return nil, errors.Wrap(err, "cannot reconstruct bellatrix block from nil data")
	}
	if !blindedBlock.Block().IsBlinded() {
		return nil, errors.New("can only reconstruct block from blinded block format")
	}
	header, err := blindedBlock.Block().Body().Execution()
	if err != nil {
		return nil, err
	}
	if header.IsNil() {
		return nil, errors.New("execution payload header in blinded block was nil")
	}

	// If the payload header has a block hash of 0x0, it means we are pre-merge and should
	// simply return the block with an empty execution payload.
	if bytes.Equal(header.BlockHash(), params.BeaconConfig().ZeroHash[:]) {
		var payload protoreflect.ProtoMessage
		payload, err = buildEmptyExecutionPayload(blindedBlock.Version())
		if err != nil {
			return nil, err
		}
		return blocks.BuildSignedBeaconBlockFromExecutionPayload(blindedBlock, payload)
	}

	executionBlockHash := common.BytesToHash(header.BlockHash())
	payload, err := s.retrievePayloadFromExecutionHash(ctx,
		executionBlockHash, header, blindedBlock.Version())
	if err != nil {
		return nil, err
	}
	fullBlock, err := blocks.BuildSignedBeaconBlockFromExecutionPayload(blindedBlock,
		payload.Proto())
	if err != nil {
		return nil, err
	}
	// reconstructedExecutionPayloadCount.Add(1)
	return fullBlock, nil
}

// ReconstructFullBellatrixBlockBatch takes in a batch of blinded beacon blocks and reconstructs
// them with a full execution payload for each block via the engine API.
func (s *Service) ReconstructFullBellatrixBlockBatch(
	ctx context.Context, blindedBlocks []interfaces.ReadOnlySignedBeaconBlock,
) ([]interfaces.SignedBeaconBlock, error) {
	if len(blindedBlocks) == 0 {
		return []interfaces.SignedBeaconBlock{}, nil
	}
	executionHashes := []common.Hash{}
	validExecPayloads := []int{}
	zeroExecPayloads := []int{}
	for i, b := range blindedBlocks {
		if err := blocks.BeaconBlockIsNil(b); err != nil {
			return nil, errors.Wrap(err, "cannot reconstruct bellatrix block from nil data")
		}
		if !b.Block().IsBlinded() {
			return nil, errors.New("can only reconstruct block from blinded block format")
		}
		header, err := b.Block().Body().Execution()
		if err != nil {
			return nil, err
		}
		if header.IsNil() {
			return nil, errors.New("execution payload header in blinded block was nil")
		}
		// Determine if the block is pre-merge or post-merge. Depending on the result,
		// we will ask the execution engine for the full payload.
		if bytes.Equal(header.BlockHash(), params.BeaconConfig().ZeroHash[:]) {
			zeroExecPayloads = append(zeroExecPayloads, i)
		} else {
			executionBlockHash := common.BytesToHash(header.BlockHash())
			validExecPayloads = append(validExecPayloads, i)
			executionHashes = append(executionHashes, executionBlockHash)
		}
	}
	fullBlocks, err := s.retrievePayloadsFromExecutionHashes(ctx,
		executionHashes, validExecPayloads, blindedBlocks)
	if err != nil {
		return nil, err
	}
	// For blocks that are pre-merge we simply reconstruct them via an empty
	// execution payload.
	for _, realIdx := range zeroExecPayloads {
		bblock := blindedBlocks[realIdx]
		var payload protoreflect.ProtoMessage
		payload, err = buildEmptyExecutionPayload(bblock.Version())
		if err != nil {
			return nil, err
		}
		var fullBlock interfaces.SignedBeaconBlock
		fullBlock, err = blocks.BuildSignedBeaconBlockFromExecutionPayload(
			blindedBlocks[realIdx], payload,
		)
		if err != nil {
			return nil, err
		}
		fullBlocks[realIdx] = fullBlock
	}
	// reconstructedExecutionPayloadCount.Add(float64(len(blindedBlocks)))
	return fullBlocks, nil
}

func (s *Service) retrievePayloadFromExecutionHash(ctx context.Context,
	executionBlockHash common.Hash, header interfaces.ExecutionData,
	version int) (interfaces.ExecutionData, error) {
	if features.Get().EnableOptionalEngineMethods {
		pBodies, err := s.GetPayloadBodiesByHash(ctx, []common.Hash{executionBlockHash})
		if err != nil {
			return nil, fmt.Errorf("could not get payload body by hash %#x: %w", executionBlockHash, err)
		}
		if len(pBodies) != 1 {
			return nil, errors.Errorf(
				"could not retrieve the correct number of payload bodies: wanted 1 but got %d",
				len(pBodies),
			)
		}
		bdy := pBodies[0]
		return fullPayloadFromPayloadBody(header, bdy, version)
	}

	executionBlock, err := s.ExecutionBlockByHash(ctx, executionBlockHash, true /* with txs */)
	if err != nil {
		return nil, fmt.Errorf("could not fetch execution block with txs by hash %#x: %w",
			executionBlockHash, err)
	}
	if executionBlock == nil {
		return nil, fmt.Errorf("received nil execution block for request by hash %#x",
			executionBlockHash)
	}
	if bytes.Equal(executionBlock.Hash.Bytes(), []byte{}) {
		return nil, ErrEmptyBlockHash
	}

	executionBlock.Version = version
	return fullPayloadFromExecutionBlock(version, header, executionBlock)
}

//nolint:gocognit // from prysm.
func (s *Service) retrievePayloadsFromExecutionHashes(
	ctx context.Context,
	executionHashes []common.Hash,
	validExecPayloads []int,
	blindedBlocks []interfaces.ReadOnlySignedBeaconBlock) ([]interfaces.SignedBeaconBlock, error) {
	fullBlocks := make([]interfaces.SignedBeaconBlock, len(blindedBlocks))
	var execBlocks []*pb.ExecutionBlock
	var payloadBodies []*pb.ExecutionPayloadBodyV1
	var err error
	if features.Get().EnableOptionalEngineMethods {
		payloadBodies, err = s.GetPayloadBodiesByHash(ctx, executionHashes)
		if err != nil {
			return nil, fmt.Errorf("could not fetch payload bodies by hash %#x: %w",
				executionHashes, err)
		}
	} else {
		execBlocks, err = s.ExecutionBlocksByHashes(ctx, executionHashes, true /* with txs*/)
		if err != nil {
			return nil, fmt.Errorf("could not fetch execution blocks with txs by hash %#x: %w",
				executionHashes, err)
		}
	}

	// For each valid payload, we reconstruct the full block from it with the
	// blinded block.
	for sliceIdx, realIdx := range validExecPayloads {
		var payload interfaces.ExecutionData
		bblock := blindedBlocks[realIdx]
		//nolint:nestif // from prysm.
		if features.Get().EnableOptionalEngineMethods {
			b := payloadBodies[sliceIdx]
			if b == nil {
				return nil, fmt.Errorf("received nil payload body for request by hash %#x",
					executionHashes[sliceIdx])
			}
			var header interfaces.ExecutionData
			header, err = bblock.Block().Body().Execution()
			if err != nil {
				return nil, err
			}
			payload, err = fullPayloadFromPayloadBody(header, b, bblock.Version())
			if err != nil {
				return nil, err
			}
		} else {
			b := execBlocks[sliceIdx]
			if b == nil {
				return nil, fmt.Errorf("received nil execution block for request by hash %#x",
					executionHashes[sliceIdx])
			}
			var header interfaces.ExecutionData
			header, err = bblock.Block().Body().Execution()
			if err != nil {
				return nil, err
			}
			payload, err = fullPayloadFromExecutionBlock(bblock.Version(), header, b)
			if err != nil {
				return nil, err
			}
		}
		var fullBlock interfaces.SignedBeaconBlock
		fullBlock, err = blocks.BuildSignedBeaconBlockFromExecutionPayload(bblock,
			payload.Proto())
		if err != nil {
			return nil, err
		}
		fullBlocks[realIdx] = fullBlock
	}
	return fullBlocks, nil
}

func fullPayloadFromExecutionBlock(
	blockVersion int, header interfaces.ExecutionData, block *pb.ExecutionBlock,
) (interfaces.ExecutionData, error) {
	if header.IsNil() || block == nil {
		return nil, errors.New("execution block and header cannot be nil")
	}
	blockHash := block.Hash
	if !bytes.Equal(header.BlockHash(), blockHash[:]) {
		return nil, fmt.Errorf(
			"block hash field in execution header %#x does not match execution block hash %#x",
			header.BlockHash(),
			blockHash,
		)
	}
	blockTransactions := block.Transactions
	txs := make([][]byte, len(blockTransactions))
	for i, tx := range blockTransactions {
		txBin, err := tx.MarshalBinary()
		if err != nil {
			return nil, err
		}
		txs[i] = txBin
	}

	switch blockVersion {
	case version.Bellatrix:
		return blocks.WrappedExecutionPayload(&pb.ExecutionPayload{
			ParentHash:    header.ParentHash(),
			FeeRecipient:  header.FeeRecipient(),
			StateRoot:     header.StateRoot(),
			ReceiptsRoot:  header.ReceiptsRoot(),
			LogsBloom:     header.LogsBloom(),
			PrevRandao:    header.PrevRandao(),
			BlockNumber:   header.BlockNumber(),
			GasLimit:      header.GasLimit(),
			GasUsed:       header.GasUsed(),
			Timestamp:     header.Timestamp(),
			ExtraData:     header.ExtraData(),
			BaseFeePerGas: header.BaseFeePerGas(),
			BlockHash:     blockHash[:],
			Transactions:  txs,
		})
	case version.Capella:
		return blocks.WrappedExecutionPayloadCapella(&pb.ExecutionPayloadCapella{
			ParentHash:    header.ParentHash(),
			FeeRecipient:  header.FeeRecipient(),
			StateRoot:     header.StateRoot(),
			ReceiptsRoot:  header.ReceiptsRoot(),
			LogsBloom:     header.LogsBloom(),
			PrevRandao:    header.PrevRandao(),
			BlockNumber:   header.BlockNumber(),
			GasLimit:      header.GasLimit(),
			GasUsed:       header.GasUsed(),
			Timestamp:     header.Timestamp(),
			ExtraData:     header.ExtraData(),
			BaseFeePerGas: header.BaseFeePerGas(),
			BlockHash:     blockHash[:],
			Transactions:  txs,
			Withdrawals:   block.Withdrawals,
		}, 0) // We can't get the block value and don't care about the block value for this instance
	case version.Deneb:
		ebg, err := header.ExcessBlobGas()
		if err != nil {
			return nil, errors.Wrap(err,
				"unable to extract ExcessBlobGas attribute from excution payload header")
		}
		bgu, err := header.BlobGasUsed()
		if err != nil {
			return nil, errors.Wrap(err,
				"unable to extract BlobGasUsed attribute from excution payload header")
		}
		return blocks.WrappedExecutionPayloadDeneb(
			&pb.ExecutionPayloadDeneb{
				ParentHash:    header.ParentHash(),
				FeeRecipient:  header.FeeRecipient(),
				StateRoot:     header.StateRoot(),
				ReceiptsRoot:  header.ReceiptsRoot(),
				LogsBloom:     header.LogsBloom(),
				PrevRandao:    header.PrevRandao(),
				BlockNumber:   header.BlockNumber(),
				GasLimit:      header.GasLimit(),
				GasUsed:       header.GasUsed(),
				Timestamp:     header.Timestamp(),
				ExtraData:     header.ExtraData(),
				BaseFeePerGas: header.BaseFeePerGas(),
				BlockHash:     blockHash[:],
				Transactions:  txs,
				Withdrawals:   block.Withdrawals,
				ExcessBlobGas: ebg,
				BlobGasUsed:   bgu,
			}, 0) // We can't get the block value and don't care about the block value for this instance
	default:
		return nil, fmt.Errorf("unknown execution block version %d", block.Version)
	}
}

func fullPayloadFromPayloadBody(
	header interfaces.ExecutionData, body *pb.ExecutionPayloadBodyV1, bVersion int,
) (interfaces.ExecutionData, error) {
	if header.IsNil() || body == nil {
		return nil, errors.New("execution block and header cannot be nil")
	}

	switch bVersion {
	case version.Bellatrix:
		return blocks.WrappedExecutionPayload(&pb.ExecutionPayload{
			ParentHash:    header.ParentHash(),
			FeeRecipient:  header.FeeRecipient(),
			StateRoot:     header.StateRoot(),
			ReceiptsRoot:  header.ReceiptsRoot(),
			LogsBloom:     header.LogsBloom(),
			PrevRandao:    header.PrevRandao(),
			BlockNumber:   header.BlockNumber(),
			GasLimit:      header.GasLimit(),
			GasUsed:       header.GasUsed(),
			Timestamp:     header.Timestamp(),
			ExtraData:     header.ExtraData(),
			BaseFeePerGas: header.BaseFeePerGas(),
			BlockHash:     header.BlockHash(),
			Transactions:  body.Transactions,
		})
	case version.Capella:
		return blocks.WrappedExecutionPayloadCapella(&pb.ExecutionPayloadCapella{
			ParentHash:    header.ParentHash(),
			FeeRecipient:  header.FeeRecipient(),
			StateRoot:     header.StateRoot(),
			ReceiptsRoot:  header.ReceiptsRoot(),
			LogsBloom:     header.LogsBloom(),
			PrevRandao:    header.PrevRandao(),
			BlockNumber:   header.BlockNumber(),
			GasLimit:      header.GasLimit(),
			GasUsed:       header.GasUsed(),
			Timestamp:     header.Timestamp(),
			ExtraData:     header.ExtraData(),
			BaseFeePerGas: header.BaseFeePerGas(),
			BlockHash:     header.BlockHash(),
			Transactions:  body.Transactions,
			Withdrawals:   body.Withdrawals,
		}, 0) // We can't get the block value and don't care about the
		// block value for this instance
	case version.Deneb:
		ebg, err := header.ExcessBlobGas()
		if err != nil {
			return nil, errors.Wrap(err,
				"unable to extract ExcessBlobGas attribute from excution payload header")
		}
		bgu, err := header.BlobGasUsed()
		if err != nil {
			return nil, errors.Wrap(err,
				"unable to extract BlobGasUsed attribute from excution payload header")
		}
		return blocks.WrappedExecutionPayloadDeneb(
			&pb.ExecutionPayloadDeneb{
				ParentHash:    header.ParentHash(),
				FeeRecipient:  header.FeeRecipient(),
				StateRoot:     header.StateRoot(),
				ReceiptsRoot:  header.ReceiptsRoot(),
				LogsBloom:     header.LogsBloom(),
				PrevRandao:    header.PrevRandao(),
				BlockNumber:   header.BlockNumber(),
				GasLimit:      header.GasLimit(),
				GasUsed:       header.GasUsed(),
				Timestamp:     header.Timestamp(),
				ExtraData:     header.ExtraData(),
				BaseFeePerGas: header.BaseFeePerGas(),
				BlockHash:     header.BlockHash(),
				Transactions:  body.Transactions,
				Withdrawals:   body.Withdrawals,
				ExcessBlobGas: ebg,
				BlobGasUsed:   bgu,
			}, 0) // We can't get the block value and don't care about the
		// block value for this instance
	default:
		return nil, fmt.Errorf("unknown execution block version for payload %d", bVersion)
	}
}

// Handles errors received from the RPC server according to the specification.
func handleRPCError(err error) error {
	if err == nil {
		return nil
	}
	if isTimeout(err) {
		return execution.ErrHTTPTimeout
	}
	e, ok := err.(gethRPC.Error) //nolint:errorlint // from prysm.
	if !ok {
		if strings.Contains(err.Error(), "401 Unauthorized") {
			log.Error("HTTP authentication to your execution client is not working. " +
				"Please ensure you are setting a correct value for the --jwt-secret flag in " +
				"Prysm, or use an IPC connection if on the same machine. Please see our" +
				"documentation for more information on authenticating connections " +
				"here https://docs.prylabs.network/docs/execution-node/authentication")
			return fmt.Errorf("could not authenticate connection to execution client: %w", err)
		}
		return errors.Wrapf(err, "got an unexpected error in JSON-RPC response")
	}
	switch e.ErrorCode() {
	case -32700:
		// errParseCount.Inc()
		return execution.ErrParse
	case -32600:
		// errInvalidRequestCount.Inc()
		return execution.ErrInvalidRequest
	case -32601:
		// errMethodNotFoundCount.Inc()
		return execution.ErrMethodNotFound
	case -32602:
		// errInvalidParamsCount.Inc()
		return execution.ErrInvalidParams
	case -32603:
		// errInternalCount.Inc()
		return execution.ErrInternal
	case -38001:
		// errUnknownPayloadCount.Inc()
		return execution.ErrUnknownPayload
	case -38002:
		// errInvalidForkchoiceStateCount.Inc()
		return execution.ErrInvalidForkchoiceState
	case -38003:
		// errInvalidPayloadAttributesCount.Inc()
		return execution.ErrInvalidPayloadAttributes
	case -38004:
		// errRequestTooLargeCount.Inc()
		return execution.ErrRequestTooLarge
	case -32000:
		// errServerErrorCount.Inc()
		// Only -32000 status codes are data errors in the RPC specification.
		var errWithData gethRPC.DataError
		errWithData, ok = err.(gethRPC.DataError) //nolint:errorlint // from prysm.
		if !ok {
			return errors.Wrapf(err, "got an unexpected error in JSON-RPC response")
		}
		return errors.Wrapf(execution.ErrServer, "%v", errWithData.Error())
	default:
		return err
	}
}

// ErrHTTPTimeout returns true if the error is a http.Client timeout error.
var ErrHTTPTimeout = errors.New("timeout from http.Client")

type httpTimeoutError interface {
	Error() string
	Timeout() bool
}

func isTimeout(e error) bool {
	t, ok := e.(httpTimeoutError) //nolint:errorlint // from prysm.
	return ok && t.Timeout()
}

func tDStringToUint256(td string) (*uint256.Int, error) {
	b, err := hexutil.DecodeBig(td)
	if err != nil {
		return nil, err
	}
	i, overflows := uint256.FromBig(b)
	if overflows {
		return nil, errors.New("total difficulty overflowed")
	}
	return i, nil
}

func buildEmptyExecutionPayload(v int) (proto.Message, error) {
	switch v {
	case version.Bellatrix:
		return &pb.ExecutionPayload{
			ParentHash:    make([]byte, fieldparams.RootLength),
			FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
			StateRoot:     make([]byte, fieldparams.RootLength),
			ReceiptsRoot:  make([]byte, fieldparams.RootLength),
			LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
			PrevRandao:    make([]byte, fieldparams.RootLength),
			BaseFeePerGas: make([]byte, fieldparams.RootLength),
			BlockHash:     make([]byte, fieldparams.RootLength),
			Transactions:  make([][]byte, 0),
			ExtraData:     make([]byte, 0),
		}, nil
	case version.Capella:
		return &pb.ExecutionPayloadCapella{
			ParentHash:    make([]byte, fieldparams.RootLength),
			FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
			StateRoot:     make([]byte, fieldparams.RootLength),
			ReceiptsRoot:  make([]byte, fieldparams.RootLength),
			LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
			PrevRandao:    make([]byte, fieldparams.RootLength),
			BaseFeePerGas: make([]byte, fieldparams.RootLength),
			BlockHash:     make([]byte, fieldparams.RootLength),
			Transactions:  make([][]byte, 0),
			ExtraData:     make([]byte, 0),
			Withdrawals:   make([]*pb.Withdrawal, 0),
		}, nil
	case version.Deneb:
		return &pb.ExecutionPayloadDeneb{
			ParentHash:    make([]byte, fieldparams.RootLength),
			FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
			StateRoot:     make([]byte, fieldparams.RootLength),
			ReceiptsRoot:  make([]byte, fieldparams.RootLength),
			LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
			PrevRandao:    make([]byte, fieldparams.RootLength),
			BaseFeePerGas: make([]byte, fieldparams.RootLength),
			BlockHash:     make([]byte, fieldparams.RootLength),
			Transactions:  make([][]byte, 0),
			ExtraData:     make([]byte, 0),
			Withdrawals:   make([]*pb.Withdrawal, 0),
		}, nil
	default:
		return nil, errors.Wrapf(execution.ErrUnsupportedVersion, "version=%s", version.String(v))
	}
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	pending := big.NewInt(-1)
	if number.Cmp(pending) == 0 {
		return "pending"
	}
	finalized := big.NewInt(int64(gethRPC.FinalizedBlockNumber))
	if number.Cmp(finalized) == 0 {
		return "finalized"
	}
	safe := big.NewInt(int64(gethRPC.SafeBlockNumber))
	if number.Cmp(safe) == 0 {
		return "safe"
	}
	return hexutil.EncodeBig(number)
}
