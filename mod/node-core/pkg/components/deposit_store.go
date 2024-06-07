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

package components

import (
	"cosmossdk.io/depinject"
	storev2 "cosmossdk.io/store/v2/db"
	"github.com/berachain/beacon-kit/mod/consensus-types/pkg/types"
	"github.com/berachain/beacon-kit/mod/storage/pkg/deposit"
	"github.com/cosmos/cosmos-sdk/client/flags"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/spf13/cast"
)

// TrustedSetupInput is the input for the dep inject framework.
type DepositStoreInput struct {
	depinject.In
	AppOpts servertypes.AppOptions
}

// ProvideDepositStore is a function that provides the module to the
// application.
func ProvideDepositStore(
	in DepositStoreInput,
) (*deposit.KVStore[*types.Deposit], error) {
	name := "deposits"
	dir := cast.ToString(in.AppOpts.Get(flags.FlagHome)) + "/data"
	kvp, err := storev2.NewDB(storev2.DBTypePebbleDB, name, dir, nil)
	if err != nil {
		return nil, err
	}

	return deposit.NewStore[*types.Deposit](&deposit.KVStoreProvider{
		KVStoreWithBatch: kvp,
	}), nil
}
