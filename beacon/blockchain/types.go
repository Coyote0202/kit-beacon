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

package blockchain

import (
	"context"

	"github.com/itsdevbear/bolaris/beacon/execution"
	"github.com/itsdevbear/bolaris/third_party/go-ethereum/common"
	enginev1 "github.com/itsdevbear/bolaris/third_party/prysm/proto/engine/v1"
	"github.com/itsdevbear/bolaris/types/consensus/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
)

type ExecutionService interface {
	// NotifyForkchoiceUpdate notifies the execution client of a forkchoice update.
	NotifyForkchoiceUpdate(
		ctx context.Context, fcuConfig *execution.FCUConfig,
	) error

	// NotifyNewPayload notifies the execution client of a new payload.
	NotifyNewPayload(ctx context.Context /*preStateVersion*/, _ int,
		preStateHeader interfaces.ExecutionData, /*, blk interfaces.ReadOnlySignedBeaconBlock*/
	) (bool, error)

	// GetBuiltPayload returns the payload and blobs bundle for the given slot.
	GetBuiltPayload(
		ctx context.Context, slot primitives.Slot, headHash common.Hash,
	) (interfaces.ExecutionData, *enginev1.BlobsBundle, bool, error)
}
