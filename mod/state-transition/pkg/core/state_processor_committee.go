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

package core

import (
	"github.com/berachain/beacon-kit/mod/primitives/pkg/transition"
)

func (sp *StateProcessor[
	BeaconBlockT, BeaconBlockBodyT, BeaconBlockHeaderT,
	BeaconStateT, BlobSidecarsT, ContextT,
	DepositT, ExecutionPayloadT, ExecutionPayloadHeaderT,
	ForkT, ForkDataT, ValidatorT, WithdrawalT, WithdrawalCredentialsT,
]) processSyncCommitteeUpdates(
	st BeaconStateT,
) ([]*transition.ValidatorUpdate, error) {
	vals, err := st.GetValidatorsByEffectiveBalance()
	if err != nil {
		return nil, err
	}

	// Create a list of validator updates.
	//
	// TODO: This is a trivial implementation that is to improved upon later.
	updates := make([]*transition.ValidatorUpdate, 0)
	for _, val := range vals {
		updates = append(updates, &transition.ValidatorUpdate{
			Pubkey:           val.GetPubkey(),
			EffectiveBalance: val.GetEffectiveBalance(),
		})
	}

	return updates, nil
}
