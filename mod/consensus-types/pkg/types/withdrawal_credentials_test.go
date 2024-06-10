// SPDX-License-Identifier: BUSL-1.1
//
// Copyright (C) 2024, Berachain Foundation. All rights reserved.
// Use of this software is governed by the Business Source License included
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

package types_test

import (
	"testing"

	"github.com/berachain/beacon-kit/mod/consensus-types/pkg/types"
	"github.com/berachain/beacon-kit/mod/primitives/pkg/common"
	"github.com/stretchr/testify/require"
)

func TestNewCredentialsFromExecutionAddress(t *testing.T) {
	address := common.ExecutionAddress{0xde, 0xad, 0xbe, 0xef}
	expectedCredentials := types.WithdrawalCredentials{}
	expectedCredentials[0] = 0x01 // EthSecp256k1CredentialPrefix
	copy(expectedCredentials[12:], address[:])
	for i := 1; i < 12; i++ {
		expectedCredentials[i] = 0x00
	}
	require.Len(
		t,
		expectedCredentials,
		32,
		"Expected credentials to be 32 bytes long",
	)
	require.Equal(
		t,
		byte(0x01),
		expectedCredentials[0],
		"Expected prefix to be 0x01",
	)
	require.Equal(
		t,
		address,
		common.ExecutionAddress(expectedCredentials[12:]),
		"Expected address to be set correctly",
	)
	credentials := types.
		NewCredentialsFromExecutionAddress(address)
	require.Equal(
		t,
		expectedCredentials,
		credentials,
		"Generated credentials do not match expected",
	)
}

func TestToExecutionAddress(t *testing.T) {
	expectedAddress := common.ExecutionAddress{0xde, 0xad, 0xbe, 0xef}
	credentials := types.WithdrawalCredentials{}
	for i := range credentials {
		// First byte should be 0x01
		switch {
		case i == 0:
			credentials[i] = 0x01 // EthSecp256k1CredentialPrefix
		case i > 0 && i < 12:
			credentials[i] = 0x00 // then we have 11 bytes of padding
		default:
			credentials[i] = expectedAddress[i-12] // then the address
		}
	}

	address, err := credentials.ToExecutionAddress()
	require.NoError(t, err, "Conversion to execution address should not error")
	require.Equal(
		t,
		expectedAddress,
		address,
		"Converted address does not match expected",
	)
}

func TestToExecutionAddress_InvalidPrefix(t *testing.T) {
	credentials := types.WithdrawalCredentials{}
	for i := range credentials {
		credentials[i] = 0x00 // Invalid prefix
	}

	_, err := credentials.ToExecutionAddress()

	require.Error(t, err, "Expected an error due to invalid prefix")
}
