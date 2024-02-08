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
	"github.com/itsdevbear/bolaris/types/primitives"
)

// TODO: move these? It feels coupled to this x/beacon. But it's okay for now.
// Slot returns the current slot of the beacon chain by converting the block height to a slot.
func (s *BeaconStore) Slot() primitives.Slot {
	return primitives.Slot(s.sdkCtx.BlockHeight())
}

// TODO: move these? It feels coupled to this x/beacon. But it's okay for now.
// Time returns the current time of the beacon chain in Unix timestamp format.
func (s *BeaconStore) Time() uint64 {
	return uint64(s.sdkCtx.BlockTime().Unix()) //#nosec:G701 // won't realistically overflow.
}

// Version returns the active fork version of the beacon chain based on the current slot.
// It utilizes the beacon configuration to determine the active fork version.
func (s *BeaconStore) Version() int {
	// TODO: properly do the SlotsPerEpoch math.
	return s.cfg.ActiveForkVersion(primitives.Epoch(s.Slot()))
}
